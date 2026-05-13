// Package outbox implements the transactional outbox pattern for guaranteed
// event delivery. Messages are written to the database in the same transaction
// as the business operation, then a background processor publishes them to NATS.
package outbox

import (
	"time"

	"github.com/google/uuid"
)

// Status represents the processing state of an outbox message.
type Status string

const (
	StatusPending   Status = "pending"
	StatusProcessed Status = "processed"
	StatusFailed    Status = "failed"
)

// Message represents a single outbox entry that will be published to NATS.
type Message struct {
	ID            string     `db:"id" json:"id"`
	AggregateType string     `db:"aggregate_type" json:"aggregate_type"`
	AggregateID   string     `db:"aggregate_id" json:"aggregate_id"`
	EventType     string     `db:"event_type" json:"event_type"`
	Payload       []byte     `db:"payload" json:"payload"`
	CreatedAt     time.Time  `db:"created_at" json:"created_at"`
	ProcessedAt   *time.Time `db:"processed_at" json:"processed_at,omitempty"`
	RetryCount    int        `db:"retry_count" json:"retry_count"`
	MaxRetries    int        `db:"max_retries" json:"max_retries"`
	Status        Status     `db:"status" json:"status"`
}

// NewMessage creates a new outbox message with default values.
func NewMessage(aggregateType, aggregateID, eventType string, payload []byte) *Message {
	return &Message{
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
}
