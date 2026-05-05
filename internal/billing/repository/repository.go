// Package repository provides the data access layer for the billing domain
// module using sqlx with parameterized queries for SQL injection prevention.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"parkir-pintar/internal/billing/model"
)

// ErrNotFound is returned when a billing record or penalty is not found.
var ErrNotFound = errors.New("billing record not found")

// Repository defines the data access interface for billing records and penalties.
//
//go:generate mockgen -destination=../mocks/mock_repository.go -package=mocks parkir-pintar/internal/billing/repository Repository
type Repository interface {
	CreateBillingRecord(ctx context.Context, record *model.BillingRecord) error
	GetByReservationID(ctx context.Context, reservationID string) (*model.BillingRecord, error)
	GetByIdempotencyKey(ctx context.Context, key string) (*model.BillingRecord, error)
	UpdateBillingRecord(ctx context.Context, record *model.BillingRecord) error
	CreatePenalty(ctx context.Context, penalty *model.Penalty) error
	AddPenaltyAmount(ctx context.Context, reservationID string, amount int64) (*model.BillingRecord, error)
}

// sqlxRepository is the sqlx-backed implementation of Repository.
type sqlxRepository struct {
	db *sqlx.DB
}

// NewRepository creates a new Repository backed by the given sqlx.DB.
func NewRepository(db *sqlx.DB) Repository {
	return &sqlxRepository{db: db}
}

// CreateBillingRecord inserts a new billing record.
func (r *sqlxRepository) CreateBillingRecord(ctx context.Context, record *model.BillingRecord) error {
	_, err := r.db.NamedExecContext(ctx,
		`INSERT INTO billing_records (id, reservation_id, booking_fee, parking_fee, overnight_fee,
		 cancellation_fee, penalty_amount, total_amount, duration_minutes, billed_hours,
		 is_overnight, idempotency_key, status, created_at, updated_at)
		 VALUES (:id, :reservation_id, :booking_fee, :parking_fee, :overnight_fee,
		 :cancellation_fee, :penalty_amount, :total_amount, :duration_minutes, :billed_hours,
		 :is_overnight, :idempotency_key, :status, :created_at, :updated_at)`,
		record,
	)
	if err != nil {
		return fmt.Errorf("create billing record: %w", err)
	}
	return nil
}

// GetByReservationID retrieves a billing record by its reservation ID.
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

// GetByIdempotencyKey retrieves a billing record by its idempotency key.
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

// UpdateBillingRecord updates an existing billing record's mutable fields.
func (r *sqlxRepository) UpdateBillingRecord(ctx context.Context, record *model.BillingRecord) error {
	_, err := r.db.NamedExecContext(ctx,
		`UPDATE billing_records SET booking_fee = :booking_fee, parking_fee = :parking_fee,
		 overnight_fee = :overnight_fee, cancellation_fee = :cancellation_fee,
		 penalty_amount = :penalty_amount, total_amount = :total_amount,
		 duration_minutes = :duration_minutes, billed_hours = :billed_hours,
		 is_overnight = :is_overnight, status = :status, updated_at = :updated_at
		 WHERE id = :id`,
		record,
	)
	if err != nil {
		return fmt.Errorf("update billing record: %w", err)
	}
	return nil
}

// AddPenaltyAmount atomically increments penalty_amount and recalculates total_amount
// in a single SQL UPDATE, returning the updated record. This prevents lost updates
// under concurrent ApplyPenalty calls for the same reservation.
func (r *sqlxRepository) AddPenaltyAmount(ctx context.Context, reservationID string, amount int64) (*model.BillingRecord, error) {
	var record model.BillingRecord
	err := r.db.GetContext(ctx, &record,
		`UPDATE billing_records
		 SET penalty_amount = penalty_amount + $1,
		     total_amount = booking_fee + parking_fee + overnight_fee + cancellation_fee + (penalty_amount + $1),
		     updated_at = NOW()
		 WHERE reservation_id = $2
		 RETURNING *`,
		amount, reservationID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: reservation_id=%s", ErrNotFound, reservationID)
		}
		return nil, fmt.Errorf("add penalty amount: %w", err)
	}
	return &record, nil
}

// CreatePenalty inserts a new penalty record.
func (r *sqlxRepository) CreatePenalty(ctx context.Context, penalty *model.Penalty) error {
	_, err := r.db.NamedExecContext(ctx,
		`INSERT INTO penalties (id, reservation_id, penalty_type, amount, description, created_at)
		 VALUES (:id, :reservation_id, :penalty_type, :amount, :description, :created_at)`,
		penalty,
	)
	if err != nil {
		return fmt.Errorf("create penalty: %w", err)
	}
	return nil
}
