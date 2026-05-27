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

func (m *MockRepository) RecordEvent(ctx context.Context, event model.ReservationEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockRepository) UpsertSpotSnapshot(ctx context.Context, spot model.SpotSnapshot) error {
	args := m.Called(ctx, spot)
	return args.Error(0)
}

func TestGetPeakHours_ShouldReturnAboveAverageHours_WhenDataExists(t *testing.T) {
	repo := new(MockRepository)

	stats := []model.PeakHourStats{
		{Hour: 8, DayOfWeek: 1, AvgOccupancy: 0.9, AvgReservations: 20, PeakScore: 5.0},
		{Hour: 9, DayOfWeek: 1, AvgOccupancy: 0.85, AvgReservations: 18, PeakScore: 4.5},
		{Hour: 14, DayOfWeek: 1, AvgOccupancy: 0.3, AvgReservations: 5, PeakScore: 1.0},
		{Hour: 2, DayOfWeek: 1, AvgOccupancy: 0.1, AvgReservations: 2, PeakScore: 0.5},
	}
	repo.On("GetHourlyStats", mock.Anything, mock.Anything, mock.Anything).Return(stats, nil)

	uc := NewUsecase(repo)

	result, err := uc.GetPeakHours(t.Context())

	require.NoError(t, err)
	assert.Len(t, result, 2)
	for _, r := range result {
		assert.Greater(t, r.AvgOccupancy, 0.5375)
	}
	repo.AssertExpectations(t)
}

func TestGetPeakHours_ShouldReturnEmpty_WhenNoData(t *testing.T) {
	repo := new(MockRepository)
	repo.On("GetHourlyStats", mock.Anything, mock.Anything, mock.Anything).
		Return([]model.PeakHourStats{}, nil)

	uc := NewUsecase(repo)

	result, err := uc.GetPeakHours(t.Context())

	require.NoError(t, err)
	assert.Empty(t, result)
	repo.AssertExpectations(t)
}

func TestGetPeakHours_ShouldReturnError_WhenRepositoryFails(t *testing.T) {
	repo := new(MockRepository)
	repo.On("GetHourlyStats", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, assert.AnError)

	uc := NewUsecase(repo)

	result, err := uc.GetPeakHours(t.Context())

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to retrieve peak hour data")
	repo.AssertExpectations(t)
}

func TestPredictResources_ShouldReturnPredictions_WhenSufficientData(t *testing.T) {
	repo := new(MockRepository)

	now := time.Now()
	dailyData := []model.DailyOccupancy{
		{Date: now.AddDate(0, 0, -3), AvgOccupancy: 0.4, TotalSpots: 100, OccupiedSpots: 40},
		{Date: now.AddDate(0, 0, -2), AvgOccupancy: 0.5, TotalSpots: 100, OccupiedSpots: 50},
		{Date: now.AddDate(0, 0, -1), AvgOccupancy: 0.6, TotalSpots: 100, OccupiedSpots: 60},
	}
	repo.On("GetDailyOccupancy", mock.Anything, 30).Return(dailyData, nil)

	uc := NewUsecase(repo)

	result, err := uc.PredictResources(t.Context(), 3*time.Hour)

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

func TestPredictResources_ShouldReturnError_WhenInsufficientData(t *testing.T) {
	repo := new(MockRepository)

	dailyData := []model.DailyOccupancy{
		{Date: time.Now().AddDate(0, 0, -1), AvgOccupancy: 0.5, TotalSpots: 100, OccupiedSpots: 50},
	}
	repo.On("GetDailyOccupancy", mock.Anything, 30).Return(dailyData, nil)

	uc := NewUsecase(repo)

	result, err := uc.PredictResources(t.Context(), 24*time.Hour)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "at least 2 days")
	repo.AssertExpectations(t)
}

func TestPredictResources_ShouldReturnError_WhenHorizonNegative(t *testing.T) {
	repo := new(MockRepository)
	uc := NewUsecase(repo)

	result, err := uc.PredictResources(t.Context(), -1*time.Hour)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "horizon must be positive")
	repo.AssertExpectations(t)
}

func TestGetUsagePatterns_ShouldReturnPattern_WhenDataExists(t *testing.T) {
	repo := new(MockRepository)

	stats := []model.PeakHourStats{
		{Hour: 8, DayOfWeek: 1, AvgOccupancy: 0.9, AvgReservations: 20, PeakScore: 5.0},
		{Hour: 9, DayOfWeek: 1, AvgOccupancy: 0.85, AvgReservations: 18, PeakScore: 4.5},
		{Hour: 2, DayOfWeek: 1, AvgOccupancy: 0.1, AvgReservations: 2, PeakScore: 0.5},
		{Hour: 3, DayOfWeek: 1, AvgOccupancy: 0.15, AvgReservations: 3, PeakScore: 0.6},
	}
	repo.On("GetHourlyStats", mock.Anything, mock.Anything, mock.Anything).Return(stats, nil)

	uc := NewUsecase(repo)

	result, err := uc.GetUsagePatterns(t.Context())

	require.NoError(t, err)
	assert.Equal(t, "last_30_days", result.Period)
	assert.Greater(t, result.AvgUtilization, 0.0)
	assert.NotEmpty(t, result.PeakHours)
	assert.NotEmpty(t, result.IdleHours)
	assert.Contains(t, result.IdleHours, 2)
	assert.Contains(t, result.IdleHours, 3)
	repo.AssertExpectations(t)
}

func TestGetUsagePatterns_ShouldReturnEmptyPattern_WhenNoData(t *testing.T) {
	repo := new(MockRepository)
	repo.On("GetHourlyStats", mock.Anything, mock.Anything, mock.Anything).
		Return([]model.PeakHourStats{}, nil)

	uc := NewUsecase(repo)

	result, err := uc.GetUsagePatterns(t.Context())

	require.NoError(t, err)
	assert.Equal(t, "last_30_days", result.Period)
	assert.Equal(t, 0.0, result.AvgUtilization)
	assert.Empty(t, result.PeakHours)
	assert.Empty(t, result.IdleHours)
	repo.AssertExpectations(t)
}

func TestGetUsagePatterns_ShouldReturnError_WhenRepositoryFails(t *testing.T) {
	repo := new(MockRepository)
	repo.On("GetHourlyStats", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, assert.AnError)

	uc := NewUsecase(repo)

	result, err := uc.GetUsagePatterns(t.Context())

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to retrieve usage pattern data")
	repo.AssertExpectations(t)
}
