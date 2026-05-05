// Package model defines domain structs and geofence detection functions
// for the presence module.
//
// Best practices applied (from coding standards KB):
// - Table-driven tests with t.Run() subtests for comprehensive geofence coverage
// - AAA pattern: Arrange → Act → Assert
// - testify/assert for assertions
// - t.Context() for context (Go 1.24+)
// - Each subtest is isolated and self-descriptive
package model

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectArrival_ShouldReturnCorrectResult(t *testing.T) {
	// Geofence center: Monas, Jakarta (-6.1754, 106.8272)
	centerLat := -6.1754
	centerLng := 106.8272
	radius := 50.0 // 50 meters

	tests := []struct {
		name     string
		lat      float64
		lng      float64
		expected bool
	}{
		{
			name:     "inside radius returns true",
			lat:      -6.1754,
			lng:      106.8272,
			expected: true,
		},
		{
			name:     "outside radius returns false",
			lat:      -6.1770,
			lng:      106.8290,
			expected: false,
		},
		{
			name:     "on boundary returns true",
			lat:      -6.1754 + (50.0 / earthRadiusMeters * (180.0 / math.Pi)),
			lng:      106.8272,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			_ = t.Context()

			// Act
			result := DetectArrival(tt.lat, tt.lng, centerLat, centerLng, radius)

			// Assert
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHaversineDistance_ShouldComputeCorrectDistance(t *testing.T) {
	tests := []struct {
		name             string
		lat1, lng1       float64
		lat2, lng2       float64
		expectedDistance float64
		tolerancePercent float64
	}{
		{
			name:             "identical coordinates returns zero",
			lat1:             -6.1754,
			lng1:             106.8272,
			lat2:             -6.1754,
			lng2:             106.8272,
			expectedDistance: 0,
			tolerancePercent: 0,
		},
		{
			name: "Jakarta to Bandung approx 116km",
			// Jakarta: -6.2088, 106.8456
			// Bandung: -6.9175, 107.6191
			lat1:             -6.2088,
			lng1:             106.8456,
			lat2:             -6.9175,
			lng2:             107.6191,
			expectedDistance: 116_236, // ~116 km (Haversine)
			tolerancePercent: 1,       // within 1%
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			_ = t.Context()

			// Act
			distance := HaversineDistance(tt.lat1, tt.lng1, tt.lat2, tt.lng2)

			// Assert
			if tt.expectedDistance == 0 {
				assert.Equal(t, 0.0, distance, "distance for identical coordinates must be 0")
			} else {
				tolerance := tt.expectedDistance * tt.tolerancePercent / 100
				assert.InDelta(t, tt.expectedDistance, distance, tolerance,
					"distance should be within %.0f%% of expected %.0fm", tt.tolerancePercent, tt.expectedDistance)
			}
		})
	}
}
