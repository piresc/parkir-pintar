// Package repository provides the data access layer for the analytics module
// using sqlx with parameterized queries against the reservation schema tables.
package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"parkir-pintar/internal/analytics/model"
)

// Repository defines the data access interface for analytics queries.
type Repository interface {
	// GetHourlyStats returns aggregated reservation and occupancy statistics
	// grouped by hour and day-of-week for the given date range.
	GetHourlyStats(ctx context.Context, startDate, endDate time.Time) ([]model.PeakHourStats, error)

	// GetDailyOccupancy returns average spot occupancy per day for the last N days.
	GetDailyOccupancy(ctx context.Context, days int) ([]model.DailyOccupancy, error)

	// RecordEvent persists a reservation event for analytics reporting.
	RecordEvent(ctx context.Context, event model.ReservationEvent) error
}

// sqlxRepository is the PostgreSQL-backed implementation of Repository.
type sqlxRepository struct {
	db *sqlx.DB
}

// NewRepository creates a new analytics Repository backed by the given sqlx.DB.
func NewRepository(db *sqlx.DB) Repository {
	return &sqlxRepository{db: db}
}

// GetHourlyStats queries reservation counts grouped by hour and day-of-week,
// calculating average occupancy and a peak score based on reservation density.
func (r *sqlxRepository) GetHourlyStats(ctx context.Context, startDate, endDate time.Time) ([]model.PeakHourStats, error) {
	query := `
		SELECT
			EXTRACT(HOUR FROM r.created_at)::int AS hour,
			EXTRACT(DOW FROM r.created_at)::int AS day_of_week,
			COALESCE(AVG(
				CASE WHEN ps.status IN ('reserved', 'occupied') THEN 1.0 ELSE 0.0 END
			), 0) AS avg_occupancy,
			COUNT(r.id)::int AS avg_reservations,
			(COUNT(r.id)::float / GREATEST(EXTRACT(EPOCH FROM ($2::timestamp - $1::timestamp)) / 3600, 1)) AS peak_score
		FROM reservations r
		JOIN parking_spots ps ON ps.id = r.spot_id
		WHERE r.created_at >= $1 AND r.created_at < $2
		GROUP BY EXTRACT(HOUR FROM r.created_at), EXTRACT(DOW FROM r.created_at)
		ORDER BY peak_score DESC`

	var stats []model.PeakHourStats
	if err := r.db.SelectContext(ctx, &stats, query, startDate, endDate); err != nil {
		return nil, fmt.Errorf("get hourly stats: %w", err)
	}
	return stats, nil
}

// GetDailyOccupancy calculates average spot occupancy per day for the last N days
// by comparing occupied/reserved spots against total capacity.
func (r *sqlxRepository) GetDailyOccupancy(ctx context.Context, days int) ([]model.DailyOccupancy, error) {
	query := `
		WITH daily AS (
			SELECT
				DATE(r.created_at) AS date,
				COUNT(DISTINCT r.spot_id) AS occupied_spots
			FROM reservations r
			WHERE r.created_at >= NOW() - make_interval(days => $1)
			  AND r.status IN ('confirmed', 'checked_in', 'checked_out')
			GROUP BY DATE(r.created_at)
		),
		capacity AS (
			SELECT COUNT(*) AS total_spots FROM parking_spots
		)
		SELECT
			d.date,
			d.occupied_spots,
			c.total_spots,
			CASE WHEN c.total_spots > 0
				THEN d.occupied_spots::float / c.total_spots
				ELSE 0
			END AS avg_occupancy
		FROM daily d
		CROSS JOIN capacity c
		ORDER BY d.date DESC`

	var occupancy []model.DailyOccupancy
	if err := r.db.SelectContext(ctx, &occupancy, query, days); err != nil {
		return nil, fmt.Errorf("get daily occupancy: %w", err)
	}
	return occupancy, nil
}

// RecordEvent inserts a reservation event into the reservation_events table
// for later analytics aggregation (peak hours, occupancy trends).
func (r *sqlxRepository) RecordEvent(ctx context.Context, event model.ReservationEvent) error {
	query := `
		INSERT INTO reservation_events (reservation_id, driver_id, spot_id, vehicle_type, status, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := r.db.ExecContext(ctx, query,
		event.ReservationID,
		event.DriverID,
		event.SpotID,
		event.VehicleType,
		event.Status,
		event.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("record event: %w", err)
	}
	return nil
}
