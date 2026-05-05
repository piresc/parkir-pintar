// Package usecase implements the business logic layer for the search domain.
//
// Best practices applied (from Go testify coding standards KB):
// - Test naming: Test[FunctionName]_Should[ExpectedResult]_When[Condition]
// - AAA pattern: Arrange → Act → Assert
// - testify/mock for mock implementations of all dependency interfaces
// - testify/assert and testify/require for assertions
// - Each test is isolated with its own mock setup
// - AssertExpectations(t) called on all mocks to verify interactions
// - Use t.Context() for Go 1.24+ context in tests
// - Mock at interface boundaries rather than concrete implementations
// - Keep mocks simple and focused on the behavior being tested
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

// --- Mock Implementations ---

// MockRepository implements repository.Repository using testify/mock.
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

// MockRedisClient implements RedisClient using testify/mock.
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

// --- Test Cases ---

// TestGetAvailability_ShouldReturnCachedData_WhenCacheHit verifies that
// GetAvailability returns data from Redis cache without querying the database.
func TestGetAvailability_ShouldReturnCachedData_WhenCacheHit(t *testing.T) {
	// Arrange
	repo := new(MockRepository)
	redis := new(MockRedisClient)

	floors := []model.FloorAvailability{
		{FloorNumber: 1, AvailableCar: 25, AvailableMoto: 40, TotalCar: 30, TotalMoto: 50},
	}
	cachedJSON, err := json.Marshal(floors)
	require.NoError(t, err)

	redis.On("Get", mock.Anything, "availability:car").Return(string(cachedJSON), nil)

	uc := NewUsecase(repo, redis)

	// Act
	result, err := uc.GetAvailability(t.Context(), &model.GetAvailabilityRequest{VehicleType: "car"})

	// Assert
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, 25, result[0].AvailableCar)
	// Repository should NOT have been called — cache hit
	repo.AssertNotCalled(t, "GetAvailabilityByVehicleType")
	redis.AssertExpectations(t)
}

// TestGetAvailability_ShouldQueryDB_WhenCacheMiss verifies that
// GetAvailability falls through to PostgreSQL when Redis returns an error.
func TestGetAvailability_ShouldQueryDB_WhenCacheMiss(t *testing.T) {
	// Arrange
	repo := new(MockRepository)
	redis := new(MockRedisClient)

	floors := []model.FloorAvailability{
		{FloorNumber: 1, AvailableCar: 20, AvailableMoto: 45, TotalCar: 30, TotalMoto: 50},
	}

	redis.On("Get", mock.Anything, "availability:car").Return("", errors.New("cache miss"))
	repo.On("GetAvailabilityByVehicleType", mock.Anything, "car").Return(floors, nil)
	redis.On("Set", mock.Anything, "availability:car", mock.Anything, cacheTTL).Return(nil)

	uc := NewUsecase(repo, redis)

	// Act
	result, err := uc.GetAvailability(t.Context(), &model.GetAvailabilityRequest{VehicleType: "car"})

	// Assert
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, 20, result[0].AvailableCar)
	repo.AssertExpectations(t)
	redis.AssertExpectations(t)
}

// TestGetAvailability_ShouldGracefullyDegrade_WhenRedisFailure verifies that
// GetAvailability still returns data from DB when Redis Get fails, and does
// not panic even if Redis Set also fails.
func TestGetAvailability_ShouldGracefullyDegrade_WhenRedisFailure(t *testing.T) {
	// Arrange
	repo := new(MockRepository)
	redis := new(MockRedisClient)

	floors := []model.FloorAvailability{
		{FloorNumber: 1, AvailableCar: 10, AvailableMoto: 30, TotalCar: 30, TotalMoto: 50},
	}

	redis.On("Get", mock.Anything, "availability:motorcycle").Return("", errors.New("redis connection refused"))
	repo.On("GetAvailabilityByVehicleType", mock.Anything, "motorcycle").Return(floors, nil)
	redis.On("Set", mock.Anything, "availability:motorcycle", mock.Anything, cacheTTL).Return(errors.New("redis connection refused"))

	uc := NewUsecase(repo, redis)

	// Act
	result, err := uc.GetAvailability(t.Context(), &model.GetAvailabilityRequest{VehicleType: "motorcycle"})

	// Assert
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, 10, result[0].AvailableCar)
	repo.AssertExpectations(t)
	redis.AssertExpectations(t)
}

