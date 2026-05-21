package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	billingconstants "parkir-pintar/internal/billing/constants"
	"parkir-pintar/internal/billing/model"
	"parkir-pintar/internal/billing/repository"
	"parkir-pintar/internal/pricing"
)

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

func TestStartBilling_ShouldCreateRecord_WhenNewReservation(t *testing.T) {
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

	result, err := uc.StartBilling(t.Context(), req)

	require.NoError(t, err)
	assert.Equal(t, "res-1", result.ReservationID)
	assert.Equal(t, pricing.BookingFee, result.BookingFee)
	assert.Equal(t, string(billingconstants.BillingStatusPending), result.Status)
	assert.Equal(t, pricing.BookingFee, result.TotalAmount)
	assert.NotEmpty(t, result.ID)
	repo.AssertExpectations(t)
}

func TestStartBilling_ShouldReturnExisting_WhenDuplicateIdempotencyKey(t *testing.T) {
	repo := new(MockRepository)

	existing := &model.BillingRecord{
		ID:             "existing-billing-id",
		ReservationID:  "res-1",
		BookingFee:     pricing.BookingFee,
		TotalAmount:    pricing.BookingFee,
		IdempotencyKey: "billing-res-1",
		Status:         string(billingconstants.BillingStatusPending),
	}
	repo.On("GetByIdempotencyKey", mock.Anything, "billing-res-1").Return(existing, nil)

	uc := NewUsecase(repo)
	req := &model.StartBillingRequest{
		ReservationID:  "res-1",
		BookingFee:     pricing.BookingFee,
		IdempotencyKey: "billing-res-1",
	}

	result, err := uc.StartBilling(t.Context(), req)

	require.NoError(t, err)
	assert.Equal(t, "existing-billing-id", result.ID)
	repo.AssertNotCalled(t, "CreateBillingRecord")
	repo.AssertExpectations(t)
}

func TestCalculateFee_ShouldComputeCorrectFees_WhenStandardSession(t *testing.T) {
	repo := new(MockRepository)

	existingRecord := &model.BillingRecord{
		ID:            "billing-1",
		ReservationID: "res-1",
		BookingFee:    pricing.BookingFee,
		Status:        string(billingconstants.BillingStatusPending),
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

	result, err := uc.CalculateFee(t.Context(), req)

	require.NoError(t, err)
	assert.Equal(t, int64(10_000), result.ParkingFee) // 2h × 5000
	assert.Equal(t, int64(0), result.OvernightFee)    // same day
	assert.Equal(t, 120, result.DurationMinutes)      // 2 hours
	assert.Equal(t, 2, result.BilledHours)
	assert.False(t, result.IsOvernight)
	assert.Equal(t, string(billingconstants.BillingStatusCalculated), result.Status)
	expectedTotal := pricing.BookingFee + int64(10_000)
	assert.Equal(t, expectedTotal, result.TotalAmount)
	repo.AssertExpectations(t)
}

func TestCalculateFee_ShouldApplyOvernightFee_WhenSessionCrossesMidnight(t *testing.T) {
	repo := new(MockRepository)

	existingRecord := &model.BillingRecord{
		ID:            "billing-2",
		ReservationID: "res-2",
		BookingFee:    pricing.BookingFee,
		Status:        string(billingconstants.BillingStatusPending),
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

	result, err := uc.CalculateFee(t.Context(), req)

	require.NoError(t, err)
	assert.Equal(t, int64(40_000), result.ParkingFee)               // 8h × 5000
	assert.Equal(t, pricing.OvernightPerNight, result.OvernightFee) // 20,000
	assert.Equal(t, 480, result.DurationMinutes)                    // 8 hours
	assert.Equal(t, 8, result.BilledHours)
	assert.True(t, result.IsOvernight)
	assert.Equal(t, int64(65_000), result.TotalAmount)
	repo.AssertExpectations(t)
}

func TestGenerateInvoice_ShouldReturnExisting_WhenAlreadyInvoiced(t *testing.T) {
	repo := new(MockRepository)

	existing := &model.BillingRecord{
		ID:             "billing-3",
		ReservationID:  "res-3",
		BookingFee:     pricing.BookingFee,
		ParkingFee:     10_000,
		TotalAmount:    15_000,
		IdempotencyKey: "invoice-res-3",
		Status:         string(billingconstants.BillingStatusInvoiced),
	}
	repo.On("GetByReservationID", mock.Anything, "res-3").Return(existing, nil)

	uc := NewUsecase(repo)
	req := &model.GenerateInvoiceRequest{
		ReservationID:  "res-3",
		IdempotencyKey: "invoice-res-3",
	}

	result, err := uc.GenerateInvoice(t.Context(), req)

	require.NoError(t, err)
	assert.Equal(t, "billing-3", result.ID)
	assert.Equal(t, string(billingconstants.BillingStatusInvoiced), result.Status)
	repo.AssertNotCalled(t, "UpdateBillingRecord")
	repo.AssertExpectations(t)
}

func TestGenerateInvoice_ShouldUpdateStatus_WhenNewInvoice(t *testing.T) {
	repo := new(MockRepository)

	existingRecord := &model.BillingRecord{
		ID:             "billing-4",
		ReservationID:  "res-4",
		BookingFee:     pricing.BookingFee,
		ParkingFee:     10_000,
		TotalAmount:    15_000,
		IdempotencyKey: "billing-res-4",
		Status:         string(billingconstants.BillingStatusCalculated),
	}
	repo.On("GetByReservationID", mock.Anything, "res-4").Return(existingRecord, nil)
	repo.On("UpdateBillingRecord", mock.Anything, mock.AnythingOfType("*model.BillingRecord")).Return(nil)

	uc := NewUsecase(repo)
	req := &model.GenerateInvoiceRequest{
		ReservationID:  "res-4",
		IdempotencyKey: "invoice-res-4",
	}

	result, err := uc.GenerateInvoice(t.Context(), req)

	require.NoError(t, err)
	assert.Equal(t, string(billingconstants.BillingStatusInvoiced), result.Status)
	repo.AssertExpectations(t)
}

// that total_amount always equals the sum of all fee fields.
func TestBillingTotalInvariant_ShouldEqualSumOfFees(t *testing.T) {
	record := &model.BillingRecord{
		BookingFee:   pricing.BookingFee,
		ParkingFee:   15_000,
		OvernightFee: pricing.OvernightPerNight,
	}

	total := pricing.CalculateTotal(record.BookingFee, record.ParkingFee, record.OvernightFee)

	expectedTotal := pricing.BookingFee + int64(15_000) + pricing.OvernightPerNight
	assert.Equal(t, expectedTotal, total)
	assert.Equal(t, record.BookingFee+record.ParkingFee+record.OvernightFee, total)
	assert.GreaterOrEqual(t, total, pricing.BookingFee)
}
