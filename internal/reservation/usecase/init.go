package usecase

import (
	"context"
	"time"

	"parkir-pintar/internal/reservation"
	"parkir-pintar/internal/reservation/constants"
	"parkir-pintar/internal/reservation/gateway"
	"parkir-pintar/internal/reservation/repository"
	"parkir-pintar/pkg/redislock"
)

type lockerAdapter struct {
	inner *redislock.Locker
}

func (a *lockerAdapter) Acquire(ctx context.Context, key string) (Lock, error) {
	l, err := a.inner.Acquire(ctx, key)
	if err != nil {
		return nil, err
	}
	return l, nil
}

// NewLockerAdapter wraps a *redislock.Locker to satisfy the Locker interface.
func NewLockerAdapter(l *redislock.Locker) Locker {
	return &lockerAdapter{inner: l}
}

// reservationUsecase is the concrete implementation of Usecase.
type reservationUsecase struct {
	repo           repository.Repository
	locker         Locker
	billingClient  BillingClient
	paymentClient  PaymentClient
	presenceClient PresenceClient
	taskEnqueuer   TaskEnqueuer
	eventPublisher gateway.EventPublisher
	expiryTimeout  time.Duration
	paymentTimeout time.Duration
}

// NewUsecase creates a new reservation Usecase with all required dependencies.
// taskEnqueuer, eventPublisher, and presenceClient are optional (nil-safe); when nil,
// no async tasks are enqueued, no events are published, and presence verification
// is skipped respectively.
func NewUsecase(
	repo repository.Repository,
	locker Locker,
	billingClient BillingClient,
	paymentClient PaymentClient,
	presenceClient PresenceClient,
	taskEnqueuer TaskEnqueuer,
	eventPublisher gateway.EventPublisher,
	expiryTimeoutMinutes int,
	paymentTimeoutMinutes int,
) reservation.Usecase {
	timeout := time.Duration(expiryTimeoutMinutes) * time.Minute
	if timeout <= 0 {
		timeout = constants.DefaultExpiryTimeout
	}
	paymentTimeout := time.Duration(paymentTimeoutMinutes) * time.Minute
	if paymentTimeout <= 0 {
		paymentTimeout = constants.DefaultPaymentTimeout
	}
	return &reservationUsecase{
		repo:           repo,
		locker:         locker,
		billingClient:  billingClient,
		paymentClient:  paymentClient,
		presenceClient: presenceClient,
		taskEnqueuer:   taskEnqueuer,
		eventPublisher: eventPublisher,
		expiryTimeout:  timeout,
		paymentTimeout: paymentTimeout,
	}
}
