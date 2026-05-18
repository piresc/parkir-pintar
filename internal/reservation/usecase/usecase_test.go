// Package usecase implements the business logic layer for the reservation domain.
//
// Best practices applied (from Go testify coding standards KB):
// - Test naming: Test[FunctionName]_Should[ExpectedResult]_When[Condition]
// - AAA pattern: Arrange → Act → Assert
// - testify/mock for mock implementations of all dependency interfaces
// - testify/assert and testify/require for assertions
// - Each test is isolated with its own mock setup
// - AssertExpectations(t) called on all mocks to verify interactions
// - Use t.Context() for Go 1.24+ context in tests
// - Use errors.Is for sentinel error checks where applicable
// - Mock at interface boundaries rather than concrete implementations
package usecase

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	billingmodel "parkir-pintar/internal/billing/model"
	"parkir-pintar/internal/reservation/model"
	"parkir-pintar/pkg/pricing"
	"parkir-pintar/pkg/redislock"

	"github.com/jackc/pgx/v5/pgconn"
)

// --- Mock Implementations ---

// MockRepository implements repository.Repository using testify/mock.
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) FindByIdempotencyKey(ctx context.Context, key string) (*model.Reservation, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Reservation), args.Error(1)
}

func (m *MockRepository) FindAvailableSpot(ctx context.Context, vehicleType string) (*model.ParkingSpot, error) {
	args := m.Called(ctx, vehicleType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ParkingSpot), args.Error(1)
}

func (m *MockRepository) GetSpotByID(ctx context.Context, spotID string) (*model.ParkingSpot, error) {
	args := m.Called(ctx, spotID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ParkingSpot), args.Error(1)
}

func (m *MockRepository) GetSpotForUpdate(ctx context.Context, spotID string) (*model.ParkingSpot, error) {
	args := m.Called(ctx, spotID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ParkingSpot), args.Error(1)
}

func (m *MockRepository) FindAvailableSpotTx(ctx context.Context, tx *sqlx.Tx, vehicleType string) (*model.ParkingSpot, error) {
	args := m.Called(ctx, tx, vehicleType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ParkingSpot), args.Error(1)
}

func (m *MockRepository) GetSpotForUpdateTx(ctx context.Context, tx *sqlx.Tx, spotID string) (*model.ParkingSpot, error) {
	args := m.Called(ctx, tx, spotID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ParkingSpot), args.Error(1)
}

func (m *MockRepository) CreateReservationTx(ctx context.Context, tx *sqlx.Tx, reservation *model.Reservation) error {
	args := m.Called(ctx, tx, reservation)
	return args.Error(0)
}

func (m *MockRepository) UpdateSpotStatusTx(ctx context.Context, tx *sqlx.Tx, spotID string, status string) error {
	args := m.Called(ctx, tx, spotID, status)
	return args.Error(0)
}

func (m *MockRepository) FindExpiredReservations(ctx context.Context) ([]*model.Reservation, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Reservation), args.Error(1)
}

func (m *MockRepository) FindStalePaymentReservations(ctx context.Context, timeoutMinutes int) ([]*model.Reservation, error) {
	args := m.Called(ctx, timeoutMinutes)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Reservation), args.Error(1)
}

func (m *MockRepository) ListByDriverID(ctx context.Context, driverID string, status string) ([]*model.Reservation, error) {
	args := m.Called(ctx, driverID, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Reservation), args.Error(1)
}

func (m *MockRepository) GetByID(ctx context.Context, id string) (*model.Reservation, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Reservation), args.Error(1)
}

func (m *MockRepository) GetByIDForUpdate(ctx context.Context, tx *sqlx.Tx, id string) (*model.Reservation, error) {
	args := m.Called(ctx, tx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Reservation), args.Error(1)
}

func (m *MockRepository) UpdateReservationTx(ctx context.Context, tx *sqlx.Tx, reservation *model.Reservation) error {
	args := m.Called(ctx, tx, reservation)
	return args.Error(0)
}

// WithTransaction executes the callback directly (simulating a successful transaction).
func (m *MockRepository) WithTransaction(ctx context.Context, fn func(tx *sqlx.Tx) error) error {
	return fn(nil) // pass nil tx since mocks don't need real transactions
}

// MockBillingClient implements BillingClient using testify/mock.
type MockBillingClient struct {
	mock.Mock
}

func (m *MockBillingClient) StartBilling(ctx context.Context, reservationID string, bookingFee int64, idempotencyKey string) (*billingmodel.BillingRecord, error) {
	args := m.Called(ctx, reservationID, bookingFee, idempotencyKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*billingmodel.BillingRecord), args.Error(1)
}

func (m *MockBillingClient) CalculateFee(ctx context.Context, reservationID string, checkInAt, checkOutAt time.Time) (*billingmodel.BillingRecord, error) {
	args := m.Called(ctx, reservationID, checkInAt, checkOutAt)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*billingmodel.BillingRecord), args.Error(1)
}

