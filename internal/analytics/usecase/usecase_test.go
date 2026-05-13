// Package usecase implements the business logic layer for the analytics module.
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
package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"parkir-pintar/internal/analytics/model"
)

// --- Mock Implementations ---

// MockRepository implements repository.Repository using testify/mock.
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) GetHourlyStats(ctx context.Context, startDate, endDate time.Time) ([]model.PeakHourStats, error) {
	args := m.Called(ctx, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.PeakHourStats), args.Error(1)
}

func (m *MockRepository) GetDailyOccupancy(ctx context.Context, days int) ([]model.DailyOccupancy, error) {
	args := m.Called(ctx, days)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.DailyOccupancy), args.Error(1)
}

// --- Test Cases ---

// TestGetPeakHours_ShouldReturnAboveAverageHours_WhenDataExists verifies that
// GetPeakHours returns only hours with above-average occupancy.
func TestGetPeakHours_ShouldReturnAboveAverageHours_WhenDataExists(t *testing.T) {
	// Arrange
	repo := new(MockRepository)

	stats := []model.PeakHourStats{
		{Hour: 8, DayOfWeek: 1, AvgOccupancy: 0.9, AvgReservations: 20, PeakScore: 5.0},
		{Hour: 9, DayOfWeek: 1, AvgOccupancy: 0.85, AvgReservations: 18, PeakScore: 4.5},
		{Hour: 14, DayOfWeek: 1, AvgOccupancy: 0.3, AvgReservations: 5, PeakScore: 1.0},
		{Hour: 2, DayOfWeek: 1, AvgOccupancy: 0.1, AvgReservations: 2, PeakScore: 0.5},
	}
	repo.On("GetHourlyStats", mock.Anything, mock.Anything, mock.Anything).Return(stats, nil)

	uc := NewUsecase(repo)

	// Act
	result, err := uc.GetPeakHours(t.Context())

	// Assert
	require.NoError(t, err)
	// Average occupancy = (0.9 + 0.85 + 0.3 + 0.1) / 4 = 0.5375
	// Hours above average: 8 (0.9) and 9 (0.85)
	assert.Len(t, result, 2)
	for _, r := range result {
		assert.Greater(t, r.AvgOccupancy, 0.5375)
	}
	repo.AssertExpectations(t)
}

// TestGetPeakHours_ShouldReturnEmpty_WhenNoData verifies that GetPeakHours
// returns an empty slice when no historical data exists.
func TestGetPeakHours_ShouldReturnEmpty_WhenNoData(t *testing.T) {
	// Arrange
	repo := new(MockRepository)
	repo.On("GetHourlyStats", mock.Anything, mock.Anything, mock.Anything).
		Return([]model.PeakHourStats{}, nil)

	uc := NewUsecase(repo)

	// Act
	result, err := uc.GetPeakHours(t.Context())

	// Assert
	require.NoError(t, err)
	assert.Empty(t, result)
	repo.AssertExpectations(t)
}

// TestGetPeakHours_ShouldReturnError_WhenRepositoryFails verifies that
// GetPeakHours propagates repository errors as internal app errors.
func TestGetPeakHours_ShouldReturnError_WhenRepositoryFails(t *testing.T) {
	// Arrange
	repo := new(MockRepository)
	repo.On("GetHourlyStats", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, assert.AnError)

	uc := NewUsecase(repo)

	// Act
	result, err := uc.GetPeakHours(t.Context())

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to retrieve peak hour data")
	repo.AssertExpectations(t)
}

