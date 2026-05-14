// Package usecase implements the business logic layer for the billing domain.
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
// - Keep mocks simple and focused on the behavior being tested
package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"parkir-pintar/internal/billing/model"
	"parkir-pintar/internal/billing/repository"
	"parkir-pintar/pkg/pricing"
)

// --- Mock Implementations ---

// MockRepository implements repository.Repository using testify/mock.
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) CreateBillingRecord(ctx context.Context, record *model.BillingRecord) error {
	args := m.Called(ctx, record)
	return args.Error(0)
}

func (m *MockRepository) GetByReservationID(ctx context.Context, reservationID string) (*model.BillingRecord, error) {
	args := m.Called(ctx, reservationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.BillingRecord), args.Error(1)
}

func (m *MockRepository) GetByIdempotencyKey(ctx context.Context, key string) (*model.BillingRecord, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.BillingRecord), args.Error(1)
}

func (m *MockRepository) UpdateBillingRecord(ctx context.Context, record *model.BillingRecord) error {
	args := m.Called(ctx, record)
	return args.Error(0)
}

// --- Test Cases ---

// TestStartBilling_ShouldCreateRecord_WhenNewReservation verifies that
// StartBilling creates a billing record with booking_fee=5000 and status "pending".
func TestStartBilling_ShouldCreateRecord_WhenNewReservation(t *testing.T) {
	// Arrange
	repo := new(MockRepository)

	repo.On("GetByIdempotencyKey", mock.Anything, "billing-res-1").Return(nil, repository.ErrNotFound)
	repo.On("GetByReservationID", mock.Anything, "res-1").Return(nil, repository.ErrNotFound)
	repo.On("CreateBillingRecord", mock.Anything, mock.AnythingOfType("*model.BillingRecord")).Return(nil)

	uc := NewUsecase(repo)
	req := &model.StartBillingRequest{
		ReservationID:  "res-1",
		BookingFee:     pricing.BookingFee,
		IdempotencyKey: "billing-res-1",
	}

	// Act
	result, err := uc.StartBilling(t.Context(), req)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "res-1", result.ReservationID)
	assert.Equal(t, pricing.BookingFee, result.BookingFee)
	assert.Equal(t, model.BillingStatusPending, result.Status)
	assert.Equal(t, pricing.BookingFee, result.TotalAmount)
	assert.NotEmpty(t, result.ID)
	repo.AssertExpectations(t)
}

// TestStartBilling_ShouldReturnExisting_WhenDuplicateIdempotencyKey verifies
// that a duplicate idempotency key returns the existing record without side effects.
func TestStartBilling_ShouldReturnExisting_WhenDuplicateIdempotencyKey(t *testing.T) {
	// Arrange
	repo := new(MockRepository)

	existing := &model.BillingRecord{
		ID:             "existing-billing-id",
		ReservationID:  "res-1",
		BookingFee:     pricing.BookingFee,
		TotalAmount:    pricing.BookingFee,
		IdempotencyKey: "billing-res-1",
		Status:         model.BillingStatusPending,
	}
	repo.On("GetByIdempotencyKey", mock.Anything, "billing-res-1").Return(existing, nil)

	uc := NewUsecase(repo)
	req := &model.StartBillingRequest{
		ReservationID:  "res-1",
		BookingFee:     pricing.BookingFee,
		IdempotencyKey: "billing-res-1",
	}

	// Act
	result, err := uc.StartBilling(t.Context(), req)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "existing-billing-id", result.ID)
	// CreateBillingRecord should NOT have been called
	repo.AssertNotCalled(t, "CreateBillingRecord")
	repo.AssertExpectations(t)
}

// TestCalculateFee_ShouldComputeCorrectFees_WhenStandardSession verifies
// that CalculateFee computes correct parking_fee, overnight_fee, and total
// for a standard 2-hour same-day session.
func TestCalculateFee_ShouldComputeCorrectFees_WhenStandardSession(t *testing.T) {
	// Arrange
	repo := new(MockRepository)

	existingRecord := &model.BillingRecord{
		ID:            "billing-1",
		ReservationID: "res-1",
		BookingFee:    pricing.BookingFee,
		Status:        model.BillingStatusPending,
	}
	repo.On("GetByReservationID", mock.Anything, "res-1").Return(existingRecord, nil)
	repo.On("UpdateBillingRecord", mock.Anything, mock.AnythingOfType("*model.BillingRecord")).Return(nil)

	loc := time.FixedZone("WIB", 7*60*60)
	checkIn := time.Date(2026, 4, 24, 10, 0, 0, 0, loc)
	checkOut := time.Date(2026, 4, 24, 12, 0, 0, 0, loc)

	uc := NewUsecase(repo)
	req := &model.CalculateFeeRequest{
		ReservationID: "res-1",
		CheckInAt:     checkIn,
		CheckOutAt:    checkOut,
	}

	// Act
	result, err := uc.CalculateFee(t.Context(), req)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, int64(10_000), result.ParkingFee) // 2h × 5000
	assert.Equal(t, int64(0), result.OvernightFee)    // same day
	assert.Equal(t, 120, result.DurationMinutes)      // 2 hours
	assert.Equal(t, 2, result.BilledHours)
	assert.False(t, result.IsOvernight)
	assert.Equal(t, model.BillingStatusCalculated, result.Status)
	// total = booking_fee + parking_fee + overnight_fee
	expectedTotal := pricing.BookingFee + int64(10_000)
	assert.Equal(t, expectedTotal, result.TotalAmount)
	repo.AssertExpectations(t)
}

