package grpc

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"parkir-pintar/internal/presence"
	presencev1 "parkir-pintar/proto/presence/v1"
)

type Handler struct {
	presencev1.UnimplementedPresenceServiceServer
	uc presence.Usecase
}

func NewHandler(uc presence.Usecase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) RegisterService(s *grpc.Server) {
	presencev1.RegisterPresenceServiceServer(s, h)
}

func (h *Handler) VerifyPresence(ctx context.Context, req *presencev1.VerifyPresenceRequest) (*presencev1.VerifyPresenceResponse, error) {
	if req.GetDriverId() == "" {
		return nil, status.Error(codes.InvalidArgument, "driver_id is required")
	}
	if req.GetReservationId() == "" {
		return nil, status.Error(codes.InvalidArgument, "reservation_id is required")
	}

	result, err := h.uc.VerifyPresence(
		ctx,
		req.GetReservationId(),
		int(req.GetFloorNumber()),
		int(req.GetSpotNumber()),
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "verify presence: %v", err)
	}

	return &presencev1.VerifyPresenceResponse{
		Verified: result.Verified,
		Message:  result.Message,
	}, nil
}
