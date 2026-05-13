package outbox

import (
	"context"

	"github.com/jmoiron/sqlx"
)

// Repository defines the persistence interface for outbox messages.
type Repository interface {
	// Create inserts a new outbox message. Can be called within an existing
	// transaction (tx) to ensure atomicity with the business operation.
	Create(ctx context.Context, tx *sqlx.Tx, msg *Message) error

	// GetUnprocessed returns pending messages ordered by creation time.
	// The limit parameter controls batch size for the processor.
	GetUnprocessed(ctx context.Context, limit int) ([]*Message, error)

	// MarkProcessed updates a message status to processed with a timestamp.
	MarkProcessed(ctx context.Context, id string) error

	// MarkFailed increments the retry count and marks the message as failed
	// if max retries have been exceeded.
	MarkFailed(ctx context.Context, id string) error
}

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	db *sqlx.DB
}

// NewPostgresRepository creates a new PostgresRepository.
func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// Create inserts a new outbox message within the given transaction.
func (r *PostgresRepository) Create(ctx context.Context, tx *sqlx.Tx, msg *Message) error {
	query := `
		INSERT INTO outbox_messages (id, aggregate_type, aggregate_id, event_type, payload, created_at, retry_count, max_retries, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := tx.ExecContext(ctx, query,
		msg.ID,
		msg.AggregateType,
		msg.AggregateID,
		msg.EventType,
		msg.Payload,
		msg.CreatedAt,
		msg.RetryCount,
		msg.MaxRetries,
		msg.Status,
	)
	return err
}

// GetUnprocessed retrieves pending messages ordered by creation time.
func (r *PostgresRepository) GetUnprocessed(ctx context.Context, limit int) ([]*Message, error) {
	var messages []*Message
	query := `
		SELECT id, aggregate_type, aggregate_id, event_type, payload, created_at, processed_at, retry_count, max_retries, status
		FROM outbox_messages
		WHERE status = $1
		ORDER BY created_at ASC
		LIMIT $2`

	err := r.db.SelectContext(ctx, &messages, query, StatusPending, limit)
	if err != nil {
		return nil, err
	}
	return messages, nil
}

// MarkProcessed updates the message status to processed.
func (r *PostgresRepository) MarkProcessed(ctx context.Context, id string) error {
	query := `
		UPDATE outbox_messages
		SET status = $1, processed_at = NOW()
		WHERE id = $2`

	_, err := r.db.ExecContext(ctx, query, StatusProcessed, id)
	return err
}

// MarkFailed increments retry_count and sets status to failed if max retries exceeded.
func (r *PostgresRepository) MarkFailed(ctx context.Context, id string) error {
	query := `
		UPDATE outbox_messages
		SET retry_count = retry_count + 1,
		    status = CASE WHEN retry_count + 1 >= max_retries THEN 'failed' ELSE 'pending' END
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id)
	return err
}
