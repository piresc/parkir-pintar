// Package model defines domain structs and state machine for the reservation module.
//
// Best practices applied (from coding standards KB):
// - Table-driven tests with t.Run() subtests for comprehensive state machine coverage
// - AAA pattern: Arrange → Act → Assert
// - testify/assert for assertions
// - errors.Is for sentinel error checking
// - t.Context() for context (Go 1.24+)
// - Each subtest is isolated and self-descriptive
package model

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateTransition_ShouldReturnNil_WhenTransitionIsValid(t *testing.T) {
	tests := []struct {
		name string
		from string
		to   string
	}{
		{name: "pending to confirmed", from: StatusPending, to: StatusConfirmed},
		{name: "confirmed to checked_in", from: StatusConfirmed, to: StatusCheckedIn},
		{name: "confirmed to expired", from: StatusConfirmed, to: StatusExpired},
		{name: "confirmed to cancelled", from: StatusConfirmed, to: StatusCancelled},
		{name: "checked_in to checked_out", from: StatusCheckedIn, to: StatusCheckedOut},
		{name: "checked_out to completed", from: StatusCheckedOut, to: StatusCompleted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			_ = t.Context()

			// Act
			err := ValidateTransition(tt.from, tt.to)

			// Assert
			assert.NoError(t, err)
		})
	}
}

func TestValidateTransition_ShouldReturnError_WhenTransitionIsInvalid(t *testing.T) {
	tests := []struct {
		name string
		from string
		to   string
	}{
		{name: "pending to checked_in", from: StatusPending, to: StatusCheckedIn},
		{name: "pending to checked_out", from: StatusPending, to: StatusCheckedOut},
		{name: "pending to expired", from: StatusPending, to: StatusExpired},
		{name: "pending to cancelled", from: StatusPending, to: StatusCancelled},
		{name: "confirmed to pending", from: StatusConfirmed, to: StatusPending},
		{name: "confirmed to confirmed", from: StatusConfirmed, to: StatusConfirmed},
		{name: "confirmed to checked_out", from: StatusConfirmed, to: StatusCheckedOut},
		{name: "checked_in to pending", from: StatusCheckedIn, to: StatusPending},
		{name: "checked_in to confirmed", from: StatusCheckedIn, to: StatusConfirmed},
		{name: "checked_in to expired", from: StatusCheckedIn, to: StatusExpired},
		{name: "checked_in to cancelled", from: StatusCheckedIn, to: StatusCancelled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			_ = t.Context()

			// Act
			err := ValidateTransition(tt.from, tt.to)

			// Assert
			assert.Error(t, err)
			assert.True(t, errors.Is(err, ErrInvalidTransition))
		})
	}
}

func TestValidateTransition_ShouldReturnError_WhenFromTerminalState(t *testing.T) {
	terminalStates := []string{StatusCompleted, StatusExpired, StatusCancelled, StatusFailed}
	allStatuses := []string{
		StatusPending, StatusConfirmed, StatusCheckedIn,
		StatusCheckedOut, StatusCompleted, StatusExpired, StatusCancelled, StatusFailed,
	}

	for _, terminal := range terminalStates {
		for _, target := range allStatuses {
			name := terminal + " to " + target
			t.Run(name, func(t *testing.T) {
				// Arrange
				_ = t.Context()

				// Act
				err := ValidateTransition(terminal, target)

				// Assert
				assert.Error(t, err)
				assert.True(t, errors.Is(err, ErrInvalidTransition))
			})
		}
	}
}
