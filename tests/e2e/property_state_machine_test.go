// Package e2e_test — Property 3: State Machine Enforcement Through Usecase Layer.
//
// Uses pgregory.net/rapid to generate reservations in terminal states, then
// attempts random transitions and asserts they all fail.
//
// Best practices applied (from Go testify testing standards):
// - Use require for assertions that must pass to continue (fail-fast)
// - Follow AAA (Arrange-Act-Assert) structure
// - Do not mock the database — these are integration tests against real PostgreSQL
package e2e_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"

	"parkir-pintar/internal/reservation/model"
	"parkir-pintar/tests/testhelpers"
)

// TestProperty3_StateMachineEnforcement verifies that for any reservation in a
// terminal state (expired, cancelled), attempting any transition through the
// usecase layer returns an error and the status remains unchanged in the DB.
//
// **Validates: Requirements 4.5, 4.6**
func TestProperty3_StateMachineEnforcement(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		ctx := context.Background()

		// Clean tables for isolation
		err := testhelpers.TruncateTables(ctx, env.db,
			"presence_logs", "penalties", "payments", "billing_records", "reservations", "drivers")
		require.NoError(t, err)

		env.paymentGW.ShouldFail = false

		// Insert a test driver
		driverID, err := testhelpers.InsertTestDriver(ctx, env.db, "car")
		require.NoError(t, err)

		// Create reservation
		reservation, err := env.reservationUC.CreateReservation(ctx, &model.CreateReservationRequest{
			DriverID:       driverID,
			VehicleType:    "car",
			AssignmentMode: model.AssignmentSystemAssigned,
			IdempotencyKey: uuid.New().String(),
		})
		require.NoError(t, err)
		require.NotNil(t, reservation)

		// Move to a random terminal state
		terminalOp := rapid.SampledFrom([]string{"expire", "cancel"}).Draw(rt, "terminalOp")

		switch terminalOp {
		case "expire":
			err = env.reservationUC.ExpireReservation(ctx, &model.ExpireReservationRequest{
				ReservationID: reservation.ID,
			})
			require.NoError(t, err)
		case "cancel":
			_, err = env.reservationUC.CancelReservation(ctx, &model.CancelReservationRequest{
				ReservationID: reservation.ID,
			})
			require.NoError(t, err)
		}

		// Verify the reservation is in a terminal state
		var currentStatus string
		err = env.db.QueryRowContext(ctx,
			"SELECT status FROM reservations WHERE id = $1", reservation.ID).Scan(&currentStatus)
		require.NoError(t, err)

		// Attempt a random transition — all should fail
		transition := rapid.SampledFrom([]string{"checkin", "cancel", "expire"}).Draw(rt, "transition")

		switch transition {
		case "checkin":
			_, transErr := env.reservationUC.CheckIn(ctx, &model.CheckInRequest{
				ReservationID: reservation.ID,
			})
			assert.Error(t, transErr, "check-in from terminal state %s should fail", currentStatus)

		case "cancel":
			_, transErr := env.reservationUC.CancelReservation(ctx, &model.CancelReservationRequest{
				ReservationID: reservation.ID,
			})
			assert.Error(t, transErr, "cancel from terminal state %s should fail", currentStatus)

		case "expire":
			transErr := env.reservationUC.ExpireReservation(ctx, &model.ExpireReservationRequest{
				ReservationID: reservation.ID,
			})
			assert.Error(t, transErr, "expire from terminal state %s should fail", currentStatus)
		}

		// Assert status remains unchanged in DB
		var statusAfter string
		err = env.db.QueryRowContext(ctx,
			"SELECT status FROM reservations WHERE id = $1", reservation.ID).Scan(&statusAfter)
		require.NoError(t, err)
		assert.Equal(t, currentStatus, statusAfter,
			"status should remain %s after failed transition attempt", currentStatus)
	})
}
