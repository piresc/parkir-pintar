// Package integration provides integration tests for the ParkirPintar reservation
// expiry flow, testing the create → expire path through the usecase layer.
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
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	billingmodel "parkir-pintar/internal/billing/model"
	"parkir-pintar/internal/reservation/constants"
	"parkir-pintar/internal/reservation/model"
	"parkir-pintar/internal/reservation/usecase"
)

// TestExpiryFlow_ShouldReleaseSpot_WhenReservationExpires tests the full
// create → expire flow. Per PRD, the booking fee (5,000 IDR, already charged
// at confirmation) is the only cost — no additional no-show penalty is applied.
// Verifies: spot released to "available", no ApplyPenalty call.
//
// Validates: Requirements 6.2, 21.3, 25.4
func TestExpiryFlow_ShouldReleaseSpot_WhenReservationExpires(t *testing.T) {
	// Arrange — set up all mocks
	repo := new(MockRepository)
	locker := new(MockLocker)
	billing := new(MockBillingClient)
	payment := new(MockPaymentClient)

	uc := usecase.NewUsecase(repo, locker, billing, payment, nil, nil, nil, 60, 10)

	// --- Phase 1: Create Reservation ---

	repo.On("FindByIdempotencyKey", mock.Anything, "expire-key-1").Return(nil, model.ErrNotFound)
	repo.On("FindAvailableSpot", mock.Anything, "car").Return(&model.ParkingSpot{
		ID:          "spot-expire-1",
		VehicleType: "car",
		Status:      "available",
	}, nil)
	lock := new(MockLock)
	locker.On("Acquire", mock.Anything, "spot:spot-expire-1").Return(lock, nil)
	lock.On("Release", mock.Anything).Return(nil)
	repo.On("GetSpotForUpdateTx", mock.Anything, (*sqlx.Tx)(nil), "spot-expire-1").Return(&model.ParkingSpot{
		ID:     "spot-expire-1",
		Status: "available",
	}, nil)
	repo.On("CreateReservationTx", mock.Anything, (*sqlx.Tx)(nil), mock.AnythingOfType("*model.Reservation")).Return(nil)
	repo.On("UpdateSpotStatusTx", mock.Anything, (*sqlx.Tx)(nil), "spot-expire-1", "reserved").Return(nil)
	billing.On("StartBilling", mock.Anything, mock.AnythingOfType("string"), constants.BookingFee, mock.AnythingOfType("string")).Return(&billingmodel.BillingRecord{ID: "billing-test-id"}, nil)

	// Act: create reservation
	reservation, err := uc.CreateReservation(t.Context(), &model.CreateReservationRequest{
		DriverID:       "driver-expire-1",
		VehicleType:    "car",
		AssignmentMode: constants.AssignmentSystemAssigned,
		IdempotencyKey: "expire-key-1",
	})
	require.NoError(t, err)
	require.NotNil(t, reservation)

	// --- Phase 1b: Confirm reservation ---
	confirmedAt := time.Now().Add(-2 * time.Hour)
	repo.On("GetByIDForUpdate", mock.Anything, (*sqlx.Tx)(nil), reservation.ID).Return(&model.Reservation{
		ID:          reservation.ID,
		DriverID:    "driver-expire-1",
		SpotID:      "spot-expire-1",
		Status:      constants.StatusWaitingPayment,
		ConfirmedAt: nil,
	}, nil).Once()
	// Second GetByIDForUpdate: re-check inside confirmation transaction (TOCTOU fix)
	repo.On("GetByIDForUpdate", mock.Anything, (*sqlx.Tx)(nil), reservation.ID).Return(&model.Reservation{
		ID:          reservation.ID,
		DriverID:    "driver-expire-1",
		SpotID:      "spot-expire-1",
		Status:      constants.StatusWaitingPayment,
		ConfirmedAt: nil,
	}, nil).Once()
	repo.On("UpdateReservationTx", mock.Anything, (*sqlx.Tx)(nil), mock.MatchedBy(func(r *model.Reservation) bool {
		return r.Status == constants.StatusConfirmed
	})).Return(nil).Once()
	payment.On("ProcessPayment", mock.Anything, "billing-test-id", constants.BookingFee, "qris", mock.AnythingOfType("string")).Return("pay-booking", nil).Once()

	_, err = uc.ConfirmReservation(t.Context(), &model.ConfirmReservationRequest{
		ReservationID: reservation.ID,
	})
	require.NoError(t, err)

	// --- Phase 2: Expire Reservation (simulating 1h+ elapsed) ---

	// Arrange: return reservation as confirmed but past expiry
	expiresAt := time.Now().Add(-1 * time.Hour)
	repo.On("GetByIDForUpdate", mock.Anything, (*sqlx.Tx)(nil), reservation.ID).Return(&model.Reservation{
		ID:          reservation.ID,
		DriverID:    "driver-expire-1",
		SpotID:      "spot-expire-1",
		Status:      constants.StatusConfirmed,
		ConfirmedAt: &confirmedAt,
		ExpiresAt:   &expiresAt,
	}, nil)
	repo.On("UpdateReservationTx", mock.Anything, (*sqlx.Tx)(nil), mock.MatchedBy(func(r *model.Reservation) bool {
		return r.Status == constants.StatusExpired
	})).Return(nil)
	repo.On("UpdateSpotStatusTx", mock.Anything, (*sqlx.Tx)(nil), "spot-expire-1", "available").Return(nil)

	// Act: expire reservation
	err = uc.ExpireReservation(t.Context(), &model.ExpireReservationRequest{
		ReservationID: reservation.ID,
	})

	// Assert: expiry succeeded
	require.NoError(t, err)

	// Verify: spot released to "available"
	repo.AssertCalled(t, "UpdateSpotStatusTx", mock.Anything, (*sqlx.Tx)(nil), "spot-expire-1", "available")

	// Verify all mock expectations
	repo.AssertExpectations(t)
	locker.AssertExpectations(t)
	lock.AssertExpectations(t)
	billing.AssertExpectations(t)
	payment.AssertExpectations(t)
}
