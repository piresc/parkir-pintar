// Package usecase implements the business logic layer for the payment domain
// module. It orchestrates payment processing, QRIS payments, refunds, and
// status queries, coordinating with the repository, payment gateway, and NATS.
//
// Best practices applied (from Go coding standards KB):
// - Document all exported functions and types with proper Godoc format
// - Use context.Context as first parameter for consistency
// - Return errors as the last value from functions
// - Wrap errors with context using fmt.Errorf with %w
// - Keep interfaces small and define them where they're used
// - Never ignore errors; always handle them explicitly
package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"parkir-pintar/internal/payment/gateway"
	"parkir-pintar/internal/payment/model"
	"parkir-pintar/internal/payment/repository"
)

// NATSClient defines the interface for NATS JetStream event publishing.
type NATSClient interface {
	Publish(subject string, data []byte) error
}

// Usecase defines the business logic interface for payment operations.
type Usecase interface {
	ProcessPayment(ctx context.Context, req *model.ProcessPaymentRequest) (*model.Payment, error)
	ProcessQRIS(ctx context.Context, req *model.ProcessQRISRequest) (*model.Payment, error)
	RefundPayment(ctx context.Context, req *model.RefundPaymentRequest) (*model.Payment, error)
	GetPaymentStatus(ctx context.Context, req *model.GetPaymentStatusRequest) (*model.Payment, error)
}

// paymentUsecase is the concrete implementation of Usecase.
type paymentUsecase struct {
	repo    repository.Repository
	gw      gateway.PaymentGateway
	nats    NATSClient
}

// NewUsecase creates a new payment Usecase with all required dependencies.
func NewUsecase(repo repository.Repository, gw gateway.PaymentGateway, nats NATSClient) Usecase {
	return &paymentUsecase{
		repo: repo,
		gw:   gw,
		nats: nats,
	}
}

// ProcessPayment processes a payment with idempotency check and circuit breaker
// pattern (retry 3x with exponential backoff 100ms/200ms/400ms).
func (uc *paymentUsecase) ProcessPayment(ctx context.Context, req *model.ProcessPaymentRequest) (*model.Payment, error) {
	// Idempotency check
	existing, err := uc.repo.GetByIdempotencyKey(ctx, req.IdempotencyKey)
	if err == nil && existing != nil {
		return existing, nil
	}

	now := time.Now()
	payment := &model.Payment{
		ID:             uuid.New().String(),
		BillingID:      req.BillingID,
		Amount:         req.Amount,
		PaymentMethod:  req.PaymentMethod,
		PaymentGateway: "stub-gateway",
		IdempotencyKey: req.IdempotencyKey,
		Status:         model.PaymentStatusPending,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := uc.repo.CreatePayment(ctx, payment); err != nil {
		return nil, fmt.Errorf("process payment create: %w", err)
	}

	// Circuit breaker: retry gateway.Charge up to 3 times with exponential backoff
	var txnRef string
	var chargeErr error
	backoffs := []time.Duration{100 * time.Millisecond, 200 * time.Millisecond, 400 * time.Millisecond}

	for i := range 3 {
		txnRef, chargeErr = uc.gw.Charge(ctx, req.Amount, req.PaymentMethod)
		if chargeErr == nil {
			break
		}
		slog.Warn("payment gateway charge failed, retrying",
			slog.Int("attempt", i+1),
			slog.Any("error", chargeErr))
		if i < 2 {
			select {
			case <-time.After(backoffs[i]):
			case <-ctx.Done():
				payment.Status = model.PaymentStatusFailed
				payment.UpdatedAt = time.Now()
				if updateErr := uc.repo.UpdatePayment(ctx, payment); updateErr != nil {
					slog.Error("failed to update payment status", slog.Any("error", updateErr))
				}
				return payment, ctx.Err()
			}
		}
	}

	if chargeErr != nil {
		// All retries exhausted — mark payment as failed
		payment.Status = model.PaymentStatusFailed
		payment.UpdatedAt = time.Now()
		if updateErr := uc.repo.UpdatePayment(ctx, payment); updateErr != nil {
			slog.Error("failed to update payment status to failed", slog.Any("error", updateErr))
		}
		uc.publishEvent("payment.failed", payment)
		return payment, nil
	}

	// Gateway succeeded
	paidAt := time.Now()
	payment.TransactionRef = txnRef
	payment.Status = model.PaymentStatusSuccess
	payment.PaidAt = &paidAt
	payment.UpdatedAt = paidAt

	if err := uc.repo.UpdatePayment(ctx, payment); err != nil {
		return nil, fmt.Errorf("process payment update success: %w", err)
	}

	uc.publishEvent("payment.success", payment)
	return payment, nil
}

// ProcessQRIS delegates to ProcessPayment with method="qris".
func (uc *paymentUsecase) ProcessQRIS(ctx context.Context, req *model.ProcessQRISRequest) (*model.Payment, error) {
	return uc.ProcessPayment(ctx, &model.ProcessPaymentRequest{
		BillingID:      req.BillingID,
		Amount:         req.Amount,
		PaymentMethod:  "qris",
		IdempotencyKey: req.IdempotencyKey,
	})
}

// RefundPayment refunds a previously completed payment via the gateway.
func (uc *paymentUsecase) RefundPayment(ctx context.Context, req *model.RefundPaymentRequest) (*model.Payment, error) {
	payment, err := uc.repo.GetByID(ctx, req.PaymentID)
	if err != nil {
		return nil, fmt.Errorf("refund payment get: %w", err)
	}

	if err := uc.gw.Refund(ctx, payment.TransactionRef); err != nil {
		return nil, fmt.Errorf("refund payment gateway: %w", err)
	}

	payment.Status = model.PaymentStatusRefunded
	payment.UpdatedAt = time.Now()

	if err := uc.repo.UpdatePayment(ctx, payment); err != nil {
		return nil, fmt.Errorf("refund payment update: %w", err)
	}

	return payment, nil
}

// GetPaymentStatus retrieves the current payment by ID.
func (uc *paymentUsecase) GetPaymentStatus(ctx context.Context, req *model.GetPaymentStatusRequest) (*model.Payment, error) {
	payment, err := uc.repo.GetByID(ctx, req.PaymentID)
	if err != nil {
		return nil, fmt.Errorf("get payment status: %w", err)
	}
	return payment, nil
}

// publishEvent serializes the payment and publishes it to NATS.
// Errors are logged but do not fail the operation.
func (uc *paymentUsecase) publishEvent(subject string, payment *model.Payment) {
	data, err := json.Marshal(payment)
	if err != nil {
		slog.Error("failed to marshal payment event", slog.String("subject", subject), slog.Any("error", err))
		return
	}
	if err := uc.nats.Publish(subject, data); err != nil {
		slog.Error("failed to publish payment event", slog.String("subject", subject), slog.Any("error", err))
	}
}
