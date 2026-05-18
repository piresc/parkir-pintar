// Package integration provides integration tests for the ParkirPintar reservation
// lifecycle, testing the full flow through the usecase layer with mocked dependencies.
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
// - Use mock.AnythingOfType() for type-safe argument matching
// - Use AssertNotCalled() to verify methods weren't called
package integration

import (
	"context"
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

func (m *MockRepository) FindStalePaymentReservations(ctx context.Context, timeoutMinutes int) ([]*model.Reservation, error) {
	args := m.Called(ctx, timeoutMinutes)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Reservation), args.Error(1)
}

func (m *MockRepository) UpdateSpotStatusTx(ctx context.Context, tx *sqlx.Tx, spotID string, status string) error {
	args := m.Called(ctx, tx, spotID, status)
	return args.Error(0)
}

func (m *MockRepository) UpdateReservationTx(ctx context.Context, tx *sqlx.Tx, reservation *model.Reservation) error {
	args := m.Called(ctx, tx, reservation)
	return args.Error(0)
}

func (m *MockRepository) GetByIDForUpdate(ctx context.Context, tx *sqlx.Tx, id string) (*model.Reservation, error) {
	args := m.Called(ctx, tx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Reservation), args.Error(1)
}

func (m *MockRepository) FindExpiredReservations(ctx context.Context) ([]*model.Reservation, error) {
	args := m.Called(ctx)
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

func (m *MockRepository) ListByDriverID(ctx context.Context, driverID string, status string) ([]*model.Reservation, error) {
	args := m.Called(ctx, driverID, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Reservation), args.Error(1)
}

func (m *MockRepository) WithTransaction(ctx context.Context, fn func(tx *sqlx.Tx) error) error {
	return fn(nil)
}

// MockBillingClient implements usecase.BillingClient using testify/mock.
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

// MockPaymentClient implements usecase.PaymentClient using testify/mock.
type MockPaymentClient struct {
	mock.Mock
}

func (m *MockPaymentClient) ProcessPayment(ctx context.Context, billingID string, amount int64, paymentMethod string, idempotencyKey string) (string, error) {
	args := m.Called(ctx, billingID, amount, paymentMethod, idempotencyKey)
	return args.String(0), args.Error(1)
}

type MockLock struct {
	mock.Mock
}

func (m *MockLock) Release(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

type MockLocker struct {
	mock.Mock
}

func (m *MockLocker) Acquire(ctx context.Context, key string) (usecase.Lock, error) {
	args := m.Called(ctx, key)
	if lock, ok := args.Get(0).(usecase.Lock); ok {
		return lock, args.Error(1)
	}
	return nil, args.Error(1)
}

// --- Integration Test: Full Reservation-to-Billing Flow ---

// TestReservationToBillingFlow_ShouldCompleteFullLifecycle_WhenHappyPath tests the
// complete reservation lifecycle: create → check-in → check-out, verifying all
// billing and payment interactions at each stage.
//
// Validates: Requirements 2.1, 2.4, 4.1, 4.2, 5.1, 25.4
func TestReservationToBillingFlow_ShouldCompleteFullLifecycle_WhenHappyPath(t *testing.T) {
	// Arrange — set up all mocks
	repo := new(MockRepository)
	locker := new(MockLocker)
	billing := new(MockBillingClient)
	payment := new(MockPaymentClient)

	uc := usecase.NewUsecase(repo, locker, billing, payment, nil, nil, nil, 60)

	// --- Phase 1: Create Reservation ---

	// Arrange: idempotency check returns not found
	repo.On("FindByIdempotencyKey", mock.Anything, "integ-key-1").Return(nil, model.ErrNotFound)
	repo.On("FindAvailableSpot", mock.Anything, "car").Return(&model.ParkingSpot{
		ID:          "spot-integ-1",
		VehicleType: "car",
		Status:      "available",
	}, nil)
	lock := new(MockLock)
	locker.On("Acquire", mock.Anything, "spot:spot-integ-1").Return(lock, nil)
	lock.On("Release", mock.Anything).Return(nil)
	repo.On("GetSpotForUpdateTx", mock.Anything, (*sqlx.Tx)(nil), "spot-integ-1").Return(&model.ParkingSpot{
		ID:     "spot-integ-1",
		Status: "available",
	}, nil)
	repo.On("ListByDriverID", mock.Anything, "driver-integ-1", "").Return([]*model.Reservation{}, nil)
	repo.On("CreateReservationTx", mock.Anything, (*sqlx.Tx)(nil), mock.AnythingOfType("*model.Reservation")).Return(nil)
	repo.On("UpdateSpotStatusTx", mock.Anything, (*sqlx.Tx)(nil), "spot-integ-1", "reserved").Return(nil)
	billing.On("StartBilling", mock.Anything, mock.AnythingOfType("string"), pricing.BookingFee, mock.AnythingOfType("string")).Return(&billingmodel.BillingRecord{ID: "billing-test-id"}, nil)

	// Act: create reservation
	createReq := &model.CreateReservationRequest{
		DriverID:       "driver-integ-1",
		VehicleType:    "car",
		AssignmentMode: model.AssignmentSystemAssigned,
		IdempotencyKey: "integ-key-1",
	}
	reservation, err := uc.CreateReservation(t.Context(), createReq)

	// Assert: reservation created with waiting_payment status
	require.NoError(t, err)
	require.NotNil(t, reservation)
	assert.Equal(t, model.StatusWaitingPayment, reservation.Status)
	assert.Equal(t, "spot-integ-1", reservation.SpotID)

	// Verify billing StartBilling was called with booking_fee=5000
	billing.AssertCalled(t, "StartBilling", mock.Anything, reservation.ID, pricing.BookingFee, mock.AnythingOfType("string"))

	// --- Phase 1b: Confirm reservation ---
	confirmedAt := time.Now().Add(-2 * time.Hour)
	repo.On("GetByIDForUpdate", mock.Anything, (*sqlx.Tx)(nil), reservation.ID).Return(&model.Reservation{
		ID:          reservation.ID,
		DriverID:    "driver-integ-1",
		SpotID:      "spot-integ-1",
		Status:      model.StatusWaitingPayment,
		ConfirmedAt: nil,
	}, nil).Once()
	// Second GetByIDForUpdate: re-check inside confirmation transaction (TOCTOU fix)
	repo.On("GetByIDForUpdate", mock.Anything, (*sqlx.Tx)(nil), reservation.ID).Return(&model.Reservation{
		ID:          reservation.ID,
		DriverID:    "driver-integ-1",
		SpotID:      "spot-integ-1",
		Status:      model.StatusWaitingPayment,
		ConfirmedAt: nil,
	}, nil).Once()
	repo.On("UpdateReservationTx", mock.Anything, (*sqlx.Tx)(nil), mock.MatchedBy(func(r *model.Reservation) bool {
		return r.Status == model.StatusConfirmed
	})).Return(nil).Once()
	payment.On("ProcessPayment", mock.Anything, "billing-test-id", pricing.BookingFee, "qris", mock.AnythingOfType("string")).Return("pay-booking", nil).Once()

	confirmed, err := uc.ConfirmReservation(t.Context(), &model.ConfirmReservationRequest{
		ReservationID: reservation.ID,
	})
	require.NoError(t, err)
	require.NotNil(t, confirmed)
	assert.Equal(t, model.StatusConfirmed, confirmed.Status)

	// --- Phase 2: Check-In ---

	// Arrange: return the created reservation for check-in
	repo.On("GetByIDForUpdate", mock.Anything, (*sqlx.Tx)(nil), reservation.ID).Return(&model.Reservation{
		ID:          reservation.ID,
		DriverID:    "driver-integ-1",
		SpotID:      "spot-integ-1",
		Status:      model.StatusConfirmed,
		ConfirmedAt: &confirmedAt,
	}, nil).Once()
	repo.On("UpdateReservationTx", mock.Anything, (*sqlx.Tx)(nil), mock.MatchedBy(func(r *model.Reservation) bool {
		return r.Status == model.StatusCheckedIn
	})).Return(nil).Once()
	repo.On("UpdateSpotStatusTx", mock.Anything, (*sqlx.Tx)(nil), "spot-integ-1", "occupied").Return(nil)
	billing.On("StartBilling", mock.Anything, reservation.ID, int64(0), mock.AnythingOfType("string")).Return(&billingmodel.BillingRecord{ID: "billing-checkin-id"}, nil)

	// Act: check in
	checkInReq := &model.CheckInRequest{ReservationID: reservation.ID}
	checkedIn, err := uc.CheckIn(t.Context(), checkInReq)

	// Assert: reservation transitioned to checked_in
	require.NoError(t, err)
	require.NotNil(t, checkedIn)
	assert.Equal(t, model.StatusCheckedIn, checkedIn.Reservation.Status)
	assert.NotNil(t, checkedIn.Reservation.CheckedInAt)

	// Verify billing activation was called (StartBilling with 0 fee)
	billing.AssertCalled(t, "StartBilling", mock.Anything, reservation.ID, int64(0), mock.AnythingOfType("string"))

	// --- Phase 3: Check-Out ---

	// Arrange: return checked-in reservation for checkout
	checkedInAt := *checkedIn.Reservation.CheckedInAt
	billingRecord := &billingmodel.BillingRecord{
		ID:          "billing-integ-1",
		TotalAmount: 15000,
		BookingFee:  pricing.BookingFee,
		ParkingFee:  10000,
	}

	repo.On("GetByIDForUpdate", mock.Anything, (*sqlx.Tx)(nil), reservation.ID).Return(&model.Reservation{
		ID:          reservation.ID,
		DriverID:    "driver-integ-1",
		SpotID:      "spot-integ-1",
		Status:      model.StatusCheckedIn,
		ConfirmedAt: &confirmedAt,
		CheckedInAt: &checkedInAt,
	}, nil).Once()
	repo.On("UpdateReservationTx", mock.Anything, (*sqlx.Tx)(nil), mock.MatchedBy(func(r *model.Reservation) bool {
		return r.Status == model.StatusCheckedOut
	})).Return(nil).Once()
	billing.On("CalculateFee", mock.Anything, reservation.ID, checkedInAt, mock.AnythingOfType("time.Time")).Return(billingRecord, nil)
	billing.On("GenerateInvoice", mock.Anything, reservation.ID, mock.AnythingOfType("string")).Return(billingRecord, nil)

	// Act: check out
	checkOutReq := &model.CheckOutRequest{ReservationID: reservation.ID}
	checkOutResult, err := uc.CheckOut(t.Context(), checkOutReq)

	// Assert: reservation transitioned to checked_out with billing details
	require.NoError(t, err)
	require.NotNil(t, checkOutResult)
	assert.Equal(t, model.StatusCheckedOut, checkOutResult.Reservation.Status)
	assert.Equal(t, int64(15000), checkOutResult.TotalAmount)
	assert.Equal(t, "billing-integ-1", checkOutResult.BillingID)
	assert.Equal(t, "", checkOutResult.PaymentID)
	assert.Equal(t, pricing.BookingFee, checkOutResult.BookingFee)
	assert.Equal(t, int64(10000), checkOutResult.ParkingFee)
	assert.Equal(t, int64(0), checkOutResult.OvernightFee)

	// Verify: payment NOT processed during checkout (happens in CompleteCheckout)
	payment.AssertNotCalled(t, "ProcessPayment")

	// Verify all mock expectations
	repo.AssertExpectations(t)
	locker.AssertExpectations(t)
	lock.AssertExpectations(t)
	billing.AssertExpectations(t)
	payment.AssertExpectations(t)
}
