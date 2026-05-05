// Package usecase provides preservation property tests for cache hit behavior.
//
// Best practices applied (from Go testify coding standards KB):
// - Test naming: Test[FunctionName]_Should[ExpectedResult]_When[Condition]
// - AAA pattern: Arrange → Act → Assert
// - testify/mock for mock implementations of all dependency interfaces
// - testify/assert and testify/require for assertions
// - Each test is isolated with its own mock setup
// - AssertExpectations(t) called on all mocks to verify interactions
// - Mock at interface boundaries rather than concrete implementations
//
// **Validates: Requirements 3.10** (Preservation Property 14 from design)
//
// Non-bug condition: cache hit
// These tests verify that cached availability data is returned without DB query
// on unfixed code. They must PASS on unfixed code.
package usecase

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"parkir-pintar/internal/search/model"

	"pgregory.net/rapid"
)

// TestGetAvailability_ShouldReturnCached_WhenCacheHit verifies that
// GetAvailability returns cached data without querying the database.
// Non-bug condition: cache hit.
//
// **Validates: Requirements 3.10**
func TestGetAvailability_ShouldReturnCached_WhenCacheHit(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Arrange — generate random availability data
		availableCar := rapid.IntRange(0, 50).Draw(t, "availableCar")
		availableMoto := rapid.IntRange(0, 80).Draw(t, "availableMoto")
		floorNumber := rapid.IntRange(1, 5).Draw(t, "floorNumber")
		vehicleType := rapid.SampledFrom([]string{"car", "motorcycle"}).Draw(t, "vehicleType")

		floors := []model.FloorAvailability{
			{
				FloorNumber:   floorNumber,
				AvailableCar:  availableCar,
				AvailableMoto: availableMoto,
				TotalCar:      50,
				TotalMoto:     80,
			},
		}
		cachedJSON, err := json.Marshal(floors)
		require.NoError(t, err)

		repo := new(MockRepository)
		redis := new(MockRedisClient)

		cacheKey := "availability:" + vehicleType
		redis.On("Get", mock.Anything, cacheKey).Return(string(cachedJSON), nil)

		uc := NewUsecase(repo, redis)

		// Act
		result, err := uc.GetAvailability(t.Context(), &model.GetAvailabilityRequest{
			VehicleType: vehicleType,
		})

		// Assert — should return cached data without DB query
		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, availableCar, result[0].AvailableCar)
		assert.Equal(t, availableMoto, result[0].AvailableMoto)
		assert.Equal(t, floorNumber, result[0].FloorNumber)

		// Repository should NOT have been called — cache hit
		repo.AssertNotCalled(t, "GetAvailabilityByVehicleType")
		redis.AssertExpectations(t)
	})
}

// TestGetFloorMap_ShouldReturnCached_WhenCacheHit verifies that
// GetFloorMap returns cached data without querying the database.
// Non-bug condition: cache hit.
//
// **Validates: Requirements 3.10**
func TestGetFloorMap_ShouldReturnCached_WhenCacheHit(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Arrange
		floorNumber := rapid.IntRange(1, 5).Draw(t, "floorNumber")
		spotCode := rapid.StringMatching(`F[1-5]-[CM]-[0-9]{3}`).Draw(t, "spotCode")

		spots := []model.SpotDetails{
			{
				ID:          "spot-cached",
				SpotCode:    spotCode,
				FloorNumber: floorNumber,
				VehicleType: "car",
				Status:      "available",
			},
		}
		cachedJSON, err := json.Marshal(spots)
		require.NoError(t, err)

		repo := new(MockRepository)
		redis := new(MockRedisClient)

		redis.On("Get", mock.Anything, mock.MatchedBy(func(key string) bool {
			return true // match any key for this test
		})).Return(string(cachedJSON), nil)

		uc := NewUsecase(repo, redis)

		// Act
		result, err := uc.GetFloorMap(t.Context(), &model.GetFloorMapRequest{
			FloorNumber: floorNumber,
		})

		// Assert — should return cached data without DB query
		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, spotCode, result[0].SpotCode)

		// Repository should NOT have been called — cache hit
		repo.AssertNotCalled(t, "GetFloorSpots")
	})
}
