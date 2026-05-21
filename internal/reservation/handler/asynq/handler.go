package asynq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"

	pkgasynq "parkir-pintar/pkg/asynq"
)

// Task type constants for the reservation domain.
const (
	TypeReservationExpire  = "task:reservation:expire"
	TypePaymentHoldTimeout = "task:payment:hold_timeout"
)

// ReservationExpiryPayload is the payload for reservation expiry tasks.
type ReservationExpiryPayload struct {
	ReservationID string `json:"reservation_id"`
}

// PaymentHoldTimeoutPayload is the payload for payment hold timeout tasks.
type PaymentHoldTimeoutPayload struct {
	ReservationID string `json:"reservation_id"`
	PaymentID     string `json:"payment_id"`
}

// Expirer is the interface that the reservation usecase must satisfy.
type Expirer interface {
	ExpireReservation(ctx context.Context, reservationID string) error
}

// Failer is the interface that the reservation usecase must satisfy.
type Failer interface {
	FailReservation(ctx context.Context, reservationID string, paymentID string) error
}

// ExpiryHandler processes reservation expiry tasks.
type ExpiryHandler struct {
	expirer Expirer
}

func NewExpiryHandler(expirer Expirer) *ExpiryHandler {
	return &ExpiryHandler{expirer: expirer}
}

func (h *ExpiryHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var payload ReservationExpiryPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal reservation expiry payload: %w: %w", err, asynq.SkipRetry)
	}
	if payload.ReservationID == "" {
		return fmt.Errorf("reservation_id is required: %w", asynq.SkipRetry)
	}
	return h.expirer.ExpireReservation(ctx, payload.ReservationID)
}

// PaymentTimeoutHandler processes payment hold timeout tasks.
type PaymentTimeoutHandler struct {
	failer Failer
}

func NewPaymentTimeoutHandler(failer Failer) *PaymentTimeoutHandler {
	return &PaymentTimeoutHandler{failer: failer}
}

func (h *PaymentTimeoutHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var payload PaymentHoldTimeoutPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal payment hold timeout payload: %w: %w", err, asynq.SkipRetry)
	}
	if payload.ReservationID == "" {
		return fmt.Errorf("reservation_id is required: %w", asynq.SkipRetry)
	}
	return h.failer.FailReservation(ctx, payload.ReservationID, payload.PaymentID)
}

// TaskEnqueuer wraps the generic asynq client with domain-specific enqueue methods.
type TaskEnqueuer struct {
	client *pkgasynq.Client
}

func NewTaskEnqueuer(client *pkgasynq.Client) *TaskEnqueuer {
	return &TaskEnqueuer{client: client}
}

func (e *TaskEnqueuer) EnqueueReservationExpiry(ctx context.Context, reservationID string, delay time.Duration) (string, error) {
	payload, err := json.Marshal(ReservationExpiryPayload{ReservationID: reservationID})
	if err != nil {
		return "", fmt.Errorf("marshal reservation expiry payload: %w", err)
	}
	return e.client.Enqueue(ctx, TypeReservationExpire, payload, delay)
}

func (e *TaskEnqueuer) EnqueuePaymentHoldTimeout(ctx context.Context, reservationID string, paymentID string, delay time.Duration) (string, error) {
	payload, err := json.Marshal(PaymentHoldTimeoutPayload{ReservationID: reservationID, PaymentID: paymentID})
	if err != nil {
		return "", fmt.Errorf("marshal payment hold timeout payload: %w", err)
	}
	taskID := fmt.Sprintf("payment-hold:%s", reservationID)
	return e.client.Enqueue(ctx, TypePaymentHoldTimeout, payload, delay, asynq.TaskID(taskID))
}

func (e *TaskEnqueuer) CancelTask(ctx context.Context, taskID string) error {
	return e.client.CancelTask(ctx, taskID)
}