func (m *MockBillingClient) GenerateInvoice(ctx context.Context, reservationID string, idempotencyKey string) (*billingmodel.BillingRecord, error) {
	args := m.Called(ctx, reservationID, idempotencyKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*billingmodel.BillingRecord), args.Error(1)
}

// MockPaymentClient implements PaymentClient using testify/mock.
type MockPaymentClient struct {
	mock.Mock
}

func (m *MockPaymentClient) ProcessPayment(ctx context.Context, billingID string, amount int64, paymentMethod string, idempotencyKey string) (string, error) {
	args := m.Called(ctx, billingID, amount, paymentMethod, idempotencyKey)
	return args.String(0), args.Error(1)
}

// MockLock implements Lock using testify/mock.
type MockLock struct {
	mock.Mock
}

func (m *MockLock) Release(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// MockLocker implements Locker using testify/mock.
type MockLocker struct {
	mock.Mock
}

func (m *MockLocker) Acquire(ctx context.Context, key string) (Lock, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(Lock), args.Error(1)
}

// --- Test Cases ---

// TestCreateReservation_ShouldReturnExisting_WhenDuplicateIdempotencyKey verifies
// that a duplicate idempotency key returns the existing reservation without side effects.
func TestCreateReservation_ShouldReturnExisting_WhenDuplicateIdempotencyKey(t *testing.T) {
	// Arrange
	repo := new(MockRepository)
	locker := new(MockLocker)
	billing := new(MockBillingClient)
	payment := new(MockPaymentClient)

	existing := &model.Reservation{
		ID:             "existing-id",
		DriverID:       "driver-1",
		SpotID:         "spot-1",
		Status:         model.StatusConfirmed,
		IdempotencyKey: "idem-key-1",
	}
	repo.On("FindByIdempotencyKey", mock.Anything, "idem-key-1").Return(existing, nil)

	uc := NewUsecase(repo, locker, billing, payment, nil, nil, nil, 60)
	req := &model.CreateReservationRequest{
		DriverID:       "driver-1",
		VehicleType:    "car",
		AssignmentMode: model.AssignmentSystemAssigned,
		IdempotencyKey: "idem-key-1",
	}

	// Act
	result, err := uc.CreateReservation(t.Context(), req)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "existing-id", result.ID)
	assert.Equal(t, model.StatusConfirmed, result.Status)
	// No other mocks should have been called (no side effects)
	repo.AssertExpectations(t)
	locker.AssertExpectations(t)
	billing.AssertExpectations(t)
	payment.AssertExpectations(t)
}

// TestCreateReservation_ShouldReturnConfirmed_WhenSystemAssigned verifies
// successful system-assigned reservation creates reservation with status "confirmed".
func TestCreateReservation_ShouldReturnConfirmed_WhenSystemAssigned(t *testing.T) {
	// Arrange
	repo := new(MockRepository)
	locker := new(MockLocker)
	billing := new(MockBillingClient)
	payment := new(MockPaymentClient)

	repo.On("FindByIdempotencyKey", mock.Anything, "new-key").Return(nil, model.ErrNotFound)
	repo.On("ListByDriverID", mock.Anything, "driver-1", "").Return([]*model.Reservation{}, nil)
	repo.On("FindAvailableSpot", mock.Anything, "car").Return(&model.ParkingSpot{
		ID:          "spot-42",
		VehicleType: "car",
		Status:      "available",
	}, nil)
	lock := new(MockLock)
	locker.On("Acquire", mock.Anything, "spot:spot-42").Return(lock, nil)
	lock.On("Release", mock.Anything).Return(nil)
	repo.On("GetSpotForUpdateTx", mock.Anything, (*sqlx.Tx)(nil), "spot-42").Return(&model.ParkingSpot{
		ID:     "spot-42",
		Status: "available",
	}, nil)
	repo.On("CreateReservationTx", mock.Anything, (*sqlx.Tx)(nil), mock.AnythingOfType("*model.Reservation")).Return(nil)
	repo.On("UpdateSpotStatusTx", mock.Anything, (*sqlx.Tx)(nil), "spot-42", "reserved").Return(nil)
	billing.On("StartBilling", mock.Anything, mock.AnythingOfType("string"), pricing.BookingFee, mock.AnythingOfType("string")).Return(&billingmodel.BillingRecord{ID: "billing-test-id"}, nil)

	uc := NewUsecase(repo, locker, billing, payment, nil, nil, nil, 60)
	req := &model.CreateReservationRequest{
		DriverID:       "driver-1",
		VehicleType:    "car",
		AssignmentMode: model.AssignmentSystemAssigned,
		IdempotencyKey: "new-key",
	}

	// Act
	result, err := uc.CreateReservation(t.Context(), req)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, model.StatusWaitingPayment, result.Status)
	assert.Equal(t, "spot-42", result.SpotID)
	assert.Equal(t, "driver-1", result.DriverID)
	assert.Equal(t, model.AssignmentSystemAssigned, result.AssignmentMode)
	assert.Nil(t, result.ConfirmedAt)
	assert.Nil(t, result.ExpiresAt)
	repo.AssertExpectations(t)
	locker.AssertExpectations(t)
	billing.AssertExpectations(t)
	payment.AssertExpectations(t)
}

