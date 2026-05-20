// Package handler provides gRPC and NATS handlers for the reservation domain module.
package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"parkir-pintar/pkg/logger"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"parkir-pintar/internal/reservation/model"
	"parkir-pintar/pkg/events"
	pkgnats "parkir-pintar/pkg/nats"
)

// ReservationConfirmer defines the minimal usecase interface needed by the NATS
// handler to react to payment results.
type ReservationConfirmer interface {
	ConfirmReservation(ctx context.Context, req *model.ConfirmReservationRequest) (*model.Reservation, error)
	FailReservation(ctx context.Context, req *model.FailReservationRequest) error
}

// PaymentResultEvent is the canonical event from pkg/events.
type PaymentResultEvent = events.PaymentResultEvent

// NATSHandler consumes payment result messages from NATS JetStream and
// delegates to the reservation usecase.
type NATSHandler struct {
	uc     ReservationConfirmer
	client *pkgnats.Client
	cc     jetstream.ConsumeContext
}

// NewNATSHandler creates a new NATSHandler.
func NewNATSHandler(uc ReservationConfirmer, client *pkgnats.Client) *NATSHandler {
	return &NATSHandler{
		uc:     uc,
		client: client,
	}
}

// Start begins consuming messages from the reservation-payment-consumer.
// It returns an error if the consumer cannot be started.
func (h *NATSHandler) Start() error {
	cc, err := h.client.Consume(pkgnats.ConsumerReservationPayment, h.handleMessage)
	if err != nil {
		return err
	}
	h.cc = cc
	return nil
}

// Stop gracefully stops the NATS consumer.
func (h *NATSHandler) Stop() {
	if h.cc != nil {
		h.cc.Stop()
	}
}

func (h *NATSHandler) handleMessage(msg jetstream.Msg) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var event PaymentResultEvent
	if err := json.Unmarshal(msg.Data(), &event); err != nil {
		slog.Error("failed to unmarshal payment result event",
			logger.Err(err),
			slog.String("subject", msg.Subject()),
		)
		// Nak so the message can be redelivered or sent to dead-letter
		_ = msg.Nak()
		return
	}

	slog.Info("received payment result",
		slog.String("reservation_id", event.ReservationID),
		slog.String("payment_id", event.PaymentID),
		slog.String("status", event.Status),
	)

	var err error
	switch event.Status {
	case "success":
		_, err = h.uc.ConfirmReservation(ctx, &model.ConfirmReservationRequest{
			ReservationID: event.ReservationID,
		})
	case "failed":
		err = h.uc.FailReservation(ctx, &model.FailReservationRequest{
			ReservationID: event.ReservationID,
		})
	default:
		slog.Warn("unknown payment status, acking to discard",
			slog.String("status", event.Status),
			slog.String("reservation_id", event.ReservationID),
		)
		_ = msg.Ack()
		return
	}

	if err != nil {
		slog.Error("failed to process payment result",
			slog.String("reservation_id", event.ReservationID),
			slog.String("status", event.Status),
			logger.Err(err),
		)
		_ = msg.Nak()
		return
	}

	_ = msg.Ack()
}
