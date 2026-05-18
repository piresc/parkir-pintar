// Package usecase implements the business logic for the presence domain.
// It verifies driver presence at an assigned parking spot using sensor data.
package usecase

import (
	"context"
	"log/slog"

	"parkir-pintar/internal/presence/repository"
)

// VerifyResult holds the outcome of a presence verification check.
type VerifyResult struct {
	Verified bool
	Message  string
}

// Usecase defines the business logic interface for presence operations.
type Usecase interface {
	VerifyPresence(ctx context.Context, reservationID string, floorNumber int, spotNumber int) (*VerifyResult, error)
}

// presenceUsecase is the concrete implementation of Usecase.
type presenceUsecase struct {
	sensor repository.SensorGateway
}

// NewUsecase creates a new presence Usecase with the given sensor gateway.
func NewUsecase(sensor repository.SensorGateway) Usecase {
	return &presenceUsecase{
		sensor: sensor,
	}
}

// VerifyPresence checks if the assigned parking spot is occupied using sensor data.
//
// Graceful degradation: if the sensor call fails, the driver is considered
// verified (benefit of the doubt) and a warning is logged.
func (uc *presenceUsecase) VerifyPresence(ctx context.Context, reservationID string, floorNumber int, spotNumber int) (*VerifyResult, error) {
	reading, err := uc.sensor.CheckSpotOccupancy(ctx, floorNumber, spotNumber)
	if err != nil {
		slog.Warn("sensor check failed, assuming presence verified (graceful degradation)",
			slog.String("reservation_id", reservationID),
			slog.Int("floor_number", floorNumber),
			slog.Int("spot_number", spotNumber),
			slog.Any("error", err))
		return &VerifyResult{
			Verified: true,
			Message:  "sensor unavailable, presence assumed",
		}, nil
	}

	if reading.Occupied {
		return &VerifyResult{
			Verified: true,
			Message:  "spot occupied, presence confirmed",
		}, nil
	}

	return &VerifyResult{
		Verified: false,
		Message:  "spot not occupied, driver may be at wrong spot",
	}, nil
}