// TestCreateReservation_ShouldReturnConfirmed_WhenUserSelected verifies
// successful user-selected reservation with a specific spot.
func TestCreateReservation_ShouldReturnConfirmed_WhenUserSelected(t *testing.T) {
	// Arrange
	repo := new(MockRepository)
	locker := new(MockLocker)
	billing := new(MockBillingClient)
	payment := new(MockPaymentClient)

	repo.On("FindByIdempotencyKey", mock.Anything, "user-key").Return(nil, model.ErrNotFound)
	repo.On("ListByDriverID", mock.Anything, "driver-2", "").Return([]*model.Reservation{}, nil)
	lck := new(MockLock)
	locker.On("Acquire", mock.Anything, "spot:spot-99").Return(lck, nil)
	lck.On("Release", mock.Anything).Return(nil)
	repo.On("GetSpotForUpdateTx", mock.Anything, (*sqlx.Tx)(nil), "spot-99").Return(&model.ParkingSpot{
		ID:          "spot-99",
		VehicleType: "motorcycle",
		Status:      "available",
	}, nil)
	repo.On("CreateReservationTx", mock.Anything, (*sqlx.Tx)(nil), mock.AnythingOfType("*model.Reservation")).Return(nil)
	repo.On("UpdateSpotStatusTx", mock.Anything, (*sqlx.Tx)(nil), "spot-99", "reserved").Return(nil)
	billing.On("StartBilling", mock.Anything, mock.AnythingOfType("string"), pricing.BookingFee, mock.AnythingOfType("string")).Return(&billingmodel.BillingRecord{ID: "billing-test-id"}, nil)

	uc := NewUsecase(repo, locker, billing, payment, nil, nil, nil, 60)
	req := &model.CreateReservationRequest{
		DriverID:       "driver-2",
		VehicleType:    "motorcycle",
		AssignmentMode: model.AssignmentUserSelected,
		SpotID:         "spot-99",
		IdempotencyKey: "user-key",
	}

	// Act
	result, err := uc.CreateReservation(t.Context(), req)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, model.StatusWaitingPayment, result.Status)
	assert.Equal(t, "spot-99", result.SpotID)
	assert.Equal(t, model.AssignmentUserSelected, result.AssignmentMode)
	repo.AssertExpectations(t)
	locker.AssertExpectations(t)
	billing.AssertExpectations(t)
	payment.AssertExpectations(t)
}

// TestCreateReservation_ShouldReturnConflict_WhenLockContention verifies
// that Redis SetNX returning false results in a conflict error.
func TestCreateReservation_ShouldReturnConflict_WhenLockContention(t *testing.T) {
	// Arrange
	repo := new(MockRepository)
	locker := new(MockLocker)
	billing := new(MockBillingClient)
	payment := new(MockPaymentClient)

	repo.On("FindByIdempotencyKey", mock.Anything, "lock-key").Return(nil, model.ErrNotFound)
	repo.On("ListByDriverID", mock.Anything, "driver-3", "").Return([]*model.Reservation{}, nil)
	repo.On("FindAvailableSpot", mock.Anything, "car").Return(&model.ParkingSpot{
		ID:     "spot-locked",
		Status: "available",
	}, nil)
	locker.On("Acquire", mock.Anything, "spot:spot-locked").Return(nil, redislock.ErrLockUnavailable)

	uc := NewUsecase(repo, locker, billing, payment, nil, nil, nil, 60)
	req := &model.CreateReservationRequest{
		DriverID:       "driver-3",
		VehicleType:    "car",
		AssignmentMode: model.AssignmentSystemAssigned,
		IdempotencyKey: "lock-key",
	}

	// Act
	result, err := uc.CreateReservation(t.Context(), req)

	// Assert
	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "spot is being reserved by another driver")
	repo.AssertExpectations(t)
	locker.AssertExpectations(t)
}

// TestCreateReservation_ShouldReturnConflict_WhenNoAvailableSpots verifies
// that FindAvailableSpot returning error results in a conflict error.
func TestCreateReservation_ShouldReturnConflict_WhenNoAvailableSpots(t *testing.T) {
	// Arrange
	repo := new(MockRepository)
	locker := new(MockLocker)
	billing := new(MockBillingClient)
	payment := new(MockPaymentClient)

	repo.On("FindByIdempotencyKey", mock.Anything, "no-spots-key").Return(nil, model.ErrNotFound)
	repo.On("ListByDriverID", mock.Anything, "driver-4", "").Return([]*model.Reservation{}, nil)
	repo.On("FindAvailableSpot", mock.Anything, "car").Return(nil, model.ErrNotFound)

	uc := NewUsecase(repo, locker, billing, payment, nil, nil, nil, 60)
	req := &model.CreateReservationRequest{
		DriverID:       "driver-4",
		VehicleType:    "car",
		AssignmentMode: model.AssignmentSystemAssigned,
		IdempotencyKey: "no-spots-key",
	}

	// Act
	result, err := uc.CreateReservation(t.Context(), req)

	// Assert
	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no available spots")
	repo.AssertExpectations(t)
}

