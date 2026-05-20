package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"parkir-pintar/internal/payment/model"
	"parkir-pintar/internal/payment/repository"
)

func TestProcessPayment_ShouldReturnPromptly_WhenContextCancelledDuringRetry(t *testing.T) {
	repo := new(MockRepository)
	gw := new(MockPaymentGateway)

	gatewayErr := errors.New("gateway unavailable")
	repo.On("GetByIdempotencyKey", mock.Anything, "pay-ctx-cancel").Return(nil, repository.ErrNotFound)
	repo.On("CreatePayment", mock.Anything, mock.AnythingOfType("*model.Payment")).Return(nil)
	// Gateway always fails to force retries
	gw.On("Charge", mock.Anything, int64(5000), "qris").Return("", gatewayErr)
	repo.On("UpdatePayment", mock.Anything, mock.AnythingOfType("*model.Payment")).Return(nil)

	uc := NewUsecase(repo, gw, nil)
	req := &model.ProcessPaymentRequest{
		BillingID:      "billing-ctx",
		Amount:         5000,
		PaymentMethod:  "qris",
		IdempotencyKey: "pay-ctx-cancel",
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	result, err := uc.ProcessPayment(ctx, req)
	elapsed := time.Since(start)

	maxAllowed := 200 * time.Millisecond
	assert.Less(t, elapsed, maxAllowed,
		"ProcessPayment should return within %v after context cancellation, but took %v — time.Sleep ignores ctx.Done()",
		maxAllowed, elapsed)

	if err != nil {
		require.ErrorIs(t, err, context.Canceled)
	} else {
		require.NotNil(t, result)
	}
}
