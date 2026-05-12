// Package integration provides integration tests for the ParkirPintar reservation
// cancellation flow, testing the full create → cancel path through the usecase layer.
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
// - Use AssertNotCalled() to verify methods weren't called
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

// TestCancellationFlow_ShouldNotChargeFee_WhenCancelledWithin2Min tests the full
// create → cancel flow where cancellation happens within the 2-minute free window.
// Verifies: no ApplyPenalty call, spot released to "available".
//
// Validates: Requirements 3.1, 9.1, 21.3, 25.4
func TestCancellationFlow_ShouldNotChargeFee_WhenCancelledWithin2Min(t *testing.T) {
	// Arrange — set up all mocks
	repo := new(MockRepository)
	locker := new(MockLocker)
	natsClient := new(MockNATSClient)
	billing := new(MockBillingClient)
	payment := new(MockPaymentClient)

	uc := usecase.NewUsecase(repo, locker, natsClient, billing, payment)

	// --- Phase 1: Create Reservation ---

	repo.On("FindByIdempotencyKey", mock.Anything, "cancel-free-key").Return(nil, model.ErrNotFound)
	repo.On("FindAvailableSpot", mock.Anything, "motorcycle").Return(&model.ParkingSpot{
		ID:          "spot-cancel-1",
		VehicleType: "motorcycle",
		Status:      "available",
	}, nil)
	lock := new(MockLock)
	locker.On("Acquire", mock.Anything, "spot:spot-cancel-1").Return(lock, nil)
	lock.On("Release", mock.Anything).Return(nil)
	repo.On("GetSpotForUpdate", mock.Anything, "spot-cancel-1").Return(&model.ParkingSpot{
		ID:     "spot-cancel-1",
		Status: "available",
	}, nil)
	repo.On("CreateReservationTx", mock.Anything, (*sqlx.Tx)(nil), mock.AnythingOfType("*model.Reservation")).Return(nil)
	repo.On("UpdateSpotStatusTx", mock.Anything, (*sqlx.Tx)(nil), "spot-cancel-1", "reserved").Return(nil)
	billing.On("StartBilling", mock.Anything, mock.AnythingOfType("string"), billingmodel.BookingFee, mock.AnythingOfType("string")).Return(&billingmodel.BillingRecord{ID: "billing-test-id"}, nil)

	// Act: create reservation
	reservation, err := uc.CreateReservation(t.Context(), &model.CreateReservationRequest{
		DriverID:       "driver-cancel-1",
		VehicleType:    "motorcycle",
		AssignmentMode: model.AssignmentSystemAssigned,
		IdempotencyKey: "cancel-free-key",
	})
	require.NoError(t, err)
	require.NotNil(t, reservation)

	// Confirm the reservation so cancellation fee logic applies
	confirmedAt := time.Now().Add(-1 * time.Minute)
	repo.On("GetByIDForUpdate", mock.Anything, (*sqlx.Tx)(nil), reservation.ID).Return(&model.Reservation{
		ID:          reservation.ID,
		DriverID:    "driver-cancel-1",
		SpotID:      "spot-cancel-1",
		Status:      model.StatusWaitingPayment,
		ConfirmedAt: nil,
	}, nil).Once()
	repo.On("GetByIDForUpdate", mock.Anything, (*sqlx.Tx)(nil), reservation.ID).Return(&model.Reservation{
		ID:          reservation.ID,
		DriverID:    "driver-cancel-1",
		SpotID:      "spot-cancel-1",
		Status:      model.StatusConfirmed,
		ConfirmedAt: &confirmedAt,
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

	// --- Phase 2: Cancel within 2 minutes ---

	// Arrange: return reservation with confirmedAt = 1 minute ago (within free window)
	repo.On("GetByIDForUpdate", mock.Anything, (*sqlx.Tx)(nil), reservation.ID).Return(&model.Reservation{
		ID:          reservation.ID,
		DriverID:    "driver-cancel-1",
		SpotID:      "spot-cancel-1",
		Status:      model.StatusConfirmed,
		ConfirmedAt: &confirmedAt,
	}, nil)
	repo.On("UpdateReservationTx", mock.Anything, (*sqlx.Tx)(nil), mock.MatchedBy(func(r *model.Reservation) bool {
		return r.Status == model.StatusCancelled
	})).Return(nil)
	repo.On("UpdateSpotStatusTx", mock.Anything, (*sqlx.Tx)(nil), "spot-cancel-1", "available").Return(nil)
	natsClient.On("Publish", "reservation.cancelled", mock.Anything).Return(nil)

	// Act: cancel reservation
	cancelled, err := uc.CancelReservation(t.Context(), &model.CancelReservationRequest{
		ReservationID: reservation.ID,
	})

	// Assert: cancellation succeeded with no fee
	require.NoError(t, err)
	require.NotNil(t, cancelled)
	assert.Equal(t, model.StatusCancelled, cancelled.Status)
	assert.NotNil(t, cancelled.CancelledAt)

	// Verify: ApplyPenalty was NOT called (free cancellation)
	billing.AssertNotCalled(t, "ApplyPenalty")

	// Verify: spot released to "available"
	repo.AssertCalled(t, "UpdateSpotStatusTx", mock.Anything, (*sqlx.Tx)(nil), "spot-cancel-1", "available")

	// Verify all mock expectations
	repo.AssertExpectations(t)
	locker.AssertExpectations(t)
	lock.AssertExpectations(t)
	natsClient.AssertExpectations(t)
	billing.AssertExpectations(t)
	payment.AssertExpectations(t)
}

// TestCancellationFlow_ShouldCharge5000IDR_WhenCancelledAfter2Min tests the full
// create → cancel flow where cancellation happens after the 2-minute free window.
// Verifies: ApplyPenalty called with "cancellation" and 5000, spot released to "available".
//
// Validates: Requirements 3.2, 9.2, 21.3, 25.4
func TestCancellationFlow_ShouldCharge5000IDR_WhenCancelledAfter2Min(t *testing.T) {
	// Arrange — set up all mocks
	repo := new(MockRepository)
	locker := new(MockLocker)
	natsClient := new(MockNATSClient)
	billing := new(MockBillingClient)
	payment := new(MockPaymentClient)

	uc := usecase.NewUsecase(repo, locker, natsClient, billing, payment)

	// --- Phase 1: Create Reservation ---

	repo.On("FindByIdempotencyKey", mock.Anything, "cancel-paid-key").Return(nil, model.ErrNotFound)
	repo.On("FindAvailableSpot", mock.Anything, "car").Return(&model.ParkingSpot{
		ID:          "spot-cancel-2",
		VehicleType: "car",
		Status:      "available",
	}, nil)
	lock := new(MockLock)
	locker.On("Acquire", mock.Anything, "spot:spot-cancel-2").Return(lock, nil)
	lock.On("Release", mock.Anything).Return(nil)
	repo.On("GetSpotForUpdate", mock.Anything, "spot-cancel-2").Return(&model.ParkingSpot{
		ID:     "spot-cancel-2",
		Status: "available",
	}, nil)
	repo.On("CreateReservationTx", mock.Anything, (*sqlx.Tx)(nil), mock.AnythingOfType("*model.Reservation")).Return(nil)
	repo.On("UpdateSpotStatusTx", mock.Anything, (*sqlx.Tx)(nil), "spot-cancel-2", "reserved").Return(nil)
	billing.On("StartBilling", mock.Anything, mock.AnythingOfType("string"), billingmodel.BookingFee, mock.AnythingOfType("string")).Return(&billingmodel.BillingRecord{ID: "billing-test-id"}, nil)

	// Act: create reservation
	reservation, err := uc.CreateReservation(t.Context(), &model.CreateReservationRequest{
		DriverID:       "driver-cancel-2",
		VehicleType:    "car",
		AssignmentMode: model.AssignmentSystemAssigned,
		IdempotencyKey: "cancel-paid-key",
	})
	require.NoError(t, err)
	require.NotNil(t, reservation)

	// --- Phase 1b: Confirm reservation ---
	confirmedAt := time.Now().Add(-5 * time.Minute)
	repo.On("GetByIDForUpdate", mock.Anything, (*sqlx.Tx)(nil), reservation.ID).Return(&model.Reservation{
		ID:          reservation.ID,
		DriverID:    "driver-cancel-2",
		SpotID:      "spot-cancel-2",
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

	// --- Phase 2: Cancel after 2 minutes ---

	// Arrange: return reservation with confirmedAt = 5 minutes ago (past free window)
	repo.On("GetByIDForUpdate", mock.Anything, (*sqlx.Tx)(nil), reservation.ID).Return(&model.Reservation{
		ID:          reservation.ID,
		DriverID:    "driver-cancel-2",
		SpotID:      "spot-cancel-2",
		Status:      model.StatusConfirmed,
		ConfirmedAt: &confirmedAt,
	}, nil)
	repo.On("UpdateReservationTx", mock.Anything, (*sqlx.Tx)(nil), mock.MatchedBy(func(r *model.Reservation) bool {
		return r.Status == model.StatusCancelled
	})).Return(nil)
	repo.On("UpdateSpotStatusTx", mock.Anything, (*sqlx.Tx)(nil), "spot-cancel-2", "available").Return(nil)
	billing.On("ApplyPenalty", mock.Anything, reservation.ID, "cancellation", billingmodel.CancelFee, "cancellation fee").Return(nil)
	natsClient.On("Publish", "reservation.cancelled", mock.Anything).Return(nil)

	// Act: cancel reservation
	cancelled, err := uc.CancelReservation(t.Context(), &model.CancelReservationRequest{
		ReservationID: reservation.ID,
	})

	// Assert: cancellation succeeded with fee applied
	require.NoError(t, err)
	require.NotNil(t, cancelled)
	assert.Equal(t, model.StatusCancelled, cancelled.Status)
	assert.NotNil(t, cancelled.CancelledAt)

	// Verify: ApplyPenalty WAS called with "cancellation" and 5000 IDR
	billing.AssertCalled(t, "ApplyPenalty", mock.Anything, reservation.ID, "cancellation", billingmodel.CancelFee, "cancellation fee")

	// Verify: spot released to "available"
	repo.AssertCalled(t, "UpdateSpotStatusTx", mock.Anything, (*sqlx.Tx)(nil), "spot-cancel-2", "available")

	// Verify all mock expectations
	repo.AssertExpectations(t)
	locker.AssertExpectations(t)
	lock.AssertExpectations(t)
	natsClient.AssertExpectations(t)
	billing.AssertExpectations(t)
	payment.AssertExpectations(t)
}
