package presence

import (
	"context"
	"time"
)

//go:generate mockgen -destination=mocks/mock_usecase.go -package=mocks parkir-pintar/internal/presence Usecase
type Usecase interface {
	VerifyPresence(ctx context.Context, reservationID string, floorNumber int, spotNumber int) (*VerifyResult, error)
}

type VerifyResult struct {
	Verified bool
	Message  string
}

type SensorReading struct {
	Occupied   bool
	DetectedAt time.Time
}

//go:generate mockgen -destination=mocks/mock_sensor_gateway.go -package=mocks parkir-pintar/internal/presence SensorGateway
type SensorGateway interface {
	CheckSpotOccupancy(ctx context.Context, floorNumber int, spotNumber int) (*SensorReading, error)
}
