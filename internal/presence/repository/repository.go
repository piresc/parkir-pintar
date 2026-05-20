package repository

import (
	"context"
	"time"
)

type SensorReading struct {
	Occupied   bool
	DetectedAt time.Time
}

//go:generate mockgen -destination=../mocks/mock_sensor_gateway.go -package=mocks parkir-pintar/internal/presence/repository SensorGateway
type SensorGateway interface {
	CheckSpotOccupancy(ctx context.Context, floorNumber int, spotNumber int) (*SensorReading, error)
}

type StubSensorGateway struct {
	OccupiedResult bool
	ErrResult      error
}

func NewStubSensorGateway() *StubSensorGateway {
	return &StubSensorGateway{OccupiedResult: true}
}

func (g *StubSensorGateway) CheckSpotOccupancy(_ context.Context, _ int, _ int) (*SensorReading, error) {
	if g.ErrResult != nil {
		return nil, g.ErrResult
	}
	return &SensorReading{
		Occupied:   g.OccupiedResult,
		DetectedAt: time.Now(),
	}, nil
}