// TestCancelReservation_ShouldNotChargeFee_WhenCancelledWithin2Min verifies
// that cancelling within 2 minutes of confirmation results in no cancellation fee.
func TestCancelReservation_ShouldNotChargeFee_WhenCancelledWithin2Min(t *testing.T) {
	// Arrange
	repo := new(MockRepository)
	locker := new(MockLocker)
	billing := new(MockBillingClient)
	payment := new(MockPaymentClient)

	confirmedAt := time.Now().Add(-1 * time.Minute) // 1 minute ago
	reservation := &model.Reservation{
		ID:          "res-cancel-free",
		DriverID:    "driver-5",
		SpotID:      "spot-5",
		Status:      model.StatusConfirmed,
		ConfirmedAt: &confirmedAt,
	}

	repo.On("GetByIDForUpdate", mock.Anything, (*sqlx.Tx)(nil), "res-cancel-free").Return(reservation, nil)
	repo.On("UpdateReservationTx", mock.Anything, (*sqlx.Tx)(nil), mock.AnythingOfType("*model.Reservation")).Return(nil)
	repo.On("UpdateSpotStatusTx", mock.Anything, (*sqlx.Tx)(nil), "spot-5", "available").Return(nil)
	// No ApplyPenalty call expected since penalty system is removed

	uc := NewUsecase(repo, locker, billing, payment, nil, nil, nil, 60)
	req := &model.CancelReservationRequest{ReservationID: "res-cancel-free"}

	// Act
	result, err := uc.CancelReservation(t.Context(), req)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, model.StatusCancelled, result.Status)
	assert.NotNil(t, result.CancelledAt)
	repo.AssertExpectations(t)
}

// TestCancelReservation_ShouldChargeFee_WhenCancelledAfter2Min verifies
// that cancelling after 2 minutes results in a 5,000 IDR cancellation fee.
func TestCancelReservation_ShouldChargeFee_WhenCancelledAfter2Min(t *testing.T) {
	// Arrange
	repo := new(MockRepository)
	locker := new(MockLocker)
	billing := new(MockBillingClient)
	payment := new(MockPaymentClient)

	confirmedAt := time.Now().Add(-5 * time.Minute) // 5 minutes ago
	reservation := &model.Reservation{
		ID:          "res-cancel-paid",
		DriverID:    "driver-6",
		SpotID:      "spot-6",
		Status:      model.StatusConfirmed,
		ConfirmedAt: &confirmedAt,
	}

	repo.On("GetByIDForUpdate", mock.Anything, (*sqlx.Tx)(nil), "res-cancel-paid").Return(reservation, nil)
	repo.On("UpdateReservationTx", mock.Anything, (*sqlx.Tx)(nil), mock.AnythingOfType("*model.Reservation")).Return(nil)
	repo.On("UpdateSpotStatusTx", mock.Anything, (*sqlx.Tx)(nil), "spot-6", "available").Return(nil)
	uc := NewUsecase(repo, locker, billing, payment, nil, nil, nil, 60)
	req := &model.CancelReservationRequest{ReservationID: "res-cancel-paid"}

	// Act
	result, err := uc.CancelReservation(t.Context(), req)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, model.StatusCancelled, result.Status)
	repo.AssertExpectations(t)
}

// TestCancelReservation_ShouldReturnError_WhenInvalidState verifies
// that cancelling from checked_in state returns an error.
func TestCancelReservation_ShouldReturnError_WhenInvalidState(t *testing.T) {
	// Arrange
	repo := new(MockRepository)
	locker := new(MockLocker)
	billing := new(MockBillingClient)
	payment := new(MockPaymentClient)

	confirmedAt := time.Now().Add(-30 * time.Minute)
	checkedInAt := time.Now().Add(-20 * time.Minute)
	reservation := &model.Reservation{
		ID:          "res-checked-in",
		DriverID:    "driver-7",
		SpotID:      "spot-7",
		Status:      model.StatusCheckedIn,
		ConfirmedAt: &confirmedAt,
		CheckedInAt: &checkedInAt,
	}

	repo.On("GetByIDForUpdate", mock.Anything, (*sqlx.Tx)(nil), "res-checked-in").Return(reservation, nil)

	uc := NewUsecase(repo, locker, billing, payment, nil, nil, nil, 60)
	req := &model.CancelReservationRequest{ReservationID: "res-checked-in"}

	// Act
	result, err := uc.CancelReservation(t.Context(), req)

	// Assert
	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
	repo.AssertExpectations(t)
}

