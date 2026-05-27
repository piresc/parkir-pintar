package nats

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"parkir-pintar/internal/analytics"
	"parkir-pintar/internal/analytics/constants"
	"parkir-pintar/internal/analytics/model"
	pkgnats "parkir-pintar/pkg/nats"
)

const natsHandlerTimeout = 15 * time.Second

type SpotUpdatedEvent struct {
	SpotID      string    `json:"spot_id"`
	FloorNumber int       `json:"floor_number"`
	SpotNumber  int       `json:"spot_number"`
	VehicleType string    `json:"vehicle_type"`
	SpotCode    string    `json:"spot_code"`
	Status      string    `json:"status"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Handler struct {
	uc     analytics.Usecase
	client *pkgnats.Client
}

func NewHandler(uc analytics.Usecase, client *pkgnats.Client) *Handler {
	return &Handler{uc: uc, client: client}
}

func (h *Handler) InitConsumers() (jetstream.ConsumeContext, error) {
	return h.client.Consume(constants.ConsumerAnalytics, h.handleReservationEvent)
}

func (h *Handler) InitSpotConsumer() (jetstream.ConsumeContext, error) {
	return h.client.Consume(constants.ConsumerAnalyticsSpot, h.handleSpotUpdated)
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

func (h *Handler) handleSpotUpdated(msg jetstream.Msg) {
	var event SpotUpdatedEvent
	if err := json.Unmarshal(msg.Data(), &event); err != nil {
		slog.Error("failed to unmarshal spot updated event (terminating poison message)",
			slog.String("subject", msg.Subject()),
			slog.String("error", err.Error()))
		_ = msg.Term()
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), natsHandlerTimeout)
	defer cancel()

	spot := model.SpotSnapshot{
		ID:          event.SpotID,
		FloorNumber: event.FloorNumber,
		SpotNumber:  event.SpotNumber,
		VehicleType: event.VehicleType,
		SpotCode:    event.SpotCode,
		Status:      event.Status,
	}
	if err := h.uc.HandleSpotUpdated(ctx, spot); err != nil {
		slog.Error("failed to handle spot updated",
			slog.String("spot_id", event.SpotID),
			slog.String("error", err.Error()))
		_ = msg.Nak()
		return
	}

	_ = msg.Ack()
	slog.Info("processed spot snapshot update",
		slog.String("spot_id", event.SpotID),
		slog.String("status", event.Status))
}
