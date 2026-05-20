// - Never ignore errors; always handle them explicitly
package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"parkir-pintar/pkg/logger"
	"time"

	"github.com/google/uuid"

	paymenterrors "parkir-pintar/internal/payment/errors"
	"parkir-pintar/internal/payment/gateway"
	"parkir-pintar/internal/payment/model"
	"parkir-pintar/internal/payment/repository"
	"parkir-pintar/pkg/idempotency"

	paymentconstants "parkir-pintar/internal/payment/constants"
)

const paymentMethodQRIS = string(paymentconstants.PaymentMethodQRIS)

//go:generate mockgen -destination=../mocks/mock_usecase.go -package=mocks parkir-pintar/internal/payment/usecase Usecase
type Usecase interface {
	ProcessPayment(ctx context.Context, req *model.ProcessPaymentRequest) (*model.Payment, error)
	ProcessQRIS(ctx context.Context, req *model.ProcessQRISRequest) (*model.Payment, error)
	RefundPayment(ctx context.Context, req *model.RefundPaymentRequest) (*model.Payment, error)
	GetPaymentStatus(ctx context.Context, req *model.GetPaymentStatusRequest) (*model.Payment, error)
}

//go:generate mockgen -destination=../mocks/mock_event_publisher.go -package=mocks parkir-pintar/internal/payment/usecase EventPublisher
type EventPublisher interface {
	PublishPaymentSuccess(ctx context.Context, event gateway.PaymentResultEvent) error
	PublishPaymentFailed(ctx context.Context, event gateway.PaymentResultEvent) error
}

func (uc *paymentUsecase) ProcessPayment(ctx context.Context, req *model.ProcessPaymentRequest) (*model.Payment, error) {
	res, err := idempotency.Check(ctx, req.IdempotencyKey, uc.repo.GetByIdempotencyKey, repository.ErrNotFound, "process payment")
	if err != nil {
		return nil, err
	}
	if res.Found {
		return res.Record, nil
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
		if errors.Is(err, repository.ErrConflict) {
			existing, fetchErr := uc.repo.GetByIdempotencyKey(ctx, req.IdempotencyKey)
			if fetchErr != nil {
				return nil, fmt.Errorf("process payment fetch after conflict: %w", fetchErr)
			}
			return existing, nil
		}
		return nil, fmt.Errorf("process payment create: %w", err)
	}

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
			logger.Err(chargeErr))
		if i < 2 {
			select {
			case <-time.After(backoffs[i]):
			case <-ctx.Done():
				payment.Status = model.PaymentStatusFailed
				payment.UpdatedAt = time.Now()
				cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
				if updateErr := uc.repo.UpdatePayment(cleanupCtx, payment); updateErr != nil { //nolint:contextcheck // intentional: parent ctx is cancelled, need fresh context
					slog.Error("failed to update payment status on context cancel",
						slog.String("payment_id", payment.ID),
						logger.Err(updateErr))
				}
				cleanupCancel()
				return nil, fmt.Errorf("%w: %w", paymenterrors.ErrCancelled, ctx.Err())
			}
		}
	}

	if chargeErr != nil {
		payment.Status = model.PaymentStatusFailed
		payment.UpdatedAt = time.Now()
		if updateErr := uc.repo.UpdatePayment(ctx, payment); updateErr != nil {
			slog.Error("failed to update payment status to failed", logger.Err(updateErr))
		}
		if uc.eventPublisher != nil {
			pubErr := uc.eventPublisher.PublishPaymentFailed(ctx, gateway.PaymentResultEvent{
				PaymentID: payment.ID,
				Status:    string(model.PaymentStatusFailed),
			})
			if pubErr != nil {
				slog.Error("failed to publish payment failed event",
					slog.String("payment_id", payment.ID),
					logger.Err(pubErr))
			}
		}
		return payment, nil
	}

	paidAt := time.Now()
	payment.TransactionRef = txnRef
	payment.Status = model.PaymentStatusSuccess
	payment.PaidAt = &paidAt
	payment.UpdatedAt = paidAt

	if err := uc.repo.UpdatePayment(ctx, payment); err != nil {
		return nil, fmt.Errorf("process payment update success: %w", err)
	}

	if uc.eventPublisher != nil {
		pubErr := uc.eventPublisher.PublishPaymentSuccess(ctx, gateway.PaymentResultEvent{
			PaymentID: payment.ID,
			Status:    string(model.PaymentStatusSuccess),
		})
		if pubErr != nil {
			slog.Error("failed to publish payment success event",
				slog.String("payment_id", payment.ID),
				logger.Err(pubErr))
		}
	}

	return payment, nil
}

