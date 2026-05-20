// Package e2e_test — Property 2: Billing Total Consistency Through Real DB.
//
// Uses pgregory.net/rapid to generate random create→check-in→check-out
// lifecycles with varying durations, then asserts billing total consistency.
//
// Best practices applied (from Go testify testing standards):
// - Use require for assertions that must pass to continue (fail-fast)
// - Follow AAA (Arrange-Act-Assert) structure
// - Do not mock the database — these are integration tests against real PostgreSQL
package e2e_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"

	"parkir-pintar/internal/reservation/constants"
	"parkir-pintar/internal/reservation/model"
	"parkir-pintar/tests/testhelpers"
)

// TestProperty2_BillingTotalConsistency verifies that for any completed
// reservation lifecycle (create → check-in → check-out), the billing total
// equals booking_fee + parking_fee + overnight_fee + cancellation_fee + penalty_amount.
//
// **Validates: Requirements 5.4, 12.3, 13.4**
func TestProperty2_BillingTotalConsistency(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		ctx := context.Background()

		// Clean tables for isolation
		err := testhelpers.TruncateTables(ctx, env.db,
			"payments", "billing_records", "reservations", "drivers")
		require.NoError(t, err)
		err = testhelpers.ResetSpots(ctx, env.db)
		require.NoError(t, err)

		// Ensure payment gateway succeeds
		env.paymentGW.ShouldFail = false

		// Insert a test driver
		driverID, err := testhelpers.InsertTestDriver(ctx, env.db, "car")
		require.NoError(t, err)

		// Create reservation
		reservation, err := env.reservationUC.CreateReservation(ctx, &model.CreateReservationRequest{
			DriverID:       driverID,
			VehicleType:    "car",
			AssignmentMode: constants.AssignmentSystemAssigned,
			IdempotencyKey: uuid.New().String(),
		})
		require.NoError(t, err)
		require.NotNil(t, reservation)

		// Confirm reservation
		_, err = env.reservationUC.ConfirmReservation(ctx, &model.ConfirmReservationRequest{
			ReservationID: reservation.ID,
		})
		require.NoError(t, err)

		// Check in
		_, err = env.reservationUC.CheckIn(ctx, &model.CheckInRequest{
			ReservationID: reservation.ID,
		})
		require.NoError(t, err)

		// Manipulate checked_in_at to a random number of hours ago (1-24)
		hoursAgo := rapid.IntRange(1, 24).Draw(rt, "hoursAgo")
		pastTime := time.Now().Add(-time.Duration(hoursAgo) * time.Hour)
		_, err = env.db.ExecContext(ctx,
			"UPDATE reservations SET checked_in_at = $1 WHERE id = $2",
			pastTime, reservation.ID)
		require.NoError(t, err)

		// Check out
		_, err = env.reservationUC.CheckOut(ctx, &model.CheckOutRequest{
			ReservationID: reservation.ID,
		})
		require.NoError(t, err)

		// Assert billing total consistency
		testhelpers.AssertBillingTotal(t, env.db, reservation.ID)
	})
}
