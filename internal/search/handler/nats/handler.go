package nats

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"parkir-pintar/internal/events"
	"parkir-pintar/internal/search"
	pkgnats "parkir-pintar/pkg/nats"
)

type SpotUpdatedEvent = events.SpotUpdatedEvent

const natsHandlerTimeout = 15 * time.Second

type Handler struct {
	uc     search.Usecase
	client *pkgnats.Client
}

func NewHandler(uc search.Usecase, client *pkgnats.Client) *Handler {
	return &Handler{uc: uc, client: client}
}

func (h *Handler) InitConsumers() (jetstream.ConsumeContext, error) {
	return h.client.Consume(events.ConsumerSearchSpot, h.handleSpotUpdated)
}

func (h *Handler) handleSpotUpdated(msg jetstream.Msg) {
	var event SpotUpdatedEvent
	if err := json.Unmarshal(msg.Data(), &event); err != nil {
		slog.Error("failed to unmarshal spot updated event", slog.String("error", err.Error()))
		_ = msg.Term()
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), natsHandlerTimeout)
	defer cancel()

	spot := search.SpotData{
		ID:          event.SpotID,
		FloorNumber: event.FloorNumber,
		SpotNumber:  event.SpotNumber,
		VehicleType: event.VehicleType,
		SpotCode:    event.SpotCode,
		Status:      event.Status,
	}
	if err := h.uc.HandleSpotUpdated(ctx, spot); err != nil {
		slog.Error("failed to handle spot updated", slog.String("spot_id", event.SpotID), slog.String("error", err.Error()))
		_ = msg.Nak()
		return
	}

	_ = msg.Ack()
	slog.Info("processed spot updated event", slog.String("spot_id", event.SpotID), slog.String("status", event.Status))
}
