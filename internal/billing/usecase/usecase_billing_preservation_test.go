// Package usecase provides preservation property tests for non-ApplyPenalty billing operations.
//
// Best practices applied (from Go testify coding standards KB):
// - Test naming: Test[FunctionName]_Should[ExpectedResult]_When[Condition]
// - AAA pattern: Arrange → Act → Assert
// - testify/mock for mock implementations of all dependency interfaces
// - testify/assert and testify/require for assertions
// - Each test is isolated with its own mock setup
// - AssertExpectations(t) called on all mocks to verify interactions
// - Mock at interface boundaries rather than concrete implementations
//
// **Validates: Requirements 3.14** (Preservation Property 14 from design)
//
// Non-bug condition: operation != ApplyPenalty
// These tests verify that StartBilling, CalculateFee, GenerateInvoice, and
// ApplyOvernightFee function identically on unfixed code. They must PASS.
package usecase

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"parkir-pintar/internal/billing/model"
	"parkir-pintar/internal/billing/repository"

	"pgregory.net/rapid"
)

// TestStartBilling_ShouldCreateRecord_WhenNewKey verifies that StartBilling
// creates a billing record with correct booking_fee and status "pending".
// Non-bug condition: operation != ApplyPenalty.
//
// **Validates: Requirements 3.14**
func TestStartBilling_ShouldCreateRecord_WhenNewKey(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Arrange
		bookingFee := rapid.Int64Range(1_000, 50_000).Draw(t, "bookingFee")
		idempotencyKey := rapid.StringMatching(`[a-z0-9]{16}`).Draw(t, "idempotencyKey")

		repo := new(MockRepository)
		natsClient := new(MockNATSClient)

		repo.On("GetByIdempotencyKey", mock.Anything, idempotencyKey).Return(nil, repository.ErrNotFound)
		repo.On("CreateBillingRecord", mock.Anything, mock.AnythingOfType("*model.BillingRecord")).Return(nil)

		uc := NewUsecase(repo, natsClient)

		// Act
		result, err := uc.StartBilling(t.Context(), &model.StartBillingRequest{
			ReservationID:  "res-pres-billing",
			BookingFee:     bookingFee,
			IdempotencyKey: idempotencyKey,
		})

		// Assert
		require.NoError(t, err)
		assert.Equal(t, bookingFee, result.BookingFee)
		assert.Equal(t, model.BillingStatusPending, result.Status)
		assert.Equal(t, bookingFee, result.TotalAmount, "total should equal booking_fee for new record")
		assert.NotEmpty(t, result.ID)
		repo.AssertExpectations(t)
	})
}

// TestCalculateFee_ShouldComputeCorrectly_WhenStandardSession verifies that
// CalculateFee computes correct parking_fee based on duration.
// Non-bug condition: operation != ApplyPenalty.
//
// **Validates: Requirements 3.14**
func TestCalculateFee_ShouldComputeCorrectly_WhenStandardSession(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Arrange — generate random session duration (1-12 hours)
		hours := rapid.IntRange(1, 12).Draw(t, "hours")
		bookingFee := rapid.Int64Range(1_000, 10_000).Draw(t, "bookingFee")

		loc := time.FixedZone("WIB", 7*60*60)
		checkIn := time.Date(2026, 4, 24, 8, 0, 0, 0, loc)
		checkOut := checkIn.Add(time.Duration(hours) * time.Hour)

		existingRecord := &model.BillingRecord{
			ID:            "billing-calc",
			ReservationID: "res-calc",
			BookingFee:    bookingFee,
			Status:        model.BillingStatusPending,
		}

		repo := new(MockRepository)
		natsClient := new(MockNATSClient)

		repo.On("GetByReservationID", mock.Anything, "res-calc").Return(existingRecord, nil)
		repo.On("UpdateBillingRecord", mock.Anything, mock.AnythingOfType("*model.BillingRecord")).Return(nil)
		natsClient.On("Publish", mock.Anything, mock.Anything).Return(nil)

		uc := NewUsecase(repo, natsClient)

		// Act
		result, err := uc.CalculateFee(t.Context(), &model.CalculateFeeRequest{
			ReservationID: "res-calc",
			CheckInAt:     checkIn,
			CheckOutAt:    checkOut,
		})

		// Assert
		require.NoError(t, err)
		expectedParkingFee := int64(hours) * model.HourlyRate
		assert.Equal(t, expectedParkingFee, result.ParkingFee,
			"parking_fee should be %d hours × %d = %d", hours, model.HourlyRate, expectedParkingFee)
		assert.Equal(t, model.BillingStatusCalculated, result.Status)
		assert.Equal(t, hours*60, result.DurationMinutes)
		repo.AssertExpectations(t)
	})
}

