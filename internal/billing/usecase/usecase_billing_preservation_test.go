// ApplyOvernightFee function identically on unfixed code. They must PASS.
package usecase

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"parkir-pintar/internal/billing/model"
	billingconstants "parkir-pintar/internal/billing/constants"
	"parkir-pintar/internal/billing/repository"
	"parkir-pintar/pkg/pricing"

	"pgregory.net/rapid"
)

func TestStartBilling_ShouldCreateRecord_WhenNewKey(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		bookingFee := rapid.Int64Range(1_000, 50_000).Draw(t, "bookingFee")
		idempotencyKey := rapid.StringMatching(`[a-z0-9]{16}`).Draw(t, "idempotencyKey")

		repo := new(MockRepository)

		repo.On("GetByIdempotencyKey", mock.Anything, idempotencyKey).Return(nil, repository.ErrNotFound)
		repo.On("GetByReservationID", mock.Anything, "res-pres-billing").Return(nil, repository.ErrNotFound)
		repo.On("CreateBillingRecord", mock.Anything, mock.AnythingOfType("*model.BillingRecord")).Return(nil)

		uc := NewUsecase(repo)

		result, err := uc.StartBilling(t.Context(), &model.StartBillingRequest{
			ReservationID:  "res-pres-billing",
			BookingFee:     bookingFee,
			IdempotencyKey: idempotencyKey,
		})

		require.NoError(t, err)
		assert.Equal(t, bookingFee, result.BookingFee)
		assert.Equal(t, string(billingconstants.BillingStatusPending), result.Status)
		assert.Equal(t, bookingFee, result.TotalAmount, "total should equal booking_fee for new record")
		assert.NotEmpty(t, result.ID)
		repo.AssertExpectations(t)
	})
}

func TestCalculateFee_ShouldComputeCorrectly_WhenStandardSession(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		hours := rapid.IntRange(1, 12).Draw(t, "hours")
		bookingFee := rapid.Int64Range(1_000, 10_000).Draw(t, "bookingFee")

		loc := time.FixedZone("WIB", 7*60*60)
		checkIn := time.Date(2026, 4, 24, 8, 0, 0, 0, loc)
		checkOut := checkIn.Add(time.Duration(hours) * time.Hour)

		existingRecord := &model.BillingRecord{
			ID:            "billing-calc",
			ReservationID: "res-calc",
			BookingFee:    bookingFee,
			Status:        string(billingconstants.BillingStatusPending),
		}

		repo := new(MockRepository)

		repo.On("GetByReservationID", mock.Anything, "res-calc").Return(existingRecord, nil)
		repo.On("UpdateBillingRecord", mock.Anything, mock.AnythingOfType("*model.BillingRecord")).Return(nil)

		uc := NewUsecase(repo)

		result, err := uc.CalculateFee(t.Context(), &model.CalculateFeeRequest{
			ReservationID: "res-calc",
			CheckInAt:     checkIn,
			CheckOutAt:    checkOut,
		})

		require.NoError(t, err)
		expectedParkingFee := int64(hours) * pricing.HourlyRate
		assert.Equal(t, expectedParkingFee, result.ParkingFee,
			"parking_fee should be %d hours × %d = %d", hours, pricing.HourlyRate, expectedParkingFee)
		assert.Equal(t, string(billingconstants.BillingStatusCalculated), result.Status)
		assert.Equal(t, hours*60, result.DurationMinutes)
		repo.AssertExpectations(t)
	})
}

func TestGenerateInvoice_ShouldUpdateStatus_WhenNewInvoicePreservation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		_ = rapid.StringMatching(`[a-z0-9]{16}`).Draw(t, "idempotencyKey")

		repo := new(MockRepository)

		existingRecord := &model.BillingRecord{
			ID:            "billing-inv",
			ReservationID: "res-inv",
			BookingFee:    pricing.BookingFee,
			ParkingFee:    10_000,
			TotalAmount:   15_000,
			Status:        string(billingconstants.BillingStatusCalculated),
		}
		repo.On("GetByReservationID", mock.Anything, "res-inv").Return(existingRecord, nil)
		repo.On("UpdateBillingRecord", mock.Anything, mock.AnythingOfType("*model.BillingRecord")).Return(nil)

		uc := NewUsecase(repo)

		result, err := uc.GenerateInvoice(t.Context(), &model.GenerateInvoiceRequest{
			ReservationID:  "res-inv",
			IdempotencyKey: "any-key",
		})

		require.NoError(t, err)
		assert.Equal(t, string(billingconstants.BillingStatusInvoiced), result.Status)
		repo.AssertExpectations(t)
	})
}

func TestApplyOvernightFee_ShouldSetOvernightFields_WhenCalled(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		bookingFee := rapid.Int64Range(1_000, 10_000).Draw(t, "bookingFee")
		parkingFee := rapid.Int64Range(5_000, 50_000).Draw(t, "parkingFee")

		existingRecord := &model.BillingRecord{
			ID:            "billing-overnight",
			ReservationID: "res-overnight",
			BookingFee:    bookingFee,
			ParkingFee:    parkingFee,
			TotalAmount:   bookingFee + parkingFee,
			Status:        string(billingconstants.BillingStatusCalculated),
		}

		repo := new(MockRepository)

		repo.On("GetByReservationID", mock.Anything, "res-overnight").Return(existingRecord, nil)
		repo.On("UpdateBillingRecord", mock.Anything, mock.AnythingOfType("*model.BillingRecord")).Return(nil)

		uc := NewUsecase(repo)

		result, err := uc.ApplyOvernightFee(t.Context(), &model.ApplyOvernightFeeRequest{
			ReservationID: "res-overnight",
		})

		require.NoError(t, err)
		assert.Equal(t, pricing.OvernightPerNight, result.OvernightFee)
		assert.True(t, result.IsOvernight)
		expectedTotal := bookingFee + parkingFee + pricing.OvernightPerNight
		assert.Equal(t, expectedTotal, result.TotalAmount,
			"total should include overnight fee")
		repo.AssertExpectations(t)
	})
}
