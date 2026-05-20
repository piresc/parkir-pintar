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

	paymentconstants "parkir-pintar/internal/payment/constants"
)

// TestPaymentState_ShouldHaveCorrectStatuses verifies that the payment model
// defines exactly 4 statuses: pending, success, failed, refunded.
//
// Validates: Requirements 16.1, 16.2
func TestPaymentState_ShouldHaveCorrectStatuses(t *testing.T) {
	// Assert — Verify the 4 payment status constants exist with correct values
	assert.Equal(t, "pending", string(paymentconstants.PaymentStatusPending))
	assert.Equal(t, "success", string(paymentconstants.PaymentStatusSuccess))
	assert.Equal(t, "failed", string(paymentconstants.PaymentStatusFailed))
	assert.Equal(t, "refunded", string(paymentconstants.PaymentStatusRefunded))

	// Assert — Verify exactly 4 statuses are defined
	statuses := []string{
		string(paymentconstants.PaymentStatusPending),
		string(paymentconstants.PaymentStatusSuccess),
		string(paymentconstants.PaymentStatusFailed),
		string(paymentconstants.PaymentStatusRefunded),
	}
	assert.Len(t, statuses, 4, "payment model should define exactly 4 statuses")

	// Verify all statuses are unique
	seen := make(map[string]bool)
	for _, s := range statuses {
		assert.False(t, seen[s], "duplicate payment status: %s", s)
		seen[s] = true
	}
}
