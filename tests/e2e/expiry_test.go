// Package e2e_test — reservation expiry integration test.
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

// TestExpiry_ShouldReleaseSpot_WhenReservationExpires verifies that
// expiring a reservation transitions it to EXPIRED and releases the spot.
// Per PRD, the booking fee (5,000 IDR, already charged at confirmation) is
// the only cost — no additional no-show penalty is applied.
//
// Validates: Requirements 8.1, 8.2, 8.3, 8.4
func TestExpiry_ShouldReleaseSpot_WhenReservationExpires(t *testing.T) {
	// Arrange
	ctx := context.Background()
	err := testhelpers.TruncateTables(ctx, env.db, "penalties", "payments", "billing_records", "reservations", "drivers")
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

	reservation, err = env.reservationUC.ConfirmReservation(ctx, &model.ConfirmReservationRequest{
		ReservationID: reservation.ID,
	})
	require.NoError(t, err)
	require.NotNil(t, reservation)

	// Act — Expire the reservation
	err = env.reservationUC.ExpireReservation(ctx, &model.ExpireReservationRequest{
		ReservationID: reservation.ID,
	})

	// Assert — Status EXPIRED
	require.NoError(t, err)

	var status string
	err = env.db.QueryRowContext(ctx,
		"SELECT status FROM reservations WHERE id = $1", reservation.ID).Scan(&status)
	require.NoError(t, err)
	assert.Equal(t, model.StatusExpired, status)

	// Assert — Spot back to "available"
	var spotStatus string
	err = env.db.QueryRowContext(ctx,
		"SELECT status FROM parking_spots WHERE id = $1", reservation.SpotID).Scan(&spotStatus)
	require.NoError(t, err)
	assert.Equal(t, "available", spotStatus)

	// Assert — No additional no-show penalty applied (booking fee is the only cost)
	testhelpers.AssertNoPenaltyExists(t, env.db, reservation.ID, "no_show")
}
