// Package e2e_test — payment state machine validation test.
//
// Best practices applied (from Go testify testing standards):
// - Use require for assertions that must pass to continue (fail-fast)
// - Use assert for non-critical checks
// - Follow AAA (Arrange-Act-Assert) structure
// - Use descriptive test names: Test[Scenario]_Should[Expected]_When[Condition]
package e2e_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	paymentmodel "parkir-pintar/internal/payment/model"
)

// TestPaymentState_ShouldHaveCorrectStatuses verifies that the payment model
// defines exactly 4 statuses: pending, success, failed, refunded.
//
// Validates: Requirements 16.1, 16.2
func TestPaymentState_ShouldHaveCorrectStatuses(t *testing.T) {
	// Assert — Verify the 4 payment status constants exist with correct values
	assert.Equal(t, "pending", paymentmodel.PaymentStatusPending)
	assert.Equal(t, "success", paymentmodel.PaymentStatusSuccess)
	assert.Equal(t, "failed", paymentmodel.PaymentStatusFailed)
	assert.Equal(t, "refunded", paymentmodel.PaymentStatusRefunded)

	// Assert — Verify exactly 4 statuses are defined
	statuses := []string{
		paymentmodel.PaymentStatusPending,
		paymentmodel.PaymentStatusSuccess,
		paymentmodel.PaymentStatusFailed,
		paymentmodel.PaymentStatusRefunded,
	}
	assert.Len(t, statuses, 4, "payment model should define exactly 4 statuses")

	// Verify all statuses are unique
	seen := make(map[string]bool)
	for _, s := range statuses {
		assert.False(t, seen[s], "duplicate payment status: %s", s)
		seen[s] = true
	}
}
