// Package e2e_test — wrong-spot penalty integration test.
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
	"github.com/stretchr/testify/require"

	billingmodel "parkir-pintar/internal/billing/model"
	"parkir-pintar/internal/reservation/model"
	"parkir-pintar/tests/testhelpers"
)

// TestWrongSpot_ShouldApplyPenalty_WhenDriverParksInWrongSpot verifies that
// applying a wrong-spot penalty creates a penalty record and updates the
// billing total to include the penalty amount.
//
// Validates: Requirements 9.1, 9.2, 9.3, 9.4
func TestWrongSpot_ShouldApplyPenalty_WhenDriverParksInWrongSpot(t *testing.T) {
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

	// Act — Apply wrong-spot penalty via billing usecase
	_, err = env.billingUC.ApplyPenalty(ctx, &billingmodel.ApplyPenaltyRequest{
		ReservationID: reservation.ID,
		PenaltyType:   "wrong_spot",
		Amount:        billingmodel.WrongSpotPenalty,
		Description:   "driver parked in wrong spot",
	})

	// Assert — Penalty record exists
	require.NoError(t, err)
	testhelpers.AssertPenaltyExists(t, env.db, reservation.ID, "wrong_spot", billingmodel.WrongSpotPenalty)

	// Assert — Billing total includes penalty
	testhelpers.AssertBillingTotal(t, env.db, reservation.ID)
}
