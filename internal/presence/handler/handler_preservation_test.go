// Package handler provides preservation property tests for valid coordinate acceptance.
//
// Best practices applied (from Go testify coding standards KB):
// - Test naming: Test[FunctionName]_Should[ExpectedResult]_When[Condition]
// - AAA pattern: Arrange → Act → Assert
// - testify/assert and testify/require for assertions
// - Mock at interface boundaries rather than concrete implementations
// - Keep mocks simple and focused on the behavior being tested
//
// **Validates: Requirements 3.9** (Preservation Property 14 from design)
//
// Non-bug condition: lat ∈ [-90,90] AND lng ∈ [-180,180]
// These tests verify that valid coordinates are processed without error
// on unfixed code. They must PASS on unfixed code.
package handler

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	presencev1 "parkir-pintar/proto/presence/v1"

	"pgregory.net/rapid"
)

// TestDetectArrival_ShouldProcessWithoutError_WhenValidCoordinates generates
// random valid lat in [-90,90] and lng in [-180,180] and verifies the presence
// handler processes them without error. Non-bug condition: valid coordinates.
//
// **Validates: Requirements 3.9**
func TestDetectArrival_ShouldProcessWithoutError_WhenValidCoordinates(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Arrange — generate valid coordinates
		lat := rapid.Float64Range(-90, 90).Draw(t, "latitude")
		lng := rapid.Float64Range(-180, 180).Draw(t, "longitude")
		centerLat := rapid.Float64Range(-90, 90).Draw(t, "centerLat")
		centerLng := rapid.Float64Range(-180, 180).Draw(t, "centerLng")
		radius := rapid.Float64Range(1, 10000).Draw(t, "radius")

		uc := &mockPresenceUsecase{}
		h := NewHandler(uc)

		req := &presencev1.DetectArrivalRequest{
			ReservationId:   "res-valid",
			Latitude:        lat,
			Longitude:       lng,
			CenterLatitude:  centerLat,
			CenterLongitude: centerLng,
			RadiusMeters:    radius,
		}

		// Act
		resp, err := h.DetectArrival(context.Background(), req)

		// Assert — valid coordinates should be processed without error
		require.NoError(t, err, "valid coordinates (lat=%f, lng=%f) should not produce error", lat, lng)
		assert.NotNil(t, resp)
		assert.Equal(t, "res-valid", resp.GetReservationId())
	})
}

// TestDetectWrongSpot_ShouldProcessWithoutError_WhenValidCoordinates verifies
// that DetectWrongSpot processes valid coordinates without error.
// Non-bug condition: valid coordinates.
//
// **Validates: Requirements 3.9**
func TestDetectWrongSpot_ShouldProcessWithoutError_WhenValidCoordinates(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Arrange
		lat := rapid.Float64Range(-90, 90).Draw(t, "latitude")
		lng := rapid.Float64Range(-180, 180).Draw(t, "longitude")
		spotLat := rapid.Float64Range(-90, 90).Draw(t, "spotLat")
		spotLng := rapid.Float64Range(-180, 180).Draw(t, "spotLng")
		threshold := rapid.Float64Range(1, 1000).Draw(t, "threshold")

		uc := &mockPresenceUsecase{}
		h := NewHandler(uc)

		req := &presencev1.DetectWrongSpotRequest{
			ReservationId:   "res-valid",
			Latitude:        lat,
			Longitude:       lng,
			SpotLatitude:    spotLat,
			SpotLongitude:   spotLng,
			ThresholdMeters: threshold,
		}

		// Act
		resp, err := h.DetectWrongSpot(context.Background(), req)

		// Assert
		require.NoError(t, err, "valid coordinates should not produce error")
		assert.NotNil(t, resp)
	})
}
