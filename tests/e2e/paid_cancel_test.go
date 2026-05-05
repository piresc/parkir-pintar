// Package e2e_test — paid cancellation after 2 minutes integration test.
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

	billingmodel "parkir-pintar/internal/billing/model"
	"parkir-pintar/internal/reservation/model"
	"parkir-pintar/tests/testhelpers"
)

// TestPaidCancel_ShouldApplyFee_WhenCancelledAfter2Minutes verifies that
// cancelling a reservation after the 2-minute free window results in
// CANCELLED status and a cancellation penalty of 5000 IDR.
//
// Validates: Requirements 11.1, 11.2, 11.3, 11.4
func TestPaidCancel_ShouldApplyFee_WhenCancelledAfter2Minutes(t *testing.T) {
	// Arrange
	ctx := context.Background()
	err := testhelpers.TruncateTables(ctx, env.db, "presence_logs", "penalties", "payments", "billing_records", "reservations", "drivers")
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

	// Manipulate confirmed_at to be 3 minutes ago so cancellation is past the free window
	_, err = env.db.ExecContext(ctx,
		"UPDATE reservations SET confirmed_at = NOW() - INTERVAL '3 minutes' WHERE id = $1",
		reservation.ID)
	require.NoError(t, err)

	// Act — Cancel reservation (now >2 minutes after confirmation)
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

	// Assert — Cancellation penalty of 5000 IDR exists
	testhelpers.AssertPenaltyExists(t, env.db, reservation.ID, "cancellation", billingmodel.CancelFee)
}
