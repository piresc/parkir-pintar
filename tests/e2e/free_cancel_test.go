// Package e2e_test — cancellation from waiting_payment state integration test.
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

	"parkir-pintar/internal/reservation/model"
	"parkir-pintar/tests/testhelpers"
)

// TestCancelReservation_ShouldReleaseSpot_WhenCancelledBeforeConfirmation verifies
// that cancelling a reservation still in waiting_payment state (before payment is
// processed) results in CANCELLED status and spot released. The billing record
// exists (created on reservation) but no payment was collected, so the driver
// loses nothing. No additional penalty is charged.
//
// Business rule: No cancellation fee. No additional penalty.
// Validates: Requirements 10.1, 10.2, 10.3, 10.4
func TestCancelReservation_ShouldReleaseSpot_WhenCancelledBeforeConfirmation(t *testing.T) {
	// Arrange
	ctx := context.Background()
	err := testhelpers.TruncateTables(ctx, env.db, "payments", "billing_records", "reservations", "drivers")
	require.NoError(t, err)

	driverID, err := testhelpers.InsertTestDriver(ctx, env.db, "car")
	require.NoError(t, err)

	reservation, err := env.reservationUC.CreateReservation(ctx, &model.CreateReservationRequest{
		DriverID:       driverID,
		VehicleType:    "car",
		AssignmentMode: model.AssignmentSystemAssigned,
		IdempotencyKey: uuid.New().String(),
	})
	require.NoError(t, err)
	require.NotNil(t, reservation)
	assert.Equal(t, model.StatusWaitingPayment, reservation.Status)

	// Count billing records before cancellation
	var billingCountBefore int
	err = env.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM billing_records WHERE reservation_id = $1",
		reservation.ID).Scan(&billingCountBefore)
	require.NoError(t, err)

	// Act — Cancel before confirmation (payment not yet processed)
	cancelled, err := env.reservationUC.CancelReservation(ctx, &model.CancelReservationRequest{
		ReservationID: reservation.ID,
	})

	// Assert — CANCELLED status
	require.NoError(t, err)
	require.NotNil(t, cancelled)
	assert.Equal(t, model.StatusCancelled, cancelled.Status)

	// Assert — Spot back to "available"
	var spotStatus string
	err = env.db.QueryRowContext(ctx,
		"SELECT status FROM parking_spots WHERE id = $1", reservation.SpotID).Scan(&spotStatus)
	require.NoError(t, err)
	assert.Equal(t, "available", spotStatus)

	// Assert — No new billing records created (no cancellation fee)
	var billingCountAfter int
	err = env.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM billing_records WHERE reservation_id = $1",
		reservation.ID).Scan(&billingCountAfter)
	require.NoError(t, err)
	assert.Equal(t, billingCountBefore, billingCountAfter,
		"no new billing record should be created on cancellation")

	// Assert — No payment was processed (booking fee not collected since not confirmed)
	var paymentCount int
	err = env.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM payments WHERE billing_id = (SELECT id FROM billing_records WHERE reservation_id = $1)`,
		reservation.ID).Scan(&paymentCount)
	require.NoError(t, err)
	assert.Equal(t, 0, paymentCount,
		"no payment should exist since reservation was never confirmed")
}
