package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"parkir-pintar/internal/reservation/constants"
	pkgnats "parkir-pintar/pkg/nats"
)

// Type aliases for convenience within this package.
type SpotUpdatedEvent = constants.SpotUpdatedEvent
type ReservationEvent = constants.ReservationEvent

// EventPublisher defines the interface for publishing domain events.
type EventPublisher interface {
	PublishSpotUpdated(ctx context.Context, event SpotUpdatedEvent) error
	PublishReservationEvent(ctx context.Context, subject string, event ReservationEvent) error
}

// Publisher implements EventPublisher using NATS JetStream.
type Publisher struct {
	publisher *pkgnats.Publisher
}

func NewPublisher(publisher *pkgnats.Publisher) *Publisher {
	return &Publisher{publisher: publisher}
}

// PublishSpotUpdated publishes a spot status change to the search service.
func (p *Publisher) PublishSpotUpdated(ctx context.Context, event SpotUpdatedEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal spot updated event: %w", err)
	}
	msgID := fmt.Sprintf("spot-%s-%s-%d", event.SpotID, event.Status, time.Now().UnixNano())
	return p.publisher.Publish(ctx, constants.SubjectSearchSpotUpdated, data, msgID)
}

// PublishReservationEvent publishes a reservation lifecycle event to analytics.
func (p *Publisher) PublishReservationEvent(ctx context.Context, subject string, event ReservationEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal reservation event: %w", err)
	}
	msgID := fmt.Sprintf("res-%s-%s-%d", event.ReservationID, event.Status, time.Now().UnixNano())
	return p.publisher.Publish(ctx, subject, data, msgID)
}
