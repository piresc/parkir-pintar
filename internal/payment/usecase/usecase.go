// - Never ignore errors; always handle them explicitly
package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"parkir-pintar/internal/payment/gateway"
	"parkir-pintar/internal/payment/model"
	"parkir-pintar/internal/payment/repository"
	"parkir-pintar/pkg/idempotency"
	"parkir-pintar/pkg/logger"
	"parkir-pintar/pkg/retry"

	paymentconstants "parkir-pintar/internal/payment/constants"
)

const paymentMethodQRIS = string(paymentconstants.PaymentMethodQRIS)

//go:generate mockgen -destination=../mocks/mock_usecase.go -package=mocks parkir-pintar/internal/payment Usecase

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
		Status:         string(paymentconstants.PaymentStatusPending),
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

	chargeErr := retry.Do(ctx, retry.DefaultConfig(), func(retryCtx context.Context) error {
		var err error
		txnRef, err = uc.gw.Charge(retryCtx, req.Amount, req.PaymentMethod)
		if err != nil {
			slog.Warn("payment gateway charge failed, retrying",
				logger.Err(err))
		}
		return err
	})

	if chargeErr != nil && (errors.Is(chargeErr, context.Canceled) || errors.Is(chargeErr, context.DeadlineExceeded)) {
		payment.Status = string(paymentconstants.PaymentStatusFailed)
		payment.UpdatedAt = time.Now()
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		if updateErr := uc.repo.UpdatePayment(cleanupCtx, payment); updateErr != nil { //nolint:contextcheck // intentional: parent ctx is cancelled, need fresh context
			slog.Error("failed to update payment status on context cancel",
				slog.String("payment_id", payment.ID),
				logger.Err(updateErr))
		}
		cleanupCancel()
		return nil, fmt.Errorf("%w: %w", paymentconstants.ErrCancelled, chargeErr)
	}

	if chargeErr != nil {
		payment.Status = string(paymentconstants.PaymentStatusFailed)
		payment.UpdatedAt = time.Now()
		if updateErr := uc.repo.UpdatePayment(ctx, payment); updateErr != nil {
			slog.Error("failed to update payment status to failed", logger.Err(updateErr))
			return nil, fmt.Errorf("process payment update failed status: %w", updateErr)
		}
		if uc.eventPublisher != nil {
			pubErr := uc.eventPublisher.PublishPaymentFailed(ctx, gateway.PaymentResultEvent{
				PaymentID: payment.ID,
				Status:    string(paymentconstants.PaymentStatusFailed),
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
	payment.Status = string(paymentconstants.PaymentStatusSuccess)
	payment.PaidAt = &paidAt
	payment.UpdatedAt = paidAt

	if err := uc.repo.UpdatePayment(ctx, payment); err != nil {
		return nil, fmt.Errorf("process payment update success: %w", err)
	}

	if uc.eventPublisher != nil {
		pubErr := uc.eventPublisher.PublishPaymentSuccess(ctx, gateway.PaymentResultEvent{
			PaymentID: payment.ID,
			Status:    string(paymentconstants.PaymentStatusSuccess),
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

	if payment.Status != string(paymentconstants.PaymentStatusSuccess) {
		return nil, fmt.Errorf("%w: current status %q", paymentconstants.ErrCannotRefund, payment.Status)
	}

	// Set intermediate "refunding" status to lock against concurrent refunds.
	payment.Status = string(paymentconstants.PaymentStatusRefunding)
	payment.UpdatedAt = time.Now()

	if err := uc.repo.UpdatePaymentWithStatusCheck(ctx, payment, string(paymentconstants.PaymentStatusSuccess)); err != nil {
		return nil, fmt.Errorf("refund payment status lock: %w", err)
	}

	refundErr := retry.Do(ctx, retry.DefaultConfig(), func(retryCtx context.Context) error {
		err := uc.gw.Refund(retryCtx, payment.TransactionRef)
		if err != nil {
			slog.Warn("payment gateway refund failed, retrying",
				slog.String("payment_id", payment.ID),
				logger.Err(err))
		}
		return err
	})

	if refundErr != nil && (errors.Is(refundErr, context.Canceled) || errors.Is(refundErr, context.DeadlineExceeded)) {
		// Revert status back to success since gateway was not called successfully.
		payment.Status = string(paymentconstants.PaymentStatusSuccess)
		payment.UpdatedAt = time.Now()
		revertCtx, revertCancel := context.WithTimeout(context.Background(), 5*time.Second)
		if revertErr := uc.repo.UpdatePayment(revertCtx, payment); revertErr != nil { //nolint:contextcheck // intentional: parent ctx is cancelled, need fresh context
			slog.Error("failed to revert payment status after context cancel",
				slog.String("payment_id", payment.ID),
				logger.Err(revertErr))
		}
		revertCancel()
		return nil, fmt.Errorf("%w: %w", paymentconstants.ErrCancelled, refundErr)
	}
	if refundErr != nil {
		payment.Status = string(paymentconstants.PaymentStatusSuccess)
		payment.UpdatedAt = time.Now()
		if revertErr := uc.repo.UpdatePayment(ctx, payment); revertErr != nil {
			slog.Error("failed to revert payment status after gateway failure",
				slog.String("payment_id", payment.ID),
				logger.Err(revertErr))
		}
		return nil, fmt.Errorf("%w: %w", paymentconstants.ErrRefundFailed, refundErr)
	}

	// Gateway succeeded — now mark as actually refunded
	payment.Status = string(paymentconstants.PaymentStatusRefunded)
	payment.UpdatedAt = time.Now()
	if err := uc.repo.UpdatePayment(ctx, payment); err != nil {
		slog.Error("failed to update payment to refunded after gateway success",
			slog.String("payment_id", payment.ID),
			logger.Err(err))
		// Gateway already refunded, so we log but don't fail — eventual consistency
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
