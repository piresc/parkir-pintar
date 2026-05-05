// Package subscriber provides NATS event subscribers for the search domain
// module. It listens to reservation lifecycle events and invalidates the
// Redis cache to keep availability data fresh.
//
// Best practices applied (from Go coding standards KB):
// - Document all exported functions and types with proper Godoc format
// - Handle errors explicitly; never ignore errors
// - Use structured logging for observability
package subscriber

import (
	"context"
	"fmt"
	"log/slog"

	"parkir-pintar/internal/search/usecase"
)

// NATSSubscriber subscribes to reservation events and invalidates Redis cache.
type NATSSubscriber struct {
	redis usecase.RedisClient
}

// NewNATSSubscriber creates a new NATSSubscriber with the given RedisClient.
func NewNATSSubscriber(redis usecase.RedisClient) *NATSSubscriber {
	return &NATSSubscriber{redis: redis}
}

// cacheKeysToInvalidate returns the known cache keys that should be deleted
// when a reservation event occurs.
func cacheKeysToInvalidate() []string {
	keys := []string{
		"availability:car",
		"availability:motorcycle",
	}
	// Invalidate floor map caches for all 5 floors.
	for floor := 1; floor <= 5; floor++ {
		keys = append(keys, fmt.Sprintf("floormap:%d", floor))
	}
	return keys
}

// InvalidateCache deletes all known availability and floor map cache keys.
// Errors are logged but do not propagate — cache invalidation is non-critical.
func (s *NATSSubscriber) InvalidateCache(ctx context.Context) {
	for _, key := range cacheKeysToInvalidate() {
		if err := s.redis.Delete(ctx, key); err != nil {
			slog.Warn("search subscriber: failed to invalidate cache key",
				slog.String("key", key),
				slog.Any("error", err))
		}
	}
}

// HandleReservationEvent is the callback for reservation lifecycle events.
// It invalidates the Redis cache on: reservation.confirmed, reservation.checked_in,
// reservation.checked_out, reservation.expired, reservation.cancelled.
func (s *NATSSubscriber) HandleReservationEvent(ctx context.Context, subject string, _ []byte) {
	slog.Info("search subscriber: received event", slog.String("subject", subject))
	s.InvalidateCache(ctx)
}
