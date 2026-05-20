package grpcgw

import (
	"context"
	"errors"
	"time"

	reservation "parkir-pintar/internal/reservation"
	"parkir-pintar/pkg/circuitbreaker"
	presencev1 "parkir-pintar/proto/presence/v1"
)

// PresenceClient defines the interface for presence service operations.
//
//go:generate mockgen -destination=../../mocks/mock_presence_client.go -package=mocks parkir-pintar/internal/reservation/gateway/grpc PresenceClient
type PresenceClient interface {
	VerifyPresence(ctx context.Context, driverID string, reservationID string, floorNumber int, spotNumber int) (*reservation.PresenceResult, error)
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

func (c *presenceClient) VerifyPresence(ctx context.Context, driverID string, reservationID string, floorNumber int, spotNumber int) (*reservation.PresenceResult, error) {
	var result *reservation.PresenceResult
	err := c.cb.Execute(func() error {
		var err error
		result, err = c.verifyPresenceInner(ctx, driverID, reservationID, floorNumber, spotNumber)
		return err
	})
	if errors.Is(err, circuitbreaker.ErrCircuitOpen) {
		return nil, errors.New("presence service temporarily unavailable")
	}
	return result, err
}

func (c *presenceClient) verifyPresenceInner(ctx context.Context, driverID string, reservationID string, floorNumber int, spotNumber int) (*reservation.PresenceResult, error) {
	resp, err := c.client.VerifyPresence(ctx, &presencev1.VerifyPresenceRequest{
		DriverId:      driverID,
		ReservationId: reservationID,
		FloorNumber:   int32(floorNumber),
		SpotNumber:    int32(spotNumber),
	})
	if err != nil {
		return nil, err
	}
	return &reservation.PresenceResult{
		Verified: resp.GetVerified(),
		Message:  resp.GetMessage(),
	}, nil
}
