// Package usecase implements the business logic layer for the analytics module.
// It provides peak/idle hour identification, resource prediction, and usage
// pattern summarization based on historical reservation data.
package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"parkir-pintar/internal/analytics/model"
	"parkir-pintar/internal/analytics/repository"
	"parkir-pintar/pkg/apperror"
)

// idleThreshold defines the occupancy percentage below which an hour is
// considered idle (30%).
const idleThreshold = 0.30

// defaultLookbackDays is the number of days of historical data used for analysis.
const defaultLookbackDays = 30

// Usecase defines the business logic interface for analytics operations.
type Usecase interface {
	// GetPeakHours identifies hours with the highest occupancy from historical data.
	GetPeakHours(ctx context.Context) ([]model.PeakHourStats, error)

	// GetIdleHours identifies hours where average occupancy is below 30%.
	GetIdleHours(ctx context.Context) ([]model.PeakHourStats, error)

	// PredictResources generates resource predictions for the given time horizon
	// using simple linear extrapolation from historical patterns.
	PredictResources(ctx context.Context, horizon time.Duration) ([]model.ResourcePrediction, error)

	// GetUsagePatterns summarizes weekly utilization patterns.
	GetUsagePatterns(ctx context.Context) (*model.UsagePattern, error)

	// RecordEvent records a reservation event for analytics reporting.
	RecordEvent(ctx context.Context, event model.ReservationEvent) error
}

// analyticsUsecase is the concrete implementation of Usecase.
type analyticsUsecase struct {
	repo repository.Repository
}

// NewUsecase creates a new analytics Usecase with the given repository dependency.
func NewUsecase(repo repository.Repository) Usecase {
	return &analyticsUsecase{repo: repo}
}

// GetPeakHours retrieves hourly stats for the last 30 days and returns hours
// sorted by peak score descending. Only hours with above-average occupancy are included.
func (uc *analyticsUsecase) GetPeakHours(ctx context.Context) ([]model.PeakHourStats, error) {
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -defaultLookbackDays)

	stats, err := uc.repo.GetHourlyStats(ctx, startDate, endDate)
	if err != nil {
		slog.Error("failed to get hourly stats for peak hours",
			slog.Any("error", err))
		return nil, apperror.Internal("failed to retrieve peak hour data")
	}

	if len(stats) == 0 {
		return []model.PeakHourStats{}, nil
	}

	// Calculate average occupancy across all hours
	var totalOccupancy float64
	for _, s := range stats {
		totalOccupancy += s.AvgOccupancy
	}
	avgOccupancy := totalOccupancy / float64(len(stats))

	// Filter to only peak hours (above average occupancy)
	var peakHours []model.PeakHourStats
	for _, s := range stats {
		if s.AvgOccupancy > avgOccupancy {
			peakHours = append(peakHours, s)
		}
	}

	return peakHours, nil
}

// GetIdleHours retrieves hourly stats and returns hours where average occupancy
// is below the idle threshold (30%).
func (uc *analyticsUsecase) GetIdleHours(ctx context.Context) ([]model.PeakHourStats, error) {
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -defaultLookbackDays)

	stats, err := uc.repo.GetHourlyStats(ctx, startDate, endDate)
	if err != nil {
		slog.Error("failed to get hourly stats for idle hours",
			slog.Any("error", err))
		return nil, apperror.Internal("failed to retrieve idle hour data")
	}

	var idleHours []model.PeakHourStats
	for _, s := range stats {
		if s.AvgOccupancy < idleThreshold {
			idleHours = append(idleHours, s)
		}
	}

	if idleHours == nil {
		idleHours = []model.PeakHourStats{}
	}

	return idleHours, nil
}

