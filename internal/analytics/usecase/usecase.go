package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"parkir-pintar/pkg/logger"
	"math"
	"time"

	"parkir-pintar/internal/analytics/model"
	"parkir-pintar/pkg/apperror"
)

const idleThreshold = 0.30

const defaultLookbackDays = 30

func (uc *analyticsUsecase) GetPeakHours(ctx context.Context) ([]model.PeakHourStats, error) {
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -defaultLookbackDays)

	stats, err := uc.repo.GetHourlyStats(ctx, startDate, endDate)
	if err != nil {
		slog.Error("failed to get hourly stats for peak hours",
			logger.Err(err))
		return nil, apperror.Internal("failed to retrieve peak hour data")
	}

	if len(stats) == 0 {
		return []model.PeakHourStats{}, nil
	}

	var totalOccupancy float64
	for _, s := range stats {
		totalOccupancy += s.AvgOccupancy
	}
	avgOccupancy := totalOccupancy / float64(len(stats))

	var peakHours []model.PeakHourStats
	for _, s := range stats {
		if s.AvgOccupancy > avgOccupancy {
			peakHours = append(peakHours, s)
		}
	}

	return peakHours, nil
}

func (uc *analyticsUsecase) GetIdleHours(ctx context.Context) ([]model.PeakHourStats, error) {
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -defaultLookbackDays)

	stats, err := uc.repo.GetHourlyStats(ctx, startDate, endDate)
	if err != nil {
		slog.Error("failed to get hourly stats for idle hours",
			logger.Err(err))
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

func (uc *analyticsUsecase) PredictResources(ctx context.Context, horizon time.Duration) ([]model.ResourcePrediction, error) {
	if horizon <= 0 {
		return nil, apperror.BadRequest("horizon must be positive")
	}

	dailyOccupancy, err := uc.repo.GetDailyOccupancy(ctx, defaultLookbackDays)
	if err != nil {
		slog.Error("failed to get daily occupancy for prediction",
			logger.Err(err))
		return nil, apperror.Internal("failed to retrieve occupancy data for prediction")
	}

	if len(dailyOccupancy) < 2 {
		return nil, apperror.New("INSUFFICIENT_DATA",
			"at least 2 days of historical data required for prediction", 422)
	}

	slope, intercept := linearRegression(dailyOccupancy)

	now := time.Now()
	hours := int(math.Ceil(horizon.Hours()))
	predictions := make([]model.ResourcePrediction, 0, hours)

	n := float64(len(dailyOccupancy))
	for i := 1; i <= hours; i++ {
		timestamp := now.Add(time.Duration(i) * time.Hour)
		dayOffset := float64(i) / 24.0
		predictedOccupancy := slope*(n+dayOffset) + intercept

		predictedOccupancy = math.Max(0, math.Min(1, predictedOccupancy))

		recommendedInstances := 1 + int(predictedOccupancy*4)

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

func (uc *analyticsUsecase) GetUsagePatterns(ctx context.Context) (*model.UsagePattern, error) {
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -defaultLookbackDays)

	stats, err := uc.repo.GetHourlyStats(ctx, startDate, endDate)
	if err != nil {
		slog.Error("failed to get hourly stats for usage patterns",
			logger.Err(err))
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

func (uc *analyticsUsecase) RecordEvent(ctx context.Context, event model.ReservationEvent) error {
	if err := uc.repo.RecordEvent(ctx, event); err != nil {
		slog.Error("failed to record reservation event",
			slog.String("reservation_id", event.ReservationID),
			slog.String("status", event.Status),
			logger.Err(err))
		return apperror.Internal("failed to record reservation event")
	}
	return nil
}

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
