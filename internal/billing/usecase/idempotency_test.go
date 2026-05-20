package usecase

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"parkir-pintar/internal/billing/model"
	"parkir-pintar/pkg/pricing"
)

func TestGenerateInvoice_ShouldCreateDifferentRecords_WhenDifferentReservations(t *testing.T) {
	repo := new(MockRepository)

	record1 := &model.BillingRecord{
		ID:             "billing-1",
		ReservationID:  "res-1",
		BookingFee:     pricing.BookingFee,
		ParkingFee:     10000,
		TotalAmount:    15000,
		IdempotencyKey: "billing-res-1",
		Status:         model.BillingStatusCalculated,
	}
	repo.On("GetByReservationID", mock.Anything, "res-1").Return(record1, nil).Once()
	repo.On("UpdateBillingRecord", mock.Anything, mock.MatchedBy(func(r *model.BillingRecord) bool {
		return r.Status == model.BillingStatusInvoiced && r.ReservationID == "res-1"
	})).Return(nil).Once()

	uc := NewUsecase(repo)
	inv1, err := uc.GenerateInvoice(t.Context(), &model.GenerateInvoiceRequest{
		ReservationID:  "res-1",
		IdempotencyKey: "invoice-key-1",
	})
	require.NoError(t, err)
	require.NotNil(t, inv1)

	record2 := &model.BillingRecord{
		ID:             "billing-2",
		ReservationID:  "res-2",
		BookingFee:     pricing.BookingFee,
		ParkingFee:     20000,
		TotalAmount:    25000,
		IdempotencyKey: "billing-res-2",
		Status:         model.BillingStatusCalculated,
	}
	repo.On("GetByReservationID", mock.Anything, "res-2").Return(record2, nil).Once()
	repo.On("UpdateBillingRecord", mock.Anything, mock.MatchedBy(func(r *model.BillingRecord) bool {
		return r.Status == model.BillingStatusInvoiced && r.ReservationID == "res-2"
	})).Return(nil).Once()

	inv2, err := uc.GenerateInvoice(t.Context(), &model.GenerateInvoiceRequest{
		ReservationID:  "res-2",
		IdempotencyKey: "invoice-key-2",
	})
	require.NoError(t, err)
	require.NotNil(t, inv2)

	assert.NotEqual(t, inv1.ID, inv2.ID, "different reservations must produce different billing records")
	assert.NotEqual(t, inv1.ReservationID, inv2.ReservationID)

	repo.AssertNumberOfCalls(t, "UpdateBillingRecord", 2)

	repo.AssertExpectations(t)
}