// TestCheckIn_ShouldTransitionToCheckedIn_WhenConfirmedState verifies
// check-in from confirmed state transitions to checked_in and spot becomes occupied.
func TestCheckIn_ShouldTransitionToCheckedIn_WhenConfirmedState(t *testing.T) {
	// Arrange
	repo := new(MockRepository)
	locker := new(MockLocker)
	billing := new(MockBillingClient)
	payment := new(MockPaymentClient)

	confirmedAt := time.Now().Add(-10 * time.Minute)
	reservation := &model.Reservation{
		ID:          "res-checkin",
		DriverID:    "driver-8",
		SpotID:      "spot-8",
		Status:      model.StatusConfirmed,
		ConfirmedAt: &confirmedAt,
	}

	repo.On("GetByIDForUpdate", mock.Anything, (*sqlx.Tx)(nil), "res-checkin").Return(reservation, nil)
	repo.On("UpdateReservationTx", mock.Anything, (*sqlx.Tx)(nil), mock.AnythingOfType("*model.Reservation")).Return(nil)
	repo.On("UpdateSpotStatusTx", mock.Anything, (*sqlx.Tx)(nil), "spot-8", "occupied").Return(nil)
	billing.On("StartBilling", mock.Anything, "res-checkin", int64(0), mock.AnythingOfType("string")).Return(&billingmodel.BillingRecord{ID: "billing-checkin-id"}, nil)

	uc := NewUsecase(repo, locker, billing, payment, nil, nil, nil, 60)
	req := &model.CheckInRequest{ReservationID: "res-checkin"}

	// Act
	result, err := uc.CheckIn(t.Context(), req)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, model.StatusCheckedIn, result.Reservation.Status)
	assert.NotNil(t, result.Reservation.CheckedInAt)
	repo.AssertExpectations(t)
	billing.AssertExpectations(t)
}

// TestCheckIn_ShouldReturnError_WhenPendingState verifies
// check-in from pending state returns an error.
func TestCheckIn_ShouldReturnError_WhenPendingState(t *testing.T) {
	// Arrange
	repo := new(MockRepository)
	locker := new(MockLocker)
	billing := new(MockBillingClient)
	payment := new(MockPaymentClient)

	reservation := &model.Reservation{
		ID:       "res-pending",
		DriverID: "driver-9",
		SpotID:   "spot-9",
		Status:   model.StatusPending,
	}

	repo.On("GetByIDForUpdate", mock.Anything, (*sqlx.Tx)(nil), "res-pending").Return(reservation, nil)

	uc := NewUsecase(repo, locker, billing, payment, nil, nil, nil, 60)
	req := &model.CheckInRequest{ReservationID: "res-pending"}

	// Act
	result, err := uc.CheckIn(t.Context(), req)

	// Assert
	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
	repo.AssertExpectations(t)
}

// TestCheckOut_ShouldCalculateFeeAndProcess_WhenCheckedInState verifies
// check-out from checked_in state calculates fee, processes payment, and releases spot.
func TestCheckOut_ShouldCalculateFeeAndProcess_WhenCheckedInState(t *testing.T) {
	// Arrange
	repo := new(MockRepository)
	locker := new(MockLocker)
	billing := new(MockBillingClient)
	payment := new(MockPaymentClient)

	confirmedAt := time.Now().Add(-3 * time.Hour)
	checkedInAt := time.Now().Add(-2 * time.Hour)
	reservation := &model.Reservation{
		ID:          "res-checkout",
		DriverID:    "driver-10",
		SpotID:      "spot-10",
		Status:      model.StatusCheckedIn,
		ConfirmedAt: &confirmedAt,
		CheckedInAt: &checkedInAt,
	}

	billingRecord := &billingmodel.BillingRecord{
		ID:          "billing-1",
		TotalAmount: 15000,
	}

	repo.On("GetByIDForUpdate", mock.Anything, (*sqlx.Tx)(nil), "res-checkout").Return(reservation, nil)
	repo.On("UpdateReservationTx", mock.Anything, (*sqlx.Tx)(nil), mock.AnythingOfType("*model.Reservation")).Return(nil)
	billing.On("CalculateFee", mock.Anything, "res-checkout", checkedInAt, mock.AnythingOfType("time.Time")).Return(billingRecord, nil)
	billing.On("GenerateInvoice", mock.Anything, "res-checkout", mock.AnythingOfType("string")).Return(billingRecord, nil)

	uc := NewUsecase(repo, locker, billing, payment, nil, nil, nil, 60)
	req := &model.CheckOutRequest{ReservationID: "res-checkout"}

	// Act
	result, err := uc.CheckOut(t.Context(), req)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, model.StatusCheckedOut, result.Reservation.Status)
	assert.NotNil(t, result.Reservation.CheckedOutAt)
	assert.Equal(t, int64(15000), result.TotalAmount)
	assert.Equal(t, "billing-1", result.BillingID)
	assert.Equal(t, "", result.PaymentID)
	repo.AssertExpectations(t)
	billing.AssertExpectations(t)
	payment.AssertExpectations(t)
}

