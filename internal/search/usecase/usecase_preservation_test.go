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

func TestGetAvailability_ShouldReturnCached_WhenCacheHit(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
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

		result, err := uc.GetAvailability(t.Context(), &model.GetAvailabilityRequest{
			VehicleType: vehicleType,
		})

		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, availableCar, result[0].AvailableCar)
		assert.Equal(t, availableMoto, result[0].AvailableMoto)
		assert.Equal(t, floorNumber, result[0].FloorNumber)

		repo.AssertNotCalled(t, "GetAvailabilityByVehicleType")
		redis.AssertExpectations(t)
	})
}

func TestGetFloorMap_ShouldReturnCached_WhenCacheHit(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
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

		result, err := uc.GetFloorMap(t.Context(), &model.GetFloorMapRequest{
			FloorNumber: floorNumber,
		})

		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, spotCode, result[0].SpotCode)

		repo.AssertNotCalled(t, "GetFloorSpots")
	})
}
