// Package usecase implements the business logic for the presence domain.
// It verifies driver location against assigned parking spots using Redis Geo.
package usecase

import (
	"context"
	"fmt"

	"parkir-pintar/internal/presence/repository"
)

const (
	// DefaultThresholdMeters is the default maximum distance (in meters)
	// a driver can be from their assigned spot to be considered verified.
	DefaultThresholdMeters = 50.0
)

// SpotInfo holds the spot code for a reservation lookup.
type SpotInfo struct {
	SpotID   string
	SpotCode string
}

// ReservationLookup provides spot info for a given reservation.
// In a full implementation this would call the reservation service.
type ReservationLookup interface {
	GetSpotForReservation(ctx context.Context, reservationID string) (*SpotInfo, error)
}

// Usecase defines the business logic interface for presence operations.
type Usecase interface {
	VerifyLocation(ctx context.Context, driverID string, lat, lng float64, reservationID string) (verified bool, distanceMeters float64, spotCode string, err error)
	UpdateDriverLocation(ctx context.Context, driverID string, lat, lng float64) error
}

// presenceUsecase is the concrete implementation of Usecase.
type presenceUsecase struct {
	repo             repository.Repository
	lookup           ReservationLookup
	thresholdMeters  float64
}

// NewUsecase creates a new presence Usecase with all required dependencies.
func NewUsecase(repo repository.Repository, lookup ReservationLookup, thresholdMeters float64) Usecase {
	if thresholdMeters <= 0 {
		thresholdMeters = DefaultThresholdMeters
	}
	return &presenceUsecase{
		repo:            repo,
		lookup:          lookup,
		thresholdMeters: thresholdMeters,
	}
}

// VerifyLocation checks if the driver's GPS coordinates are within the
// acceptable radius of their assigned parking spot.
func (uc *presenceUsecase) VerifyLocation(ctx context.Context, driverID string, lat, lng float64, reservationID string) (bool, float64, string, error) {
	// Look up which spot is assigned to this reservation.
	spotInfo, err := uc.lookup.GetSpotForReservation(ctx, reservationID)
	if err != nil {
		return false, 0, "", fmt.Errorf("lookup spot for reservation %s: %w", reservationID, err)
	}

	// Update driver location in Redis Geo.
	if err := uc.repo.UpdateDriverLocation(ctx, driverID, lat, lng); err != nil {
		return false, 0, spotInfo.SpotCode, fmt.Errorf("update driver location: %w", err)
	}

	// Calculate distance between driver and assigned spot.
	dist, err := uc.repo.GetDistanceToSpot(ctx, driverID, spotInfo.SpotID)
	if err != nil {
		return false, 0, spotInfo.SpotCode, fmt.Errorf("get distance to spot: %w", err)
	}

	verified := dist <= uc.thresholdMeters
	return verified, dist, spotInfo.SpotCode, nil
}

// UpdateDriverLocation stores the driver's current GPS coordinates.
func (uc *presenceUsecase) UpdateDriverLocation(ctx context.Context, driverID string, lat, lng float64) error {
	return uc.repo.UpdateDriverLocation(ctx, driverID, lat, lng)
}