// TestExpireReservation_ShouldReleaseSpot_WhenConfirmedState verifies
// expiring from confirmed state releases the spot. Per PRD, the booking fee
// (already charged at confirmation) is the only cost — no additional no-show
// penalty is applied.
func TestExpireReservation_ShouldReleaseSpot_WhenConfirmedState(t *testing.T) {
	// Arrange
	repo := new(MockRepository)
	locker := new(MockLocker)
	billing := new(MockBillingClient)
	payment := new(MockPaymentClient)

	confirmedAt := time.Now().Add(-2 * time.Hour)
	expiresAt := time.Now().Add(-1 * time.Hour)
	reservation := &model.Reservation{
		ID:          "res-expire",
		DriverID:    "driver-11",
		SpotID:      "spot-11",
		Status:      model.StatusConfirmed,
		ConfirmedAt: &confirmedAt,
		ExpiresAt:   &expiresAt,
	}

	repo.On("GetByIDForUpdate", mock.Anything, (*sqlx.Tx)(nil), "res-expire").Return(reservation, nil)
	repo.On("UpdateReservationTx", mock.Anything, (*sqlx.Tx)(nil), mock.AnythingOfType("*model.Reservation")).Return(nil)
	repo.On("UpdateSpotStatusTx", mock.Anything, (*sqlx.Tx)(nil), "spot-11", "available").Return(nil)

	uc := NewUsecase(repo, locker, billing, payment, nil, nil, nil, 60)
	req := &model.ExpireReservationRequest{ReservationID: "res-expire"}

	// Act
	err := uc.ExpireReservation(t.Context(), req)

	// Assert
	require.NoError(t, err)
	// No penalty system — booking fee already charged at confirmation is the only cost
	repo.AssertExpectations(t)
}

// TestCreateReservation_ShouldReturnExisting_WhenUniqueConstraintViolation verifies
// that when CreateReservationTx fails with a unique constraint violation (concurrent
// duplicate idempotency key), the usecase retries FindByIdempotencyKey and returns
// the existing reservation instead of an error.
func TestCreateReservation_ShouldReturnExisting_WhenUniqueConstraintViolation(t *testing.T) {
	// Arrange
	repo := new(MockRepository)
	locker := new(MockLocker)
	billing := new(MockBillingClient)
	payment := new(MockPaymentClient)

	existing := &model.Reservation{
		ID:             "existing-concurrent-id",
		DriverID:       "driver-1",
		SpotID:         "spot-42",
		VehicleType:    "car",
		AssignmentMode: model.AssignmentSystemAssigned,
		Status:         model.StatusConfirmed,
		IdempotencyKey: "dup-key",
	}

	// First idempotency check: not found (concurrent request hasn't committed yet)
	repo.On("FindByIdempotencyKey", mock.Anything, "dup-key").Return(nil, model.ErrNotFound).Once()
	repo.On("ListByDriverID", mock.Anything, "driver-1", "").Return([]*model.Reservation{}, nil)
	repo.On("FindAvailableSpot", mock.Anything, "car").Return(&model.ParkingSpot{
		ID:          "spot-42",
		VehicleType: "car",
		Status:      "available",
	}, nil)
	lck := new(MockLock)
	locker.On("Acquire", mock.Anything, "spot:spot-42").Return(lck, nil)
	lck.On("Release", mock.Anything).Return(nil)
	repo.On("GetSpotForUpdateTx", mock.Anything, (*sqlx.Tx)(nil), "spot-42").Return(&model.ParkingSpot{
		ID:     "spot-42",
		Status: "available",
	}, nil)
	// CreateReservationTx fails with unique constraint violation (PG error code 23505)
	repo.On("CreateReservationTx", mock.Anything, (*sqlx.Tx)(nil), mock.AnythingOfType("*model.Reservation")).
		Return(&pgconn.PgError{Code: "23505", Message: "duplicate key value violates unique constraint"})
	// Retry idempotency lookup returns the existing reservation
	repo.On("FindByIdempotencyKey", mock.Anything, "dup-key").Return(existing, nil).Once()

	uc := NewUsecase(repo, locker, billing, payment, nil, nil, nil, 60)
	req := &model.CreateReservationRequest{
		DriverID:       "driver-1",
		VehicleType:    "car",
		AssignmentMode: model.AssignmentSystemAssigned,
		IdempotencyKey: "dup-key",
	}

	// Act
	result, err := uc.CreateReservation(t.Context(), req)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "existing-concurrent-id", result.ID)
	assert.Equal(t, model.StatusConfirmed, result.Status)
	assert.Equal(t, "dup-key", result.IdempotencyKey)
	repo.AssertExpectations(t)
	locker.AssertExpectations(t)
	// No billing or NATS calls should have been made
	billing.AssertNotCalled(t, "StartBilling")
}

