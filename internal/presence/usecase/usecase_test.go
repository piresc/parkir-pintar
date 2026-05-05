// Package usecase implements the business logic layer for the presence domain.
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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"parkir-pintar/internal/presence/model"
	"parkir-pintar/internal/presence/repository"
)

// --- Mock Implementations ---

// MockRepository implements repository.Repository using testify/mock.
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) SavePresenceLog(ctx context.Context, log *model.PresenceLog) error {
	args := m.Called(ctx, log)
	return args.Error(0)
}

func (m *MockRepository) GetPresenceByReservation(ctx context.Context, reservationID string) (*model.PresenceLog, error) {
	args := m.Called(ctx, reservationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.PresenceLog), args.Error(1)
}

func (m *MockRepository) CleanupPresence(ctx context.Context, reservationID string) error {
	args := m.Called(ctx, reservationID)
	return args.Error(0)
}

// MockRedisClient implements RedisClient using testify/mock.
type MockRedisClient struct {
	mock.Mock
}

func (m *MockRedisClient) XAdd(ctx context.Context, stream string, values map[string]interface{}) error {
	args := m.Called(ctx, stream, values)
	return args.Error(0)
}

func (m *MockRedisClient) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

// MockNATSClient implements NATSClient using testify/mock.
type MockNATSClient struct {
	mock.Mock
}

func (m *MockNATSClient) Publish(subject string, data []byte) error {
	args := m.Called(subject, data)
	return args.Error(0)
}

// --- Test Cases ---

// TestStreamLocation_ShouldStoreInRedisAndPostgres_WhenValidUpdate verifies
// that StreamLocation saves to Redis stream and persists to PostgreSQL.
func TestStreamLocation_ShouldStoreInRedisAndPostgres_WhenValidUpdate(t *testing.T) {
	// Arrange
	repo := new(MockRepository)
	redis := new(MockRedisClient)
	nats := new(MockNATSClient)

	ts := time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC)
	update := &model.LocationUpdate{
		ReservationID: "res-1",
		Latitude:      -6.2088,
		Longitude:     106.8456,
		Accuracy:      10.0,
		Timestamp:     ts,
	}

	redis.On("XAdd", mock.Anything, "presence:res-1", mock.MatchedBy(func(v map[string]interface{}) bool {
		return v["reservation_id"] == "res-1" && v["latitude"] == -6.2088
	})).Return(nil)

	repo.On("SavePresenceLog", mock.Anything, mock.MatchedBy(func(log *model.PresenceLog) bool {
		return log.ReservationID == "res-1" &&
			log.Latitude == -6.2088 &&
			log.Longitude == 106.8456 &&
			log.Accuracy == 10.0 &&
			log.ID != ""
	})).Return(nil)

	uc := NewUsecase(repo, redis, nats)

	// Act
	err := uc.StreamLocation(t.Context(), update)

	// Assert
	require.NoError(t, err)
	redis.AssertExpectations(t)
	repo.AssertExpectations(t)
}

// TestDetectArrival_ShouldPublishEvent_WhenInsideGeofence verifies that
// DetectArrival returns arrived=true and publishes presence.arrival event
// when the driver is inside the geofence radius.
func TestDetectArrival_ShouldPublishEvent_WhenInsideGeofence(t *testing.T) {
	// Arrange
	repo := new(MockRepository)
	redis := new(MockRedisClient)
	natsClient := new(MockNATSClient)

	// Same coordinates → distance = 0, well within 100m radius
	natsClient.On("Publish", "presence.arrival", mock.Anything).Return(nil)

	uc := NewUsecase(repo, redis, natsClient)

	// Act
	result, err := uc.DetectArrival(t.Context(),
		-6.2088, 106.8456, // driver position
		-6.2088, 106.8456, // geofence center (same point)
		100.0,             // radius meters
		"res-1",
	)

	// Assert
	require.NoError(t, err)
	assert.True(t, result.Arrived)
	assert.Equal(t, "res-1", result.ReservationID)
	assert.False(t, result.DetectedAt.IsZero())
	natsClient.AssertExpectations(t)
}

