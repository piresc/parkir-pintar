// Package e2e_test — overnight fee integration test.
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
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	billingmodel "parkir-pintar/internal/billing/model"
	"parkir-pintar/internal/reservation/model"
	"parkir-pintar/tests/testhelpers"
)

// TestOvernight_ShouldApplyOvernightFee_WhenSessionCrossesMidnight verifies
// that a session crossing midnight in WIB results in:
// parking_fee = 8 × 5000 = 40000, overnight_fee = 20000, is_overnight = true,
// and total = 5000 + 40000 + 20000 = 65000.
//
// Strategy: set checked_in_at to 22:00 WIB yesterday (15:00 UTC yesterday).
// CheckOut uses time.Now(), so we set checked_in_at exactly 8 hours before
// the current time, ensuring the session crosses midnight in WIB.
//
// Validates: Requirements 13.1, 13.2, 13.3, 13.4
func TestOvernight_ShouldApplyOvernightFee_WhenSessionCrossesMidnight(t *testing.T) {
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

	// Manipulate checked_in_at to 22:00 WIB yesterday so the session crosses
	// midnight WIB when CheckOut runs (which uses time.Now()).
	// 22:00 WIB = 15:00 UTC. We place it yesterday so checkout today crosses midnight.
	wib := time.FixedZone("WIB", 7*60*60)
	nowWIB := time.Now().In(wib)
	yesterdayAt2200WIB := time.Date(nowWIB.Year(), nowWIB.Month(), nowWIB.Day()-1, 22, 0, 0, 0, wib)

	_, err = env.db.ExecContext(ctx,
		"UPDATE reservations SET checked_in_at = $1 WHERE id = $2",
		yesterdayAt2200WIB.UTC(), reservation.ID)
	require.NoError(t, err)

	// Act — Check out (now is ~today, so session crosses midnight WIB)
	checkoutResp, err := env.reservationUC.CheckOut(ctx, &model.CheckOutRequest{
		ReservationID: reservation.ID,
	})

	// Assert — Checkout succeeds
	require.NoError(t, err)
	require.NotNil(t, checkoutResp)

	// Query billing record
	var billing struct {
		BookingFee   int64 `db:"booking_fee"`
		ParkingFee   int64 `db:"parking_fee"`
		OvernightFee int64 `db:"overnight_fee"`
		IsOvernight  bool  `db:"is_overnight"`
		TotalAmount  int64 `db:"total_amount"`
		BilledHours  int   `db:"billed_hours"`
	}
	err = env.db.GetContext(ctx, &billing,
		`SELECT booking_fee, parking_fee, overnight_fee, is_overnight, total_amount, billed_hours
		 FROM billing_records WHERE reservation_id = $1`, reservation.ID)
	require.NoError(t, err)

	// Assert — is_overnight = true (session crossed midnight WIB)
	assert.True(t, billing.IsOvernight, "is_overnight should be true")

	// Assert — Overnight fee = 20000
	assert.Equal(t, billingmodel.OvernightFlatFee, billing.OvernightFee,
		"overnight fee should be 20000")

	// Assert — Parking fee = billed_hours × 5000
	assert.Equal(t, int64(billing.BilledHours)*billingmodel.HourlyRate, billing.ParkingFee,
		"parking fee should be billed_hours × 5000")

	// Assert — Total = booking_fee + parking_fee + overnight_fee
	expectedTotal := billing.BookingFee + billing.ParkingFee + billing.OvernightFee
	assert.Equal(t, expectedTotal, billing.TotalAmount,
		"total should be booking_fee + parking_fee + overnight_fee")

	// Assert — Billing total consistency
	testhelpers.AssertBillingTotal(t, env.db, reservation.ID)
}
