package reservation

import (
	"context"
	"time"

	billingmodel "parkir-pintar/internal/billing/model"
	"parkir-pintar/pkg/events"
)

// BillingClient defines the interface for billing service operations.
//
//go:generate mockgen -destination=mocks/mock_billing_client.go -package=mocks parkir-pintar/internal/reservation BillingClient
type BillingClient interface {
	StartBilling(ctx context.Context, reservationID string, bookingFee int64, idempotencyKey string) (*billingmodel.BillingRecord, error)
	CalculateFee(ctx context.Context, reservationID string, checkInAt, checkOutAt time.Time) (*billingmodel.BillingRecord, error)
	GenerateInvoice(ctx context.Context, reservationID string, idempotencyKey string) (*billingmodel.BillingRecord, error)
}

// PaymentClient defines the interface for payment service operations.
//
//go:generate mockgen -destination=mocks/mock_payment_client.go -package=mocks parkir-pintar/internal/reservation PaymentClient
type PaymentClient interface {
	ProcessPayment(ctx context.Context, billingID string, amount int64, paymentMethod string, idempotencyKey string) (string, error)
	RefundPayment(ctx context.Context, paymentID string) error
}

// PresenceClient defines the interface for presence verification operations.
// This is optional — if nil, presence verification is skipped (graceful degradation).
//
//go:generate mockgen -destination=mocks/mock_presence_client.go -package=mocks parkir-pintar/internal/reservation PresenceClient
type PresenceClient interface {
	VerifyPresence(ctx context.Context, driverID string, reservationID string, floorNumber int, spotNumber int) (*PresenceResult, error)
}

// PresenceResult holds the result of a presence verification check.
type PresenceResult struct {
	Verified bool
	Message  string
}

// TaskEnqueuer enqueues asynchronous tasks (e.g. reservation expiry).
// Implementations must be safe to call concurrently.
//
//go:generate mockgen -destination=mocks/mock_task_enqueuer.go -package=mocks parkir-pintar/internal/reservation TaskEnqueuer
type TaskEnqueuer interface {
	EnqueueReservationExpiry(ctx context.Context, reservationID string, delay time.Duration) (string, error)
	EnqueuePaymentHoldTimeout(ctx context.Context, reservationID string, paymentID string, delay time.Duration) (string, error)
	CancelTask(ctx context.Context, taskID string) error
}

// EventPublisher defines the interface for publishing domain events.
//
//go:generate mockgen -destination=mocks/mock_event_publisher.go -package=mocks parkir-pintar/internal/reservation EventPublisher
type EventPublisher interface {
	PublishSpotUpdated(ctx context.Context, event events.SpotUpdatedEvent) error
	PublishReservationEvent(ctx context.Context, subject string, event events.ReservationEvent) error
}

// Lock represents an acquired distributed lock.
type Lock interface {
	Release(ctx context.Context) error
}

// Locker manages distributed lock acquisition.
type Locker interface {
	Acquire(ctx context.Context, key string) (Lock, error)
}
