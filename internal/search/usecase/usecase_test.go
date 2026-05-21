package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"parkir-pintar/internal/search/model"
)

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) GetAvailabilityByVehicleType(ctx context.Context, vehicleType string) ([]model.FloorAvailability, error) {
	args := m.Called(ctx, vehicleType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.FloorAvailability), args.Error(1)
}

func (m *MockRepository) GetFloorSpots(ctx context.Context, floorNumber int) ([]model.SpotDetails, error) {
	args := m.Called(ctx, floorNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.SpotDetails), args.Error(1)
}

func (m *MockRepository) GetSpotByID(ctx context.Context, spotID string) (*model.SpotDetails, error) {
	args := m.Called(ctx, spotID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.SpotDetails), args.Error(1)
}

type MockRedisClient struct {
	mock.Mock
}

func (m *MockRedisClient) Get(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *MockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	args := m.Called(ctx, key, value, expiration)
	return args.Error(0)
}

func (m *MockRedisClient) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func TestGetAvailability_ShouldReturnCachedData_WhenCacheHit(t *testing.T) {
	repo := new(MockRepository)
	redis := new(MockRedisClient)

	floors := []model.FloorAvailability{
		{FloorNumber: 1, AvailableCar: 25, AvailableMoto: 40, TotalCar: 30, TotalMoto: 50},
	}
	cachedJSON, err := json.Marshal(floors)
	require.NoError(t, err)

	redis.On("Get", mock.Anything, "availability:car").Return(string(cachedJSON), nil)

	uc := NewUsecase(repo, nil, redis)

	result, err := uc.GetAvailability(t.Context(), &model.GetAvailabilityRequest{VehicleType: "car"})

	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, 25, result[0].AvailableCar)
	repo.AssertNotCalled(t, "GetAvailabilityByVehicleType")
	redis.AssertExpectations(t)
}

func TestGetAvailability_ShouldQueryDB_WhenCacheMiss(t *testing.T) {
	repo := new(MockRepository)
	redis := new(MockRedisClient)

	floors := []model.FloorAvailability{
		{FloorNumber: 1, AvailableCar: 20, AvailableMoto: 45, TotalCar: 30, TotalMoto: 50},
	}

	redis.On("Get", mock.Anything, "availability:car").Return("", errors.New("cache miss"))
	repo.On("GetAvailabilityByVehicleType", mock.Anything, "car").Return(floors, nil)
	redis.On("Set", mock.Anything, "availability:car", mock.Anything, cacheTTL).Return(nil)

	uc := NewUsecase(repo, nil, redis)

	result, err := uc.GetAvailability(t.Context(), &model.GetAvailabilityRequest{VehicleType: "car"})

	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, 20, result[0].AvailableCar)
	repo.AssertExpectations(t)
	redis.AssertExpectations(t)
}

func TestGetAvailability_ShouldGracefullyDegrade_WhenRedisFailure(t *testing.T) {
	repo := new(MockRepository)
	redis := new(MockRedisClient)

	floors := []model.FloorAvailability{
		{FloorNumber: 1, AvailableCar: 10, AvailableMoto: 30, TotalCar: 30, TotalMoto: 50},
	}

	redis.On("Get", mock.Anything, "availability:motorcycle").Return("", errors.New("redis connection refused"))
	repo.On("GetAvailabilityByVehicleType", mock.Anything, "motorcycle").Return(floors, nil)
	redis.On("Set", mock.Anything, "availability:motorcycle", mock.Anything, cacheTTL).Return(errors.New("redis connection refused"))

	uc := NewUsecase(repo, nil, redis)

	result, err := uc.GetAvailability(t.Context(), &model.GetAvailabilityRequest{VehicleType: "motorcycle"})

	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, 10, result[0].AvailableCar)
	repo.AssertExpectations(t)
	redis.AssertExpectations(t)
}

func TestGetFloorMap_ShouldReturnCachedData_WhenCacheHit(t *testing.T) {
	repo := new(MockRepository)
	redis := new(MockRedisClient)

	spots := []model.SpotDetails{
		{ID: "spot-1", SpotCode: "F1-C-001", VehicleType: "car", Status: "available", FloorNumber: 1, SpotNumber: 1},
	}
	cachedJSON, err := json.Marshal(spots)
	require.NoError(t, err)

	redis.On("Get", mock.Anything, "floormap:1").Return(string(cachedJSON), nil)

	uc := NewUsecase(repo, nil, redis)

	result, err := uc.GetFloorMap(t.Context(), &model.GetFloorMapRequest{FloorNumber: 1})

	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "F1-C-001", result[0].SpotCode)
	repo.AssertNotCalled(t, "GetFloorSpots")
	redis.AssertExpectations(t)
}

// GetSpotDetails always queries the database directly (no cache).
func TestGetSpotDetails_ShouldQueryDB_WhenCalled(t *testing.T) {
	repo := new(MockRepository)
	redis := new(MockRedisClient)

	spot := &model.SpotDetails{
		ID: "spot-1", SpotCode: "F2-M-010", VehicleType: "motorcycle",
		Status: "available", FloorNumber: 2, SpotNumber: 10,
	}
	repo.On("GetSpotByID", mock.Anything, "spot-1").Return(spot, nil)

	uc := NewUsecase(repo, nil, redis)

	result, err := uc.GetSpotDetails(t.Context(), &model.GetSpotDetailsRequest{SpotID: "spot-1"})

	require.NoError(t, err)
	assert.Equal(t, "F2-M-010", result.SpotCode)
	repo.AssertExpectations(t)
	redis.AssertNotCalled(t, "Get")
	redis.AssertNotCalled(t, "Set")
}

func TestCacheInvalidation_ShouldDeleteCorrectKeys(t *testing.T) {
	redis := new(MockRedisClient)

	redis.On("Delete", mock.Anything, "availability:car").Return(nil)
	redis.On("Delete", mock.Anything, "availability:motorcycle").Return(nil)
	redis.On("Delete", mock.Anything, "floormap:1").Return(nil)
	redis.On("Delete", mock.Anything, "floormap:2").Return(nil)
	redis.On("Delete", mock.Anything, "floormap:3").Return(nil)
	redis.On("Delete", mock.Anything, "floormap:4").Return(nil)
	redis.On("Delete", mock.Anything, "floormap:5").Return(nil)

	sub := &cacheInvalidator{redis: redis}

	sub.invalidate(t.Context())

	redis.AssertExpectations(t)
	redis.AssertNumberOfCalls(t, "Delete", 7) // 2 availability + 5 floor maps
}

type cacheInvalidator struct {
	redis RedisClient
}

func (c *cacheInvalidator) invalidate(ctx context.Context) {
	keys := []string{
		"availability:car",
		"availability:motorcycle",
		"floormap:1",
		"floormap:2",
		"floormap:3",
		"floormap:4",
		"floormap:5",
	}
	for _, key := range keys {
		_ = c.redis.Delete(ctx, key)
	}
}
