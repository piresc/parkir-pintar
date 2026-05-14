// Package usecase implements the business logic layer for the billing domain.
package usecase

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"parkir-pintar/internal/billing/model"
	"parkir-pintar/internal/billing/repository"
	"parkir-pintar/pkg/pricing"
)

// TestGenerateInvoice_ShouldCreateDifferentRecords_WhenDifferentIdempotencyKeys
// verifies PRD §17.1: "different keys create different records" for billing.
func TestGenerateInvoice_ShouldCreateDifferentRecords_WhenDifferentIdempotencyKeys(t *testing.T) {
	// Arrange — first invoice
	repo := new(MockRepository)

	repo.On("GetByIdempotencyKey", mock.Anything, "invoice-key-1").Return(nil, repository.ErrNotFound).Once()
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
		return r.Status == model.BillingStatusInvoiced
	})).Return(nil).Once()

	uc := NewUsecase(repo)
	inv1, err := uc.GenerateInvoice(t.Context(), &model.GenerateInvoiceRequest{
		ReservationID:  "res-1",
		IdempotencyKey: "invoice-key-1",
	})
	require.NoError(t, err)
	require.NotNil(t, inv1)

	// Arrange — second invoice for different reservation
	repo.On("GetByIdempotencyKey", mock.Anything, "invoice-key-2").Return(nil, repository.ErrNotFound).Once()
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
		return r.Status == model.BillingStatusInvoiced
	})).Return(nil).Once()

	inv2, err := uc.GenerateInvoice(t.Context(), &model.GenerateInvoiceRequest{
		ReservationID:  "res-2",
		IdempotencyKey: "invoice-key-2",
	})
	require.NoError(t, err)
	require.NotNil(t, inv2)

	// Assert: different keys → different invoice records
	assert.NotEqual(t, inv1.ID, inv2.ID, "different idempotency keys must produce different billing records")
	assert.NotEqual(t, inv1.ReservationID, inv2.ReservationID)

	// Assert: UpdateBillingRecord was called twice (once per key)
	repo.AssertNumberOfCalls(t, "UpdateBillingRecord", 2)

	repo.AssertExpectations(t)
}
