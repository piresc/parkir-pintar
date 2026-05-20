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

	"parkir-pintar/internal/reservation/constants"
)

func TestValidateTransition_ShouldReturnNil_WhenTransitionIsValid(t *testing.T) {
	tests := []struct {
		name string
		from string
		to   string
	}{
		{name: "pending to confirmed", from: constants.StatusPending, to: constants.StatusConfirmed},
		{name: "confirmed to checked_in", from: constants.StatusConfirmed, to: constants.StatusCheckedIn},
		{name: "confirmed to expired", from: constants.StatusConfirmed, to: constants.StatusExpired},
		{name: "confirmed to cancelled", from: constants.StatusConfirmed, to: constants.StatusCancelled},
		{name: "checked_in to checked_out", from: constants.StatusCheckedIn, to: constants.StatusCheckedOut},
		{name: "checked_out to completed", from: constants.StatusCheckedOut, to: constants.StatusCompleted},
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
		{name: "pending to checked_in", from: constants.StatusPending, to: constants.StatusCheckedIn},
		{name: "pending to checked_out", from: constants.StatusPending, to: constants.StatusCheckedOut},
		{name: "pending to expired", from: constants.StatusPending, to: constants.StatusExpired},
		{name: "pending to cancelled", from: constants.StatusPending, to: constants.StatusCancelled},
		{name: "confirmed to pending", from: constants.StatusConfirmed, to: constants.StatusPending},
		{name: "confirmed to confirmed", from: constants.StatusConfirmed, to: constants.StatusConfirmed},
		{name: "confirmed to checked_out", from: constants.StatusConfirmed, to: constants.StatusCheckedOut},
		{name: "checked_in to pending", from: constants.StatusCheckedIn, to: constants.StatusPending},
		{name: "checked_in to confirmed", from: constants.StatusCheckedIn, to: constants.StatusConfirmed},
		{name: "checked_in to expired", from: constants.StatusCheckedIn, to: constants.StatusExpired},
		{name: "checked_in to cancelled", from: constants.StatusCheckedIn, to: constants.StatusCancelled},
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
	terminalStates := []string{constants.StatusCompleted, constants.StatusExpired, constants.StatusCancelled, constants.StatusFailed}
	allStatuses := []string{
		constants.StatusPending, constants.StatusConfirmed, constants.StatusCheckedIn,
		constants.StatusCheckedOut, constants.StatusCompleted, constants.StatusExpired, constants.StatusCancelled, constants.StatusFailed,
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
