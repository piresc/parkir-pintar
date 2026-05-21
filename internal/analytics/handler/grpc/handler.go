package grpc

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"parkir-pintar/internal/analytics"
	analyticsv1 "parkir-pintar/proto/analytics/v1"
)

type Handler struct {
	analyticsv1.UnimplementedAnalyticsServiceServer
	uc analytics.Usecase
}

func NewHandler(uc analytics.Usecase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) RegisterService(s *grpc.Server) {
	analyticsv1.RegisterAnalyticsServiceServer(s, h)
}

func (h *Handler) GetPeakHours(ctx context.Context, _ *analyticsv1.GetPeakHoursRequest) (*analyticsv1.GetPeakHoursResponse, error) {
	stats, err := h.uc.GetPeakHours(ctx)
	if err != nil {
		return nil, mapError(err)
	}

	protoStats := make([]*analyticsv1.PeakHourStats, 0, len(stats))
	for _, s := range stats {
		protoStats = append(protoStats, &analyticsv1.PeakHourStats{
			Hour:            int32(s.Hour),
			DayOfWeek:       int32(s.DayOfWeek),
			AvgOccupancy:    s.AvgOccupancy,
			AvgReservations: int32(s.AvgReservations),
			PeakScore:       s.PeakScore,
		})
	}

	return &analyticsv1.GetPeakHoursResponse{Stats: protoStats}, nil
}

func (h *Handler) GetUsagePatterns(ctx context.Context, _ *analyticsv1.GetUsagePatternsRequest) (*analyticsv1.GetUsagePatternsResponse, error) {
	pattern, err := h.uc.GetUsagePatterns(ctx)
	if err != nil {
		return nil, mapError(err)
	}

	peakHours := make([]int32, 0, len(pattern.PeakHours))
	for _, ph := range pattern.PeakHours {
		peakHours = append(peakHours, int32(ph))
	}

	idleHours := make([]int32, 0, len(pattern.IdleHours))
	for _, ih := range pattern.IdleHours {
		idleHours = append(idleHours, int32(ih))
	}

	return &analyticsv1.GetUsagePatternsResponse{
		Period:         pattern.Period,
		AvgUtilization: pattern.AvgUtilization,
		PeakHours:      peakHours,
		IdleHours:      idleHours,
	}, nil
}

const defaultPredictionHorizonMinutes = 60
const maxHorizonMinutes = 10080 // 7 days

func (h *Handler) PredictResources(ctx context.Context, req *analyticsv1.PredictResourcesRequest) (*analyticsv1.PredictResourcesResponse, error) {
	horizonMinutes := int(req.GetHorizonMinutes())
	if horizonMinutes <= 0 {
		horizonMinutes = defaultPredictionHorizonMinutes
	}
	if horizonMinutes > maxHorizonMinutes {
		return nil, status.Error(codes.InvalidArgument, "horizon_minutes cannot exceed 10080 (7 days)")
	}

	predictions, err := h.uc.PredictResources(ctx, time.Duration(horizonMinutes)*time.Minute)
	if err != nil {
		return nil, mapError(err)
	}

	protoPredictions := make([]*analyticsv1.ResourcePrediction, 0, len(predictions))
	for _, p := range predictions {
		protoPredictions = append(protoPredictions, &analyticsv1.ResourcePrediction{
			Timestamp:            p.Timestamp.Format(time.RFC3339),
			PredictedOccupancy:   p.PredictedOccupancy,
			RecommendedInstances: int32(p.RecommendedInstances),
			Confidence:           p.Confidence,
		})
	}

	return &analyticsv1.PredictResourcesResponse{Predictions: protoPredictions}, nil
}