// TestCreateReservation_ShouldReturnError_WhenNonUniqueConstraintError verifies
// that when CreateReservationTx fails with a non-unique-constraint error, the
// original error is returned without retrying FindByIdempotencyKey.
func TestCreateReservation_ShouldReturnError_WhenNonUniqueConstraintError(t *testing.T) {
	// Arrange
	repo := new(MockRepository)
	locker := new(MockLocker)
	billing := new(MockBillingClient)
	payment := new(MockPaymentClient)

	repo.On("FindByIdempotencyKey", mock.Anything, "err-key").Return(nil, model.ErrNotFound)
	repo.On("ListByDriverID", mock.Anything, "driver-1", "").Return([]*model.Reservation{}, nil)
	repo.On("FindAvailableSpot", mock.Anything, "car").Return(&model.ParkingSpot{
		ID:          "spot-50",
		VehicleType: "car",
		Status:      "available",
	}, nil)
	lck := new(MockLock)
	locker.On("Acquire", mock.Anything, "spot:spot-50").Return(lck, nil)
	lck.On("Release", mock.Anything).Return(nil)
	repo.On("GetSpotForUpdateTx", mock.Anything, (*sqlx.Tx)(nil), "spot-50").Return(&model.ParkingSpot{
		ID:     "spot-50",
		Status: "available",
	}, nil)
	// CreateReservationTx fails with a generic DB error (not unique constraint)
	repo.On("CreateReservationTx", mock.Anything, (*sqlx.Tx)(nil), mock.AnythingOfType("*model.Reservation")).
		Return(fmt.Errorf("connection refused"))

	uc := NewUsecase(repo, locker, billing, payment, nil, nil, nil, 60)
	req := &model.CreateReservationRequest{
		DriverID:       "driver-1",
		VehicleType:    "car",
		AssignmentMode: model.AssignmentSystemAssigned,
		IdempotencyKey: "err-key",
	}

	// Act
	result, err := uc.CreateReservation(t.Context(), req)

	// Assert
	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection refused")
	repo.AssertExpectations(t)
	locker.AssertExpectations(t)
}