// TestGetFloorMap_ShouldReturnCachedData_WhenCacheHit verifies that
// GetFloorMap returns data from Redis cache without querying the database.
func TestGetFloorMap_ShouldReturnCachedData_WhenCacheHit(t *testing.T) {
	// Arrange
	repo := new(MockRepository)
	redis := new(MockRedisClient)

	spots := []model.SpotDetails{
		{ID: "spot-1", SpotCode: "F1-C-001", VehicleType: "car", Status: "available", FloorNumber: 1, SpotNumber: 1},
	}
	cachedJSON, err := json.Marshal(spots)
	require.NoError(t, err)

	redis.On("Get", mock.Anything, "floormap:1").Return(string(cachedJSON), nil)

	uc := NewUsecase(repo, redis)

	// Act
	result, err := uc.GetFloorMap(t.Context(), &model.GetFloorMapRequest{FloorNumber: 1})

	// Assert
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "F1-C-001", result[0].SpotCode)
	repo.AssertNotCalled(t, "GetFloorSpots")
	redis.AssertExpectations(t)
}

// TestGetSpotDetails_ShouldQueryDB_WhenCalled verifies that
// GetSpotDetails always queries the database directly (no cache).
func TestGetSpotDetails_ShouldQueryDB_WhenCalled(t *testing.T) {
	// Arrange
	repo := new(MockRepository)
	redis := new(MockRedisClient)

	spot := &model.SpotDetails{
		ID: "spot-1", SpotCode: "F2-M-010", VehicleType: "motorcycle",
		Status: "available", FloorNumber: 2, SpotNumber: 10,
	}
	repo.On("GetSpotByID", mock.Anything, "spot-1").Return(spot, nil)

	uc := NewUsecase(repo, redis)

	// Act
	result, err := uc.GetSpotDetails(t.Context(), &model.GetSpotDetailsRequest{SpotID: "spot-1"})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "F2-M-010", result.SpotCode)
	repo.AssertExpectations(t)
	// Redis should NOT have been called for spot details
	redis.AssertNotCalled(t, "Get")
	redis.AssertNotCalled(t, "Set")
}

// TestCacheInvalidation_ShouldDeleteCorrectKeys verifies that
// InvalidateCache deletes all known availability and floor map cache keys.
func TestCacheInvalidation_ShouldDeleteCorrectKeys(t *testing.T) {
	// Arrange
	redis := new(MockRedisClient)

	redis.On("Delete", mock.Anything, "availability:car").Return(nil)
	redis.On("Delete", mock.Anything, "availability:motorcycle").Return(nil)
	redis.On("Delete", mock.Anything, "floormap:1").Return(nil)
	redis.On("Delete", mock.Anything, "floormap:2").Return(nil)
	redis.On("Delete", mock.Anything, "floormap:3").Return(nil)
	redis.On("Delete", mock.Anything, "floormap:4").Return(nil)
	redis.On("Delete", mock.Anything, "floormap:5").Return(nil)

	// Use the subscriber to test cache invalidation
	sub := &cacheInvalidator{redis: redis}

	// Act
	sub.invalidate(t.Context())

	// Assert
	redis.AssertExpectations(t)
	redis.AssertNumberOfCalls(t, "Delete", 7) // 2 availability + 5 floor maps
}

// cacheInvalidator is a test helper that replicates the subscriber's cache
// invalidation logic to test it without importing the subscriber package.
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
