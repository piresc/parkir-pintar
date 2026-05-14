// Package e2e_test — reservation state machine validation tests.
//
// Best practices applied (from Go testify testing standards):
// - Follow AAA (Arrange-Act-Assert) structure
// - Use descriptive test names: Test[Scenario]_Should[Expected]_When[Condition]
// - Use table-driven tests for exhaustive transition coverage
// - Keep tests isolated, repeatable, and focused on a single responsibility
// - Use require for fatal assertions, assert for non-fatal checks
// - No mocks needed — these tests validate the pure state machine function
package e2e_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	reservationmodel "parkir-pintar/internal/reservation/model"
)

// TestStateMachine_ShouldAllowConfirmedToCheckedIn verifies that the
// transition CONFIRMED → CHECKED_IN is allowed.
//
// Validates: Requirement 4.1
func TestStateMachine_ShouldAllowConfirmedToCheckedIn(t *testing.T) {
	// Act
	err := reservationmodel.ValidateTransition("confirmed", "checked_in")

	// Assert
	require.NoError(t, err, "CONFIRMED → CHECKED_IN should be allowed")
}

// TestStateMachine_ShouldAllowConfirmedToExpired verifies that the
// transition CONFIRMED → EXPIRED is allowed.
//
// Validates: Requirement 4.2
func TestStateMachine_ShouldAllowConfirmedToExpired(t *testing.T) {
	// Act
	err := reservationmodel.ValidateTransition("confirmed", "expired")

	// Assert
	require.NoError(t, err, "CONFIRMED → EXPIRED should be allowed")
}

// TestStateMachine_ShouldAllowConfirmedToCancelled verifies that the
// transition CONFIRMED → CANCELLED is allowed.
//
// Validates: Requirement 4.3
func TestStateMachine_ShouldAllowConfirmedToCancelled(t *testing.T) {
	// Act
	err := reservationmodel.ValidateTransition("confirmed", "cancelled")

	// Assert
	require.NoError(t, err, "CONFIRMED → CANCELLED should be allowed")
}

// TestStateMachine_ShouldAllowCheckedInToCheckedOut verifies that the
// transition CHECKED_IN → CHECKED_OUT is allowed.
//
// Validates: Requirement 4.4
func TestStateMachine_ShouldAllowCheckedInToCheckedOut(t *testing.T) {
	// Act
	err := reservationmodel.ValidateTransition("checked_in", "checked_out")

	// Assert
	require.NoError(t, err, "CHECKED_IN → CHECKED_OUT should be allowed")
}

// TestStateMachine_ShouldRejectTerminalStateTransitions verifies that terminal
// states (checked_out, expired, cancelled) have no outgoing transitions.
// Uses table-driven tests for exhaustive coverage.
//
// Validates: Requirement 4.5
func TestStateMachine_ShouldRejectTerminalStateTransitions(t *testing.T) {
	// Arrange
	terminalStates := []string{"checked_out", "expired", "cancelled"}
	allStates := []string{"pending", "confirmed", "checked_in", "checked_out", "expired", "cancelled"}

	for _, from := range terminalStates {
		for _, to := range allStates {
			t.Run(from+"_to_"+to, func(t *testing.T) {
				// Act
				err := reservationmodel.ValidateTransition(from, to)

				// Assert
				assert.Error(t, err,
					"transition from terminal state %q to %q should be rejected", from, to)
			})
		}
	}
}

// TestStateMachine_ShouldRejectInvalidTransitions verifies that invalid
// transitions return an error. Uses table-driven tests for specific cases.
//
// Validates: Requirement 4.6
func TestStateMachine_ShouldRejectInvalidTransitions(t *testing.T) {
	// Arrange
	invalidTransitions := []struct {
		name string
		from string
		to   string
	}{
		{name: "confirmed_to_checked_out", from: "confirmed", to: "checked_out"},
		{name: "confirmed_to_pending", from: "confirmed", to: "pending"},
		{name: "checked_in_to_cancelled", from: "checked_in", to: "cancelled"},
		{name: "checked_in_to_expired", from: "checked_in", to: "expired"},
		{name: "checked_in_to_confirmed", from: "checked_in", to: "confirmed"},
		{name: "checked_in_to_pending", from: "checked_in", to: "pending"},
		{name: "pending_to_checked_in", from: "pending", to: "checked_in"},
		{name: "pending_to_checked_out", from: "pending", to: "checked_out"},
		{name: "pending_to_expired", from: "pending", to: "expired"},
		{name: "pending_to_cancelled", from: "pending", to: "cancelled"},
	}

	for _, tc := range invalidTransitions {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			err := reservationmodel.ValidateTransition(tc.from, tc.to)

			// Assert
			assert.Error(t, err,
				"transition from %q to %q should be rejected", tc.from, tc.to)
		})
	}
}
