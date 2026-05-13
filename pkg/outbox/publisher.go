package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Publisher writes events to the outbox table instead of directly publishing
// to NATS. This ensures the event is persisted atomically with the business
// operation when called within the same database transaction.
type Publisher struct {
	repo Repository
}

// NewPublisher creates a new outbox Publisher.
func NewPublisher(repo Repository) *Publisher {
	return &Publisher{repo: repo}
}

// Publish writes an event to the outbox table within the given transaction.
// The event will be picked up by the OutboxProcessor and published to NATS.
func (p *Publisher) Publish(ctx context.Context, tx *sqlx.Tx, aggregateType, aggregateID, eventType string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("outbox publisher marshal payload: %w", err)
	}

	return p.PublishRaw(ctx, tx, aggregateType, aggregateID, eventType, data)
}

// PublishRaw writes a pre-serialized event payload to the outbox table.
func (p *Publisher) PublishRaw(ctx context.Context, tx *sqlx.Tx, aggregateType, aggregateID, eventType string, payload []byte) error {
	msg := &Message{
		ID:            uuid.New().String(),
		AggregateType: aggregateType,
		AggregateID:   aggregateID,
		EventType:     eventType,
		Payload:       payload,
		CreatedAt:     time.Now(),
		RetryCount:    0,
		MaxRetries:    5,
		Status:        StatusPending,
	}

	if err := p.repo.Create(ctx, tx, msg); err != nil {
		return fmt.Errorf("outbox publisher create message: %w", err)
	}

	return nil
}