// TestGetIdleHours_ShouldReturnBelowThresholdHours_WhenDataExists verifies that
// GetIdleHours returns only hours with occupancy below 30%.
func TestGetIdleHours_ShouldReturnBelowThresholdHours_WhenDataExists(t *testing.T) {
	// Arrange
	repo := new(MockRepository)

	stats := []model.PeakHourStats{
		{Hour: 8, DayOfWeek: 1, AvgOccupancy: 0.9, AvgReservations: 20, PeakScore: 5.0},
		{Hour: 2, DayOfWeek: 1, AvgOccupancy: 0.1, AvgReservations: 2, PeakScore: 0.5},
		{Hour: 3, DayOfWeek: 1, AvgOccupancy: 0.15, AvgReservations: 3, PeakScore: 0.6},
		{Hour: 14, DayOfWeek: 1, AvgOccupancy: 0.5, AvgReservations: 10, PeakScore: 2.0},
	}
	repo.On("GetHourlyStats", mock.Anything, mock.Anything, mock.Anything).Return(stats, nil)

	uc := NewUsecase(repo)

	// Act
	result, err := uc.GetIdleHours(t.Context())

	// Assert
	require.NoError(t, err)
	// Hours below 30%: 2 (0.1) and 3 (0.15)
	assert.Len(t, result, 2)
	for _, r := range result {
		assert.Less(t, r.AvgOccupancy, 0.30)
	}
	repo.AssertExpectations(t)
}

// TestGetIdleHours_ShouldReturnEmpty_WhenAllHoursAboveThreshold verifies that
// GetIdleHours returns an empty slice when no hours are below the idle threshold.
func TestGetIdleHours_ShouldReturnEmpty_WhenAllHoursAboveThreshold(t *testing.T) {
	// Arrange
	repo := new(MockRepository)

	stats := []model.PeakHourStats{
		{Hour: 8, DayOfWeek: 1, AvgOccupancy: 0.5, AvgReservations: 10, PeakScore: 2.0},
		{Hour: 9, DayOfWeek: 1, AvgOccupancy: 0.7, AvgReservations: 15, PeakScore: 3.0},
	}
	repo.On("GetHourlyStats", mock.Anything, mock.Anything, mock.Anything).Return(stats, nil)

	uc := NewUsecase(repo)

	// Act
	result, err := uc.GetIdleHours(t.Context())

	// Assert
	require.NoError(t, err)
	assert.Empty(t, result)
	repo.AssertExpectations(t)
}

// TestPredictResources_ShouldReturnPredictions_WhenSufficientData verifies that
// PredictResources generates hourly predictions for the given horizon.
func TestPredictResources_ShouldReturnPredictions_WhenSufficientData(t *testing.T) {
	// Arrange
	repo := new(MockRepository)

	now := time.Now()
	dailyData := []model.DailyOccupancy{
		{Date: now.AddDate(0, 0, -3), AvgOccupancy: 0.4, TotalSpots: 100, OccupiedSpots: 40},
		{Date: now.AddDate(0, 0, -2), AvgOccupancy: 0.5, TotalSpots: 100, OccupiedSpots: 50},
		{Date: now.AddDate(0, 0, -1), AvgOccupancy: 0.6, TotalSpots: 100, OccupiedSpots: 60},
	}
	repo.On("GetDailyOccupancy", mock.Anything, 30).Return(dailyData, nil)

	uc := NewUsecase(repo)

	// Act — predict 3 hours ahead
	result, err := uc.PredictResources(t.Context(), 3*time.Hour)

	// Assert
	require.NoError(t, err)
	assert.Len(t, result, 3)
	for _, p := range result {
		assert.True(t, p.Timestamp.After(now))
		assert.GreaterOrEqual(t, p.PredictedOccupancy, 0.0)
		assert.LessOrEqual(t, p.PredictedOccupancy, 1.0)
		assert.GreaterOrEqual(t, p.RecommendedInstances, 1)
		assert.Greater(t, p.Confidence, 0.0)
		assert.LessOrEqual(t, p.Confidence, 1.0)
	}
	repo.AssertExpectations(t)
}