// PredictResources generates resource predictions at hourly intervals for the
// given horizon duration. Uses simple linear regression on daily occupancy data
// to extrapolate future resource needs.
func (uc *analyticsUsecase) PredictResources(ctx context.Context, horizon time.Duration) ([]model.ResourcePrediction, error) {
	if horizon <= 0 {
		return nil, apperror.BadRequest("horizon must be positive")
	}

	dailyOccupancy, err := uc.repo.GetDailyOccupancy(ctx, defaultLookbackDays)
	if err != nil {
		slog.Error("failed to get daily occupancy for prediction",
			slog.Any("error", err))
		return nil, apperror.Internal("failed to retrieve occupancy data for prediction")
	}

	if len(dailyOccupancy) < 2 {
		return nil, apperror.New("INSUFFICIENT_DATA",
			"at least 2 days of historical data required for prediction", 422)
	}

	// Simple linear regression: y = slope*x + intercept
	slope, intercept := linearRegression(dailyOccupancy)

	// Generate predictions at hourly intervals
	now := time.Now()
	hours := int(math.Ceil(horizon.Hours()))
	predictions := make([]model.ResourcePrediction, 0, hours)

	n := float64(len(dailyOccupancy))
	for i := 1; i <= hours; i++ {
		timestamp := now.Add(time.Duration(i) * time.Hour)
		// Project occupancy using day-fraction offset from last data point
		dayOffset := float64(i) / 24.0
		predictedOccupancy := slope*(n+dayOffset) + intercept

		// Clamp between 0 and 1
		predictedOccupancy = math.Max(0, math.Min(1, predictedOccupancy))

		// Recommend instances: 1 base + 1 per 25% occupancy
		recommendedInstances := 1 + int(predictedOccupancy*4)

		// Confidence decreases with distance from known data
		confidence := math.Max(0.1, 1.0-dayOffset/float64(defaultLookbackDays))

		predictions = append(predictions, model.ResourcePrediction{
			Timestamp:            timestamp,
			PredictedOccupancy:   math.Round(predictedOccupancy*1000) / 1000,
			RecommendedInstances: recommendedInstances,
			Confidence:           math.Round(confidence*1000) / 1000,
		})
	}

	return predictions, nil
}

// GetUsagePatterns summarizes weekly utilization by identifying peak and idle
// hours from the last 30 days of data.
func (uc *analyticsUsecase) GetUsagePatterns(ctx context.Context) (*model.UsagePattern, error) {
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -defaultLookbackDays)

	stats, err := uc.repo.GetHourlyStats(ctx, startDate, endDate)
	if err != nil {
		slog.Error("failed to get hourly stats for usage patterns",
			slog.Any("error", err))
		return nil, apperror.Internal("failed to retrieve usage pattern data")
	}

	if len(stats) == 0 {
		return &model.UsagePattern{
			Period:         fmt.Sprintf("last_%d_days", defaultLookbackDays),
			AvgUtilization: 0,
			PeakHours:      []int{},
			IdleHours:      []int{},
		}, nil
	}

	// Aggregate by hour (across all days of week)
	hourOccupancy := make(map[int][]float64)
	var totalUtilization float64

	for _, s := range stats {
		hourOccupancy[s.Hour] = append(hourOccupancy[s.Hour], s.AvgOccupancy)
		totalUtilization += s.AvgOccupancy
	}

	avgUtilization := totalUtilization / float64(len(stats))

	var peakHours []int
	var idleHours []int

	for hour, occupancies := range hourOccupancy {
		var sum float64
		for _, o := range occupancies {
			sum += o
		}
		avg := sum / float64(len(occupancies))

		if avg > avgUtilization {
			peakHours = append(peakHours, hour)
		}
		if avg < idleThreshold {
			idleHours = append(idleHours, hour)
		}
	}

	return &model.UsagePattern{
		Period:         fmt.Sprintf("last_%d_days", defaultLookbackDays),
		AvgUtilization: math.Round(avgUtilization*1000) / 1000,
		PeakHours:      peakHours,
		IdleHours:      idleHours,
	}, nil
}

// RecordEvent delegates event persistence to the repository layer.
func (uc *analyticsUsecase) RecordEvent(ctx context.Context, event model.ReservationEvent) error {
	if err := uc.repo.RecordEvent(ctx, event); err != nil {
		slog.Error("failed to record reservation event",
			slog.String("reservation_id", event.ReservationID),
			slog.String("status", event.Status),
			slog.Any("error", err))
		return apperror.Internal("failed to record reservation event")
	}
	return nil
}

// linearRegression performs simple linear regression on daily occupancy data.
// Returns slope and intercept for the line y = slope*x + intercept.
func linearRegression(data []model.DailyOccupancy) (slope, intercept float64) {
	n := float64(len(data))
	if n == 0 {
		return 0, 0
	}

	var sumX, sumY, sumXY, sumX2 float64
	for i, d := range data {
		x := float64(i)
		y := d.AvgOccupancy
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	denominator := n*sumX2 - sumX*sumX
	if denominator == 0 {
		return 0, sumY / n
	}

	slope = (n*sumXY - sumX*sumY) / denominator
	intercept = (sumY - slope*sumX) / n
	return slope, intercept
}
