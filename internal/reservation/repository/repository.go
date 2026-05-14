// Package repository provides the data access layer for the reservation domain
// module using sqlx with parameterized queries for SQL injection prevention.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"parkir-pintar/internal/reservation/model"
)

// Repository defines the data access interface for reservations and parking spots.
//
//go:generate mockgen -destination=../mocks/mock_repository.go -package=mocks parkir-pintar/internal/reservation/repository Repository
type Repository interface {
	FindByIdempotencyKey(ctx context.Context, key string) (*model.Reservation, error)
	FindAvailableSpot(ctx context.Context, vehicleType string) (*model.ParkingSpot, error)
	GetSpotByID(ctx context.Context, spotID string) (*model.ParkingSpot, error)
	GetSpotForUpdate(ctx context.Context, spotID string) (*model.ParkingSpot, error)
	CreateReservationTx(ctx context.Context, tx *sqlx.Tx, reservation *model.Reservation) error
	UpdateSpotStatusTx(ctx context.Context, tx *sqlx.Tx, spotID string, status string) error
	UpdateReservationTx(ctx context.Context, tx *sqlx.Tx, reservation *model.Reservation) error
	FindExpiredReservations(ctx context.Context) ([]*model.Reservation, error)
	GetByID(ctx context.Context, id string) (*model.Reservation, error)
	GetByIDForUpdate(ctx context.Context, tx *sqlx.Tx, id string) (*model.Reservation, error)
	WithTransaction(ctx context.Context, fn func(tx *sqlx.Tx) error) error
	FindStalePaymentReservations(ctx context.Context, timeoutMinutes int) ([]*model.Reservation, error)
	ListByDriverID(ctx context.Context, driverID string, status string) ([]*model.Reservation, error)
}

// sqlxRepository is the sqlx-backed implementation of Repository.
type sqlxRepository struct {
	db *sqlx.DB
}

// NewRepository creates a new Repository backed by the given sqlx.DB.
func NewRepository(db *sqlx.DB) Repository {
	return &sqlxRepository{db: db}
}

// FindByIdempotencyKey retrieves a reservation by its idempotency key.
func (r *sqlxRepository) FindByIdempotencyKey(ctx context.Context, key string) (*model.Reservation, error) {
	var reservation model.Reservation
	err := r.db.GetContext(ctx, &reservation, "SELECT * FROM reservations WHERE idempotency_key = $1", key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: idempotency_key=%s", model.ErrNotFound, key)
		}
		return nil, fmt.Errorf("find reservation by idempotency key: %w", err)
	}
	return &reservation, nil
}

// FindAvailableSpot retrieves the first available parking spot matching the vehicle type.
// Uses FOR UPDATE SKIP LOCKED to avoid lock contention during concurrent reservations.
func (r *sqlxRepository) FindAvailableSpot(ctx context.Context, vehicleType string) (*model.ParkingSpot, error) {
	var spot model.ParkingSpot
	err := r.db.GetContext(ctx, &spot,
		`SELECT * FROM parking_spots
		 WHERE vehicle_type = $1 AND status = 'available'
		 ORDER BY floor_number, spot_number
		 LIMIT 1
		 FOR UPDATE SKIP LOCKED`,
		vehicleType,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: vehicle_type=%s", model.ErrNotFound, vehicleType)
		}
		return nil, fmt.Errorf("find available spot: %w", err)
	}
	return &spot, nil
}

// GetSpotByID retrieves a parking spot by ID (read-only, no lock).
func (r *sqlxRepository) GetSpotByID(ctx context.Context, spotID string) (*model.ParkingSpot, error) {
	var spot model.ParkingSpot
	err := r.db.GetContext(ctx, &spot, "SELECT * FROM parking_spots WHERE id = $1", spotID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: spot_id=%s", model.ErrNotFound, spotID)
		}
		return nil, fmt.Errorf("get spot by id: %w", err)
	}
	return &spot, nil
}

// GetSpotForUpdate retrieves a parking spot by ID with a row-level lock for update.
func (r *sqlxRepository) GetSpotForUpdate(ctx context.Context, spotID string) (*model.ParkingSpot, error) {
	var spot model.ParkingSpot
	err := r.db.GetContext(ctx, &spot, "SELECT * FROM parking_spots WHERE id = $1 FOR UPDATE", spotID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: spot_id=%s", model.ErrNotFound, spotID)
		}
		return nil, fmt.Errorf("get spot for update: %w", err)
	}
	return &spot, nil
}

// CreateReservationTx inserts a new reservation within the given transaction.
func (r *sqlxRepository) CreateReservationTx(ctx context.Context, tx *sqlx.Tx, reservation *model.Reservation) error {
	_, err := tx.NamedExecContext(ctx,
		`INSERT INTO reservations (id, driver_id, spot_id, vehicle_type, assignment_mode, status,
		 idempotency_key, confirmed_at, expires_at, checked_in_at, checked_out_at, cancelled_at,
		 created_at, updated_at)
		 VALUES (:id, :driver_id, :spot_id, :vehicle_type, :assignment_mode, :status,
		 :idempotency_key, :confirmed_at, :expires_at, :checked_in_at, :checked_out_at, :cancelled_at,
		 :created_at, :updated_at)`,
		reservation,
	)
	if err != nil {
		return fmt.Errorf("create reservation: %w", err)
	}
	return nil
}

