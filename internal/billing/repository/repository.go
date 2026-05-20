// module using sqlx with parameterized queries for SQL injection prevention.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"parkir-pintar/internal/billing/model"
	"parkir-pintar/pkg/database"
)

var ErrNotFound = errors.New("billing record not found")

var ErrConflict = errors.New("conflict: duplicate record")

// ErrConcurrentModification is returned when an optimistic lock fails due to a version mismatch.
var ErrConcurrentModification = errors.New("concurrent modification: version mismatch")

//go:generate mockgen -destination=../mocks/mock_repository.go -package=mocks parkir-pintar/internal/billing/repository Repository
type Repository interface {
	CreateBillingRecord(ctx context.Context, record *model.BillingRecord) error
	GetByReservationID(ctx context.Context, reservationID string) (*model.BillingRecord, error)
	GetByIdempotencyKey(ctx context.Context, key string) (*model.BillingRecord, error)
	UpdateBillingRecord(ctx context.Context, record *model.BillingRecord) error
}

func (r *sqlxRepository) CreateBillingRecord(ctx context.Context, record *model.BillingRecord) error {
	_, err := r.db.NamedExecContext(ctx,
		`INSERT INTO billing_records (id, reservation_id, booking_fee, parking_fee, overnight_fee,
		 total_amount, duration_minutes, billed_hours,
		 is_overnight, idempotency_key, status, version, created_at, updated_at)
		 VALUES (:id, :reservation_id, :booking_fee, :parking_fee, :overnight_fee,
		 :total_amount, :duration_minutes, :billed_hours,
		 :is_overnight, :idempotency_key, :status, :version, :created_at, :updated_at)`,
		record,
	)
	if err != nil {
		if database.IsUniqueViolation(err) {
			return ErrConflict
		}
		return fmt.Errorf("create billing record: %w", err)
	}
	return nil
}

func (r *sqlxRepository) GetByReservationID(ctx context.Context, reservationID string) (*model.BillingRecord, error) {
	var record model.BillingRecord
	err := r.db.GetContext(ctx, &record, "SELECT * FROM billing_records WHERE reservation_id = $1", reservationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: reservation_id=%s", ErrNotFound, reservationID)
		}
		return nil, fmt.Errorf("get billing record by reservation_id: %w", err)
	}
	return &record, nil
}

func (r *sqlxRepository) GetByIdempotencyKey(ctx context.Context, key string) (*model.BillingRecord, error) {
	var record model.BillingRecord
	err := r.db.GetContext(ctx, &record, "SELECT * FROM billing_records WHERE idempotency_key = $1", key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: idempotency_key=%s", ErrNotFound, key)
		}
		return nil, fmt.Errorf("get billing record by idempotency_key: %w", err)
	}
	return &record, nil
}

// It uses optimistic locking via the version column to prevent lost updates.
func (r *sqlxRepository) UpdateBillingRecord(ctx context.Context, record *model.BillingRecord) error {
	currentVersion := record.Version
	record.Version = currentVersion + 1

	result, err := r.db.NamedExecContext(ctx,
		`UPDATE billing_records SET booking_fee = :booking_fee, parking_fee = :parking_fee,
		 overnight_fee = :overnight_fee, total_amount = :total_amount,
		 duration_minutes = :duration_minutes, billed_hours = :billed_hours,
		 is_overnight = :is_overnight, status = :status, version = :version,
		 updated_at = :updated_at
		 WHERE id = :id AND version = `+fmt.Sprintf("%d", currentVersion),
		record,
	)
	if err != nil {
		record.Version = currentVersion // rollback in-memory version on error
		return fmt.Errorf("update billing record: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		record.Version = currentVersion
		return fmt.Errorf("update billing record rows affected: %w", err)
	}
	if rows == 0 {
		record.Version = currentVersion
		return ErrConcurrentModification
	}
	return nil
}
