package nats

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"parkir-pintar/internal/analytics"
	"parkir-pintar/internal/analytics/model"
	"parkir-pintar/internal/events"
	pkgnats "parkir-pintar/pkg/nats"
)

const natsHandlerTimeout = 15 * time.Second

type Handler struct {
	uc     analytics.Usecase
	client *pkgnats.Client
}

func NewHandler(uc analytics.Usecase, client *pkgnats.Client) *Handler {
	return &Handler{uc: uc, client: client}
}

func (h *Handler) InitConsumers() (jetstream.ConsumeContext, error) {
	return h.client.Consume(events.ConsumerAnalytics, h.handleReservationEvent)
}

func (h *Handler) handleReservationEvent(msg jetstream.Msg) {
	var event model.ReservationEvent
	if err := json.Unmarshal(msg.Data(), &event); err != nil {
		slog.Error("failed to unmarshal reservation event (terminating poison message)",
			slog.String("subject", msg.Subject()),
			slog.String("error", err.Error()))
		_ = msg.Term()
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), natsHandlerTimeout)
	defer cancel()

	if err := h.uc.RecordEvent(ctx, event); err != nil {
		slog.Error("failed to record reservation event",
			slog.String("reservation_id", event.ReservationID),
			slog.String("status", event.Status),
			slog.String("error", err.Error()))
		_ = msg.Nak()
		return
	}

	if err := msg.Ack(); err != nil {
		slog.Error("failed to ack reservation event",
			slog.String("reservation_id", event.ReservationID),
			slog.String("error", err.Error()))
	}
	slog.Info("recorded analytics event",
		slog.String("reservation_id", event.ReservationID),
		slog.String("status", event.Status),
		slog.String("spot_id", event.SpotID))
}