// TestDetectArrival_ShouldNotPublish_WhenOutsideGeofence verifies that
// DetectArrival returns arrived=false and does NOT publish an event
// when the driver is outside the geofence radius.
func TestDetectArrival_ShouldNotPublish_WhenOutsideGeofence(t *testing.T) {
	// Arrange
	repo := new(MockRepository)
	redis := new(MockRedisClient)
	natsClient := new(MockNATSClient)

	uc := NewUsecase(repo, redis, natsClient)

	// Act — driver is far from center (different city)
	result, err := uc.DetectArrival(t.Context(),
		-7.2575, 112.7521, // Surabaya
		-6.2088, 106.8456, // Jakarta
		100.0,             // 100m radius
		"res-2",
	)

	// Assert
	require.NoError(t, err)
	assert.False(t, result.Arrived)
	// Publish should NOT have been called
	natsClient.AssertNotCalled(t, "Publish")
}

// TestDetectWrongSpot_ShouldPublishEvent_WhenDistanceExceedsThreshold verifies
// that DetectWrongSpot returns is_wrong_spot=true and publishes presence.wrong_spot
// event when the distance exceeds the threshold.
func TestDetectWrongSpot_ShouldPublishEvent_WhenDistanceExceedsThreshold(t *testing.T) {
	// Arrange
	repo := new(MockRepository)
	redis := new(MockRedisClient)
	natsClient := new(MockNATSClient)

	natsClient.On("Publish", "presence.wrong_spot", mock.Anything).Return(nil)

	uc := NewUsecase(repo, redis, natsClient)

	// Act — driver is far from assigned spot
	result, err := uc.DetectWrongSpot(t.Context(),
		-7.2575, 112.7521, // driver position (Surabaya)
		-6.2088, 106.8456, // assigned spot (Jakarta)
		50.0,              // 50m threshold
		"res-3",
	)

	// Assert
	require.NoError(t, err)
	assert.True(t, result.IsWrongSpot)
	assert.Greater(t, result.DistanceMeters, 50.0)
	natsClient.AssertExpectations(t)
}

// TestDetectWrongSpot_ShouldNotPublish_WhenWithinThreshold verifies that
// DetectWrongSpot returns is_wrong_spot=false when the driver is within threshold.
func TestDetectWrongSpot_ShouldNotPublish_WhenWithinThreshold(t *testing.T) {
	// Arrange
	repo := new(MockRepository)
	redis := new(MockRedisClient)
	natsClient := new(MockNATSClient)

	uc := NewUsecase(repo, redis, natsClient)

	// Act — same coordinates, distance = 0
	result, err := uc.DetectWrongSpot(t.Context(),
		-6.2088, 106.8456,
		-6.2088, 106.8456,
		50.0,
		"res-4",
	)

	// Assert
	require.NoError(t, err)
	assert.False(t, result.IsWrongSpot)
	assert.Equal(t, 0.0, result.DistanceMeters)
	natsClient.AssertNotCalled(t, "Publish")
}

// TestGetPresence_ShouldReturnLog_WhenExists verifies that GetPresence
// returns the latest presence log from the repository.
func TestGetPresence_ShouldReturnLog_WhenExists(t *testing.T) {
	// Arrange
	repo := new(MockRepository)
	redis := new(MockRedisClient)
	natsClient := new(MockNATSClient)

	expected := &model.PresenceLog{
		ID:            "log-1",
		ReservationID: "res-1",
		Latitude:      -6.2088,
		Longitude:     106.8456,
		Accuracy:      5.0,
		RecordedAt:    time.Now(),
	}
	repo.On("GetPresenceByReservation", mock.Anything, "res-1").Return(expected, nil)

	uc := NewUsecase(repo, redis, natsClient)

	// Act
	result, err := uc.GetPresence(t.Context(), "res-1")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "log-1", result.ID)
	assert.Equal(t, "res-1", result.ReservationID)
	assert.Equal(t, -6.2088, result.Latitude)
	repo.AssertExpectations(t)
}

// TestGetPresence_ShouldReturnError_WhenNotFound verifies that GetPresence
// returns an error when no presence log exists for the reservation.
func TestGetPresence_ShouldReturnError_WhenNotFound(t *testing.T) {
	// Arrange
	repo := new(MockRepository)
	redis := new(MockRedisClient)
	natsClient := new(MockNATSClient)

	repo.On("GetPresenceByReservation", mock.Anything, "res-missing").Return(nil, repository.ErrNotFound)

	uc := NewUsecase(repo, redis, natsClient)

	// Act
	result, err := uc.GetPresence(t.Context(), "res-missing")

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	repo.AssertExpectations(t)
}
