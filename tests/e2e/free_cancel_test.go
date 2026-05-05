// Package e2e_test — free cancellation within 2 minutes integration test.
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

// TestFreeCancel_ShouldNotApplyPenalty_WhenCancelledWithin2Minutes verifies
// that cancelling a reservation immediately (within the 2-minute free window)
// results in CANCELLED status, spot released, and no penalty.
//
// Validates: Requirements 10.1, 10.2, 10.3, 10.4
func TestFreeCancel_ShouldNotApplyPenalty_WhenCancelledWithin2Minutes(t *testing.T) {
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

	// Act — Cancel immediately (within 2-minute free window)
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

	// Assert — No penalty record exists
	testhelpers.AssertNoPenalty(t, env.db, reservation.ID)
}