// TestGenerateInvoice_ShouldUpdateStatus_WhenNewInvoicePreservation verifies that
// GenerateInvoice updates the billing record status to "invoiced".
// Non-bug condition: operation != ApplyPenalty.
//
// **Validates: Requirements 3.14**
func TestGenerateInvoice_ShouldUpdateStatus_WhenNewInvoicePreservation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Arrange
		idempotencyKey := rapid.StringMatching(`[a-z0-9]{16}`).Draw(t, "idempotencyKey")

		repo := new(MockRepository)
		natsClient := new(MockNATSClient)

		repo.On("GetByIdempotencyKey", mock.Anything, idempotencyKey).Return(nil, repository.ErrNotFound)
		existingRecord := &model.BillingRecord{
			ID:            "billing-inv",
			ReservationID: "res-inv",
			BookingFee:    model.BookingFee,
			ParkingFee:    10_000,
			TotalAmount:   15_000,
			Status:        model.BillingStatusCalculated,
		}
		repo.On("GetByReservationID", mock.Anything, "res-inv").Return(existingRecord, nil)
		repo.On("UpdateBillingRecord", mock.Anything, mock.AnythingOfType("*model.BillingRecord")).Return(nil)
		natsClient.On("Publish", mock.Anything, mock.Anything).Return(nil)

		uc := NewUsecase(repo, natsClient)

		// Act
		result, err := uc.GenerateInvoice(t.Context(), &model.GenerateInvoiceRequest{
			ReservationID:  "res-inv",
			IdempotencyKey: idempotencyKey,
		})

		// Assert
		require.NoError(t, err)
		assert.Equal(t, model.BillingStatusInvoiced, result.Status)
		repo.AssertExpectations(t)
	})
}

// TestApplyOvernightFee_ShouldSetOvernightFields_WhenCalled verifies that
// ApplyOvernightFee sets overnight_fee and is_overnight correctly.
// Non-bug condition: operation != ApplyPenalty.
//
// **Validates: Requirements 3.14**
func TestApplyOvernightFee_ShouldSetOvernightFields_WhenCalled(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Arrange
		bookingFee := rapid.Int64Range(1_000, 10_000).Draw(t, "bookingFee")
		parkingFee := rapid.Int64Range(5_000, 50_000).Draw(t, "parkingFee")

		existingRecord := &model.BillingRecord{
			ID:            "billing-overnight",
			ReservationID: "res-overnight",
			BookingFee:    bookingFee,
			ParkingFee:    parkingFee,
			TotalAmount:   bookingFee + parkingFee,
			Status:        model.BillingStatusCalculated,
		}

		repo := new(MockRepository)
		natsClient := new(MockNATSClient)

		repo.On("GetByReservationID", mock.Anything, "res-overnight").Return(existingRecord, nil)
		repo.On("UpdateBillingRecord", mock.Anything, mock.AnythingOfType("*model.BillingRecord")).Return(nil)

		uc := NewUsecase(repo, natsClient)

		// Act
		result, err := uc.ApplyOvernightFee(t.Context(), &model.ApplyOvernightFeeRequest{
			ReservationID: "res-overnight",
		})

		// Assert
		require.NoError(t, err)
		assert.Equal(t, model.OvernightFlatFee, result.OvernightFee)
		assert.True(t, result.IsOvernight)
		expectedTotal := bookingFee + parkingFee + model.OvernightFlatFee
		assert.Equal(t, expectedTotal, result.TotalAmount,
			"total should include overnight fee")
		repo.AssertExpectations(t)
	})
}