// TestPredictResources_ShouldReturnError_WhenInsufficientData verifies that
// PredictResources returns an error when fewer than 2 days of data exist.
func TestPredictResources_ShouldReturnError_WhenInsufficientData(t *testing.T) {
	// Arrange
	repo := new(MockRepository)

	dailyData := []model.DailyOccupancy{
		{Date: time.Now().AddDate(0, 0, -1), AvgOccupancy: 0.5, TotalSpots: 100, OccupiedSpots: 50},
	}
	repo.On("GetDailyOccupancy", mock.Anything, 30).Return(dailyData, nil)

	uc := NewUsecase(repo)

	// Act
	result, err := uc.PredictResources(t.Context(), 24*time.Hour)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "at least 2 days")
	repo.AssertExpectations(t)
}

// TestPredictResources_ShouldReturnError_WhenHorizonNegative verifies that
// PredictResources rejects a non-positive horizon duration.
func TestPredictResources_ShouldReturnError_WhenHorizonNegative(t *testing.T) {
	// Arrange
	repo := new(MockRepository)
	uc := NewUsecase(repo)

	// Act
	result, err := uc.PredictResources(t.Context(), -1*time.Hour)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "horizon must be positive")
	repo.AssertExpectations(t)
}

// TestGetUsagePatterns_ShouldReturnPattern_WhenDataExists verifies that
// GetUsagePatterns correctly identifies peak and idle hours.
func TestGetUsagePatterns_ShouldReturnPattern_WhenDataExists(t *testing.T) {
	// Arrange
	repo := new(MockRepository)

	stats := []model.PeakHourStats{
		{Hour: 8, DayOfWeek: 1, AvgOccupancy: 0.9, AvgReservations: 20, PeakScore: 5.0},
		{Hour: 9, DayOfWeek: 1, AvgOccupancy: 0.85, AvgReservations: 18, PeakScore: 4.5},
		{Hour: 2, DayOfWeek: 1, AvgOccupancy: 0.1, AvgReservations: 2, PeakScore: 0.5},
		{Hour: 3, DayOfWeek: 1, AvgOccupancy: 0.15, AvgReservations: 3, PeakScore: 0.6},
	}
	repo.On("GetHourlyStats", mock.Anything, mock.Anything, mock.Anything).Return(stats, nil)

	uc := NewUsecase(repo)

	// Act
	result, err := uc.GetUsagePatterns(t.Context())

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "last_30_days", result.Period)
	assert.Greater(t, result.AvgUtilization, 0.0)
	assert.NotEmpty(t, result.PeakHours)
	assert.NotEmpty(t, result.IdleHours)
	// Hours 2 and 3 should be idle (< 30%)
	assert.Contains(t, result.IdleHours, 2)
	assert.Contains(t, result.IdleHours, 3)
	repo.AssertExpectations(t)
}

// TestGetUsagePatterns_ShouldReturnEmptyPattern_WhenNoData verifies that
// GetUsagePatterns returns a zero-value pattern when no data exists.
func TestGetUsagePatterns_ShouldReturnEmptyPattern_WhenNoData(t *testing.T) {
	// Arrange
	repo := new(MockRepository)
	repo.On("GetHourlyStats", mock.Anything, mock.Anything, mock.Anything).
		Return([]model.PeakHourStats{}, nil)

	uc := NewUsecase(repo)

	// Act
	result, err := uc.GetUsagePatterns(t.Context())

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "last_30_days", result.Period)
	assert.Equal(t, 0.0, result.AvgUtilization)
	assert.Empty(t, result.PeakHours)
	assert.Empty(t, result.IdleHours)
	repo.AssertExpectations(t)
}

// TestGetUsagePatterns_ShouldReturnError_WhenRepositoryFails verifies that
// GetUsagePatterns propagates repository errors.
func TestGetUsagePatterns_ShouldReturnError_WhenRepositoryFails(t *testing.T) {
	// Arrange
	repo := new(MockRepository)
	repo.On("GetHourlyStats", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, assert.AnError)

	uc := NewUsecase(repo)

	// Act
	result, err := uc.GetUsagePatterns(t.Context())

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to retrieve usage pattern data")
	repo.AssertExpectations(t)
}
