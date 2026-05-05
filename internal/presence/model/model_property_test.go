// Package model defines domain structs and geofence detection functions
// for the presence module.
//
// Property-based tests for geofence detection using pgregory.net/rapid.
// These tests verify Property 8 from the design document.
//
// Best practices applied (from coding standards KB):
// - rapid.Custom generators for constrained GPS coordinate inputs
// - AAA pattern: Arrange → Act → Assert
// - testify/assert for assertions
// - t.Context() for context (Go 1.24+)
// - No mocks — tests exercise pure functions directly
package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"pgregory.net/rapid"
)

// genLatLng generates a random (latitude, longitude) pair within valid GPS ranges.
func genLatLng() *rapid.Generator[[2]float64] {
	return rapid.Custom[[2]float64](func(t *rapid.T) [2]float64 {
		lat := rapid.Float64Range(-90, 90).Draw(t, "lat")
		lng := rapid.Float64Range(-180, 180).Draw(t, "lng")
		return [2]float64{lat, lng}
	})
}

// genRadius generates a random geofence radius between 1 and 10,000 meters.
func genRadius() *rapid.Generator[float64] {
	return rapid.Float64Range(1, 10_000)
}

// --- Property 8: Geofence Detection ---

// TestProperty8_DetectArrivalConsistentWithHaversine verifies that for any random
// GPS coordinate, center, and radius, DetectArrival returns true if and only if
// HaversineDistance(lat, lng, centerLat, centerLng) <= radius.
//
// **Validates: Requirements 12.2, 12.3**
func TestProperty8_DetectArrivalConsistentWithHaversine(t *testing.T) {
	_ = t.Context()

	rapid.Check(t, func(t *rapid.T) {
		// Arrange
		point := genLatLng().Draw(t, "point")
		center := genLatLng().Draw(t, "center")
		radius := genRadius().Draw(t, "radius")

		lat, lng := point[0], point[1]
		centerLat, centerLng := center[0], center[1]

		// Act
		arrived := DetectArrival(lat, lng, centerLat, centerLng, radius)
		distance := HaversineDistance(lat, lng, centerLat, centerLng)

		// Assert — DetectArrival == (distance <= radius)
		expected := distance <= radius
		assert.Equal(t, expected, arrived,
			"DetectArrival should return %v when distance=%.2f, radius=%.2f",
			expected, distance, radius)
	})
}

// TestProperty8_HaversineDistanceNonNegative verifies that for any two GPS coordinates,
// the Haversine distance is always non-negative.
//
// **Validates: Requirements 12.2, 12.3**
func TestProperty8_HaversineDistanceNonNegative(t *testing.T) {
	_ = t.Context()

	rapid.Check(t, func(t *rapid.T) {
		// Arrange
		p1 := genLatLng().Draw(t, "p1")
		p2 := genLatLng().Draw(t, "p2")

		// Act
		distance := HaversineDistance(p1[0], p1[1], p2[0], p2[1])

		// Assert
		assert.GreaterOrEqual(t, distance, 0.0,
			"Haversine distance must be non-negative")
	})
}

// TestProperty8_HaversineDistanceZeroForIdentical verifies that for any GPS coordinate,
// the Haversine distance to itself is exactly 0.
//
// **Validates: Requirements 12.2, 12.3**
func TestProperty8_HaversineDistanceZeroForIdentical(t *testing.T) {
	_ = t.Context()

	rapid.Check(t, func(t *rapid.T) {
		// Arrange
		point := genLatLng().Draw(t, "point")

		// Act
		distance := HaversineDistance(point[0], point[1], point[0], point[1])

		// Assert
		assert.Equal(t, 0.0, distance,
			"distance to self must be 0 for lat=%.6f, lng=%.6f", point[0], point[1])
	})
}
