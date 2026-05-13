package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"

	"parkir-pintar/internal/search/model"
)

// BenchmarkGetAvailability_CacheHit measures performance of the hot path
// when data is served from Redis cache (expected ~99% of requests).
func BenchmarkGetAvailability_CacheHit(b *testing.B) {
	repo := new(MockRepository)
	redis := new(MockRedisClient)

	floors := []model.FloorAvailability{
		{FloorNumber: 1, AvailableCar: 25, AvailableMoto: 40, TotalCar: 30, TotalMoto: 50},
		{FloorNumber: 2, AvailableCar: 20, AvailableMoto: 35, TotalCar: 30, TotalMoto: 50},
		{FloorNumber: 3, AvailableCar: 15, AvailableMoto: 30, TotalCar: 30, TotalMoto: 50},
		{FloorNumber: 4, AvailableCar: 10, AvailableMoto: 25, TotalCar: 30, TotalMoto: 50},
		{FloorNumber: 5, AvailableCar: 5, AvailableMoto: 20, TotalCar: 30, TotalMoto: 50},
	}
	cachedJSON, _ := json.Marshal(floors)

	redis.On("Get", mock.Anything, "availability:car").Return(string(cachedJSON), nil)

	uc := NewUsecase(repo, redis)
	ctx := context.Background()
	req := &model.GetAvailabilityRequest{VehicleType: "car"}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = uc.GetAvailability(ctx, req)
	}
}

// BenchmarkGetAvailability_CacheMiss measures performance when cache misses
// and data must be fetched from the repository then cached.
func BenchmarkGetAvailability_CacheMiss(b *testing.B) {
	repo := new(MockRepository)
	redis := new(MockRedisClient)

	floors := []model.FloorAvailability{
		{FloorNumber: 1, AvailableCar: 25, AvailableMoto: 40, TotalCar: 30, TotalMoto: 50},
		{FloorNumber: 2, AvailableCar: 20, AvailableMoto: 35, TotalCar: 30, TotalMoto: 50},
		{FloorNumber: 3, AvailableCar: 15, AvailableMoto: 30, TotalCar: 30, TotalMoto: 50},
	}

	redis.On("Get", mock.Anything, mock.Anything).Return("", errors.New("cache miss"))
	repo.On("GetAvailabilityByVehicleType", mock.Anything, "car").Return(floors, nil)
	redis.On("Set", mock.Anything, mock.Anything, mock.Anything, cacheTTL).Return(nil)

	uc := NewUsecase(repo, redis)
	ctx := context.Background()
	req := &model.GetAvailabilityRequest{VehicleType: "car"}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = uc.GetAvailability(ctx, req)
	}
}

// BenchmarkGetFloorMap_CacheHit measures floor map retrieval from cache.
func BenchmarkGetFloorMap_CacheHit(b *testing.B) {
	repo := new(MockRepository)
	redis := new(MockRedisClient)

	spots := make([]model.SpotDetails, 80)
	for i := range spots {
		spots[i] = model.SpotDetails{
			ID:          "spot-" + string(rune('a'+i%26)),
			FloorNumber: 1,
			SpotNumber:  i + 1,
			VehicleType: "car",
			SpotCode:    "F1-C-" + string(rune('0'+i%10)),
			Status:      "available",
		}
	}
	cachedJSON, _ := json.Marshal(spots)

	redis.On("Get", mock.Anything, "floormap:1").Return(string(cachedJSON), nil)

	uc := NewUsecase(repo, redis)
	ctx := context.Background()
	req := &model.GetFloorMapRequest{FloorNumber: 1}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = uc.GetFloorMap(ctx, req)
	}
}
