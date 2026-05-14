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

	"github.com/stretchr/testify/assert"

	"parkir-pintar/pkg/pricing"
)

// TestPricing_ShouldHaveCorrectBookingFee verifies that the booking fee
// constant equals 5,000 IDR as defined in PRD Section 9.
//
// Validates: Requirement 3.1
func TestPricing_ShouldHaveCorrectBookingFee(t *testing.T) {
	// Assert
	assert.Equal(t, int64(5000), pricing.BookingFee,
		"BookingFee should be 5000 IDR")
}

// TestPricing_ShouldHaveCorrectHourlyRate verifies that the hourly parking
// rate constant equals 5,000 IDR as defined in PRD Section 9.
//
// Validates: Requirement 3.2
func TestPricing_ShouldHaveCorrectHourlyRate(t *testing.T) {
	// Assert
	assert.Equal(t, int64(5000), pricing.HourlyRate,
		"HourlyRate should be 5000 IDR")
}

// TestPricing_ShouldHaveCorrectOvernightFlatFee verifies that the overnight
// flat fee constant equals 20,000 IDR as defined in PRD Section 9.
//
// Validates: Requirement 3.3
func TestPricing_ShouldHaveCorrectOvernightFlatFee(t *testing.T) {
	// Assert
	assert.Equal(t, int64(20000), pricing.OvernightPerNight,
		"OvernightPerNight should be 20000 IDR")
}