// TestCalculateFee_ShouldApplyOvernightFee_WhenSessionCrossesMidnight verifies
// that CalculateFee applies the overnight fee when the session crosses midnight in WIB.
func TestCalculateFee_ShouldApplyOvernightFee_WhenSessionCrossesMidnight(t *testing.T) {
	// Arrange
	repo := new(MockRepository)

	existingRecord := &model.BillingRecord{
		ID:            "billing-2",
		ReservationID: "res-2",
		BookingFee:    pricing.BookingFee,
		Status:        model.BillingStatusPending,
	}
	repo.On("GetByReservationID", mock.Anything, "res-2").Return(existingRecord, nil)
	repo.On("UpdateBillingRecord", mock.Anything, mock.AnythingOfType("*model.BillingRecord")).Return(nil)

	loc := time.FixedZone("WIB", 7*60*60)
	checkIn := time.Date(2026, 4, 24, 22, 0, 0, 0, loc)
	checkOut := time.Date(2026, 4, 25, 6, 0, 0, 0, loc)

	uc := NewUsecase(repo)
	req := &model.CalculateFeeRequest{
		ReservationID: "res-2",
		CheckInAt:     checkIn,
		CheckOutAt:    checkOut,
	}

	// Act
	result, err := uc.CalculateFee(t.Context(), req)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, int64(40_000), result.ParkingFee)               // 8h × 5000
	assert.Equal(t, pricing.OvernightPerNight, result.OvernightFee) // 20,000
	assert.Equal(t, 480, result.DurationMinutes)                    // 8 hours
	assert.Equal(t, 8, result.BilledHours)
	assert.True(t, result.IsOvernight)
	// total = 5000 + 40000 + 20000 = 65000
	assert.Equal(t, int64(65_000), result.TotalAmount)
	repo.AssertExpectations(t)
}

// TestGenerateInvoice_ShouldReturnExisting_WhenDuplicateIdempotencyKey verifies
// that GenerateInvoice returns the existing record when the idempotency key already exists.
func TestGenerateInvoice_ShouldReturnExisting_WhenDuplicateIdempotencyKey(t *testing.T) {
	// Arrange
	repo := new(MockRepository)

	existing := &model.BillingRecord{
		ID:             "billing-3",
		ReservationID:  "res-3",
		BookingFee:     pricing.BookingFee,
		ParkingFee:     10_000,
		TotalAmount:    15_000,
		IdempotencyKey: "invoice-res-3",
		Status:         model.BillingStatusInvoiced,
	}
	repo.On("GetByIdempotencyKey", mock.Anything, "invoice-res-3").Return(existing, nil)

	uc := NewUsecase(repo)
	req := &model.GenerateInvoiceRequest{
		ReservationID:  "res-3",
		IdempotencyKey: "invoice-res-3",
	}

	// Act
	result, err := uc.GenerateInvoice(t.Context(), req)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "billing-3", result.ID)
	assert.Equal(t, model.BillingStatusInvoiced, result.Status)
	// UpdateBillingRecord should NOT have been called (idempotent return)
	repo.AssertNotCalled(t, "UpdateBillingRecord")
	repo.AssertExpectations(t)
}

// TestGenerateInvoice_ShouldUpdateStatus_WhenNewInvoice verifies
// that GenerateInvoice updates the billing record status to "invoiced".
func TestGenerateInvoice_ShouldUpdateStatus_WhenNewInvoice(t *testing.T) {
	// Arrange
	repo := new(MockRepository)

	repo.On("GetByIdempotencyKey", mock.Anything, "invoice-res-4").Return(nil, repository.ErrNotFound)
	existingRecord := &model.BillingRecord{
		ID:             "billing-4",
		ReservationID:  "res-4",
		BookingFee:     pricing.BookingFee,
		ParkingFee:     10_000,
		TotalAmount:    15_000,
		IdempotencyKey: "billing-res-4",
		Status:         model.BillingStatusCalculated,
	}
	repo.On("GetByReservationID", mock.Anything, "res-4").Return(existingRecord, nil)
	repo.On("UpdateBillingRecord", mock.Anything, mock.AnythingOfType("*model.BillingRecord")).Return(nil)

	uc := NewUsecase(repo)
	req := &model.GenerateInvoiceRequest{
		ReservationID:  "res-4",
		IdempotencyKey: "invoice-res-4",
	}

	// Act
	result, err := uc.GenerateInvoice(t.Context(), req)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, model.BillingStatusInvoiced, result.Status)
	repo.AssertExpectations(t)
}

// TestBillingTotalInvariant_ShouldEqualSumOfFees verifies
// that total_amount always equals the sum of all fee fields.
func TestBillingTotalInvariant_ShouldEqualSumOfFees(t *testing.T) {
	// Arrange
	record := &model.BillingRecord{
		BookingFee:   pricing.BookingFee,
		ParkingFee:   15_000,
		OvernightFee: pricing.OvernightPerNight,
	}

	// Act
	total := pricing.CalculateTotal(record.BookingFee, record.ParkingFee, record.OvernightFee)

	// Assert
	expectedTotal := pricing.BookingFee + int64(15_000) + pricing.OvernightPerNight
	assert.Equal(t, expectedTotal, total)
	assert.Equal(t, record.BookingFee+record.ParkingFee+record.OvernightFee, total)
	assert.GreaterOrEqual(t, total, pricing.BookingFee)
}
