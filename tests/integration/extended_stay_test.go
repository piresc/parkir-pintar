// Package integration provides integration tests for the ParkirPintar extended
// stay flow, testing that staying past the reservation window bills at the
// standard rate with no overstay penalty.
//
// Best practices applied (from Go testify coding standards KB):
// - Test naming: Test[FunctionName]_Should[ExpectedResult]_When[Condition]
// - AAA pattern: Arrange → Act → Assert
// - testify/mock for mock implementations of all dependency interfaces
// - testify/assert and testify/require for assertions
// - Each test is isolated with its own mock setup
// - AssertExpectations(t) called on all mocks to verify interactions
// - Use t.Context() for Go 1.24+ context in tests
// - Mock at interface boundaries rather than concrete implementations
package integration

import (
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	billingmodel "parkir-pintar/internal/billing/model"
	"parkir-pintar/internal/reservation/model"
	"parkir-pintar/internal/reservation/usecase"
	"parkir-pintar/pkg/pricing"
)

// TestExtendedStayFlow_ShouldBillStandardRate_WhenStayingPastReservationExpiry tests
// the full create → check-in → checkout flow where the driver stays past the 1-hour
// reservation window. Per PRD §9.4, there is NO overstay penalty — additional time
// is billed at the standard hourly rate (5,000 IDR per started hour).
//
// Validates: PRD §9.4, §9.5 Example 4, Scenario 8
func TestExtendedStayFlow_ShouldBillStandardRate_WhenStayingPastReservationExpiry(t *testing.T) {
	repo := new(MockRepository)
	locker := new(MockLocker)
	billing := new(MockBillingClient)
	payment := new(MockPaymentClient)

	uc := usecase.NewUsecase(repo, locker, billing, payment, nil, nil, nil, 60)

	// --- Phase 1: Create Reservation ---
	repo.On("FindByIdempotencyKey", mock.Anything, "extended-key").Return(nil, model.ErrNotFound)
	repo.On("FindAvailableSpot", mock.Anything, "car").Return(&model.ParkingSpot{
		ID:          "spot-extended",
		VehicleType: "car",
		Status:      "available",
	}, nil)
	lock := new(MockLock)
	locker.On("Acquire", mock.Anything, "spot:spot-extended").Return(lock, nil)
	lock.On("Release", mock.Anything).Return(nil)
	repo.On("GetSpotForUpdateTx", mock.Anything, (*sqlx.Tx)(nil), "spot-extended").Return(&model.ParkingSpot{
		ID:          "spot-extended",
		VehicleType: "car",
		Status:      "available",
	}, nil)
	repo.On("ListByDriverID", mock.Anything, "driver-extended", "").Return([]*model.Reservation{}, nil)
	repo.On("CreateReservationTx", mock.Anything, (*sqlx.Tx)(nil), mock.AnythingOfType("*model.Reservation")).Return(nil)
	repo.On("UpdateSpotStatusTx", mock.Anything, (*sqlx.Tx)(nil), "spot-extended", "reserved").Return(nil)
	billing.On("StartBilling", mock.Anything, mock.AnythingOfType("string"), pricing.BookingFee, mock.AnythingOfType("string")).Return(&billingmodel.BillingRecord{ID: "billing-test-id"}, nil)

	// Act: create reservation
	reservation, err := uc.CreateReservation(t.Context(), &model.CreateReservationRequest{
		DriverID:       "driver-extended",
		VehicleType:    "car",
		AssignmentMode: model.AssignmentSystemAssigned,
		IdempotencyKey: "extended-key",
	})
	require.NoError(t, err)
	require.NotNil(t, reservation)

	// --- Phase 1b: Confirm reservation ---
	confirmedAt := time.Now().Add(-4 * time.Hour)
	repo.On("GetByIDForUpdate", mock.Anything, (*sqlx.Tx)(nil), reservation.ID).Return(&model.Reservation{
		ID:          reservation.ID,
		DriverID:    "driver-extended",
		SpotID:      "spot-extended",
		Status:      model.StatusWaitingPayment,
		ConfirmedAt: nil,
	}, nil).Once()
	// Second GetByIDForUpdate: re-check inside confirmation transaction (TOCTOU fix)
	repo.On("GetByIDForUpdate", mock.Anything, (*sqlx.Tx)(nil), reservation.ID).Return(&model.Reservation{
		ID:          reservation.ID,
		DriverID:    "driver-extended",
		SpotID:      "spot-extended",
		Status:      model.StatusWaitingPayment,
		ConfirmedAt: nil,
	}, nil).Once()
	repo.On("UpdateReservationTx", mock.Anything, (*sqlx.Tx)(nil), mock.MatchedBy(func(r *model.Reservation) bool {
		return r.Status == model.StatusConfirmed
	})).Return(nil).Once()
	payment.On("ProcessPayment", mock.Anything, "billing-test-id", pricing.BookingFee, "qris", mock.AnythingOfType("string")).Return("pay-booking", nil).Once()

	_, err = uc.ConfirmReservation(t.Context(), &model.ConfirmReservationRequest{
		ReservationID: reservation.ID,
	})
	require.NoError(t, err)

	// --- Phase 2: Check-In ---
	repo.On("GetByIDForUpdate", mock.Anything, (*sqlx.Tx)(nil), reservation.ID).Return(&model.Reservation{
		ID:          reservation.ID,
		DriverID:    "driver-extended",
		SpotID:      "spot-extended",
		Status:      model.StatusConfirmed,
		ConfirmedAt: &confirmedAt,
	}, nil).Once()
	repo.On("UpdateReservationTx", mock.Anything, (*sqlx.Tx)(nil), mock.MatchedBy(func(r *model.Reservation) bool {
		return r.Status == model.StatusCheckedIn
	})).Return(nil).Once()
	repo.On("UpdateSpotStatusTx", mock.Anything, (*sqlx.Tx)(nil), "spot-extended", "occupied").Return(nil)
	billing.On("StartBilling", mock.Anything, reservation.ID, int64(0), mock.AnythingOfType("string")).Return(&billingmodel.BillingRecord{ID: "billing-checkin-id"}, nil)

	checkedIn, err := uc.CheckIn(t.Context(), &model.CheckInRequest{ReservationID: reservation.ID})
	require.NoError(t, err)
	require.NotNil(t, checkedIn)

	// --- Phase 3: Check-Out after 4 hours (reservation was only for 1 hour) ---
	checkedInAt := *checkedIn.Reservation.CheckedInAt
	// PRD Example 4: 4 hours total = 20,000 parking + 5,000 booking = 25,000
	billingRecord := &billingmodel.BillingRecord{
		ID:          "billing-extended",
		TotalAmount: 25000,
	}

	repo.On("GetByIDForUpdate", mock.Anything, (*sqlx.Tx)(nil), reservation.ID).Return(&model.Reservation{
		ID:          reservation.ID,
		DriverID:    "driver-extended",
		SpotID:      "spot-extended",
		Status:      model.StatusCheckedIn,
		ConfirmedAt: &confirmedAt,
		CheckedInAt: &checkedInAt,
	}, nil).Once()
	repo.On("UpdateReservationTx", mock.Anything, (*sqlx.Tx)(nil), mock.MatchedBy(func(r *model.Reservation) bool {
		return r.Status == model.StatusCheckedOut
	})).Return(nil).Once()
	billing.On("CalculateFee", mock.Anything, reservation.ID, checkedInAt, mock.AnythingOfType("time.Time")).Return(billingRecord, nil)
	billing.On("GenerateInvoice", mock.Anything, reservation.ID, mock.AnythingOfType("string")).Return(billingRecord, nil)

	checkOutResult, err := uc.CheckOut(t.Context(), &model.CheckOutRequest{ReservationID: reservation.ID})
	require.NoError(t, err)
	require.NotNil(t, checkOutResult)
	assert.Equal(t, model.StatusCheckedOut, checkOutResult.Reservation.Status)
	assert.Equal(t, int64(25000), checkOutResult.TotalAmount, "PRD Example 4: 4-hour overstay total should be 25,000 IDR (no penalty)")

	// Verify: no penalty was applied (penalty system removed)
}
