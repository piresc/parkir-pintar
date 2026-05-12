// Package e2e_test — payment checkout success integration test.
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

// TestPaymentSuccess_ShouldCreateSuccessRecord_WhenCheckoutCompletes verifies
// that a full reservation lifecycle (create → check-in → check-out) with a
// successful payment gateway produces a payment record with status "success"
// and method "qris", and the billing record transitions to "invoiced".
//
// Validates: Requirements 14.1, 14.2, 14.3
func TestPaymentSuccess_ShouldCreateSuccessRecord_WhenCheckoutCompletes(t *testing.T) {
	// Arrange
	ctx := context.Background()
	err := testhelpers.TruncateTables(ctx, env.db,
		"presence_logs", "penalties", "payments", "billing_records", "reservations", "drivers")
	require.NoError(t, err)

	env.paymentGW.ShouldFail = false

	driverID, err := testhelpers.InsertTestDriver(ctx, env.db, "car")
	require.NoError(t, err)

	// Act — Create reservation
	reservation, err := env.reservationUC.CreateReservation(ctx, &model.CreateReservationRequest{
		DriverID:       driverID,
		VehicleType:    "car",
		AssignmentMode: model.AssignmentSystemAssigned,
		IdempotencyKey: uuid.New().String(),
	})
	require.NoError(t, err)
	require.NotNil(t, reservation)

	// Act — Confirm reservation
	reservation, err = env.reservationUC.ConfirmReservation(ctx, &model.ConfirmReservationRequest{
		ReservationID: reservation.ID,
	})
	require.NoError(t, err)
	require.NotNil(t, reservation)

	// Act — Check in
	_, err = env.reservationUC.CheckIn(ctx, &model.CheckInRequest{
		ReservationID: reservation.ID,
	})
	require.NoError(t, err)

	// Act — Check out
	checkoutResp, err := env.reservationUC.CheckOut(ctx, &model.CheckOutRequest{
		ReservationID: reservation.ID,
	})
	require.NoError(t, err)
	require.NotNil(t, checkoutResp)

	// Assert — Payment record with status "success" and method "qris"
	var payment struct {
		Status        string `db:"status"`
		PaymentMethod string `db:"payment_method"`
	}
	err = env.db.GetContext(ctx, &payment,
		`SELECT status, payment_method FROM payments
		 WHERE billing_id = (SELECT id FROM billing_records WHERE reservation_id = $1)`,
		reservation.ID)
	require.NoError(t, err)
	assert.Equal(t, "success", payment.Status)
	assert.Equal(t, "qris", payment.PaymentMethod)

	// Assert — Billing record status is "invoiced"
	var billingStatus string
	err = env.db.QueryRowContext(ctx,
		"SELECT status FROM billing_records WHERE reservation_id = $1",
		reservation.ID).Scan(&billingStatus)
	require.NoError(t, err)
	assert.Equal(t, "invoiced", billingStatus)
}
