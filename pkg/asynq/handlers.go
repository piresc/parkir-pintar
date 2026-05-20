package asynq

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
)

// ReservationExpirer is the interface that the reservation usecase must satisfy
// for handling reservation expiry.
type ReservationExpirer interface {
	ExpireReservation(ctx context.Context, reservationID string) error
}

// ReservationFailer is the interface that the reservation usecase must satisfy
// for handling payment hold timeouts.
type ReservationFailer interface {
	FailReservation(ctx context.Context, reservationID string, paymentID string) error
}

// ReservationExpiryHandler processes reservation expiry tasks.
type ReservationExpiryHandler struct {
	expirer ReservationExpirer
}

// NewReservationExpiryHandler creates a handler that calls ExpireReservation on timeout.
func NewReservationExpiryHandler(expirer ReservationExpirer) *ReservationExpiryHandler {
	return &ReservationExpiryHandler{expirer: expirer}
}

// ProcessTask implements asynq.Handler.
func (h *ReservationExpiryHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var payload ReservationExpiryPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal reservation expiry payload: %w: %w", err, asynq.SkipRetry)
	}
	if payload.ReservationID == "" {
		return fmt.Errorf("reservation_id is required: %w", asynq.SkipRetry)
	}
	return h.expirer.ExpireReservation(ctx, payload.ReservationID)
}

// PaymentHoldTimeoutHandler processes payment hold timeout tasks.
type PaymentHoldTimeoutHandler struct {
	failer ReservationFailer
}

// NewPaymentHoldTimeoutHandler creates a handler that calls FailReservation on payment timeout.
func NewPaymentHoldTimeoutHandler(failer ReservationFailer) *PaymentHoldTimeoutHandler {
	return &PaymentHoldTimeoutHandler{failer: failer}
}

// ProcessTask implements asynq.Handler.
func (h *PaymentHoldTimeoutHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var payload PaymentHoldTimeoutPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal payment hold timeout payload: %w: %w", err, asynq.SkipRetry)
	}
	if payload.ReservationID == "" {
		return fmt.Errorf("reservation_id is required: %w", asynq.SkipRetry)
	}
	return h.failer.FailReservation(ctx, payload.ReservationID, payload.PaymentID)
}
