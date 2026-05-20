// Package usecase implements the business logic layer for the reservation domain.
package usecase

import (
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	billingmodel "parkir-pintar/internal/billing/model"
	"parkir-pintar/internal/reservation/constants"
	reservationerrors "parkir-pintar/internal/reservation/errors"
	"parkir-pintar/internal/reservation/model"
)

// TestCreateReservation_ShouldCreateDifferentRecords_WhenDifferentIdempotencyKeys verifies
// that calling CreateReservation twice with different idempotency keys produces two
// different reservations with different IDs and different spot IDs.
func TestCreateReservation_ShouldCreateDifferentRecords_WhenDifferentIdempotencyKeys(t *testing.T) {
	// Arrange
	repo := new(MockRepository)
	locker := new(MockLocker)
	billing := new(MockBillingClient)
	payment := new(MockPaymentClient)

	// First call mocks (key-alpha -> spot-alpha)
	repo.On("FindByIdempotencyKey", mock.Anything, "key-alpha").Return(nil, reservationerrors.ErrNotFound).Once()
	repo.On("FindAvailableSpot", mock.Anything, "car").Return(&model.ParkingSpot{
		ID:          "spot-alpha",
		VehicleType: "car",
		Status:      "available",
	}, nil).Once()
	lck1 := new(MockLock)
	locker.On("Acquire", mock.Anything, "spot:spot-alpha").Return(lck1, nil).Once()
	lck1.On("Release", mock.Anything).Return(nil).Once()
	repo.On("GetSpotForUpdateTx", mock.Anything, (*sqlx.Tx)(nil), "spot-alpha").Return(&model.ParkingSpot{
		ID:     "spot-alpha",
		Status: "available",
	}, nil).Once()
	repo.On("CreateReservationTx", mock.Anything, (*sqlx.Tx)(nil), mock.AnythingOfType("*model.Reservation")).Return(nil).Once()
	repo.On("UpdateSpotStatusTx", mock.Anything, (*sqlx.Tx)(nil), "spot-alpha", "reserved").Return(nil).Once()
	billing.On("StartBilling", mock.Anything, mock.AnythingOfType("string"), constants.BookingFee, mock.AnythingOfType("string")).Return(&billingmodel.BillingRecord{ID: "billing-alpha-id"}, nil).Once()

	// Second call mocks (key-beta -> spot-beta)
	repo.On("FindByIdempotencyKey", mock.Anything, "key-beta").Return(nil, reservationerrors.ErrNotFound).Once()
	repo.On("FindAvailableSpot", mock.Anything, "car").Return(&model.ParkingSpot{
		ID:          "spot-beta",
		VehicleType: "car",
		Status:      "available",
	}, nil).Once()
	lck2 := new(MockLock)
	locker.On("Acquire", mock.Anything, "spot:spot-beta").Return(lck2, nil).Once()
	lck2.On("Release", mock.Anything).Return(nil).Once()
	repo.On("GetSpotForUpdateTx", mock.Anything, (*sqlx.Tx)(nil), "spot-beta").Return(&model.ParkingSpot{
		ID:     "spot-beta",
		Status: "available",
	}, nil).Once()
	repo.On("CreateReservationTx", mock.Anything, (*sqlx.Tx)(nil), mock.AnythingOfType("*model.Reservation")).Return(nil).Once()
	repo.On("UpdateSpotStatusTx", mock.Anything, (*sqlx.Tx)(nil), "spot-beta", "reserved").Return(nil).Once()
	billing.On("StartBilling", mock.Anything, mock.AnythingOfType("string"), constants.BookingFee, mock.AnythingOfType("string")).Return(&billingmodel.BillingRecord{ID: "billing-beta-id"}, nil).Once()

	uc := NewUsecase(repo, locker, billing, payment, nil, nil, nil, 60, 10)

	req1 := &model.CreateReservationRequest{
		DriverID:       "driver-1",
		VehicleType:    "car",
		AssignmentMode: string(constants.AssignmentSystemAssigned),
		IdempotencyKey: "key-alpha",
	}

	req2 := &model.CreateReservationRequest{
		DriverID:       "driver-1",
		VehicleType:    "car",
		AssignmentMode: string(constants.AssignmentSystemAssigned),
		IdempotencyKey: "key-beta",
	}

	// Act
	res1, err1 := uc.CreateReservation(t.Context(), req1)
	res2, err2 := uc.CreateReservation(t.Context(), req2)

	// Assert
	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.NotEqual(t, res1.ID, res2.ID)
	assert.NotEqual(t, res1.SpotID, res2.SpotID)
	repo.AssertNumberOfCalls(t, "CreateReservationTx", 2)

	repo.AssertExpectations(t)
	locker.AssertExpectations(t)
	billing.AssertExpectations(t)
	payment.AssertExpectations(t)
}
