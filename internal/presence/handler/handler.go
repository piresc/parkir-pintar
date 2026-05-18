// Package handler provides gRPC handlers for the presence domain module.
package handler

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"parkir-pintar/internal/presence/usecase"
	presencev1 "parkir-pintar/proto/presence/v1"
)

// Handler implements the presencev1.PresenceServiceServer gRPC interface.
type Handler struct {
	presencev1.UnimplementedPresenceServiceServer
	uc usecase.Usecase
}

// NewHandler creates a new presence gRPC Handler with the given usecase.
func NewHandler(uc usecase.Usecase) *Handler {
	return &Handler{uc: uc}
}

// VerifyLocation validates request fields and delegates to the usecase.
func (h *Handler) VerifyLocation(ctx context.Context, req *presencev1.VerifyLocationRequest) (*presencev1.VerifyLocationResponse, error) {
	if req.GetDriverId() == "" {
		return nil, status.Error(codes.InvalidArgument, "driver_id is required")
	}
	if req.GetReservationId() == "" {
		return nil, status.Error(codes.InvalidArgument, "reservation_id is required")
	}

	verified, dist, spotCode, err := h.uc.VerifyLocation(
		ctx,
		req.GetDriverId(),
		req.GetLatitude(),
		req.GetLongitude(),
		req.GetReservationId(),
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "verify location: %v", err)
	}

	return &presencev1.VerifyLocationResponse{
		Verified:         verified,
		DistanceMeters:   dist,
		AssignedSpotCode: spotCode,
	}, nil
}

// UpdateDriverLocation validates request fields and delegates to the usecase.
func (h *Handler) UpdateDriverLocation(ctx context.Context, req *presencev1.UpdateDriverLocationRequest) (*presencev1.UpdateDriverLocationResponse, error) {
	if req.GetDriverId() == "" {
		return nil, status.Error(codes.InvalidArgument, "driver_id is required")
	}

	err := h.uc.UpdateDriverLocation(
		ctx,
		req.GetDriverId(),
		req.GetLatitude(),
		req.GetLongitude(),
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "update driver location: %v", err)
	}

	return &presencev1.UpdateDriverLocationResponse{
		Success: true,
	}, nil
}
