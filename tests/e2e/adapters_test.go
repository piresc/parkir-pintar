// Package e2e_test provides Layer 1 E2E integration tests for ParkirPintar.
//
// This file defines thin adapter types that bridge the real usecase
// implementations to the interfaces expected by the reservation usecase,
// plus Redis adapters for each domain and a stub NATS client.
//
// Best practices applied (from Go coding standards):
// - Use keyed fields in struct literals to prevent breakages during refactors
// - Use context.Context as first parameter for consistency
// - Handle errors explicitly; never ignore errors
// - Keep interfaces small and define them where they're used
package e2e_test

import (
	"context"
	"log/slog"
	"time"

	"github.com/go-redis/redis/v8"

	billingmodel "parkir-pintar/internal/billing/model"
	billinguc "parkir-pintar/internal/billing/usecase"
	paymentmodel "parkir-pintar/internal/payment/model"
	paymentuc "parkir-pintar/internal/payment/usecase"
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
func (a *billingAdapter) StartBilling(ctx context.Context, reservationID string, bookingFee int64, idempotencyKey string) error {
	_, err := a.uc.StartBilling(ctx, &billingmodel.StartBillingRequest{
		ReservationID:  reservationID,
		BookingFee:     bookingFee,
		IdempotencyKey: idempotencyKey,
	})
	return err
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

// ApplyPenalty records a penalty against a reservation's billing record.
func (a *billingAdapter) ApplyPenalty(ctx context.Context, reservationID string, penaltyType string, amount int64, description string) error {
	_, err := a.uc.ApplyPenalty(ctx, &billingmodel.ApplyPenaltyRequest{
		ReservationID: reservationID,
		PenaltyType:   penaltyType,
		Amount:        amount,
		Description:   description,
	})
	return err
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
func (a *paymentAdapter) ProcessPayment(ctx context.Context, billingID string, amount int64, paymentMethod string, idempotencyKey string) error {
	_, err := a.uc.ProcessPayment(ctx, &paymentmodel.ProcessPaymentRequest{
		BillingID:      billingID,
		Amount:         amount,
		PaymentMethod:  paymentMethod,
		IdempotencyKey: idempotencyKey,
	})
	return err
}

// ---------------------------------------------------------------------------
// reservationRedisAdapter — wraps *redis.Client → reservation.RedisClient
// ---------------------------------------------------------------------------

// reservationRedisAdapter satisfies the reservation.RedisClient interface
// (SetNX + Delete) using a real go-redis/v8 client.
type reservationRedisAdapter struct {
	client *redis.Client
}

// SetNX sets a key only if it does not already exist (distributed lock).
func (a *reservationRedisAdapter) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	return a.client.SetNX(ctx, key, value, expiration).Result()
}

// Delete removes a key from Redis.
func (a *reservationRedisAdapter) Delete(ctx context.Context, key string) error {
	return a.client.Del(ctx, key).Err()
}

// ---------------------------------------------------------------------------
// presenceRedisAdapter — wraps *redis.Client → presence.RedisClient
// ---------------------------------------------------------------------------

// presenceRedisAdapter satisfies the presence.RedisClient interface
// (XAdd + Delete) using a real go-redis/v8 client.
type presenceRedisAdapter struct {
	client *redis.Client
}

// XAdd appends an entry to a Redis stream.
func (a *presenceRedisAdapter) XAdd(ctx context.Context, stream string, values map[string]interface{}) error {
	return a.client.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: values,
	}).Err()
}

// Delete removes a key from Redis.
func (a *presenceRedisAdapter) Delete(ctx context.Context, key string) error {
	return a.client.Del(ctx, key).Err()
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

// ---------------------------------------------------------------------------
// stubNATSClient — satisfies all domain NATSClient interfaces
// ---------------------------------------------------------------------------

// stubNATSClient is a no-op NATS client that logs publishes without failing.
// It satisfies the NATSClient interface used by reservation, billing, payment,
// and presence usecases.
type stubNATSClient struct{}

// Publish logs the subject but always returns nil.
func (s *stubNATSClient) Publish(subject string, data []byte) error {
	slog.Debug("stub NATS publish", "subject", subject, "bytes", len(data))
	return nil
}