// UpdateSpotStatusTx updates a parking spot's status within the given transaction.
func (r *sqlxRepository) UpdateSpotStatusTx(ctx context.Context, tx *sqlx.Tx, spotID string, status string) error {
	_, err := tx.ExecContext(ctx,
		"UPDATE parking_spots SET status = $1, updated_at = NOW() WHERE id = $2",
		status, spotID,
	)
	if err != nil {
		return fmt.Errorf("update spot status: %w", err)
	}
	return nil
}

// FindExpiredReservations retrieves all confirmed reservations that have passed their expiry time.
func (r *sqlxRepository) FindExpiredReservations(ctx context.Context) ([]*model.Reservation, error) {
	var reservations []*model.Reservation
	err := r.db.SelectContext(ctx, &reservations,
		"SELECT * FROM reservations WHERE status = 'confirmed' AND expires_at < NOW()",
	)
	if err != nil {
		return nil, fmt.Errorf("find expired reservations: %w", err)
	}
	return reservations, nil
}

// GetByID retrieves a single reservation by its UUID.
func (r *sqlxRepository) GetByID(ctx context.Context, id string) (*model.Reservation, error) {
	var reservation model.Reservation
	err := r.db.GetContext(ctx, &reservation, "SELECT * FROM reservations WHERE id = $1", id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: id=%s", model.ErrNotFound, id)
		}
		return nil, fmt.Errorf("get reservation by id: %w", err)
	}
	return &reservation, nil
}

// GetByIDForUpdate retrieves a reservation by ID with a row-level lock (FOR UPDATE)
// within the given transaction. This prevents TOCTOU races on state transitions.
func (r *sqlxRepository) GetByIDForUpdate(ctx context.Context, tx *sqlx.Tx, id string) (*model.Reservation, error) {
	var reservation model.Reservation
	err := tx.GetContext(ctx, &reservation, "SELECT * FROM reservations WHERE id = $1 FOR UPDATE", id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: id=%s", model.ErrNotFound, id)
		}
		return nil, fmt.Errorf("get reservation for update: %w", err)
	}
	return &reservation, nil
}

// UpdateReservationTx updates an existing reservation's mutable fields within a transaction.
func (r *sqlxRepository) UpdateReservationTx(ctx context.Context, tx *sqlx.Tx, reservation *model.Reservation) error {
	_, err := tx.NamedExecContext(ctx,
		`UPDATE reservations SET status = :status, confirmed_at = :confirmed_at,
		 expires_at = :expires_at, checked_in_at = :checked_in_at,
		 checked_out_at = :checked_out_at, cancelled_at = :cancelled_at,
		 updated_at = :updated_at
		 WHERE id = :id`,
		reservation,
	)
	if err != nil {
		return fmt.Errorf("update reservation tx: %w", err)
	}
	return nil
}

// WithTransaction executes the given function within a database transaction.
// If fn returns an error, the transaction is rolled back; otherwise it is committed.
func (r *sqlxRepository) WithTransaction(ctx context.Context, fn func(tx *sqlx.Tx) error) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback failed: %w (original: %w)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// FindStalePaymentReservations retrieves waiting_payment reservations whose
// created_at is older than the specified timeout in minutes.
func (r *sqlxRepository) FindStalePaymentReservations(ctx context.Context, timeoutMinutes int) ([]*model.Reservation, error) {
	var reservations []*model.Reservation
	query := `SELECT * FROM reservations
		WHERE status = 'waiting_payment'
		AND created_at < NOW() - make_interval(mins => $1)`
	err := r.db.SelectContext(ctx, &reservations, query, timeoutMinutes)
	if err != nil {
		return nil, fmt.Errorf("find stale payment reservations: %w", err)
	}
	return reservations, nil
}

// ListByDriverID retrieves reservations for a given driver, optionally filtered by status.
func (r *sqlxRepository) ListByDriverID(ctx context.Context, driverID string, status string) ([]*model.Reservation, error) {
	var reservations []*model.Reservation
	if status != "" {
		err := r.db.SelectContext(ctx, &reservations,
			`SELECT r.*, ps.spot_code FROM reservations r
			 JOIN parking_spots ps ON ps.id = r.spot_id
			 WHERE r.driver_id = $1 AND r.status = $2 ORDER BY r.created_at DESC`,
			driverID, status,
		)
		if err != nil {
			return nil, fmt.Errorf("list reservations by driver: %w", err)
		}
	} else {
		err := r.db.SelectContext(ctx, &reservations,
			`SELECT r.*, ps.spot_code FROM reservations r
			 JOIN parking_spots ps ON ps.id = r.spot_id
			 WHERE r.driver_id = $1 ORDER BY r.created_at DESC`,
			driverID,
		)
		if err != nil {
			return nil, fmt.Errorf("list reservations by driver: %w", err)
		}
	}
	return reservations, nil
}
