package handler

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

// SpotUpdatedEvent is the canonical event from pkg/events.
type SpotUpdatedEvent = events.SpotUpdatedEvent

// RedisCache defines the cache invalidation interface used by NATSHandler.
type RedisCache interface {
	Delete(ctx context.Context, key string) error
}

// DefaultFloorCount is the default number of floors for cache invalidation.
const DefaultFloorCount = 5

// NATSHandler handles NATS messages for the search service.
type NATSHandler struct {
	spotSync   *sync.SpotSync
	redis      RedisCache
	client     *pkgnats.Client
	floorCount int
}

// NewNATSHandler creates a new NATSHandler.
// floorCount sets the number of floors for cache invalidation; if <= 0, DefaultFloorCount is used.
func NewNATSHandler(spotSync *sync.SpotSync, redis RedisCache, client *pkgnats.Client, floorCount int) *NATSHandler {
	if floorCount <= 0 {
		floorCount = DefaultFloorCount
	}
	return &NATSHandler{spotSync: spotSync, redis: redis, client: client, floorCount: floorCount}
}

// InitConsumers starts consuming NATS messages for the search service.
func (h *NATSHandler) InitConsumers() (jetstream.ConsumeContext, error) {
	return h.client.Consume(pkgnats.ConsumerSearchSpot, h.handleSpotUpdated)
}

func (h *NATSHandler) handleSpotUpdated(msg jetstream.Msg) {
	var event SpotUpdatedEvent
	if err := json.Unmarshal(msg.Data(), &event); err != nil {
		slog.Error("failed to unmarshal spot updated event", slog.String("error", err.Error()))
		// Term permanently stops redelivery for malformed messages that will never succeed
		_ = msg.Term()
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Upsert into spot_read_model
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

	// Invalidate cache (best-effort)
	h.invalidateCache(ctx)

	_ = msg.Ack()
	slog.Info("processed spot updated event", slog.String("spot_id", event.SpotID), slog.String("status", event.Status))
}

func (h *NATSHandler) invalidateCache(ctx context.Context) {
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
