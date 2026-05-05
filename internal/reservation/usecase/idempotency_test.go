// Package usecase implements the business logic layer for the reservation domain.
package usecase

import (
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	billingmodel "parkir-pintar/internal/billing/model"
	"parkir-pintar/internal/reservation/model"
)

// TestCreateReservation_ShouldCreateDifferentRecords_WhenDifferentIdempotencyKeys verifies
// that calling CreateReservation twice with different idempotency keys produces two
// different reservations with different IDs and different spot IDs.
func TestCreateReservation_ShouldCreateDifferentRecords_WhenDifferentIdempotencyKeys(t *testing.T) {
	// Arrange
	repo := new(MockRepository)
	redis := new(MockRedisClient)
	natsClient := new(MockNATSClient)
	billing := new(MockBillingClient)
	payment := new(MockPaymentClient)

	// First call mocks (key-alpha -> spot-alpha)
	repo.On("FindByIdempotencyKey", mock.Anything, "key-alpha").Return(nil, model.ErrNotFound).Once()
	repo.On("FindAvailableSpot", mock.Anything, "car").Return(&model.ParkingSpot{
		ID:          "spot-alpha",
		VehicleType: "car",
		Status:      "available",
	}, nil).Once()
	redis.On("SetNX", mock.Anything, "lock:spot:spot-alpha", "locked", 30*time.Second).Return(true, nil).Once()
	redis.On("Delete", mock.Anything, "lock:spot:spot-alpha").Return(nil).Once()
	repo.On("GetSpotForUpdate", mock.Anything, "spot-alpha").Return(&model.ParkingSpot{
		ID:     "spot-alpha",
		Status: "available",
	}, nil).Once()
	repo.On("CreateReservationTx", mock.Anything, (*sqlx.Tx)(nil), mock.AnythingOfType("*model.Reservation")).Return(nil).Once()
	repo.On("UpdateSpotStatusTx", mock.Anything, (*sqlx.Tx)(nil), "spot-alpha", "reserved").Return(nil).Once()
	billing.On("StartBilling", mock.Anything, mock.AnythingOfType("string"), billingmodel.BookingFee, mock.AnythingOfType("string")).Return(nil).Once()
	natsClient.On("Publish", "reservation.confirmed", mock.Anything).Return(nil).Once()

	// Second call mocks (key-beta -> spot-beta)
	repo.On("FindByIdempotencyKey", mock.Anything, "key-beta").Return(nil, model.ErrNotFound).Once()
	repo.On("FindAvailableSpot", mock.Anything, "car").Return(&model.ParkingSpot{
		ID:          "spot-beta",
		VehicleType: "car",
		Status:      "available",
	}, nil).Once()
	redis.On("SetNX", mock.Anything, "lock:spot:spot-beta", "locked", 30*time.Second).Return(true, nil).Once()
	redis.On("Delete", mock.Anything, "lock:spot:spot-beta").Return(nil).Once()
	repo.On("GetSpotForUpdate", mock.Anything, "spot-beta").Return(&model.ParkingSpot{
		ID:     "spot-beta",
		Status: "available",
	}, nil).Once()
	repo.On("CreateReservationTx", mock.Anything, (*sqlx.Tx)(nil), mock.AnythingOfType("*model.Reservation")).Return(nil).Once()
	repo.On("UpdateSpotStatusTx", mock.Anything, (*sqlx.Tx)(nil), "spot-beta", "reserved").Return(nil).Once()
	billing.On("StartBilling", mock.Anything, mock.AnythingOfType("string"), billingmodel.BookingFee, mock.AnythingOfType("string")).Return(nil).Once()
	natsClient.On("Publish", "reservation.confirmed", mock.Anything).Return(nil).Once()

	uc := NewUsecase(repo, redis, natsClient, billing, payment)

	req1 := &model.CreateReservationRequest{
		DriverID:       "driver-1",
		VehicleType:    "car",
		AssignmentMode: model.AssignmentSystemAssigned,
		IdempotencyKey: "key-alpha",
	}

	req2 := &model.CreateReservationRequest{
		DriverID:       "driver-1",
		VehicleType:    "car",
		AssignmentMode: model.AssignmentSystemAssigned,
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
	redis.AssertExpectations(t)
	natsClient.AssertExpectations(t)
	billing.AssertExpectations(t)
	payment.AssertExpectations(t)
}
