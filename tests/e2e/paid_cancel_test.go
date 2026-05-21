// Package e2e_test — cancellation after confirmation integration test.
//
// Best practices applied (from Go testify testing standards):
// - Use require for assertions that must pass to continue (fail-fast)
// - Use assert for non-critical checks
// - Follow AAA (Arrange-Act-Assert) structure
// - Use descriptive test names: Test[Scenario]_Should[Expected]_When[Condition]
// - Do not mock the database — these are integration tests against real PostgreSQL
package e2e_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"parkir-pintar/internal/pricing"
	"parkir-pintar/internal/reservation/constants"
	"parkir-pintar/internal/reservation/model"
	"parkir-pintar/tests/testhelpers"
)

// TestCancelReservation_ShouldForfeitBookingFee_WhenCancelledAfterConfirmation verifies
// that cancelling a confirmed reservation results in CANCELLED status, spot released,
// and the booking fee (5,000 IDR already charged on confirmation) is NOT refunded.
// No additional cancellation fee or penalty is charged.
//
// Business rule: "No cancellation fee. Driver forfeits the 5,000 IDR booking fee
// already charged on confirmation. No additional penalty."
//
// Validates: Requirements 11.1, 11.2, 11.3
func TestCancelReservation_ShouldForfeitBookingFee_WhenCancelledAfterConfirmation(t *testing.T) {
	// Arrange
	ctx := context.Background()
	err := testhelpers.TruncateTables(ctx, env.db, "payments", "billing_records", "reservations", "drivers")
	require.NoError(t, err)

	driverID, err := testhelpers.InsertTestDriver(ctx, env.db, "car")
	require.NoError(t, err)

	reservation, err := env.reservationUC.CreateReservation(ctx, &model.CreateReservationRequest{
		DriverID:       driverID,
		VehicleType:    "car",
		AssignmentMode: string(constants.AssignmentSystemAssigned),
		IdempotencyKey: uuid.New().String(),
	})
	require.NoError(t, err)
	require.NotNil(t, reservation)

	// Confirm reservation — this charges the 5,000 IDR booking fee
	reservation, err = env.reservationUC.ConfirmReservation(ctx, &model.ConfirmReservationRequest{
		ReservationID: reservation.ID,
	})
	require.NoError(t, err)
	require.NotNil(t, reservation)
	assert.Equal(t, string(constants.StatusConfirmed), reservation.Status)

	// Verify booking fee was charged
	var bookingFee int64
	err = env.db.QueryRowContext(ctx,
		"SELECT booking_fee FROM billing_records WHERE reservation_id = $1",
		reservation.ID).Scan(&bookingFee)
	require.NoError(t, err)
	assert.Equal(t, pricing.BookingFee, bookingFee, "booking fee should be 5000 IDR")

	// Count billing records before cancellation
	var billingCountBefore int
	err = env.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM billing_records WHERE reservation_id = $1",
		reservation.ID).Scan(&billingCountBefore)
	require.NoError(t, err)

	// Act — Cancel the confirmed reservation
	cancelled, err := env.reservationUC.CancelReservation(ctx, &model.CancelReservationRequest{
		ReservationID: reservation.ID,
	})

	// Assert — CANCELLED status
	require.NoError(t, err)
	require.NotNil(t, cancelled)
	assert.Equal(t, string(constants.StatusCancelled), cancelled.Status)

	// Assert — Spot back to "available"
	var spotStatus string
	err = env.db.QueryRowContext(ctx,
		"SELECT status FROM parking_spots WHERE id = $1", reservation.SpotID).Scan(&spotStatus)
	require.NoError(t, err)
	assert.Equal(t, "available", spotStatus)

	// Assert — No new billing records created (no cancellation fee charged)
	var billingCountAfter int
	err = env.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM billing_records WHERE reservation_id = $1",
		reservation.ID).Scan(&billingCountAfter)
	require.NoError(t, err)
	assert.Equal(t, billingCountBefore, billingCountAfter,
		"no new billing record should be created on cancellation")

	// Assert — Booking fee still exists (NOT refunded)
	var bookingFeeAfter int64
	err = env.db.QueryRowContext(ctx,
		"SELECT booking_fee FROM billing_records WHERE reservation_id = $1",
		reservation.ID).Scan(&bookingFeeAfter)
	require.NoError(t, err)
	assert.Equal(t, pricing.BookingFee, bookingFeeAfter,
		"booking fee should remain charged (non-refundable)")

	// Assert — No penalty/cancellation_fee columns exist (removed by migration 000011)
	// This is implicitly verified by the schema — the columns don't exist.
}
