// Package repository provides the data access layer for the presence domain
// module using sqlx with parameterized queries for SQL injection prevention.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"parkir-pintar/internal/presence/model"
)

// ErrNotFound is returned when a presence log is not found.
var ErrNotFound = errors.New("presence log not found")

// Repository defines the data access interface for presence logs.
type Repository interface {
	SavePresenceLog(ctx context.Context, log *model.PresenceLog) error
	GetPresenceByReservation(ctx context.Context, reservationID string) (*model.PresenceLog, error)
	CleanupPresence(ctx context.Context, reservationID string) error
}

// sqlxRepository is the sqlx-backed implementation of Repository.
type sqlxRepository struct {
	db *sqlx.DB
}

// NewRepository creates a new Repository backed by the given sqlx.DB.
func NewRepository(db *sqlx.DB) Repository {
	return &sqlxRepository{db: db}
}

// SavePresenceLog inserts a new presence log record.
func (r *sqlxRepository) SavePresenceLog(ctx context.Context, log *model.PresenceLog) error {
	_, err := r.db.NamedExecContext(ctx,
		`INSERT INTO presence_logs (id, reservation_id, latitude, longitude, accuracy, recorded_at)
		 VALUES (:id, :reservation_id, :latitude, :longitude, :accuracy, :recorded_at)`,
		log,
	)
	if err != nil {
		return fmt.Errorf("save presence log: %w", err)
	}
	return nil
}

// GetPresenceByReservation retrieves the latest presence log for a reservation.
func (r *sqlxRepository) GetPresenceByReservation(ctx context.Context, reservationID string) (*model.PresenceLog, error) {
	var log model.PresenceLog
	err := r.db.GetContext(ctx, &log,
		`SELECT * FROM presence_logs WHERE reservation_id = $1 ORDER BY recorded_at DESC LIMIT 1`,
		reservationID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: reservation_id=%s", ErrNotFound, reservationID)
		}
		return nil, fmt.Errorf("get presence by reservation: %w", err)
	}
	return &log, nil
}

// CleanupPresence deletes all presence logs for a reservation.
func (r *sqlxRepository) CleanupPresence(ctx context.Context, reservationID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM presence_logs WHERE reservation_id = $1`,
		reservationID,
	)
	if err != nil {
		return fmt.Errorf("cleanup presence: %w", err)
	}
	return nil
}
