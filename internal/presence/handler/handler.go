// Package handler provides gRPC handlers for the presence domain module.
// Handlers validate request fields, delegate to the usecase layer, and map
// domain errors to gRPC status codes.
package handler

import (
	"context"
	"errors"
	"io"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"parkir-pintar/internal/presence/model"
	"parkir-pintar/internal/presence/repository"
	"parkir-pintar/internal/presence/usecase"
	"parkir-pintar/pkg/apperror"
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

// RegisterService registers this handler with the given gRPC server.
func (h *Handler) RegisterService(s *grpc.Server) {
	presencev1.RegisterPresenceServiceServer(s, h)
}

// validateCoordinates checks that latitude is within [-90, 90] and longitude
// is within [-180, 180]. Returns an InvalidArgument gRPC status error for
// out-of-range values.
func validateCoordinates(lat, lng float64) error {
	if lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return status.Error(codes.InvalidArgument, "latitude must be between -90 and 90, longitude must be between -180 and 180")
	}
	return nil
}

// StreamLocation implements the client streaming RPC. It receives LocationUpdate
// messages from the client and calls usecase.StreamLocation for each.
func (h *Handler) StreamLocation(stream presencev1.PresenceService_StreamLocationServer) error {
	ctx := stream.Context()

	for {
		msg, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return stream.SendAndClose(&presencev1.StreamLocationResponse{
				Status: "ok",
			})
		}
		if err != nil {
			return status.Errorf(codes.Internal, "receive error: %v", err)
		}

		if msg.GetReservationId() == "" {
			return status.Error(codes.InvalidArgument, "reservation_id is required")
		}

		if err := validateCoordinates(msg.GetLatitude(), msg.GetLongitude()); err != nil {
			return err
		}

		ts := time.Now()
		if msg.GetTimestamp() != nil {
			ts = msg.GetTimestamp().AsTime()
		}

		update := &model.LocationUpdate{
			ReservationID: msg.GetReservationId(),
			Latitude:      msg.GetLatitude(),
			Longitude:     msg.GetLongitude(),
			Accuracy:      msg.GetAccuracy(),
			Timestamp:     ts,
		}

		if err := h.uc.StreamLocation(ctx, update); err != nil {
			return status.Errorf(codes.Internal, "stream location: %v", err)
		}
	}
}

// DetectArrival validates required fields and delegates to the usecase.
func (h *Handler) DetectArrival(ctx context.Context, req *presencev1.DetectArrivalRequest) (*presencev1.ArrivalResponse, error) {
	if req.GetReservationId() == "" {
		return nil, status.Error(codes.InvalidArgument, "reservation_id is required")
	}

	if err := validateCoordinates(req.GetLatitude(), req.GetLongitude()); err != nil {
		return nil, err
	}

	result, err := h.uc.DetectArrival(ctx,
		req.GetLatitude(), req.GetLongitude(),
		req.GetCenterLatitude(), req.GetCenterLongitude(),
		req.GetRadiusMeters(),
		req.GetReservationId(),
	)
	if err != nil {
		return nil, mapError(err)
	}

	return &presencev1.ArrivalResponse{
		Arrived:       result.Arrived,
		ReservationId: result.ReservationID,
		DetectedAt:    timestamppb.New(result.DetectedAt),
	}, nil
}

// DetectWrongSpot validates required fields and delegates to the usecase.
func (h *Handler) DetectWrongSpot(ctx context.Context, req *presencev1.DetectWrongSpotRequest) (*presencev1.WrongSpotResponse, error) {
	if req.GetReservationId() == "" {
		return nil, status.Error(codes.InvalidArgument, "reservation_id is required")
	}

	if err := validateCoordinates(req.GetLatitude(), req.GetLongitude()); err != nil {
		return nil, err
	}

	result, err := h.uc.DetectWrongSpot(ctx,
		req.GetLatitude(), req.GetLongitude(),
		req.GetSpotLatitude(), req.GetSpotLongitude(),
		req.GetThresholdMeters(),
		req.GetReservationId(),
	)
	if err != nil {
		return nil, mapError(err)
	}

	return &presencev1.WrongSpotResponse{
		IsWrongSpot:    result.IsWrongSpot,
		DistanceMeters: result.DistanceMeters,
	}, nil
}

// GetPresence validates required fields and delegates to the usecase.
func (h *Handler) GetPresence(ctx context.Context, req *presencev1.GetPresenceRequest) (*presencev1.PresenceResponse, error) {
	if req.GetReservationId() == "" {
		return nil, status.Error(codes.InvalidArgument, "reservation_id is required")
	}

	log, err := h.uc.GetPresence(ctx, req.GetReservationId())
	if err != nil {
		return nil, mapError(err)
	}

	return &presencev1.PresenceResponse{
		ReservationId: log.ReservationID,
		Latitude:      log.Latitude,
		Longitude:     log.Longitude,
		Accuracy:      log.Accuracy,
		RecordedAt:    timestamppb.New(log.RecordedAt),
	}, nil
}

// mapError maps domain errors to gRPC status codes.
func mapError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, repository.ErrNotFound) {
		return status.Error(codes.NotFound, err.Error())
	}

	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		switch appErr.HTTPStatus {
		case 404:
			return status.Error(codes.NotFound, appErr.Message)
		case 409:
			return status.Error(codes.AlreadyExists, appErr.Message)
		case 400:
			return status.Error(codes.InvalidArgument, appErr.Message)
		default:
			return status.Error(codes.Internal, appErr.Message)
		}
	}

	return status.Error(codes.Internal, err.Error())
}
