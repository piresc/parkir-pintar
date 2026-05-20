// module using sqlx with parameterized queries for SQL injection prevention.
// - Use parameterized queries to prevent SQL injection
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"parkir-pintar/internal/search/model"
	"parkir-pintar/internal/search/sync"
)

var ErrNotFound = errors.New("spot not found")

type Repository interface {
	GetAvailabilityByVehicleType(ctx context.Context, vehicleType string) ([]model.FloorAvailability, error)
	GetFloorSpots(ctx context.Context, floorNumber int) ([]model.SpotDetails, error)
	GetSpotByID(ctx context.Context, spotID string) (*model.SpotDetails, error)
}

type ReadModelRepository interface {
	UpsertSpot(ctx context.Context, spot sync.SpotData) error
	DeleteSpot(ctx context.Context, spotID string) error
}

type sqlxRepository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) Repository {
	return &sqlxRepository{db: db}
}

func (r *sqlxRepository) GetAvailabilityByVehicleType(ctx context.Context, vehicleType string) ([]model.FloorAvailability, error) {
	baseQuery := `
		SELECT
			floor_number,
			COUNT(*) FILTER (WHERE status = 'available' AND vehicle_type = 'car') AS available_car,
			COUNT(*) FILTER (WHERE status = 'available' AND vehicle_type = 'motorcycle') AS available_moto,
			COUNT(*) FILTER (WHERE vehicle_type = 'car') AS total_car,
			COUNT(*) FILTER (WHERE vehicle_type = 'motorcycle') AS total_moto
		FROM spot_read_model`

	var floors []model.FloorAvailability
	var err error

	if vehicleType != "" {
		query := baseQuery + ` WHERE vehicle_type = $1 GROUP BY floor_number ORDER BY floor_number`
		err = r.db.SelectContext(ctx, &floors, query, vehicleType)
	} else {
		query := baseQuery + ` GROUP BY floor_number ORDER BY floor_number`
		err = r.db.SelectContext(ctx, &floors, query)
	}

	if err != nil {
		return nil, fmt.Errorf("get availability: %w", err)
	}
	return floors, nil
}

func (r *sqlxRepository) GetFloorSpots(ctx context.Context, floorNumber int) ([]model.SpotDetails, error) {
	query := `
		SELECT id, spot_code, vehicle_type, status, floor_number, spot_number
		FROM spot_read_model
		WHERE floor_number = $1
		ORDER BY spot_number`

	var spots []model.SpotDetails
	if err := r.db.SelectContext(ctx, &spots, query, floorNumber); err != nil {
		return nil, fmt.Errorf("get floor spots: %w", err)
	}
	return spots, nil
}

func (r *sqlxRepository) GetSpotByID(ctx context.Context, spotID string) (*model.SpotDetails, error) {
	query := `
		SELECT id, spot_code, vehicle_type, status, floor_number, spot_number
		FROM spot_read_model
		WHERE id = $1`

	var spot model.SpotDetails
	if err := r.db.GetContext(ctx, &spot, query, spotID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: id=%s", ErrNotFound, spotID)
		}
		return nil, fmt.Errorf("get spot by id: %w", err)
	}
	return &spot, nil
}

type sqlxReadModelRepository struct {
	db *sqlx.DB
}

func NewReadModelRepository(db *sqlx.DB) ReadModelRepository {
	return &sqlxReadModelRepository{db: db}
}

func (r *sqlxReadModelRepository) UpsertSpot(ctx context.Context, spot sync.SpotData) error {
	query := `
		INSERT INTO spot_read_model (id, floor_number, spot_number, vehicle_type, spot_code, status, updated_at)
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
		return fmt.Errorf("upsert spot read model: %w", err)
	}
	return nil
}

func (r *sqlxReadModelRepository) DeleteSpot(ctx context.Context, spotID string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM spot_read_model WHERE id = $1", spotID)
	if err != nil {
		return fmt.Errorf("delete spot read model: %w", err)
	}
	return nil
}
