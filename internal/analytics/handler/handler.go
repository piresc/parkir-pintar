// Package handler provides gRPC handlers for the analytics domain module.
// Handlers delegate to the usecase layer and map domain errors to gRPC status codes.
package handler

import (
	"context"
	"errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"parkir-pintar/internal/analytics/usecase"
	"parkir-pintar/pkg/apperror"
	analyticsv1 "parkir-pintar/proto/analytics/v1"
)

// Handler implements the analyticsv1.AnalyticsServiceServer gRPC interface.
type Handler struct {
	analyticsv1.UnimplementedAnalyticsServiceServer
	uc usecase.Usecase
}

// NewHandler creates a new analytics gRPC Handler with the given usecase.
func NewHandler(uc usecase.Usecase) *Handler {
	return &Handler{uc: uc}
}

// RegisterService registers this handler with the given gRPC server.
func (h *Handler) RegisterService(s *grpc.Server) {
	analyticsv1.RegisterAnalyticsServiceServer(s, h)
}

// GetPeakHours returns peak hour statistics from historical reservation data.
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

// GetUsagePatterns returns daily occupancy/usage patterns summarized over the last 30 days.
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

// mapError maps domain errors to gRPC status codes.
func mapError(err error) error {
	if err == nil {
		return nil
	}

	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		switch appErr.HTTPStatus {
		case 404:
			return status.Error(codes.NotFound, appErr.Message)
		case 400:
			return status.Error(codes.InvalidArgument, appErr.Message)
		default:
			return status.Error(codes.Internal, appErr.Message)
		}
	}

	return status.Error(codes.Internal, err.Error())
}
