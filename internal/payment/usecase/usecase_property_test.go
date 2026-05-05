// Package usecase provides bug condition exploration tests for payment retry context awareness.
//
// Best practices applied (from Go testify coding standards KB):
// - Test naming: Test[FunctionName]_Should[ExpectedResult]_When[Condition]
// - AAA pattern: Arrange → Act → Assert
// - testify/mock for mock implementations of all dependency interfaces
// - testify/assert and testify/require for assertions
// - Each test is isolated with its own mock setup
// - Mock at interface boundaries rather than concrete implementations
//
// **Validates: Requirements 2.10** (Property 7 from design)
//
// Bug Condition: ctx.Done() AND retryInProgress
// Expected: return within one tick (< 150ms)
// Counterexample on unfixed code: blocks for full backoff duration (100ms+200ms+400ms)
//
// CRITICAL: This test is expected to FAIL on unfixed code.
// DO NOT fix the code or the test when it fails.
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

// TestProcessPayment_ShouldReturnPromptly_WhenContextCancelledDuringRetry
// cancels the context during the retry sleep and measures return time.
// On unfixed code, time.Sleep ignores context cancellation and blocks for
// the full backoff duration.
//
// **Validates: Requirements 2.10**
func TestProcessPayment_ShouldReturnPromptly_WhenContextCancelledDuringRetry(t *testing.T) {
	// Arrange
	repo := new(MockRepository)
	gw := new(MockPaymentGateway)
	natsClient := new(MockNATSClient)

	gatewayErr := errors.New("gateway unavailable")
	repo.On("GetByIdempotencyKey", mock.Anything, "pay-ctx-cancel").Return(nil, repository.ErrNotFound)
	repo.On("CreatePayment", mock.Anything, mock.AnythingOfType("*model.Payment")).Return(nil)
	// Gateway always fails to force retries
	gw.On("Charge", mock.Anything, int64(5000), "qris").Return("", gatewayErr)
	repo.On("UpdatePayment", mock.Anything, mock.AnythingOfType("*model.Payment")).Return(nil)
	natsClient.On("Publish", mock.Anything, mock.Anything).Return(nil)

	uc := NewUsecase(repo, gw, natsClient)
	req := &model.ProcessPaymentRequest{
		BillingID:      "billing-ctx",
		Amount:         5000,
		PaymentMethod:  "qris",
		IdempotencyKey: "pay-ctx-cancel",
	}

	// Cancel context after 50ms — this should interrupt the first retry sleep (100ms)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	// Act
	start := time.Now()
	result, err := uc.ProcessPayment(ctx, req)
	elapsed := time.Since(start)

	// Assert — should return within ~150ms (50ms wait + some margin), not 700ms+
	// The full backoff without context awareness would be: 100ms + 200ms + 400ms = 700ms
	maxAllowed := 200 * time.Millisecond
	assert.Less(t, elapsed, maxAllowed,
		"ProcessPayment should return within %v after context cancellation, but took %v — time.Sleep ignores ctx.Done()",
		maxAllowed, elapsed)

	// The function should either return an error or a failed payment
	if err != nil {
		require.ErrorIs(t, err, context.Canceled)
	} else {
		require.NotNil(t, result)
	}
}
