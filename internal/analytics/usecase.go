package analytics

import (
	"context"
	"time"

	"parkir-pintar/internal/analytics/model"
)

//go:generate mockgen -destination=mocks/mock_usecase.go -package=mocks parkir-pintar/internal/analytics Usecase
type Usecase interface {
	GetPeakHours(ctx context.Context) ([]model.PeakHourStats, error)
	PredictResources(ctx context.Context, horizon time.Duration) ([]model.ResourcePrediction, error)
	GetUsagePatterns(ctx context.Context) (*model.UsagePattern, error)
	RecordEvent(ctx context.Context, event model.ReservationEvent) error
}
