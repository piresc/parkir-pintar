// Package handler provides bug condition exploration tests for presence coordinate validation.
//
// Best practices applied (from Go testify coding standards KB):
// - Test naming: Test[FunctionName]_Should[ExpectedResult]_When[Condition]
// - AAA pattern: Arrange → Act → Assert
// - testify/assert for assertions
// - Mock at interface boundaries rather than concrete implementations
// - Keep mocks simple and focused on the behavior being tested
//
// **Validates: Requirements 2.13** (Property 10 from design)
//
// Bug Condition: lat < -90 OR lat > 90 OR lng < -180 OR lng > 180
// Expected: codes.InvalidArgument
// Counterexample on unfixed code: no error returned
//
// CRITICAL: This test is expected to FAIL on unfixed code.
// DO NOT fix the code or the test when it fails.
package handler

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"parkir-pintar/internal/presence/model"
	presencev1 "parkir-pintar/proto/presence/v1"

	"pgregory.net/rapid"
)

// mockPresenceUsecase implements usecase.Usecase for testing the handler layer.
type mockPresenceUsecase struct{}

func (m *mockPresenceUsecase) StreamLocation(ctx context.Context, update *model.LocationUpdate) error {
	return nil
}

func (m *mockPresenceUsecase) DetectArrival(ctx context.Context, lat, lng, centerLat, centerLng, radiusMeters float64, reservationID string) (*model.ArrivalResult, error) {
	return &model.ArrivalResult{
		Arrived:       false,
		ReservationID: reservationID,
	}, nil
}

func (m *mockPresenceUsecase) DetectWrongSpot(ctx context.Context, lat, lng, spotLat, spotLng, thresholdMeters float64, reservationID string) (*model.WrongSpotResult, error) {
	return &model.WrongSpotResult{
		IsWrongSpot:    false,
		DistanceMeters: 0,
	}, nil
}

func (m *mockPresenceUsecase) GetPresence(ctx context.Context, reservationID string) (*model.PresenceLog, error) {
	return &model.PresenceLog{ReservationID: reservationID}, nil
}

// TestDetectArrival_ShouldReturnInvalidArgument_WhenCoordinatesOutOfRange generates
// random lat/lng values outside [-90,90]×[-180,180] and sends them to DetectArrival,
// expecting an InvalidArgument error. On unfixed code, no validation exists so the
// request is accepted without error.
//
// **Validates: Requirements 2.13**
func TestDetectArrival_ShouldReturnInvalidArgument_WhenCoordinatesOutOfRange(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Arrange — generate coordinates that are out of valid range
		// Pick one of four invalid regions: lat too high, lat too low, lng too high, lng too low
		invalidType := rapid.IntRange(0, 3).Draw(t, "invalidType")

		var lat, lng float64
		switch invalidType {
		case 0: // lat > 90
			lat = rapid.Float64Range(90.001, 1000).Draw(t, "lat")
			lng = rapid.Float64Range(-180, 180).Draw(t, "lng")
		case 1: // lat < -90
			lat = rapid.Float64Range(-1000, -90.001).Draw(t, "lat")
			lng = rapid.Float64Range(-180, 180).Draw(t, "lng")
		case 2: // lng > 180
			lat = rapid.Float64Range(-90, 90).Draw(t, "lat")
			lng = rapid.Float64Range(180.001, 1000).Draw(t, "lng")
		case 3: // lng < -180
			lat = rapid.Float64Range(-90, 90).Draw(t, "lat")
			lng = rapid.Float64Range(-1000, -180.001).Draw(t, "lng")
		}

		uc := &mockPresenceUsecase{}
		h := NewHandler(uc)

		req := &presencev1.DetectArrivalRequest{
			ReservationId:   "res-test",
			Latitude:        lat,
			Longitude:       lng,
			CenterLatitude:  0,
			CenterLongitude: 0,
			RadiusMeters:    100,
		}

		// Act
		_, err := h.DetectArrival(context.Background(), req)

		// Assert — should return InvalidArgument for out-of-range coordinates
		require.Error(t, err, "expected error for out-of-range coordinates (lat=%f, lng=%f)", lat, lng)
		st, ok := status.FromError(err)
		require.True(t, ok, "error should be a gRPC status error")
		assert.Equal(t, codes.InvalidArgument, st.Code(),
			"expected InvalidArgument for lat=%f, lng=%f, got %s", lat, lng, st.Code())
	})
}
