// Package handler provides gRPC handlers for the payment domain module.
// Handlers validate request fields, delegate to the usecase layer, and map
// domain errors to gRPC status codes.
//
// Best practices applied (from Go coding standards KB):
// - Document all exported functions and types with proper Godoc format
// - Use context.Context as first parameter for consistency
// - Handle errors explicitly with proper gRPC status mapping
// - Keep interfaces small and focused
package handler

import (
	"context"
	"errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"parkir-pintar/internal/payment/model"
	"parkir-pintar/internal/payment/repository"
	"parkir-pintar/internal/payment/usecase"
	"parkir-pintar/pkg/apperror"
	paymentv1 "parkir-pintar/proto/payment/v1"
)

// Handler implements the paymentv1.PaymentServiceServer gRPC interface.
type Handler struct {
	paymentv1.UnimplementedPaymentServiceServer
	uc usecase.Usecase
}

// NewHandler creates a new payment gRPC Handler with the given usecase.
func NewHandler(uc usecase.Usecase) *Handler {
	return &Handler{uc: uc}
}

// RegisterService registers this handler with the given gRPC server.
func (h *Handler) RegisterService(s *grpc.Server) {
	paymentv1.RegisterPaymentServiceServer(s, h)
}

// ProcessPayment validates required fields and delegates to the usecase.
func (h *Handler) ProcessPayment(ctx context.Context, req *paymentv1.ProcessPaymentRequest) (*paymentv1.PaymentResponse, error) {
	if req.GetBillingId() == "" {
		return nil, status.Error(codes.InvalidArgument, "billing_id is required")
	}
	if req.GetAmount() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "amount must be positive")
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
		return nil, mapError(err)
	}

	return paymentToProto(result), nil
}

// ProcessQRIS validates required fields and delegates to the usecase.
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
		return nil, mapError(err)
	}

	return paymentToProto(result), nil
}

// RefundPayment validates required fields and delegates to the usecase.
func (h *Handler) RefundPayment(ctx context.Context, req *paymentv1.RefundPaymentRequest) (*paymentv1.PaymentResponse, error) {
	if req.GetPaymentId() == "" {
		return nil, status.Error(codes.InvalidArgument, "payment_id is required")
	}

	result, err := h.uc.RefundPayment(ctx, &model.RefundPaymentRequest{
		PaymentID: req.GetPaymentId(),
	})
	if err != nil {
		return nil, mapError(err)
	}

	return paymentToProto(result), nil
}

// GetPaymentStatus validates required fields and delegates to the usecase.
func (h *Handler) GetPaymentStatus(ctx context.Context, req *paymentv1.GetPaymentStatusRequest) (*paymentv1.PaymentResponse, error) {
	if req.GetPaymentId() == "" {
		return nil, status.Error(codes.InvalidArgument, "payment_id is required")
	}

	result, err := h.uc.GetPaymentStatus(ctx, &model.GetPaymentStatusRequest{
		PaymentID: req.GetPaymentId(),
	})
	if err != nil {
		return nil, mapError(err)
	}

	return paymentToProto(result), nil
}

// paymentToProto converts a domain Payment to a proto PaymentResponse.
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
