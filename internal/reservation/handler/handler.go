// Package handler provides gRPC handlers for the reservation domain module.
// Handlers validate request fields, delegate to the usecase layer, and map
// domain errors to gRPC status codes.
package handler

import (
	"context"
	"errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"parkir-pintar/internal/reservation/model"
	"parkir-pintar/internal/reservation/usecase"
	"parkir-pintar/pkg/apperror"
	reservationv1 "parkir-pintar/proto/reservation/v1"
)

// Handler implements the reservationv1.ReservationServiceServer gRPC interface.
type Handler struct {
	reservationv1.UnimplementedReservationServiceServer
	uc usecase.Usecase
}

// NewHandler creates a new reservation gRPC Handler with the given usecase.
func NewHandler(uc usecase.Usecase) *Handler {
	return &Handler{uc: uc}
}

// RegisterService registers this handler with the given gRPC server.
func (h *Handler) RegisterService(s *grpc.Server) {
	reservationv1.RegisterReservationServiceServer(s, h)
}

// CreateReservation validates required fields and delegates to the usecase.
func (h *Handler) CreateReservation(ctx context.Context, req *reservationv1.CreateReservationRequest) (*reservationv1.ReservationResponse, error) {
	if req.GetDriverId() == "" {
		return nil, status.Error(codes.InvalidArgument, "driver_id is required")
	}
	if req.GetVehicleType() == "" {
		return nil, status.Error(codes.InvalidArgument, "vehicle_type is required")
	}
	if req.GetIdempotencyKey() == "" {
		return nil, status.Error(codes.InvalidArgument, "idempotency_key is required")
	}

	result, err := h.uc.CreateReservation(ctx, &model.CreateReservationRequest{
		DriverID:       req.GetDriverId(),
		VehicleType:    req.GetVehicleType(),
		AssignmentMode: req.GetAssignmentMode(),
		SpotID:         req.GetSpotId(),
		IdempotencyKey: req.GetIdempotencyKey(),
	})
	if err != nil {
		return nil, mapError(err)
	}

	return reservationToProto(result), nil
}

// GetReservation validates reservation_id and retrieves the reservation.
func (h *Handler) GetReservation(ctx context.Context, req *reservationv1.GetReservationRequest) (*reservationv1.ReservationResponse, error) {
	if req.GetReservationId() == "" {
		return nil, status.Error(codes.InvalidArgument, "reservation_id is required")
	}

	result, err := h.uc.GetReservation(ctx, req.GetReservationId())
	if err != nil {
		return nil, mapError(err)
	}

	return reservationToProto(result), nil
}

// CancelReservation validates reservation_id and delegates to the usecase.
func (h *Handler) CancelReservation(ctx context.Context, req *reservationv1.CancelReservationRequest) (*reservationv1.ReservationResponse, error) {
	if req.GetReservationId() == "" {
		return nil, status.Error(codes.InvalidArgument, "reservation_id is required")
	}

	result, err := h.uc.CancelReservation(ctx, &model.CancelReservationRequest{
		ReservationID: req.GetReservationId(),
	})
	if err != nil {
		return nil, mapError(err)
	}

	return reservationToProto(result), nil
}

// CheckIn validates reservation_id and delegates to the usecase.
func (h *Handler) CheckIn(ctx context.Context, req *reservationv1.CheckInRequest) (*reservationv1.ReservationResponse, error) {
	if req.GetReservationId() == "" {
		return nil, status.Error(codes.InvalidArgument, "reservation_id is required")
	}

	result, err := h.uc.CheckIn(ctx, &model.CheckInRequest{
		ReservationID: req.GetReservationId(),
	})
	if err != nil {
		return nil, mapError(err)
	}

	return reservationToProto(result), nil
}

// CheckOut validates reservation_id and delegates to the usecase.
func (h *Handler) CheckOut(ctx context.Context, req *reservationv1.CheckOutRequest) (*reservationv1.CheckOutResponse, error) {
	if req.GetReservationId() == "" {
		return nil, status.Error(codes.InvalidArgument, "reservation_id is required")
	}

	result, err := h.uc.CheckOut(ctx, &model.CheckOutRequest{
		ReservationID: req.GetReservationId(),
	})
	if err != nil {
		return nil, mapError(err)
	}

	return &reservationv1.CheckOutResponse{
		Reservation:   reservationToProto(result.Reservation),
		TotalAmount:   result.TotalAmount,
		BillingId:     result.BillingID,
		PaymentId:     result.PaymentID,
		BookingFee:    result.BookingFee,
		ParkingFee:    result.ParkingFee,
		OvernightFee:  result.OvernightFee,
		PenaltyAmount: result.PenaltyAmount,
	}, nil
}

