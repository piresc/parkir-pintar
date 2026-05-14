package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	pkgnats "parkir-pintar/pkg/nats"
)

// SpotUpdatedEvent is published when a spot's status changes.
type SpotUpdatedEvent struct {
	SpotID      string    `json:"spot_id"`
	FloorNumber int       `json:"floor_number"`
	SpotNumber  int       `json:"spot_number"`
	VehicleType string    `json:"vehicle_type"`
	SpotCode    string    `json:"spot_code"`
	Status      string    `json:"status"` // available, reserved, occupied
	UpdatedAt   time.Time `json:"updated_at"`
}

// ReservationEvent is published for analytics on lifecycle transitions.
type ReservationEvent struct {
	ReservationID string    `json:"reservation_id"`
	DriverID      string    `json:"driver_id"`
	SpotID        string    `json:"spot_id"`
	VehicleType   string    `json:"vehicle_type"`
	Status        string    `json:"status"`
	Timestamp     time.Time `json:"timestamp"`
}

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
	msgID := fmt.Sprintf("spot-%s-%d", event.SpotID, time.Now().UnixNano())
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
