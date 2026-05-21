package reservation

import (
	"context"
	"time"
)

// BillingRecord is a local representation of billing data returned by the
// billing service. It decouples the reservation domain from the billing
// service's internal model package.
type BillingRecord struct {
	ID              string
	ReservationID   string
	BookingFee      int64
	ParkingFee      int64
	OvernightFee    int64
	TotalAmount     int64
	DurationMinutes int
	BilledHours     int
	IsOvernight     bool
	IdempotencyKey  string
	Status          string
}

// BillingClient defines the interface for billing service operations.
//
//go:generate mockgen -destination=mocks/mock_billing_client.go -package=mocks parkir-pintar/internal/reservation BillingClient
type BillingClient interface {
	StartBilling(ctx context.Context, reservationID string, bookingFee int64, idempotencyKey string) (*BillingRecord, error)
	CalculateFee(ctx context.Context, reservationID string, checkInAt, checkOutAt time.Time) (*BillingRecord, error)
	GenerateInvoice(ctx context.Context, reservationID string, idempotencyKey string) (*BillingRecord, error)
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

// Lock represents an acquired distributed lock.
type Lock interface {
	Release(ctx context.Context) error
}

// Locker manages distributed lock acquisition.
type Locker interface {
	Acquire(ctx context.Context, key string) (Lock, error)
}
