package analytics

import (
	"context"
	"time"

	"parkir-pintar/internal/analytics/model"
)

//go:generate mockgen -destination=mocks/mock_repository.go -package=mocks parkir-pintar/internal/analytics Repository
type Repository interface {
	GetHourlyStats(ctx context.Context, startDate, endDate time.Time) ([]model.PeakHourStats, error)
	GetDailyOccupancy(ctx context.Context, days int) ([]model.DailyOccupancy, error)
	RecordEvent(ctx context.Context, event model.ReservationEvent) error
	UpsertSpotSnapshot(ctx context.Context, spot model.SpotSnapshot) error
}
