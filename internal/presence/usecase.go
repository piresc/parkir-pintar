package presence

import (
	"context"

	"parkir-pintar/internal/presence/repository"
)

//go:generate mockgen -destination=mocks/mock_usecase.go -package=mocks parkir-pintar/internal/presence Usecase
type Usecase interface {
	VerifyPresence(ctx context.Context, reservationID string, floorNumber int, spotNumber int) (*VerifyResult, error)
}

type VerifyResult struct {
	Verified bool
	Message  string
}

// SensorReading is an alias for the canonical type in the repository package.
type SensorReading = repository.SensorReading

//go:generate mockgen -destination=mocks/mock_sensor_gateway.go -package=mocks parkir-pintar/internal/presence SensorGateway
type SensorGateway interface {
	CheckSpotOccupancy(ctx context.Context, floorNumber int, spotNumber int) (*SensorReading, error)
}
