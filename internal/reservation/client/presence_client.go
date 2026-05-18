package client

import (
	"context"
	"errors"
	"time"

	"parkir-pintar/pkg/circuitbreaker"
	presencev1 "parkir-pintar/proto/presence/v1"
)

// PresenceResult holds the result of a location verification check.
type PresenceResult struct {
	Verified         bool
	DistanceMeters   float64
	AssignedSpotCode string
}

// PresenceClient defines the interface for presence service operations.
//
//go:generate mockgen -destination=../mocks/mock_presence_client.go -package=mocks parkir-pintar/internal/reservation/client PresenceClient
type PresenceClient interface {
	VerifyLocation(ctx context.Context, driverID string, lat, lng float64, reservationID string) (*PresenceResult, error)
}

// presenceClient is the concrete gRPC-backed implementation of PresenceClient.
type presenceClient struct {
	client presencev1.PresenceServiceClient
	cb     *circuitbreaker.CircuitBreaker
}

// NewPresenceClient creates a new PresenceClient with circuit breaker protection.
func NewPresenceClient(client presencev1.PresenceServiceClient) PresenceClient {
	return &presenceClient{
		client: client,
		cb: circuitbreaker.New(circuitbreaker.Config{
			FailureThreshold:  5,
			OpenTimeout:       30 * time.Second,
			HalfOpenMaxProbes: 1,
		}),
	}
}

func (c *presenceClient) VerifyLocation(ctx context.Context, driverID string, lat, lng float64, reservationID string) (*PresenceResult, error) {
	var result *PresenceResult
	err := c.cb.Execute(func() error {
		var err error
		result, err = c.verifyLocationInner(ctx, driverID, lat, lng, reservationID)
		return err
	})
	if errors.Is(err, circuitbreaker.ErrCircuitOpen) {
		return nil, errors.New("presence service temporarily unavailable")
	}
	return result, err
}

func (c *presenceClient) verifyLocationInner(ctx context.Context, driverID string, lat, lng float64, reservationID string) (*PresenceResult, error) {
	resp, err := c.client.VerifyLocation(ctx, &presencev1.VerifyLocationRequest{
		DriverId:      driverID,
		Latitude:      lat,
		Longitude:     lng,
		ReservationId: reservationID,
	})
	if err != nil {
		return nil, err
	}
	return &PresenceResult{
		Verified:         resp.GetVerified(),
		DistanceMeters:   resp.GetDistanceMeters(),
		AssignedSpotCode: resp.GetAssignedSpotCode(),
	}, nil
}
