// Package handler provides gRPC handlers for the billing domain module.
// Handlers validate request fields, delegate to the usecase layer, and map
// domain errors to gRPC status codes.
package handler

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"parkir-pintar/internal/billing/model"
	"parkir-pintar/internal/billing/usecase"
	"parkir-pintar/pkg/apperror"
	billingv1 "parkir-pintar/proto/billing/v1"
)

// Handler implements the billingv1.BillingServiceServer gRPC interface.
type Handler struct {
	billingv1.UnimplementedBillingServiceServer
	uc usecase.Usecase
}

// NewHandler creates a new billing gRPC Handler with the given usecase.
func NewHandler(uc usecase.Usecase) *Handler {
	return &Handler{uc: uc}
}

// RegisterService registers this handler with the given gRPC server.
func (h *Handler) RegisterService(s *grpc.Server) {
	billingv1.RegisterBillingServiceServer(s, h)
}

// StartBilling validates required fields and delegates to the usecase.
func (h *Handler) StartBilling(ctx context.Context, req *billingv1.StartBillingRequest) (*billingv1.BillingResponse, error) {
	if req.GetReservationId() == "" {
		return nil, status.Error(codes.InvalidArgument, "reservation_id is required")
	}
	if req.GetIdempotencyKey() == "" {
		return nil, status.Error(codes.InvalidArgument, "idempotency_key is required")
	}

	result, err := h.uc.StartBilling(ctx, &model.StartBillingRequest{
		ReservationID:  req.GetReservationId(),
		BookingFee:     req.GetBookingFee(),
		IdempotencyKey: req.GetIdempotencyKey(),
	})
	if err != nil {
		return nil, mapError(err)
	}

	return billingRecordToProto(result), nil
}

// CalculateFee validates required fields and delegates to the usecase.
func (h *Handler) CalculateFee(ctx context.Context, req *billingv1.CalculateFeeRequest) (*billingv1.BillingResponse, error) {
	if req.GetReservationId() == "" {
		return nil, status.Error(codes.InvalidArgument, "reservation_id is required")
	}
	if req.GetCheckInAt() == nil {
		return nil, status.Error(codes.InvalidArgument, "check_in_at is required")
	}
	if req.GetCheckOutAt() == nil {
		return nil, status.Error(codes.InvalidArgument, "check_out_at is required")
	}

	result, err := h.uc.CalculateFee(ctx, &model.CalculateFeeRequest{
		ReservationID: req.GetReservationId(),
		CheckInAt:     req.GetCheckInAt().AsTime(),
		CheckOutAt:    req.GetCheckOutAt().AsTime(),
	})
	if err != nil {
		return nil, mapError(err)
	}

	return billingRecordToProto(result), nil
}

// GenerateInvoice validates required fields and delegates to the usecase.
func (h *Handler) GenerateInvoice(ctx context.Context, req *billingv1.GenerateInvoiceRequest) (*billingv1.BillingResponse, error) {
	if req.GetReservationId() == "" {
		return nil, status.Error(codes.InvalidArgument, "reservation_id is required")
	}
	if req.GetIdempotencyKey() == "" {
		return nil, status.Error(codes.InvalidArgument, "idempotency_key is required")
	}

	result, err := h.uc.GenerateInvoice(ctx, &model.GenerateInvoiceRequest{
		ReservationID:  req.GetReservationId(),
		IdempotencyKey: req.GetIdempotencyKey(),
	})
	if err != nil {
		return nil, mapError(err)
	}

	return billingRecordToProto(result), nil
}

// ApplyPenalty is no longer supported. Returns Unimplemented to maintain wire compatibility.
func (h *Handler) ApplyPenalty(_ context.Context, _ *billingv1.ApplyPenaltyRequest) (*billingv1.BillingResponse, error) {
	return nil, status.Error(codes.Unimplemented, "penalty system has been removed")
}

// ApplyOvernightFee validates required fields and delegates to the usecase.
func (h *Handler) ApplyOvernightFee(ctx context.Context, req *billingv1.ApplyOvernightFeeRequest) (*billingv1.BillingResponse, error) {
	if req.GetReservationId() == "" {
		return nil, status.Error(codes.InvalidArgument, "reservation_id is required")
	}

	result, err := h.uc.ApplyOvernightFee(ctx, &model.ApplyOvernightFeeRequest{
		ReservationID: req.GetReservationId(),
	})
	if err != nil {
		return nil, mapError(err)
	}

	return billingRecordToProto(result), nil
}

// billingRecordToProto converts a domain BillingRecord to a proto BillingResponse.
func billingRecordToProto(r *model.BillingRecord) *billingv1.BillingResponse {
	if r == nil {
		return nil
	}

	return &billingv1.BillingResponse{
		Id:              r.ID,
		ReservationId:   r.ReservationID,
		BookingFee:      r.BookingFee,
		ParkingFee:      r.ParkingFee,
		OvernightFee:    r.OvernightFee,
		CancellationFee: r.CancellationFee,
		PenaltyAmount:   r.PenaltyAmount,
		TotalAmount:     r.TotalAmount,
		DurationMinutes: int32(r.DurationMinutes),
		BilledHours:     int32(r.BilledHours),
		IsOvernight:     r.IsOvernight,
		IdempotencyKey:  r.IdempotencyKey,
		Status:          r.Status,
	}
}

// mapError maps domain errors to gRPC status codes.
func mapError(err error) error {
	return apperror.MapToGRPCError(err)
}
