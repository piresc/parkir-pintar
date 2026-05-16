package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"parkir-pintar/pkg/events"
	pkgnats "parkir-pintar/pkg/nats"
)

// PaymentResultEvent is the canonical event from pkg/events.
type PaymentResultEvent = events.PaymentResultEvent

// PaymentEventPublisher publishes payment result events via NATS.
type PaymentEventPublisher struct {
	publisher *pkgnats.Publisher
}

// NewPaymentEventPublisher creates a new PaymentEventPublisher backed by the given NATS publisher.
func NewPaymentEventPublisher(publisher *pkgnats.Publisher) *PaymentEventPublisher {
	return &PaymentEventPublisher{publisher: publisher}
}

// PublishPaymentSuccess publishes a success event for the given payment.
func (p *PaymentEventPublisher) PublishPaymentSuccess(ctx context.Context, event PaymentResultEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal payment success event: %w", err)
	}
	msgID := fmt.Sprintf("pay-success-%s-%d", event.PaymentID, time.Now().UnixNano())
	return p.publisher.Publish(ctx, pkgnats.SubjectPaymentReservationSuccess, data, msgID)
}

// PublishPaymentFailed publishes a failure event for the given payment.
func (p *PaymentEventPublisher) PublishPaymentFailed(ctx context.Context, event PaymentResultEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal payment failed event: %w", err)
	}
	msgID := fmt.Sprintf("pay-failed-%s-%d", event.PaymentID, time.Now().UnixNano())
	return p.publisher.Publish(ctx, pkgnats.SubjectPaymentReservationFailed, data, msgID)
}
