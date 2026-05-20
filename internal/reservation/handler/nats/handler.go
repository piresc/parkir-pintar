// Package nats provides NATS JetStream handlers for the reservation domain module.
package nats

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"parkir-pintar/internal/reservation/constants"
	"parkir-pintar/internal/reservation/model"
	"parkir-pintar/pkg/events"
	"parkir-pintar/pkg/logger"
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

// Handler consumes payment result messages from NATS JetStream and
// delegates to the reservation usecase.
const natsHandlerTimeout = 30 * time.Second

type Handler struct {
	uc     ReservationConfirmer
	client *pkgnats.Client
	cc     jetstream.ConsumeContext
}

// NewHandler creates a new NATS Handler.
func NewHandler(uc ReservationConfirmer, client *pkgnats.Client) *Handler {
	return &Handler{
		uc:     uc,
		client: client,
	}
}

// Start begins consuming messages from the reservation-payment-consumer.
// It returns an error if the consumer cannot be started.
func (h *Handler) Start() error {
	cc, err := h.client.Consume(pkgnats.ConsumerReservationPayment, h.handleMessage)
	if err != nil {
		return err
	}
	h.cc = cc
	return nil
}

// Stop gracefully stops the NATS consumer.
func (h *Handler) Stop() {
	if h.cc != nil {
		h.cc.Stop()
	}
}

func (h *Handler) handleMessage(msg jetstream.Msg) {
	ctx, cancel := context.WithTimeout(context.Background(), natsHandlerTimeout)
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
	case string(constants.PaymentEventSuccess):
		_, err = h.uc.ConfirmReservation(ctx, &model.ConfirmReservationRequest{
			ReservationID: event.ReservationID,
		})
	case string(constants.PaymentEventFailed):
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
