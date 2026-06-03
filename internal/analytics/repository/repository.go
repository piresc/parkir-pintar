package repository

import (
	"context"
	"fmt"
	"time"

	"parkir-pintar/internal/analytics/model"
)

type Repository interface {
	GetHourlyStats(ctx context.Context, startDate, endDate time.Time) ([]model.PeakHourStats, error)

	GetDailyOccupancy(ctx context.Context, days int) ([]model.DailyOccupancy, error)

	RecordEvent(ctx context.Context, event model.ReservationEvent) error

	UpsertSpotSnapshot(ctx context.Context, spot model.SpotSnapshot) error
}

func (r *sqlxRepository) GetHourlyStats(ctx context.Context, startDate, endDate time.Time) ([]model.PeakHourStats, error) {
	query := `
		SELECT
			EXTRACT(HOUR FROM r.timestamp)::int AS hour,
			EXTRACT(DOW FROM r.timestamp)::int AS day_of_week,
			COALESCE(AVG(
				CASE WHEN ps.status IN ('reserved', 'occupied') THEN 1.0 ELSE 0.0 END
			), 0) AS avg_occupancy,
			COUNT(r.id)::int AS avg_reservations,
			(COUNT(r.id)::float / GREATEST(EXTRACT(EPOCH FROM ($2::timestamp - $1::timestamp)) / 3600, 1)) AS peak_score
		FROM analytics.reservation_events r
		JOIN analytics.spot_snapshot ps ON ps.id = r.spot_id
		WHERE r.timestamp >= $1 AND r.timestamp < $2
		GROUP BY EXTRACT(HOUR FROM r.timestamp), EXTRACT(DOW FROM r.timestamp)
		ORDER BY peak_score DESC`

	var stats []model.PeakHourStats
	if err := r.db.SelectContext(ctx, &stats, query, startDate, endDate); err != nil {
		return nil, fmt.Errorf("get hourly stats: %w", err)
	}
	return stats, nil
}

func (r *sqlxRepository) GetDailyOccupancy(ctx context.Context, days int) ([]model.DailyOccupancy, error) {
	query := `
		WITH daily AS (
			SELECT
				DATE(r.timestamp) AS date,
				COUNT(DISTINCT r.spot_id) AS occupied_spots
			FROM analytics.reservation_events r
			WHERE r.timestamp >= NOW() - make_interval(days => $1)
			  AND r.status IN ('confirmed', 'checked_in', 'checked_out')
			GROUP BY DATE(r.timestamp)
		),
		capacity AS (
			SELECT COUNT(*) AS total_spots FROM analytics.spot_snapshot
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
		ORDER BY d.date ASC`

	var occupancy []model.DailyOccupancy
	if err := r.db.SelectContext(ctx, &occupancy, query, days); err != nil {
		return nil, fmt.Errorf("get daily occupancy: %w", err)
	}
	return occupancy, nil
}

func (r *sqlxRepository) RecordEvent(ctx context.Context, event model.ReservationEvent) error {
	query := `
		INSERT INTO analytics.reservation_events (reservation_id, driver_id, spot_id, vehicle_type, status, timestamp)
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

func (r *sqlxRepository) UpsertSpotSnapshot(ctx context.Context, spot model.SpotSnapshot) error {
	query := `
		INSERT INTO analytics.spot_snapshot (id, floor_number, spot_number, vehicle_type, spot_code, status, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (id) DO UPDATE SET
			floor_number = EXCLUDED.floor_number,
			spot_number = EXCLUDED.spot_number,
			vehicle_type = EXCLUDED.vehicle_type,
			spot_code = EXCLUDED.spot_code,
			status = EXCLUDED.status,
			updated_at = NOW()`

	_, err := r.db.ExecContext(ctx, query,
		spot.ID, spot.FloorNumber, spot.SpotNumber, spot.VehicleType, spot.SpotCode, spot.Status)
	if err != nil {
		return fmt.Errorf("upsert spot snapshot: %w", err)
	}
	return nil
}
