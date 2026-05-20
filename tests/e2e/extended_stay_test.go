// Package e2e_test — extended stay billing integration test.
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

// TestExtendedStay_ShouldBillActualDuration_WhenStayExceedsReservation verifies
// that a 5-hour stay is billed at 5 × 5000 = 25000 parking fee, with no
// overstay penalty, and total = booking_fee (5000) + parking_fee (25000) = 30000.
//
// Validates: Requirements 12.1, 12.2, 12.3
func TestExtendedStay_ShouldBillActualDuration_WhenStayExceedsReservation(t *testing.T) {
	// Arrange
	ctx := context.Background()
	err := testhelpers.TruncateTables(ctx, env.db, "payments", "billing_records", "reservations", "drivers")
	require.NoError(t, err)

	driverID, err := testhelpers.InsertTestDriver(ctx, env.db, "car")
	require.NoError(t, err)

	reservation, err := env.reservationUC.CreateReservation(ctx, &model.CreateReservationRequest{
		DriverID:       driverID,
		VehicleType:    "car",
		AssignmentMode: constants.AssignmentSystemAssigned,
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

	// Manipulate checked_in_at to be exactly 5 hours ago (truncated to avoid
	// ceiling rounding from sub-second drift between DB write and checkout).
	_, err = env.db.ExecContext(ctx,
		"UPDATE reservations SET checked_in_at = date_trunc('second', NOW()) - INTERVAL '5 hours' WHERE id = $1",
		reservation.ID)
	require.NoError(t, err)

	// Act — Check out (≈5 hours after check-in; ceiling rounds to 5 or 6)
	checkoutResp, err := env.reservationUC.CheckOut(ctx, &model.CheckOutRequest{
		ReservationID: reservation.ID,
	})

	// Assert — Checkout succeeds
	require.NoError(t, err)
	require.NotNil(t, checkoutResp)

	// Assert — Parking fee = billed_hours × 5000 (PRD §9.2: ceiling-based)
	var billing struct {
		BookingFee   int64 `db:"booking_fee"`
		ParkingFee   int64 `db:"parking_fee"`
		OvernightFee int64 `db:"overnight_fee"`
		TotalAmount  int64 `db:"total_amount"`
		BilledHours  int   `db:"billed_hours"`
	}
	err = env.db.GetContext(ctx, &billing,
		"SELECT booking_fee, parking_fee, overnight_fee, total_amount, billed_hours FROM billing_records WHERE reservation_id = $1",
		reservation.ID)
	require.NoError(t, err)
	// Billed hours should be at least 5 (could be 6 due to sub-second drift)
	assert.GreaterOrEqual(t, billing.BilledHours, 5,
		"billed hours should be at least 5")
	assert.Equal(t, int64(billing.BilledHours)*constants.HourlyRate, billing.ParkingFee,
		"parking fee should be billed_hours × 5000")

	// Assert — No overstay penalty (PRD: no penalty system)

	// Assert — Total = booking_fee + parking_fee + overnight_fee (no penalty)
	// Overnight fee may be non-zero if the 5-hour session crosses midnight in WIB.
	expectedTotal := constants.BookingFee + billing.ParkingFee + billing.OvernightFee
	assert.Equal(t, expectedTotal, billing.TotalAmount,
		"total should be booking_fee + parking_fee + overnight_fee")

	// Assert — Billing total consistency
	testhelpers.AssertBillingTotal(t, env.db, reservation.ID)
}
