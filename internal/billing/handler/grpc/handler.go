package grpc

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"parkir-pintar/internal/billing"
	"parkir-pintar/internal/billing/model"
	"parkir-pintar/internal/shared/grpcerror"
	billingv1 "parkir-pintar/proto/billing/v1"
)

type Handler struct {
	billingv1.UnimplementedBillingServiceServer
	uc billing.Usecase
}

func NewHandler(uc billing.Usecase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) RegisterService(s *grpc.Server) {
	billingv1.RegisterBillingServiceServer(s, h)
}

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
		return nil, grpcerror.MapToGRPCError(err)
	}

	return billingRecordToProto(result), nil
}

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
		return nil, grpcerror.MapToGRPCError(err)
	}

	return billingRecordToProto(result), nil
}

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
		return nil, grpcerror.MapToGRPCError(err)
	}

	return billingRecordToProto(result), nil
}

func (h *Handler) ApplyOvernightFee(ctx context.Context, req *billingv1.ApplyOvernightFeeRequest) (*billingv1.BillingResponse, error) {
	if req.GetReservationId() == "" {
		return nil, status.Error(codes.InvalidArgument, "reservation_id is required")
	}

	result, err := h.uc.ApplyOvernightFee(ctx, &model.ApplyOvernightFeeRequest{
		ReservationID: req.GetReservationId(),
	})
	if err != nil {
		return nil, grpcerror.MapToGRPCError(err)
	}

	return billingRecordToProto(result), nil
}

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
		TotalAmount:     r.TotalAmount,
		DurationMinutes: int32(r.DurationMinutes),
		BilledHours:     int32(r.BilledHours),
		IsOvernight:     r.IsOvernight,
		IdempotencyKey:  r.IdempotencyKey,
		Status:          r.Status,
	}
}
