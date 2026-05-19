// Package model defines domain structs and state machine for the reservation module.
//
// Property-based tests for the reservation state machine using pgregory.net/rapid.
// These tests verify Property 4: State Machine Integrity from the design document.
//
// Best practices applied (from coding standards KB):
// - rapid.Custom generators to pick from status constants
// - errors.Is for sentinel error checking
// - t.Context() for context (Go 1.24+)
// - AAA pattern: Arrange → Act → Assert
// - testify/assert for assertions
package model

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"pgregory.net/rapid"
)

// allStatuses is the complete set of reservation statuses.
var allStatuses = []string{
	StatusPending,
	StatusWaitingPayment,
	StatusConfirmed,
	StatusCheckedIn,
	StatusCheckedOut,
	StatusCompleted,
	StatusExpired,
	StatusCancelled,
	StatusFailed,
}

// validTransitionPairs enumerates every allowed (from, to) pair.
var validTransitionPairs = [][2]string{
	{StatusPending, StatusConfirmed},
	{StatusWaitingPayment, StatusConfirmed},
	{StatusWaitingPayment, StatusFailed},
	{StatusWaitingPayment, StatusCancelled},
	{StatusConfirmed, StatusCheckedIn},
	{StatusConfirmed, StatusExpired},
	{StatusConfirmed, StatusCancelled},
	{StatusCheckedIn, StatusCheckedOut},
	{StatusCheckedOut, StatusCompleted},
}

// terminalStatuses are statuses with no outgoing transitions.
var terminalStatuses = []string{
	StatusCompleted,
	StatusExpired,
	StatusCancelled,
	StatusFailed,
}

// validTransitionSet builds a lookup set for O(1) membership checks.
var validTransitionSet = func() map[[2]string]bool {
	m := make(map[[2]string]bool, len(validTransitionPairs))
	for _, pair := range validTransitionPairs {
		m[pair] = true
	}
	return m
}()

// genValidTransition returns a rapid generator that picks a random valid (from, to) pair.
func genValidTransition() *rapid.Generator[[2]string] {
	return rapid.Custom[[2]string](func(t *rapid.T) [2]string {
		idx := rapid.IntRange(0, len(validTransitionPairs)-1).Draw(t, "validPairIdx")
		return validTransitionPairs[idx]
	})
}

// genInvalidTransition returns a rapid generator that picks a random (from, to) pair
// NOT in the allowed set.
func genInvalidTransition() *rapid.Generator[[2]string] {
	return rapid.Custom[[2]string](func(t *rapid.T) [2]string {
		for {
			fromIdx := rapid.IntRange(0, len(allStatuses)-1).Draw(t, "fromIdx")
			toIdx := rapid.IntRange(0, len(allStatuses)-1).Draw(t, "toIdx")
			pair := [2]string{allStatuses[fromIdx], allStatuses[toIdx]}
			if !validTransitionSet[pair] {
				return pair
			}
		}
	})
}

// genTerminalState returns a rapid generator that picks a random terminal state.
func genTerminalState() *rapid.Generator[string] {
	return rapid.Custom[string](func(t *rapid.T) string {
		idx := rapid.IntRange(0, len(terminalStatuses)-1).Draw(t, "terminalIdx")
		return terminalStatuses[idx]
	})
}

// genAnyStatus returns a rapid generator that picks any reservation status.
func genAnyStatus() *rapid.Generator[string] {
	return rapid.Custom[string](func(t *rapid.T) string {
		idx := rapid.IntRange(0, len(allStatuses)-1).Draw(t, "statusIdx")
		return allStatuses[idx]
	})
}

// TestProperty4_ValidTransitionsAlwaysSucceed verifies that for any valid (from, to)
// pair from the allowed set, ValidateTransition returns nil.
//
// **Validates: Requirements 7.1, 7.2, 7.3**
func TestProperty4_ValidTransitionsAlwaysSucceed(t *testing.T) {
	_ = t.Context()

	rapid.Check(t, func(t *rapid.T) {
		// Arrange
		pair := genValidTransition().Draw(t, "transition")
		from, to := pair[0], pair[1]

		// Act
		err := ValidateTransition(from, to)

		// Assert
		assert.NoError(t, err, "valid transition %q -> %q should succeed", from, to)
	})
}

// TestProperty4_InvalidTransitionsAlwaysFail verifies that for any (from, to) pair
// NOT in the allowed set, ValidateTransition returns an error wrapping ErrInvalidTransition.
//
// **Validates: Requirements 7.1, 7.2, 7.3**
func TestProperty4_InvalidTransitionsAlwaysFail(t *testing.T) {
	_ = t.Context()

	rapid.Check(t, func(t *rapid.T) {
		// Arrange
		pair := genInvalidTransition().Draw(t, "transition")
		from, to := pair[0], pair[1]

		// Act
		err := ValidateTransition(from, to)

		// Assert
		assert.Error(t, err, "invalid transition %q -> %q should fail", from, to)
		assert.True(t, errors.Is(err, ErrInvalidTransition),
			"error for %q -> %q should wrap ErrInvalidTransition", from, to)
	})
}

// TestProperty4_TerminalStatesHaveNoOutgoingTransitions verifies that for any terminal
// state (completed, expired, cancelled, failed) and any target status, ValidateTransition
// returns an error wrapping ErrInvalidTransition.
//
// **Validates: Requirements 7.1, 7.2, 7.3**
func TestProperty4_TerminalStatesHaveNoOutgoingTransitions(t *testing.T) {
	_ = t.Context()

	rapid.Check(t, func(t *rapid.T) {
		// Arrange
		from := genTerminalState().Draw(t, "terminalState")
		to := genAnyStatus().Draw(t, "targetStatus")

		// Act
		err := ValidateTransition(from, to)

		// Assert
		assert.Error(t, err, "transition from terminal state %q to %q should fail", from, to)
		assert.True(t, errors.Is(err, ErrInvalidTransition),
			"error for terminal %q -> %q should wrap ErrInvalidTransition", from, to)
	})
}
