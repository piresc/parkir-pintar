package gateway

import (
	"context"
	"encoding/json"
	"fmt"

	"parkir-pintar/pkg/events"
	pkgnats "parkir-pintar/pkg/nats"
)

type PaymentResultEvent = events.PaymentResultEvent

type PaymentEventPublisher struct {
	publisher *pkgnats.Publisher
}

func NewPaymentEventPublisher(publisher *pkgnats.Publisher) *PaymentEventPublisher {
	return &PaymentEventPublisher{publisher: publisher}
}

func (p *PaymentEventPublisher) PublishPaymentSuccess(ctx context.Context, event PaymentResultEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal payment success event: %w", err)
	}
	msgID := fmt.Sprintf("pay-success-%s", event.PaymentID)
	return p.publisher.Publish(ctx, pkgnats.SubjectPaymentReservationSuccess, data, msgID)
}

func (p *PaymentEventPublisher) PublishPaymentFailed(ctx context.Context, event PaymentResultEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal payment failed event: %w", err)
	}
	msgID := fmt.Sprintf("pay-failed-%s", event.PaymentID)
	return p.publisher.Publish(ctx, pkgnats.SubjectPaymentReservationFailed, data, msgID)
}
