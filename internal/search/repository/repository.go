// Package repository provides the data access layer for the search domain
// module using sqlx with parameterized queries for SQL injection prevention.
//
// Best practices applied (from Go coding standards KB):
// - Document all exported functions and types with proper Godoc format
// - Use context.Context as first parameter for consistency
// - Handle errors explicitly with proper wrapping
// - Keep interfaces small and focused
// - Use parameterized queries to prevent SQL injection
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"parkir-pintar/internal/search/model"
)

// ErrNotFound is returned when a parking spot is not found.
var ErrNotFound = errors.New("spot not found")

// Repository defines the data access interface for search queries.
type Repository interface {
	GetAvailabilityByVehicleType(ctx context.Context, vehicleType string) ([]model.FloorAvailability, error)
	GetFloorSpots(ctx context.Context, floorNumber int) ([]model.SpotDetails, error)
	GetSpotByID(ctx context.Context, spotID string) (*model.SpotDetails, error)
}

// sqlxRepository is the sqlx-backed implementation of Repository.
type sqlxRepository struct {
	db *sqlx.DB
}

// NewRepository creates a new Repository backed by the given sqlx.DB.
func NewRepository(db *sqlx.DB) Repository {
	return &sqlxRepository{db: db}
}

// GetAvailabilityByVehicleType returns per-floor availability counts.
// It counts available spots grouped by floor_number and vehicle_type.
func (r *sqlxRepository) GetAvailabilityByVehicleType(ctx context.Context, vehicleType string) ([]model.FloorAvailability, error) {
	query := `
		SELECT
			floor_number,
			COUNT(*) FILTER (WHERE status = 'available' AND vehicle_type = 'car') AS available_car,
			COUNT(*) FILTER (WHERE status = 'available' AND vehicle_type = 'motorcycle') AS available_moto,
			COUNT(*) FILTER (WHERE vehicle_type = 'car') AS total_car,
			COUNT(*) FILTER (WHERE vehicle_type = 'motorcycle') AS total_moto
		FROM parking_spots
		GROUP BY floor_number
		ORDER BY floor_number`

	var floors []model.FloorAvailability
	if err := r.db.SelectContext(ctx, &floors, query); err != nil {
		return nil, fmt.Errorf("get availability: %w", err)
	}
	return floors, nil
}

// GetFloorSpots returns all spots on a given floor ordered by spot_number.
func (r *sqlxRepository) GetFloorSpots(ctx context.Context, floorNumber int) ([]model.SpotDetails, error) {
	query := `
		SELECT id, spot_code, vehicle_type, status, floor_number, spot_number
		FROM parking_spots
		WHERE floor_number = $1
		ORDER BY spot_number`

	var spots []model.SpotDetails
	if err := r.db.SelectContext(ctx, &spots, query, floorNumber); err != nil {
		return nil, fmt.Errorf("get floor spots: %w", err)
	}
	return spots, nil
}

// GetSpotByID returns a single spot by its UUID.
func (r *sqlxRepository) GetSpotByID(ctx context.Context, spotID string) (*model.SpotDetails, error) {
	query := `
		SELECT id, spot_code, vehicle_type, status, floor_number, spot_number
		FROM parking_spots
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
