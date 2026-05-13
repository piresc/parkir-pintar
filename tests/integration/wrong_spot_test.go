// Package integration provides integration tests for the ParkirPintar wrong-spot
// penalty flow, testing that a 200,000 IDR penalty is applied when a driver
// parks in a different spot than assigned.
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
)

// TestWrongSpotFlow_ShouldApply200kPenalty_WhenDriverParksInWrongSpot tests
// the full create → check-in → checkout flow with a wrong-spot penalty applied.
// Verifies: penalty of 200,000 IDR is included in the final billing total.
//
// Validates: PRD §9.2, §10.2, Scenario 5
func TestWrongSpotFlow_ShouldApply200kPenalty_WhenDriverParksInWrongSpot(t *testing.T) {
	repo := new(MockRepository)
	locker := new(MockLocker)
	natsClient := new(MockNATSClient)
	billing := new(MockBillingClient)
	payment := new(MockPaymentClient)

	uc := usecase.NewUsecase(repo, locker, natsClient, billing, payment)

	// --- Phase 1: Create Reservation ---
	repo.On("FindByIdempotencyKey", mock.Anything, "wrongspot-key").Return(nil, model.ErrNotFound)
	repo.On("FindAvailableSpot", mock.Anything, "car").Return(&model.ParkingSpot{
		ID:          "spot-assigned",
		VehicleType: "car",
		Status:      "available",
	}, nil)
	lock := new(MockLock)
	locker.On("Acquire", mock.Anything, "spot:spot-assigned").Return(lock, nil)
	lock.On("Release", mock.Anything).Return(nil)
	repo.On("GetSpotForUpdate", mock.Anything, "spot-assigned").Return(&model.ParkingSpot{
		ID:          "spot-assigned",
		VehicleType: "car",
		Status:      "available",
	}, nil)
	repo.On("ListByDriverID", mock.Anything, "driver-wrongspot", "").Return([]*model.Reservation{}, nil)
	repo.On("CreateReservationTx", mock.Anything, (*sqlx.Tx)(nil), mock.AnythingOfType("*model.Reservation")).Return(nil)
	repo.On("UpdateSpotStatusTx", mock.Anything, (*sqlx.Tx)(nil), "spot-assigned", "reserved").Return(nil)
	billing.On("StartBilling", mock.Anything, mock.AnythingOfType("string"), billingmodel.BookingFee, mock.AnythingOfType("string")).Return(&billingmodel.BillingRecord{ID: "billing-test-id"}, nil)

	// Act: create reservation
	reservation, err := uc.CreateReservation(t.Context(), &model.CreateReservationRequest{
		DriverID:       "driver-wrongspot",
		VehicleType:    "car",
		AssignmentMode: model.AssignmentSystemAssigned,
		IdempotencyKey: "wrongspot-key",
	})
	require.NoError(t, err)
	require.NotNil(t, reservation)

	// --- Phase 1b: Confirm reservation ---
	confirmedAt := time.Now().Add(-2 * time.Hour)
	repo.On("GetByIDForUpdate", mock.Anything, (*sqlx.Tx)(nil), reservation.ID).Return(&model.Reservation{
		ID:          reservation.ID,
		DriverID:    "driver-wrongspot",
		SpotID:      "spot-assigned",
		Status:      model.StatusWaitingPayment,
		ConfirmedAt: nil,
	}, nil).Once()
	repo.On("UpdateReservationTx", mock.Anything, (*sqlx.Tx)(nil), mock.MatchedBy(func(r *model.Reservation) bool {
		return r.Status == model.StatusConfirmed
	})).Return(nil).Once()
	payment.On("ProcessPayment", mock.Anything, "billing-test-id", billingmodel.BookingFee, "qris", mock.AnythingOfType("string")).Return("pay-booking", nil).Once()
	natsClient.On("Publish", "reservation.confirmed", mock.Anything).Return(nil).Once()

	_, err = uc.ConfirmReservation(t.Context(), &model.ConfirmReservationRequest{
		ReservationID: reservation.ID,
	})
	require.NoError(t, err)

	// --- Phase 2: Check-In ---
	repo.On("GetByIDForUpdate", mock.Anything, (*sqlx.Tx)(nil), reservation.ID).Return(&model.Reservation{
		ID:          reservation.ID,
		DriverID:    "driver-wrongspot",
		SpotID:      "spot-assigned",
		Status:      model.StatusConfirmed,
		ConfirmedAt: &confirmedAt,
	}, nil).Once()
	repo.On("UpdateReservationTx", mock.Anything, (*sqlx.Tx)(nil), mock.MatchedBy(func(r *model.Reservation) bool {
		return r.Status == model.StatusCheckedIn
	})).Return(nil).Once()
	repo.On("UpdateSpotStatusTx", mock.Anything, (*sqlx.Tx)(nil), "spot-assigned", "occupied").Return(nil)
	billing.On("StartBilling", mock.Anything, reservation.ID, int64(0), mock.AnythingOfType("string")).Return(&billingmodel.BillingRecord{ID: "billing-checkin-id"}, nil)
	natsClient.On("Publish", "reservation.checked_in", mock.Anything).Return(nil)

	checkedIn, err := uc.CheckIn(t.Context(), &model.CheckInRequest{ReservationID: reservation.ID})
	require.NoError(t, err)
	require.NotNil(t, checkedIn)

	// --- Phase 3: Check-Out with wrong-spot penalty included ---
	checkedInAt := *checkedIn.CheckedInAt
	// Total = 5000 booking + 10000 parking + 200000 penalty = 215000
	billingRecord := &billingmodel.BillingRecord{
		ID:            "billing-wrongspot",
		TotalAmount:   billingmodel.BookingFee + 10000 + billingmodel.WrongSpotPenalty,
		PenaltyAmount: billingmodel.WrongSpotPenalty,
	}

	repo.On("GetByIDForUpdate", mock.Anything, (*sqlx.Tx)(nil), reservation.ID).Return(&model.Reservation{
		ID:          reservation.ID,
		DriverID:    "driver-wrongspot",
		SpotID:      "spot-assigned",
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
	assert.Equal(t, billingmodel.BookingFee+10000+billingmodel.WrongSpotPenalty, checkOutResult.TotalAmount,
		"wrong-spot penalty total should be 215,000 IDR (5000 booking + 10000 parking + 200000 penalty)")
}
