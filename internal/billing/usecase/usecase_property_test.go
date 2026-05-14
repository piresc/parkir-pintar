// Package usecase provides bug condition exploration tests for billing penalty atomicity.
//
// Best practices applied (from Go testify coding standards KB):
// - Test naming: Test[FunctionName]_Should[ExpectedResult]_When[Condition]
// - AAA pattern: Arrange → Act → Assert
// - testify/mock for mock implementations of all dependency interfaces
// - testify/assert for assertions
// - Each test is isolated with its own mock setup
// - Mock at interface boundaries rather than concrete implementations
//
// **Validates: Requirements 2.5** (Property 3 from design)
//
// Bug Condition: concurrentRequests > 1 AND sameReservationID
// Expected: penalty_amount == a₁ + a₂ + ... + aₙ
// Counterexample on unfixed code: final amount < expected sum (lost update)
//
// CRITICAL: This test is expected to FAIL on unfixed code.
// DO NOT fix the code or the test when it fails.
package usecase

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"parkir-pintar/internal/billing/model"
	"parkir-pintar/pkg/pricing"
)

// concurrentMockRepository is a thread-safe mock that simulates a real DB
// with in-memory state. After the fix, ApplyPenalty uses AddPenaltyAmount
// which atomically increments penalty_amount, preventing lost updates.
type concurrentMockRepository struct {
	mock.Mock
	mu            sync.Mutex
	billingRecord *model.BillingRecord
	penaltyCount  atomic.Int64
}

func (m *concurrentMockRepository) CreateBillingRecord(ctx context.Context, record *model.BillingRecord) error {
	return nil
}

func (m *concurrentMockRepository) GetByReservationID(ctx context.Context, reservationID string) (*model.BillingRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Return a copy to simulate DB read (each caller gets the current snapshot)
	copy := *m.billingRecord
	return &copy, nil
}

func (m *concurrentMockRepository) GetByIdempotencyKey(ctx context.Context, key string) (*model.BillingRecord, error) {
	return nil, nil
}

func (m *concurrentMockRepository) UpdateBillingRecord(ctx context.Context, record *model.BillingRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Simulate DB write — store the record as-is
	m.billingRecord.PenaltyAmount = record.PenaltyAmount
	m.billingRecord.TotalAmount = record.TotalAmount
	m.billingRecord.UpdatedAt = record.UpdatedAt
	return nil
}

func (m *concurrentMockRepository) CreatePenalty(ctx context.Context, penalty *model.Penalty) error {
	m.penaltyCount.Add(1)
	return nil
}

// AddPenaltyAmount simulates an atomic SQL UPDATE that increments penalty_amount
// and recalculates total_amount in a single operation, protected by a mutex.
func (m *concurrentMockRepository) AddPenaltyAmount(ctx context.Context, reservationID string, amount int64) (*model.BillingRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.billingRecord.PenaltyAmount += amount
	m.billingRecord.TotalAmount = m.billingRecord.BookingFee + m.billingRecord.ParkingFee +
		m.billingRecord.OvernightFee + m.billingRecord.CancellationFee + m.billingRecord.PenaltyAmount
	copy := *m.billingRecord
	return &copy, nil
}

// TestApplyPenalty_ShouldSumAllAmounts_WhenCalledConcurrently launches 5
// concurrent ApplyPenalty calls for the same reservation and asserts the final
// penalty_amount equals the sum of all amounts. On unfixed code the
// read-modify-write race causes lost updates, so the final amount will be less
// than expected.
//
// **Validates: Requirements 2.5**
func TestApplyPenalty_ShouldSumAllAmounts_WhenCalledConcurrently(t *testing.T) {
	// Arrange
	baseRecord := &model.BillingRecord{
		ID:            "billing-concurrent",
		ReservationID: "res-concurrent",
		BookingFee:    pricing.BookingFee,
		ParkingFee:    10_000,
		TotalAmount:   15_000,
		Status:        model.BillingStatusCalculated,
	}

	repo := &concurrentMockRepository{
		billingRecord: baseRecord,
	}


	uc := NewUsecase(repo)

	concurrentRequests := 5
	penaltyAmount := int64(10_000)
	expectedTotal := penaltyAmount * int64(concurrentRequests)

	// Act — launch concurrent ApplyPenalty calls
	var wg sync.WaitGroup
	wg.Add(concurrentRequests)
	for range concurrentRequests {
		go func() {
			defer wg.Done()
			_, _ = uc.ApplyPenalty(context.Background(), &model.ApplyPenaltyRequest{
				ReservationID: "res-concurrent",
				PenaltyType:   "wrong_spot",
				Amount:        penaltyAmount,
				Description:   "concurrent penalty test",
			})
		}()
	}
	wg.Wait()

	// Assert — final penalty_amount should equal sum of all concurrent amounts
	finalPenalty := repo.billingRecord.PenaltyAmount
	assert.Equal(t, expectedTotal, finalPenalty,
		"penalty_amount should be %d (sum of %d × %d), got %d — lost update detected",
		expectedTotal, concurrentRequests, penaltyAmount, finalPenalty)
}
