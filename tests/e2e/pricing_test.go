// Package e2e_test — pricing constants validation tests.
//
// Best practices applied (from Go testify testing standards):
// - Follow AAA (Arrange-Act-Assert) structure
// - Use descriptive test names: Test[Scenario]_Should[Expected]_When[Condition]
// - Keep tests isolated, repeatable, and focused on a single responsibility
// - Use assert for value comparisons (non-fatal, allows all checks to run)
// - No mocks needed — these tests validate compile-time constants
package e2e_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	billingmodel "parkir-pintar/internal/billing/model"
)

// TestPricing_ShouldHaveCorrectBookingFee verifies that the booking fee
// constant equals 5,000 IDR as defined in PRD Section 9.
//
// Validates: Requirement 3.1
func TestPricing_ShouldHaveCorrectBookingFee(t *testing.T) {
	// Assert
	assert.Equal(t, int64(5000), billingmodel.BookingFee,
		"BookingFee should be 5000 IDR")
}

// TestPricing_ShouldHaveCorrectHourlyRate verifies that the hourly parking
// rate constant equals 5,000 IDR as defined in PRD Section 9.
//
// Validates: Requirement 3.2
func TestPricing_ShouldHaveCorrectHourlyRate(t *testing.T) {
	// Assert
	assert.Equal(t, int64(5000), billingmodel.HourlyRate,
		"HourlyRate should be 5000 IDR")
}

// TestPricing_ShouldHaveCorrectOvernightFlatFee verifies that the overnight
// flat fee constant equals 20,000 IDR as defined in PRD Section 9.
//
// Validates: Requirement 3.3
func TestPricing_ShouldHaveCorrectOvernightFlatFee(t *testing.T) {
	// Assert
	assert.Equal(t, int64(20000), billingmodel.OvernightFlatFee,
		"OvernightFlatFee should be 20000 IDR")
}

// TestPricing_ShouldHaveCorrectWrongSpotPenalty verifies that the wrong-spot
// penalty constant equals 200,000 IDR as defined in PRD Section 9.
//
// Validates: Requirement 3.4
func TestPricing_ShouldHaveCorrectWrongSpotPenalty(t *testing.T) {
	// Assert
	assert.Equal(t, int64(200000), billingmodel.WrongSpotPenalty,
		"WrongSpotPenalty should be 200000 IDR")
}

// TestPricing_ShouldHaveCorrectCancelFreeWindow verifies that the cancellation
// free window constant equals 2 minutes as defined in PRD Section 9.
//
// Validates: Requirement 3.5
func TestPricing_ShouldHaveCorrectCancelFreeWindow(t *testing.T) {
	// Assert
	assert.Equal(t, 2*time.Minute, billingmodel.CancelFreeWindow,
		"CancelFreeWindow should be 2 minutes")
}

// TestPricing_ShouldHaveCorrectCancelFee verifies that the cancellation fee
// constant equals 5,000 IDR as defined in PRD Section 9.
//
// Validates: Requirement 3.6
func TestPricing_ShouldHaveCorrectCancelFee(t *testing.T) {
	// Assert
	assert.Equal(t, int64(5000), billingmodel.CancelFee,
		"CancelFee should be 5000 IDR")
}

// TestPricing_ShouldHaveZeroNoShowFee verifies that no additional no-show fee
// is charged. Per PRD, the booking fee (5,000 IDR, already charged at
// confirmation) is the only cost the driver forfeits on a no-show.
//
// Validates: Requirement 3.7
func TestPricing_ShouldHaveZeroNoShowFee(t *testing.T) {
	// Assert
	assert.Equal(t, int64(0), billingmodel.NoShowFee,
		"NoShowFee should be 0 IDR — booking fee is the only no-show cost")
}
