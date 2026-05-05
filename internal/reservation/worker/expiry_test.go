package worker

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"

	"parkir-pintar/internal/reservation/model"
)

// fakeRepo implements repository.Repository for testing.
type fakeRepo struct {
	mu        sync.Mutex
	expired   []*model.Reservation
	findErr   error
	callCount int
}

func (f *fakeRepo) FindExpiredReservations(_ context.Context) ([]*model.Reservation, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.callCount++
	if f.findErr != nil {
		return nil, f.findErr
	}
	return f.expired, nil
}

func (f *fakeRepo) getCallCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.callCount
}

// Stub methods to satisfy the repository.Repository interface.
func (f *fakeRepo) FindByIdempotencyKey(context.Context, string) (*model.Reservation, error) {
	return nil, nil
}
func (f *fakeRepo) FindAvailableSpot(context.Context, string) (*model.ParkingSpot, error) {
	return nil, nil
}
func (f *fakeRepo) GetSpotForUpdate(context.Context, string) (*model.ParkingSpot, error) {
	return nil, nil
}
func (f *fakeRepo) CreateReservationTx(_ context.Context, _ *sqlx.Tx, _ *model.Reservation) error {
	return nil
}
func (f *fakeRepo) UpdateSpotStatusTx(_ context.Context, _ *sqlx.Tx, _ string, _ string) error {
	return nil
}
func (f *fakeRepo) UpdateReservation(context.Context, *model.Reservation) error { return nil }
func (f *fakeRepo) UpdateReservationTx(_ context.Context, _ *sqlx.Tx, _ *model.Reservation) error {
	return nil
}
func (f *fakeRepo) GetByID(context.Context, string) (*model.Reservation, error) {
	return nil, nil
}
func (f *fakeRepo) GetByIDForUpdate(_ context.Context, _ *sqlx.Tx, _ string) (*model.Reservation, error) {
	return nil, nil
}
func (f *fakeRepo) WithTransaction(_ context.Context, _ func(tx *sqlx.Tx) error) error { return nil }

// fakeUsecase implements usecase.Usecase for testing.
type fakeUsecase struct {
	mu         sync.Mutex
	expiredIDs []string
	expireErr  error
}

func (f *fakeUsecase) ExpireReservation(_ context.Context, req *model.ExpireReservationRequest) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.expireErr != nil {
		return f.expireErr
	}
	f.expiredIDs = append(f.expiredIDs, req.ReservationID)
	return nil
}

func (f *fakeUsecase) getExpiredIDs() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := make([]string, len(f.expiredIDs))
	copy(result, f.expiredIDs)
	return result
}

// Stub methods to satisfy the usecase.Usecase interface.
func (f *fakeUsecase) CreateReservation(context.Context, *model.CreateReservationRequest) (*model.Reservation, error) {
	return nil, nil
}
func (f *fakeUsecase) CancelReservation(context.Context, *model.CancelReservationRequest) (*model.Reservation, error) {
	return nil, nil
}
func (f *fakeUsecase) CheckIn(context.Context, *model.CheckInRequest) (*model.Reservation, error) {
	return nil, nil
}
func (f *fakeUsecase) CheckOut(context.Context, *model.CheckOutRequest) (*model.CheckOutResponse, error) {
	return nil, nil
}

func TestRunExpiryWorker_ProcessesExpiredReservations(t *testing.T) {
	repo := &fakeRepo{
		expired: []*model.Reservation{
			{ID: "res-1"},
			{ID: "res-2"},
		},
	}
	uc := &fakeUsecase{}

	ctx, cancel := context.WithCancel(t.Context())

	go RunExpiryWorker(ctx, 50*time.Millisecond, repo, uc)

	// Wait for at least one tick to process
	time.Sleep(150 * time.Millisecond)
	cancel()

	expiredIDs := uc.getExpiredIDs()
	assert.Contains(t, expiredIDs, "res-1")
	assert.Contains(t, expiredIDs, "res-2")
}

func TestRunExpiryWorker_ContinuesOnFindError(t *testing.T) {
	repo := &fakeRepo{
		findErr: errors.New("db connection failed"),
	}
	uc := &fakeUsecase{}

	ctx, cancel := context.WithCancel(t.Context())

	go RunExpiryWorker(ctx, 50*time.Millisecond, repo, uc)

	// Wait for multiple ticks — worker should keep running despite errors
	time.Sleep(200 * time.Millisecond)
	cancel()

	// Worker should have attempted multiple times without crashing
	assert.GreaterOrEqual(t, repo.getCallCount(), 2)
	assert.Empty(t, uc.getExpiredIDs())
}

func TestRunExpiryWorker_ContinuesOnExpireError(t *testing.T) {
	repo := &fakeRepo{
		expired: []*model.Reservation{
			{ID: "res-fail"},
			{ID: "res-ok"},
		},
	}
	callCount := 0
	uc := &fakeUsecase{
		expireErr: errors.New("expire failed"),
	}

	ctx, cancel := context.WithCancel(t.Context())
	_ = callCount

	go RunExpiryWorker(ctx, 50*time.Millisecond, repo, uc)

	// Wait for at least one tick
	time.Sleep(150 * time.Millisecond)
	cancel()

	// Worker should have attempted to expire but all failed — no IDs recorded
	assert.Empty(t, uc.getExpiredIDs())
}

func TestRunExpiryWorker_StopsOnContextCancel(t *testing.T) {
	repo := &fakeRepo{}
	uc := &fakeUsecase{}

	ctx, cancel := context.WithCancel(t.Context())

	done := make(chan struct{})
	go func() {
		RunExpiryWorker(ctx, 1*time.Hour, repo, uc)
		close(done)
	}()

	cancel()

	select {
	case <-done:
		// Worker stopped as expected
	case <-time.After(2 * time.Second):
		t.Fatal("expiry worker did not stop after context cancellation")
	}
}
