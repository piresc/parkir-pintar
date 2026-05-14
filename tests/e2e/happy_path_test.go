// Package e2e_test — full reservation lifecycle (happy path) integration test.
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
	"parkir-pintar/pkg/pricing"
	"parkir-pintar/tests/testhelpers"
)

// TestHappyPath_ShouldCompleteFullLifecycle_WhenSystemAssigned verifies the
// complete reservation lifecycle: create → confirm → check-in → check-out with system
// assignment, billing, and payment processing.
//
// Validates: Requirements 5.1, 5.2, 5.3, 5.4, 5.5, 5.6, 5.7
func TestHappyPath_ShouldCompleteFullLifecycle_WhenSystemAssigned(t *testing.T) {
	// Arrange
	ctx := context.Background()
	err := testhelpers.TruncateTables(ctx, env.db, "penalties", "payments", "billing_records", "reservations", "drivers")
	require.NoError(t, err)

	driverID, err := testhelpers.InsertTestDriver(ctx, env.db, "car")
	require.NoError(t, err)

	// Act — Step 1: Create system-assigned reservation
	reservation, err := env.reservationUC.CreateReservation(ctx, &model.CreateReservationRequest{
		DriverID:       driverID,
		VehicleType:    "car",
		AssignmentMode: model.AssignmentSystemAssigned,
		IdempotencyKey: uuid.New().String(),
	})

	// Assert — Step 1: Reservation waiting payment, spot reserved
	require.NoError(t, err)
	require.NotNil(t, reservation)
	assert.Equal(t, model.StatusWaitingPayment, reservation.Status)
	assert.NotEmpty(t, reservation.SpotID)

	// Act — Step 2: Confirm reservation
	reservation, err = env.reservationUC.ConfirmReservation(ctx, &model.ConfirmReservationRequest{
		ReservationID: reservation.ID,
	})

	// Assert — Step 2: Reservation confirmed
	require.NoError(t, err)
	require.NotNil(t, reservation)
	assert.Equal(t, model.StatusConfirmed, reservation.Status)

	var spotStatus string
	err = env.db.QueryRowContext(ctx,
		"SELECT status FROM parking_spots WHERE id = $1", reservation.SpotID).Scan(&spotStatus)
	require.NoError(t, err)
	assert.Equal(t, "reserved", spotStatus)

	// Assert — Step 3: Billing record created with booking_fee=5000
	var billingFee int64
	err = env.db.QueryRowContext(ctx,
		"SELECT booking_fee FROM billing_records WHERE reservation_id = $1",
		reservation.ID).Scan(&billingFee)
	require.NoError(t, err)
	assert.Equal(t, pricing.BookingFee, billingFee)

	// Act — Step 4: Check in
	checkedIn, err := env.reservationUC.CheckIn(ctx, &model.CheckInRequest{
		ReservationID: reservation.ID,
	})

	// Assert — Step 4: CHECKED_IN, spot occupied
	require.NoError(t, err)
	assert.Equal(t, model.StatusCheckedIn, checkedIn.Status)

	err = env.db.QueryRowContext(ctx,
		"SELECT status FROM parking_spots WHERE id = $1", reservation.SpotID).Scan(&spotStatus)
	require.NoError(t, err)
	assert.Equal(t, "occupied", spotStatus)

	// Act — Step 5: Check out
	checkoutResp, err := env.reservationUC.CheckOut(ctx, &model.CheckOutRequest{
		ReservationID: reservation.ID,
	})

	// Assert — Step 5: CHECKED_OUT, billing calculated (spot still occupied until CompleteCheckout)
	require.NoError(t, err)
	require.NotNil(t, checkoutResp)
	assert.Equal(t, model.StatusCheckedOut, checkoutResp.Reservation.Status)
	assert.Greater(t, checkoutResp.TotalAmount, int64(0))

	err = env.db.QueryRowContext(ctx,
		"SELECT status FROM parking_spots WHERE id = $1", reservation.SpotID).Scan(&spotStatus)
	require.NoError(t, err)
	assert.Equal(t, "occupied", spotStatus)

	// Act — Step 6: Complete checkout (processes payment and releases spot)
	completeResp, err := env.reservationUC.CompleteCheckout(ctx, &model.CompleteCheckoutRequest{
		ReservationID: reservation.ID,
	})

	// Assert — Step 6: Payment processed, spot available
	require.NoError(t, err)
	require.NotNil(t, completeResp)
	assert.Equal(t, model.StatusCheckedOut, completeResp.Reservation.Status)

	err = env.db.QueryRowContext(ctx,
		"SELECT status FROM parking_spots WHERE id = $1", reservation.SpotID).Scan(&spotStatus)
	require.NoError(t, err)
	assert.Equal(t, "available", spotStatus)

	// Verify parking fee = ceil(duration_hours) × 5000
	var billing struct {
		ParkingFee  int64 `db:"parking_fee"`
		BilledHours int   `db:"billed_hours"`
	}
	err = env.db.GetContext(ctx, &billing,
		"SELECT parking_fee, billed_hours FROM billing_records WHERE reservation_id = $1",
		reservation.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(billing.BilledHours)*pricing.HourlyRate, billing.ParkingFee)

	// Verify payment record exists with status success
	var paymentStatus string
	err = env.db.QueryRowContext(ctx,
		"SELECT status FROM payments WHERE billing_id = (SELECT id FROM billing_records WHERE reservation_id = $1)",
		reservation.ID).Scan(&paymentStatus)
	require.NoError(t, err)
	assert.Equal(t, "success", paymentStatus)

	// Verify billing total consistency
	testhelpers.AssertBillingTotal(t, env.db, reservation.ID)
}
