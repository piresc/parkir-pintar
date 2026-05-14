// Package e2e_test provides Layer 1 E2E integration tests for ParkirPintar.
//
// This file defines thin adapter types that bridge the real usecase
// implementations to the interfaces expected by the reservation usecase,
// plus Redis adapters for each domain.
//
// Best practices applied (from Go coding standards):
// - Use keyed fields in struct literals to prevent breakages during refactors
// - Use context.Context as first parameter for consistency
// - Handle errors explicitly; never ignore errors
// - Keep interfaces small and define them where they're used
package e2e_test

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	billingmodel "parkir-pintar/internal/billing/model"
	billinguc "parkir-pintar/internal/billing/usecase"
	paymentmodel "parkir-pintar/internal/payment/model"
	paymentuc "parkir-pintar/internal/payment/usecase"
	reservationuc "parkir-pintar/internal/reservation/usecase"
)

// ---------------------------------------------------------------------------
// billingAdapter — adapts billing.Usecase → reservation.BillingClient
// ---------------------------------------------------------------------------

// billingAdapter wraps a real billing.Usecase to satisfy the
// reservation.BillingClient interface so the reservation usecase can call
// billing operations through the full usecase chain.
type billingAdapter struct {
	uc billinguc.Usecase
}

// StartBilling creates a billing record with the booking fee.
func (a *billingAdapter) StartBilling(ctx context.Context, reservationID string, bookingFee int64, idempotencyKey string) (*billingmodel.BillingRecord, error) {
	return a.uc.StartBilling(ctx, &billingmodel.StartBillingRequest{
		ReservationID:  reservationID,
		BookingFee:     bookingFee,
		IdempotencyKey: idempotencyKey,
	})
}

// CalculateFee computes the parking fee for a completed session.
func (a *billingAdapter) CalculateFee(ctx context.Context, reservationID string, checkInAt, checkOutAt time.Time) (*billingmodel.BillingRecord, error) {
	return a.uc.CalculateFee(ctx, &billingmodel.CalculateFeeRequest{
		ReservationID: reservationID,
		CheckInAt:     checkInAt,
		CheckOutAt:    checkOutAt,
	})
}

// GenerateInvoice finalises the billing record into an invoice.
func (a *billingAdapter) GenerateInvoice(ctx context.Context, reservationID string, idempotencyKey string) (*billingmodel.BillingRecord, error) {
	return a.uc.GenerateInvoice(ctx, &billingmodel.GenerateInvoiceRequest{
		ReservationID:  reservationID,
		IdempotencyKey: idempotencyKey,
	})
}

// ---------------------------------------------------------------------------
// paymentAdapter — adapts payment.Usecase → reservation.PaymentClient
// ---------------------------------------------------------------------------

// paymentAdapter wraps a real payment.Usecase to satisfy the
// reservation.PaymentClient interface.
type paymentAdapter struct {
	uc paymentuc.Usecase
}

// ProcessPayment processes a payment for the given billing record.
func (a *paymentAdapter) ProcessPayment(ctx context.Context, billingID string, amount int64, paymentMethod string, idempotencyKey string) (string, error) {
	payment, err := a.uc.ProcessPayment(ctx, &paymentmodel.ProcessPaymentRequest{
		BillingID:      billingID,
		Amount:         amount,
		PaymentMethod:  paymentMethod,
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		return "", err
	}
	return payment.ID, nil
}

// ---------------------------------------------------------------------------
// reservationRedisAdapter — wraps *redis.Client → reservation.RedisClient
// ---------------------------------------------------------------------------

// reservationLockerAdapter satisfies the reservation.Locker interface
// using a real go-redis client via the redislock package.
type reservationLockerAdapter struct {
	client *redis.Client
}

// Acquire attempts to acquire a distributed lock using SETNX.
func (a *reservationLockerAdapter) Acquire(ctx context.Context, key string) (reservationuc.Lock, error) {
	acquired, err := a.client.SetNX(ctx, key, "locked", 12*time.Minute).Result()
	if err != nil {
		return nil, err
	}
	if !acquired {
		return nil, fmt.Errorf("lock not acquired: %s", key)
	}
	return &redisLock{client: a.client, key: key}, nil
}

type redisLock struct {
	client *redis.Client
	key    string
}

func (l *redisLock) Release(ctx context.Context) error {
	return l.client.Del(ctx, l.key).Err()
}

// ---------------------------------------------------------------------------
// searchRedisAdapter — wraps *redis.Client → search.RedisClient
// ---------------------------------------------------------------------------

// searchRedisAdapter satisfies the search.RedisClient interface
// (Get + Set + Delete) using a real go-redis/v8 client.
type searchRedisAdapter struct {
	client *redis.Client
}

// Get retrieves a value by key from Redis.
func (a *searchRedisAdapter) Get(ctx context.Context, key string) (string, error) {
	return a.client.Get(ctx, key).Result()
}

// Set stores a key-value pair in Redis with an expiration.
func (a *searchRedisAdapter) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return a.client.Set(ctx, key, value, expiration).Err()
}

// Delete removes a key from Redis.
func (a *searchRedisAdapter) Delete(ctx context.Context, key string) error {
	return a.client.Del(ctx, key).Err()
}
