// Package e2e_test — payment checkout failure integration test.
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
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"parkir-pintar/internal/reservation/constants"
	"parkir-pintar/internal/reservation/model"
	"parkir-pintar/tests/testhelpers"
)

// TestPaymentFailure_ShouldCreateFailedRecord_WhenGatewayFails verifies that
// when the payment gateway fails, the payment record is created with status
// "failed" but the checkout still completes.
//
// Validates: Requirements 15.1, 15.2, 15.3
func TestPaymentFailure_ShouldCreateFailedRecord_WhenGatewayFails(t *testing.T) {
	// Arrange
	ctx := context.Background()
	err := testhelpers.TruncateTables(ctx, env.db,
		"payments", "billing_records", "reservations", "drivers")
	require.NoError(t, err)

	driverID, err := testhelpers.InsertTestDriver(ctx, env.db, "car")
	require.NoError(t, err)

	// Create reservation with gateway succeeding (for billing setup)
	env.paymentGW.ShouldFail = false
	reservation, err := env.reservationUC.CreateReservation(ctx, &model.CreateReservationRequest{
		DriverID:       driverID,
		VehicleType:    "car",
		AssignmentMode: constants.AssignmentSystemAssigned,
		IdempotencyKey: uuid.New().String(),
	})
	require.NoError(t, err)
	require.NotNil(t, reservation)

	// Confirm reservation
	reservation, err = env.reservationUC.ConfirmReservation(ctx, &model.ConfirmReservationRequest{
		ReservationID: reservation.ID,
	})
	require.NoError(t, err)
	require.NotNil(t, reservation)

	// Check in
	_, err = env.reservationUC.CheckIn(ctx, &model.CheckInRequest{
		ReservationID: reservation.ID,
	})
	require.NoError(t, err)

	// Act — Set gateway to fail, then complete checkout (which processes payment)
	env.paymentGW.ShouldFail = true
	defer func() { env.paymentGW.ShouldFail = false }()

	checkoutResp, err := env.reservationUC.CheckOut(ctx, &model.CheckOutRequest{
		ReservationID: reservation.ID,
	})
	require.NoError(t, err)
	require.NotNil(t, checkoutResp)

	// CompleteCheckout should fail due to payment gateway failure
	_, completeErr := env.reservationUC.CompleteCheckout(ctx, &model.CompleteCheckoutRequest{
		ReservationID: reservation.ID,
	})

	// CompleteCheckout succeeds even when payment fails — the payment record
	// is created with status "failed" but the checkout flow completes.
	require.NoError(t, completeErr, "CompleteCheckout should succeed (payment failure is recorded, not propagated)")

	// Query the checkout payment (not the booking fee payment) by idempotency key
	var paymentStatus string
	queryErr := env.db.QueryRowContext(ctx,
		`SELECT status FROM payments
		 WHERE idempotency_key = $1`,
		fmt.Sprintf("payment-%s", reservation.ID)).Scan(&paymentStatus)
	require.NoError(t, queryErr)

	assert.Equal(t, "failed", paymentStatus)
}
