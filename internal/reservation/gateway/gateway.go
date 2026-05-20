package gateway

import (
	"context"

	"parkir-pintar/pkg/events"
)

type SpotUpdatedEvent = events.SpotUpdatedEvent
type ReservationEvent = events.ReservationEvent

//go:generate mockgen -destination=../mocks/mock_event_publisher.go -package=mocks parkir-pintar/internal/reservation/gateway EventPublisher
type EventPublisher interface {
	PublishSpotUpdated(ctx context.Context, event SpotUpdatedEvent) error
	PublishReservationEvent(ctx context.Context, subject string, event ReservationEvent) error
}