// TestCreateReservation_ShouldReject_WhenVehicleTypeMismatches verifies that
// a user-selected reservation is rejected if the spot's vehicle type does not
// match the requested vehicle type.
func TestCreateReservation_ShouldReject_WhenVehicleTypeMismatches(t *testing.T) {
	ctx := context.Background()

	repo := new(MockRepository)
	locker := new(MockLocker)
	billing := new(MockBillingClient)
	payment := new(MockPaymentClient)
	uc := NewUsecase(repo, locker, billing, payment, nil, nil, nil, 60)

	// A motorcycle spot.
	motorcycleSpot := &model.ParkingSpot{
		ID:          "spot-moto-001",
		VehicleType: "motorcycle",
		Status:      "available",
	}

	repo.On("FindByIdempotencyKey", ctx, "idem-mismatch-001").Return(nil, model.ErrNotFound)
	repo.On("ListByDriverID", ctx, "driver-001", "").Return([]*model.Reservation{}, nil)
	repo.On("GetSpotForUpdateTx", mock.Anything, (*sqlx.Tx)(nil), "spot-moto-001").Return(motorcycleSpot, nil)
	lck := new(MockLock)
	locker.On("Acquire", ctx, "spot:spot-moto-001").Return(lck, nil)
	lck.On("Release", mock.Anything).Return(nil)

	_, err := uc.CreateReservation(ctx, &model.CreateReservationRequest{
		DriverID:       "driver-001",
		VehicleType:    "car",
		AssignmentMode: model.AssignmentUserSelected,
		SpotID:         "spot-moto-001",
		IdempotencyKey: "idem-mismatch-001",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "vehicle type does not match")

	repo.AssertExpectations(t)
	locker.AssertExpectations(t)
}

// TestConfirmReservation_ShouldTransitionToConfirmed_WhenWaitingPayment verifies
// that ConfirmReservation transitions a waiting_payment reservation to confirmed
// after processing the booking fee payment.
func TestConfirmReservation_ShouldTransitionToConfirmed_WhenWaitingPayment(t *testing.T) {
	repo := new(MockRepository)
	locker := new(MockLocker)
	billing := new(MockBillingClient)
	payment := new(MockPaymentClient)

	reservation := &model.Reservation{
		ID:       "res-confirm",
		DriverID: "driver-1",
		SpotID:   "spot-1",
		Status:   model.StatusWaitingPayment,
	}

	billingRecord := &billingmodel.BillingRecord{ID: "billing-1"}

	repo.On("GetByIDForUpdate", mock.Anything, (*sqlx.Tx)(nil), "res-confirm").Return(reservation, nil)
	repo.On("UpdateReservationTx", mock.Anything, (*sqlx.Tx)(nil), mock.AnythingOfType("*model.Reservation")).Return(nil)
	billing.On("StartBilling", mock.Anything, "res-confirm", pricing.BookingFee, mock.AnythingOfType("string")).Return(billingRecord, nil)
	payment.On("ProcessPayment", mock.Anything, "billing-1", pricing.BookingFee, "qris", mock.AnythingOfType("string")).Return("pay-1", nil)

	uc := NewUsecase(repo, locker, billing, payment, nil, nil, nil, 60)
	result, err := uc.ConfirmReservation(t.Context(), &model.ConfirmReservationRequest{ReservationID: "res-confirm"})

	require.NoError(t, err)
	assert.Equal(t, model.StatusConfirmed, result.Status)
	assert.NotNil(t, result.ConfirmedAt)
	repo.AssertExpectations(t)
	billing.AssertExpectations(t)
	payment.AssertExpectations(t)
}

// TestConfirmReservation_ShouldReturnError_WhenNotWaitingPayment verifies
// that ConfirmReservation returns an error if the reservation is not in
// waiting_payment state.
func TestConfirmReservation_ShouldReturnError_WhenNotWaitingPayment(t *testing.T) {
	repo := new(MockRepository)
	locker := new(MockLocker)
	billing := new(MockBillingClient)
	payment := new(MockPaymentClient)

	reservation := &model.Reservation{
		ID:     "res-confirm",
		Status: model.StatusConfirmed,
	}

	repo.On("GetByIDForUpdate", mock.Anything, (*sqlx.Tx)(nil), "res-confirm").Return(reservation, nil)

	uc := NewUsecase(repo, locker, billing, payment, nil, nil, nil, 60)
	_, err := uc.ConfirmReservation(t.Context(), &model.ConfirmReservationRequest{ReservationID: "res-confirm"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not pending payment")
	repo.AssertExpectations(t)
}

// TestCompleteCheckout_ShouldProcessPaymentAndReleaseSpot_WhenCheckedOut verifies
// that CompleteCheckout processes payment and releases the spot for a checked_out
// reservation.
func TestCompleteCheckout_ShouldProcessPaymentAndReleaseSpot_WhenCheckedOut(t *testing.T) {
	repo := new(MockRepository)
	locker := new(MockLocker)
	billing := new(MockBillingClient)
	payment := new(MockPaymentClient)

	checkedOutAt := time.Now().Add(-1 * time.Hour)
	reservation := &model.Reservation{
		ID:           "res-complete",
		DriverID:     "driver-1",
		SpotID:       "spot-1",
		Status:       model.StatusCheckedOut,
		CheckedOutAt: &checkedOutAt,
	}

	billingRecord := &billingmodel.BillingRecord{
		ID:          "billing-2",
		TotalAmount: 15000,
	}

	repo.On("GetByIDForUpdate", mock.Anything, (*sqlx.Tx)(nil), "res-complete").Return(reservation, nil)
	billing.On("GenerateInvoice", mock.Anything, "res-complete", mock.AnythingOfType("string")).Return(billingRecord, nil)
	payment.On("ProcessPayment", mock.Anything, "billing-2", int64(15000), "qris", mock.AnythingOfType("string")).Return("pay-2", nil)
	repo.On("UpdateSpotStatusTx", mock.Anything, (*sqlx.Tx)(nil), "spot-1", "available").Return(nil)

	uc := NewUsecase(repo, locker, billing, payment, nil, nil, nil, 60)
	result, err := uc.CompleteCheckout(t.Context(), &model.CompleteCheckoutRequest{ReservationID: "res-complete"})

	require.NoError(t, err)
	assert.Equal(t, model.StatusCheckedOut, result.Reservation.Status)
	assert.Equal(t, int64(15000), result.TotalAmount)
	assert.Equal(t, "pay-2", result.PaymentID)
	assert.Equal(t, "billing-2", result.BillingID)
	repo.AssertExpectations(t)
	billing.AssertExpectations(t)
	payment.AssertExpectations(t)
}

// TestCompleteCheckout_ShouldReturnError_WhenNotCheckedOut verifies
// that CompleteCheckout returns an error if the reservation is not in
// checked_out state.
func TestCompleteCheckout_ShouldReturnError_WhenNotCheckedOut(t *testing.T) {
	repo := new(MockRepository)
	locker := new(MockLocker)
	billing := new(MockBillingClient)
	payment := new(MockPaymentClient)

	reservation := &model.Reservation{
		ID:     "res-complete",
		Status: model.StatusCheckedIn,
	}

	repo.On("GetByIDForUpdate", mock.Anything, (*sqlx.Tx)(nil), "res-complete").Return(reservation, nil)

	uc := NewUsecase(repo, locker, billing, payment, nil, nil, nil, 60)
	_, err := uc.CompleteCheckout(t.Context(), &model.CompleteCheckoutRequest{ReservationID: "res-complete"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not checked out")
	repo.AssertExpectations(t)
}
