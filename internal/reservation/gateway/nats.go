package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"parkir-pintar/pkg/events"
	pkgnats "parkir-pintar/pkg/nats"
)

// Type aliases for backward compatibility within this package.
type SpotUpdatedEvent = events.SpotUpdatedEvent
type ReservationEvent = events.ReservationEvent

// EventPublisher defines the interface for publishing domain events.
type EventPublisher interface {
	PublishSpotUpdated(ctx context.Context, event SpotUpdatedEvent) error
	PublishReservationEvent(ctx context.Context, subject string, event ReservationEvent) error
}

// NATSEventPublisher implements EventPublisher using NATS JetStream.
type NATSEventPublisher struct {
	publisher *pkgnats.Publisher
}

func NewNATSEventPublisher(publisher *pkgnats.Publisher) *NATSEventPublisher {
	return &NATSEventPublisher{publisher: publisher}
}

// PublishSpotUpdated publishes a spot status change to the search service.
func (p *NATSEventPublisher) PublishSpotUpdated(ctx context.Context, event SpotUpdatedEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal spot updated event: %w", err)
	}
	msgID := fmt.Sprintf("spot-%s-%s-%d", event.SpotID, event.Status, time.Now().UnixNano())
	return p.publisher.Publish(ctx, pkgnats.SubjectReservationSearchSpotUpdated, data, msgID)
}

// PublishReservationEvent publishes a reservation lifecycle event to analytics.
func (p *NATSEventPublisher) PublishReservationEvent(ctx context.Context, subject string, event ReservationEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal reservation event: %w", err)
	}
	msgID := fmt.Sprintf("res-%s-%s-%d", event.ReservationID, event.Status, time.Now().UnixNano())
	return p.publisher.Publish(ctx, subject, data, msgID)
}
