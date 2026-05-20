package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/mock"

	billingmodel "parkir-pintar/internal/billing/model"
	"parkir-pintar/internal/reservation/constants"
	"parkir-pintar/internal/reservation/model"
)

// BenchmarkCreateReservation measures the hot path for creating a reservation
// (idempotency check → lock → find spot → transaction → publish event).
func BenchmarkCreateReservation(b *testing.B) {
	repo := new(MockRepository)
	billing := new(MockBillingClient)
	payment := new(MockPaymentClient)
	locker := new(MockLocker)
	mockLock := new(MockLock)

	spot := &model.ParkingSpot{
		ID:          "spot-1",
		FloorNumber: 1,
		SpotNumber:  1,
		VehicleType: "car",
		SpotCode:    "F1-C-001",
		Status:      "available",
	}

	billingRecord := &billingmodel.BillingRecord{
		ID:             "billing-1",
		ReservationID:  "res-1",
		BookingFee:     5000,
		TotalAmount:    5000,
		Status:         "pending",
		IdempotencyKey: "idem-billing-1",
	}

	repo.On("FindByIdempotencyKey", mock.Anything, mock.Anything).Return(nil, nil)
	repo.On("ListByDriverID", mock.Anything, mock.Anything, mock.Anything).Return([]*model.Reservation{}, nil)
	locker.On("Acquire", mock.Anything, mock.Anything).Return(mockLock, nil)
	mockLock.On("Release", mock.Anything).Return(nil)
	repo.On("FindAvailableSpot", mock.Anything, "car").Return(spot, nil)
	repo.On("WithTransaction", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		fn := args.Get(1).(func(tx *sqlx.Tx) error)
		_ = fn(nil)
	}).Return(nil)
	repo.On("CreateReservationTx", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	repo.On("UpdateSpotStatusTx", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	billing.On("StartBilling", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(billingRecord, nil)

	uc := NewUsecase(repo, locker, billing, payment, nil, nil, nil, 60, 10)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		req := &model.CreateReservationRequest{
			DriverID:       "driver-bench",
			VehicleType:    "car",
			IdempotencyKey: "bench-key-" + string(rune(i%1000)),
		}
		_, _ = uc.CreateReservation(ctx, req)
	}
}

// BenchmarkCancelReservation measures the cancellation path performance.
func BenchmarkCancelReservation(b *testing.B) {
	repo := new(MockRepository)
	billing := new(MockBillingClient)
	payment := new(MockPaymentClient)
	locker := new(MockLocker)
	mockLock := new(MockLock)

	reservation := &model.Reservation{
		ID:          "res-cancel-1",
		DriverID:    "driver-1",
		SpotID:      "spot-1",
		VehicleType: "car",
		Status:      constants.StatusConfirmed,
		ExpiresAt:   timePtr(time.Now().Add(10 * time.Minute)),
		CreatedAt:   time.Now().Add(-5 * time.Minute),
	}

	locker.On("Acquire", mock.Anything, mock.Anything).Return(mockLock, nil)
	mockLock.On("Release", mock.Anything).Return(nil)
	repo.On("WithTransaction", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		fn := args.Get(1).(func(tx *sqlx.Tx) error)
		_ = fn(nil)
	}).Return(nil)
	repo.On("GetByIDForUpdate", mock.Anything, mock.Anything, "res-cancel-1").Return(reservation, nil)
	repo.On("UpdateReservationTx", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	repo.On("UpdateSpotStatusTx", mock.Anything, mock.Anything, "spot-1", "available").Return(nil)
	repo.On("GetSpotForUpdate", mock.Anything, "spot-1").Return(&model.ParkingSpot{
		ID: "spot-1", FloorNumber: 1, SpotNumber: 1, VehicleType: "car", SpotCode: "F1-C-001", Status: "reserved",
	}, nil)

	uc := NewUsecase(repo, locker, billing, payment, nil, nil, nil, 60, 10)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = uc.CancelReservation(ctx, &model.CancelReservationRequest{ReservationID: "res-cancel-1"})
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}
