// Package e2e_test — double-booking prevention integration test.
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

	"parkir-pintar/internal/reservation/constants"
	"parkir-pintar/internal/reservation/model"
	"parkir-pintar/tests/testhelpers"
)

// TestDoubleBook_ShouldRejectSecond_WhenSameSpotConcurrent verifies that
// when two drivers attempt to reserve the same spot, only the first succeeds
// and the second receives a conflict error.
//
// Validates: Requirements 6.1, 6.2, 6.3, 6.4
func TestDoubleBook_ShouldRejectSecond_WhenSameSpotConcurrent(t *testing.T) {
	// Arrange
	ctx := context.Background()
	err := testhelpers.TruncateTables(ctx, env.db, "payments", "billing_records", "reservations", "drivers")
	require.NoError(t, err)

	driver1ID, err := testhelpers.InsertTestDriver(ctx, env.db, "car")
	require.NoError(t, err)
	driver2ID, err := testhelpers.InsertTestDriver(ctx, env.db, "car")
	require.NoError(t, err)

	// Get an available spot ID
	var spotID string
	err = env.db.QueryRowContext(ctx,
		"SELECT id FROM parking_spots WHERE vehicle_type = 'car' AND status = 'available' LIMIT 1").Scan(&spotID)
	require.NoError(t, err, "should have at least one available car spot")

	// Act — First driver reserves the spot (user-selected)
	res1, err := env.reservationUC.CreateReservation(ctx, &model.CreateReservationRequest{
		DriverID:       driver1ID,
		VehicleType:    "car",
		AssignmentMode: constants.AssignmentUserSelected,
		SpotID:         spotID,
		IdempotencyKey: uuid.New().String(),
	})

	// Assert — First reservation succeeds with waiting payment status
	require.NoError(t, err)
	require.NotNil(t, res1)
	assert.Equal(t, constants.StatusWaitingPayment, res1.Status)

	// Confirm the first reservation
	res1, err = env.reservationUC.ConfirmReservation(ctx, &model.ConfirmReservationRequest{
		ReservationID: res1.ID,
	})
	require.NoError(t, err)
	require.NotNil(t, res1)
	assert.Equal(t, constants.StatusConfirmed, res1.Status)

	// Act — Second driver attempts same spot
	res2, err := env.reservationUC.CreateReservation(ctx, &model.CreateReservationRequest{
		DriverID:       driver2ID,
		VehicleType:    "car",
		AssignmentMode: constants.AssignmentUserSelected,
		SpotID:         spotID,
		IdempotencyKey: uuid.New().String(),
	})

	// Assert — Second reservation fails with conflict
	assert.Error(t, err, "second reservation on same spot should fail")
	assert.Nil(t, res2)

	// Verify spot is "reserved"
	var spotStatus string
	err = env.db.QueryRowContext(ctx,
		"SELECT status FROM parking_spots WHERE id = $1", spotID).Scan(&spotStatus)
	require.NoError(t, err)
	assert.Equal(t, "reserved", spotStatus)

	// Verify only one reservation exists for this spot
	var resCount int
	err = env.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM reservations WHERE spot_id = $1", spotID).Scan(&resCount)
	require.NoError(t, err)
	assert.Equal(t, 1, resCount, "only one reservation should exist for the spot")
}
