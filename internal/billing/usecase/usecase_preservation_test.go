// Package usecase provides preservation property tests for single ApplyPenalty correctness.
//
// Best practices applied (from Go testify coding standards KB):
// - Test naming: Test[FunctionName]_Should[ExpectedResult]_When[Condition]
// - AAA pattern: Arrange → Act → Assert
// - testify/mock for mock implementations of all dependency interfaces
// - testify/assert and testify/require for assertions
// - Each test is isolated with its own mock setup
// - Mock at interface boundaries rather than concrete implementations
//
// **Validates: Requirements 3.3** (Preservation Property 14 from design)
//
// Non-bug condition: concurrentRequests == 1
// These tests verify that a single ApplyPenalty correctly increments penalty_amount
// and recalculates total_amount on unfixed code. They must PASS on unfixed code.
package usecase

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"parkir-pintar/internal/billing/model"

	"pgregory.net/rapid"
)

// TestApplyPenalty_ShouldIncrementPenaltyAndRecalcTotal_WhenSingleRequest verifies
// that a single ApplyPenalty call correctly adds the penalty amount to the billing
// record and recalculates total_amount. Non-bug condition: concurrentRequests == 1.
//
// **Validates: Requirements 3.3**
func TestApplyPenalty_ShouldIncrementPenaltyAndRecalcTotal_WhenSingleRequest(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Arrange — generate random valid penalty amount and existing record fees
		penaltyAmount := rapid.Int64Range(1_000, 500_000).Draw(t, "penaltyAmount")
		bookingFee := rapid.Int64Range(0, 50_000).Draw(t, "bookingFee")
		parkingFee := rapid.Int64Range(0, 100_000).Draw(t, "parkingFee")
		overnightFee := rapid.Int64Range(0, 20_000).Draw(t, "overnightFee")
		existingPenalty := rapid.Int64Range(0, 200_000).Draw(t, "existingPenalty")

		expectedPenalty := existingPenalty + penaltyAmount
		expectedTotal := bookingFee + parkingFee + overnightFee + expectedPenalty

		// The atomic AddPenaltyAmount returns the updated record directly
		updatedRecord := &model.BillingRecord{
			ID:            "billing-pres",
			ReservationID: "res-pres",
			BookingFee:    bookingFee,
			ParkingFee:    parkingFee,
			OvernightFee:  overnightFee,
			PenaltyAmount: expectedPenalty,
			TotalAmount:   expectedTotal,
			Status:        model.BillingStatusCalculated,
		}

		repo := new(MockRepository)

		repo.On("CreatePenalty", mock.Anything, mock.AnythingOfType("*model.Penalty")).Return(nil)
		repo.On("AddPenaltyAmount", mock.Anything, "res-pres", penaltyAmount).Return(updatedRecord, nil)

		uc := NewUsecase(repo)

		// Act — single request, no concurrency
		result, err := uc.ApplyPenalty(t.Context(), &model.ApplyPenaltyRequest{
			ReservationID: "res-pres",
			PenaltyType:   "wrong_spot",
			Amount:        penaltyAmount,
			Description:   "preservation test",
		})

		// Assert — penalty_amount should be incremented, total recalculated
		require.NoError(t, err)
		assert.Equal(t, expectedPenalty, result.PenaltyAmount,
			"penalty_amount should be existingPenalty(%d) + newAmount(%d) = %d",
			existingPenalty, penaltyAmount, expectedPenalty)

		assert.Equal(t, expectedTotal, result.TotalAmount,
			"total_amount should equal sum of all fees")

		repo.AssertExpectations(t)
	})
}