func (uc *paymentUsecase) ProcessQRIS(ctx context.Context, req *model.ProcessQRISRequest) (*model.Payment, error) {
	return uc.ProcessPayment(ctx, &model.ProcessPaymentRequest{
		BillingID:      req.BillingID,
		Amount:         req.Amount,
		PaymentMethod:  paymentMethodQRIS,
		IdempotencyKey: req.IdempotencyKey,
	})
}

func (uc *paymentUsecase) RefundPayment(ctx context.Context, req *model.RefundPaymentRequest) (*model.Payment, error) {
	if req.IdempotencyKey != "" {
		res, err := idempotency.Check(ctx, req.IdempotencyKey, uc.repo.GetByIdempotencyKey, repository.ErrNotFound, "refund payment")
		if err != nil {
			return nil, err
		}
		if res.Found {
			return res.Record, nil
		}
	}

	payment, err := uc.repo.GetByID(ctx, req.PaymentID)
	if err != nil {
		return nil, fmt.Errorf("refund payment get: %w", err)
	}

	if payment.Status != model.PaymentStatusSuccess {
		return nil, fmt.Errorf("%w: current status %q", paymenterrors.ErrCannotRefund, payment.Status)
	}

	// the gateway to prevent double-refund from concurrent requests.
	payment.Status = model.PaymentStatusRefunded
	payment.UpdatedAt = time.Now()

	if err := uc.repo.UpdatePaymentWithStatusCheck(ctx, payment, model.PaymentStatusSuccess); err != nil {
		return nil, fmt.Errorf("refund payment status lock: %w", err)
	}

	var refundErr error
	backoffs := []time.Duration{100 * time.Millisecond, 200 * time.Millisecond, 400 * time.Millisecond}
	for i := range 3 {
		refundErr = uc.gw.Refund(ctx, payment.TransactionRef)
		if refundErr == nil {
			break
		}
		slog.Warn("payment gateway refund failed, retrying",
			slog.String("payment_id", payment.ID),
			slog.Int("attempt", i+1),
			logger.Err(refundErr))
		if i < 2 {
			select {
			case <-time.After(backoffs[i]):
			case <-ctx.Done():
				// Revert status back to success since gateway was not called successfully.
				payment.Status = model.PaymentStatusSuccess
				payment.UpdatedAt = time.Now()
				revertCtx, revertCancel := context.WithTimeout(context.Background(), 5*time.Second)
				if revertErr := uc.repo.UpdatePayment(revertCtx, payment); revertErr != nil { //nolint:contextcheck // intentional: parent ctx is cancelled, need fresh context
					slog.Error("failed to revert payment status after context cancel",
						slog.String("payment_id", payment.ID),
						logger.Err(revertErr))
				}
				revertCancel()
				return nil, fmt.Errorf("%w: %w", paymenterrors.ErrCancelled, ctx.Err())
			}
		}
	}
	if refundErr != nil {
		payment.Status = model.PaymentStatusSuccess
		payment.UpdatedAt = time.Now()
		if revertErr := uc.repo.UpdatePayment(ctx, payment); revertErr != nil {
			slog.Error("failed to revert payment status after gateway failure",
				slog.String("payment_id", payment.ID),
				logger.Err(revertErr))
		}
		return nil, fmt.Errorf("%w: %w", paymenterrors.ErrRefundFailed, refundErr)
	}

	return payment, nil
}

func (uc *paymentUsecase) GetPaymentStatus(ctx context.Context, req *model.GetPaymentStatusRequest) (*model.Payment, error) {
	payment, err := uc.repo.GetByID(ctx, req.PaymentID)
	if err != nil {
		return nil, fmt.Errorf("get payment status: %w", err)
	}
	return payment, nil
}
