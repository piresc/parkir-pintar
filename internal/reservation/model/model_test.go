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
	reservationerrors "parkir-pintar/internal/reservation/constants"
)

func TestValidateTransition_ShouldReturnNil_WhenTransitionIsValid(t *testing.T) {
	tests := []struct {
		name string
		from string
		to   string
	}{
		{name: "pending to confirmed", from: string(constants.StatusPending), to: string(constants.StatusConfirmed)},
		{name: "confirmed to checked_in", from: string(constants.StatusConfirmed), to: string(constants.StatusCheckedIn)},
		{name: "confirmed to expired", from: string(constants.StatusConfirmed), to: string(constants.StatusExpired)},
		{name: "confirmed to cancelled", from: string(constants.StatusConfirmed), to: string(constants.StatusCancelled)},
		{name: "checked_in to checked_out", from: string(constants.StatusCheckedIn), to: string(constants.StatusCheckedOut)},
		{name: "checked_out to completed", from: string(constants.StatusCheckedOut), to: string(constants.StatusCompleted)},
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
		{name: "pending to checked_in", from: string(constants.StatusPending), to: string(constants.StatusCheckedIn)},
		{name: "pending to checked_out", from: string(constants.StatusPending), to: string(constants.StatusCheckedOut)},
		{name: "pending to expired", from: string(constants.StatusPending), to: string(constants.StatusExpired)},
		{name: "pending to cancelled", from: string(constants.StatusPending), to: string(constants.StatusCancelled)},
		{name: "confirmed to pending", from: string(constants.StatusConfirmed), to: string(constants.StatusPending)},
		{name: "confirmed to confirmed", from: string(constants.StatusConfirmed), to: string(constants.StatusConfirmed)},
		{name: "confirmed to checked_out", from: string(constants.StatusConfirmed), to: string(constants.StatusCheckedOut)},
		{name: "checked_in to pending", from: string(constants.StatusCheckedIn), to: string(constants.StatusPending)},
		{name: "checked_in to confirmed", from: string(constants.StatusCheckedIn), to: string(constants.StatusConfirmed)},
		{name: "checked_in to expired", from: string(constants.StatusCheckedIn), to: string(constants.StatusExpired)},
		{name: "checked_in to cancelled", from: string(constants.StatusCheckedIn), to: string(constants.StatusCancelled)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			_ = t.Context()

			// Act
			err := ValidateTransition(tt.from, tt.to)

			// Assert
			assert.Error(t, err)
			assert.True(t, errors.Is(err, reservationerrors.ErrInvalidTransition))
		})
	}
}

func TestValidateTransition_ShouldReturnError_WhenFromTerminalState(t *testing.T) {
	terminalStates := []string{string(constants.StatusCompleted), string(constants.StatusExpired), string(constants.StatusCancelled), string(constants.StatusFailed)}
	allStatuses := []string{
		string(constants.StatusPending), string(constants.StatusConfirmed), string(constants.StatusCheckedIn),
		string(constants.StatusCheckedOut), string(constants.StatusCompleted), string(constants.StatusExpired), string(constants.StatusCancelled), string(constants.StatusFailed),
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
				assert.True(t, errors.Is(err, reservationerrors.ErrInvalidTransition))
			})
		}
	}
}
