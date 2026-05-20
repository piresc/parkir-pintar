package handler

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"parkir-pintar/internal/payment/model"
	"parkir-pintar/internal/payment/usecase"
	"parkir-pintar/internal/shared/grpcerror"
	paymentv1 "parkir-pintar/proto/payment/v1"
)

type Handler struct {
	paymentv1.UnimplementedPaymentServiceServer
	uc usecase.Usecase
}

func NewHandler(uc usecase.Usecase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) ProcessPayment(ctx context.Context, req *paymentv1.ProcessPaymentRequest) (*paymentv1.PaymentResponse, error) {
	if req.GetBillingId() == "" {
		return nil, status.Error(codes.InvalidArgument, "billing_id is required")
	}
	if req.GetAmount() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "amount must be positive")
	}
	const maxPaymentAmount int64 = 100_000_000 // 100M IDR
	if req.GetAmount() > maxPaymentAmount {
		return nil, status.Error(codes.InvalidArgument, "amount exceeds maximum allowed (100,000,000)")
	}
	if req.GetPaymentMethod() == "" {
		return nil, status.Error(codes.InvalidArgument, "payment_method is required")
	}
	if req.GetIdempotencyKey() == "" {
		return nil, status.Error(codes.InvalidArgument, "idempotency_key is required")
	}

	result, err := h.uc.ProcessPayment(ctx, &model.ProcessPaymentRequest{
		BillingID:      req.GetBillingId(),
		Amount:         req.GetAmount(),
		PaymentMethod:  req.GetPaymentMethod(),
		IdempotencyKey: req.GetIdempotencyKey(),
	})
	if err != nil {
		return nil, grpcerror.MapToGRPCError(err)
	}

	return paymentToProto(result), nil
}

func (h *Handler) ProcessQRIS(ctx context.Context, req *paymentv1.ProcessQRISRequest) (*paymentv1.PaymentResponse, error) {
	if req.GetBillingId() == "" {
		return nil, status.Error(codes.InvalidArgument, "billing_id is required")
	}
	if req.GetAmount() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "amount must be positive")
	}
	if req.GetIdempotencyKey() == "" {
		return nil, status.Error(codes.InvalidArgument, "idempotency_key is required")
	}

	result, err := h.uc.ProcessQRIS(ctx, &model.ProcessQRISRequest{
		BillingID:      req.GetBillingId(),
		Amount:         req.GetAmount(),
		IdempotencyKey: req.GetIdempotencyKey(),
	})
	if err != nil {
		return nil, grpcerror.MapToGRPCError(err)
	}

	return paymentToProto(result), nil
}

func (h *Handler) RefundPayment(ctx context.Context, req *paymentv1.RefundPaymentRequest) (*paymentv1.PaymentResponse, error) {
	if req.GetPaymentId() == "" {
		return nil, status.Error(codes.InvalidArgument, "payment_id is required")
	}

	result, err := h.uc.RefundPayment(ctx, &model.RefundPaymentRequest{
		PaymentID: req.GetPaymentId(),
	})
	if err != nil {
		return nil, grpcerror.MapToGRPCError(err)
	}

	return paymentToProto(result), nil
}

func (h *Handler) GetPaymentStatus(ctx context.Context, req *paymentv1.GetPaymentStatusRequest) (*paymentv1.PaymentResponse, error) {
	if req.GetPaymentId() == "" {
		return nil, status.Error(codes.InvalidArgument, "payment_id is required")
	}

	result, err := h.uc.GetPaymentStatus(ctx, &model.GetPaymentStatusRequest{
		PaymentID: req.GetPaymentId(),
	})
	if err != nil {
		return nil, grpcerror.MapToGRPCError(err)
	}

	return paymentToProto(result), nil
}

func paymentToProto(p *model.Payment) *paymentv1.PaymentResponse {
	if p == nil {
		return nil
	}

	resp := &paymentv1.PaymentResponse{
		Id:             p.ID,
		BillingId:      p.BillingID,
		Amount:         p.Amount,
		PaymentMethod:  p.PaymentMethod,
		PaymentGateway: p.PaymentGateway,
		TransactionRef: p.TransactionRef,
		IdempotencyKey: p.IdempotencyKey,
		Status:         p.Status,
	}
	if p.PaidAt != nil {
		resp.PaidAt = timestamppb.New(*p.PaidAt)
	}
	return resp
}