// ConfirmReservation validates reservation_id and delegates to the usecase.
func (h *Handler) ConfirmReservation(ctx context.Context, req *reservationv1.ConfirmReservationRequest) (*reservationv1.ReservationResponse, error) {
	if req.GetReservationId() == "" {
		return nil, status.Error(codes.InvalidArgument, "reservation_id is required")
	}

	result, err := h.uc.ConfirmReservation(ctx, &model.ConfirmReservationRequest{
		ReservationID: req.GetReservationId(),
	})
	if err != nil {
		return nil, mapError(err)
	}

	return reservationToProto(result), nil
}

// CompleteCheckout validates reservation_id and delegates to the usecase.
func (h *Handler) CompleteCheckout(ctx context.Context, req *reservationv1.CompleteCheckoutRequest) (*reservationv1.CheckOutResponse, error) {
	if req.GetReservationId() == "" {
		return nil, status.Error(codes.InvalidArgument, "reservation_id is required")
	}

	result, err := h.uc.CompleteCheckout(ctx, &model.CompleteCheckoutRequest{
		ReservationID: req.GetReservationId(),
	})
	if err != nil {
		return nil, mapError(err)
	}

	return &reservationv1.CheckOutResponse{
		Reservation:   reservationToProto(result.Reservation),
		TotalAmount:   result.TotalAmount,
		BillingId:     result.BillingID,
		PaymentId:     result.PaymentID,
		BookingFee:    result.BookingFee,
		ParkingFee:    result.ParkingFee,
		OvernightFee:  result.OvernightFee,
		PenaltyAmount: result.PenaltyAmount,
	}, nil
}

// ExpireReservation validates reservation_id and delegates to the usecase.
func (h *Handler) ExpireReservation(ctx context.Context, req *reservationv1.ExpireReservationRequest) (*reservationv1.ReservationResponse, error) {
	if req.GetReservationId() == "" {
		return nil, status.Error(codes.InvalidArgument, "reservation_id is required")
	}

	err := h.uc.ExpireReservation(ctx, &model.ExpireReservationRequest{
		ReservationID: req.GetReservationId(),
	})
	if err != nil {
		return nil, mapError(err)
	}

	return &reservationv1.ReservationResponse{}, nil
}

// ListByDriver validates driver_id and retrieves reservations for the driver.
func (h *Handler) ListByDriver(ctx context.Context, req *reservationv1.ListByDriverRequest) (*reservationv1.ListByDriverResponse, error) {
	if req.GetDriverId() == "" {
		return nil, status.Error(codes.InvalidArgument, "driver_id is required")
	}

	results, err := h.uc.ListByDriver(ctx, req.GetDriverId(), req.GetStatus())
	if err != nil {
		return nil, mapError(err)
	}

	var reservations []*reservationv1.ReservationResponse
	for _, r := range results {
		reservations = append(reservations, reservationToProto(r))
	}

	return &reservationv1.ListByDriverResponse{
		Reservations: reservations,
	}, nil
}

// reservationToProto converts a domain Reservation to a proto ReservationResponse.
func reservationToProto(r *model.Reservation) *reservationv1.ReservationResponse {
	if r == nil {
		return nil
	}

	resp := &reservationv1.ReservationResponse{
		Id:             r.ID,
		DriverId:       r.DriverID,
		SpotId:         r.SpotID,
		VehicleType:    r.VehicleType,
		AssignmentMode: r.AssignmentMode,
		Status:         r.Status,
		IdempotencyKey: r.IdempotencyKey,
		SpotCode:       r.SpotCode,
	}

	if r.ConfirmedAt != nil {
		resp.ConfirmedAt = timestamppb.New(*r.ConfirmedAt)
	}
	if r.ExpiresAt != nil {
		resp.ExpiresAt = timestamppb.New(*r.ExpiresAt)
	}
	if r.CheckedInAt != nil {
		resp.CheckedInAt = timestamppb.New(*r.CheckedInAt)
	}
	if r.CheckedOutAt != nil {
		resp.CheckedOutAt = timestamppb.New(*r.CheckedOutAt)
	}
	if r.CancelledAt != nil {
		resp.CancelledAt = timestamppb.New(*r.CancelledAt)
	}

	return resp
}

// mapError maps domain errors to gRPC status codes.
func mapError(err error) error {
	if err == nil {
		return nil
	}

	// Check domain sentinel errors first.
	if errors.Is(err, model.ErrNotFound) {
		return status.Error(codes.NotFound, err.Error())
	}
	if errors.Is(err, model.ErrConflict) {
		return status.Error(codes.AlreadyExists, err.Error())
	}
	if errors.Is(err, model.ErrInvalidTransition) {
		return status.Error(codes.FailedPrecondition, err.Error())
	}
	if errors.Is(err, model.ErrSpotUnavailable) {
		return status.Error(codes.FailedPrecondition, err.Error())
	}

	// Check apperror types.
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
