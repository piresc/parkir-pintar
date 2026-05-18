// Package repository provides the sensor gateway abstraction and a stub
// implementation for the presence domain. The stub gateway returns configurable
// responses without calling any external sensor hardware.
package repository

import (
	"context"
	"time"
)

// SensorReading holds the result of a spot occupancy check from a sensor.
type SensorReading struct {
	Occupied   bool
	DetectedAt time.Time
}

// SensorGateway defines the interface for parking spot sensor operations.
type SensorGateway interface {
	CheckSpotOccupancy(ctx context.Context, floorNumber int, spotNumber int) (*SensorReading, error)
}

// StubSensorGateway is a configurable stub implementation of SensorGateway for
// testing and development. By default it returns Occupied=true.
type StubSensorGateway struct {
	// OccupiedResult controls what the stub returns for Occupied. Default: true.
	OccupiedResult bool
	// ErrResult allows injecting an error for testing error paths.
	ErrResult error
}

// NewStubSensorGateway creates a new StubSensorGateway that reports spots as occupied.
func NewStubSensorGateway() *StubSensorGateway {
	return &StubSensorGateway{OccupiedResult: true}
}

// CheckSpotOccupancy returns the configured stub response.
func (g *StubSensorGateway) CheckSpotOccupancy(_ context.Context, _ int, _ int) (*SensorReading, error) {
	if g.ErrResult != nil {
		return nil, g.ErrResult
	}
	return &SensorReading{
		Occupied:   g.OccupiedResult,
		DetectedAt: time.Now(),
	}, nil
}
