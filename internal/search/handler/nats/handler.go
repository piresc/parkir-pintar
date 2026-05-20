package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"parkir-pintar/internal/search/sync"
	"parkir-pintar/pkg/events"
	pkgnats "parkir-pintar/pkg/nats"
)

type SpotUpdatedEvent = events.SpotUpdatedEvent

type RedisCache interface {
	Delete(ctx context.Context, key string) error
}

const DefaultFloorCount = 5

const natsHandlerTimeout = 15 * time.Second

type Handler struct {
	spotSync   *sync.SpotSync
	redis      RedisCache
	client     *pkgnats.Client
	floorCount int
}

func NewHandler(spotSync *sync.SpotSync, redis RedisCache, client *pkgnats.Client, floorCount int) *Handler {
	if floorCount <= 0 {
		floorCount = DefaultFloorCount
	}
	return &Handler{spotSync: spotSync, redis: redis, client: client, floorCount: floorCount}
}

func (h *Handler) InitConsumers() (jetstream.ConsumeContext, error) {
	return h.client.Consume(pkgnats.ConsumerSearchSpot, h.handleSpotUpdated)
}

func (h *Handler) handleSpotUpdated(msg jetstream.Msg) {
	var event SpotUpdatedEvent
	if err := json.Unmarshal(msg.Data(), &event); err != nil {
		slog.Error("failed to unmarshal spot updated event", slog.String("error", err.Error()))
		// Term permanently stops redelivery for malformed messages that will never succeed
		_ = msg.Term()
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), natsHandlerTimeout)
	defer cancel()

	spotData := sync.SpotData{
		ID:          event.SpotID,
		FloorNumber: event.FloorNumber,
		SpotNumber:  event.SpotNumber,
		VehicleType: event.VehicleType,
		SpotCode:    event.SpotCode,
		Status:      event.Status,
	}
	if err := h.spotSync.HandleSpotUpdated(ctx, spotData); err != nil {
		slog.Error("failed to upsert spot", slog.String("spot_id", event.SpotID), slog.String("error", err.Error()))
		_ = msg.Nak()
		return
	}

	h.invalidateCache(ctx)

	_ = msg.Ack()
	slog.Info("processed spot updated event", slog.String("spot_id", event.SpotID), slog.String("status", event.Status))
}

func (h *Handler) invalidateCache(ctx context.Context) {
	keys := []string{"availability:car", "availability:motorcycle"}
	for floor := 1; floor <= h.floorCount; floor++ {
		keys = append(keys, fmt.Sprintf("floormap:%d", floor))
	}
	for _, key := range keys {
		if err := h.redis.Delete(ctx, key); err != nil {
			slog.Warn("failed to invalidate cache", slog.String("key", key), slog.String("error", err.Error()))
		}
	}
}
