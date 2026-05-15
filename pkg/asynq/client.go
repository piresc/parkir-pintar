package asynq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
)

// Client wraps the asynq client for enqueuing tasks.
type Client struct {
	client    *asynq.Client
	inspector *asynq.Inspector
}

// NewClient creates a new Asynq client connected to the given Redis address.
func NewClient(redisAddr, redisPassword string) *Client {
	redisOpt := asynq.RedisClientOpt{Addr: redisAddr, Password: redisPassword}
	return &Client{
		client:    asynq.NewClient(redisOpt),
		inspector: asynq.NewInspector(redisOpt),
	}
}

// Close closes the underlying asynq client.
func (c *Client) Close() error {
	return c.client.Close()
}

// EnqueueReservationExpiry schedules a reservation expiry task after the given delay.
func (c *Client) EnqueueReservationExpiry(ctx context.Context, reservationID string, delay time.Duration) (string, error) {
	payload, err := json.Marshal(ReservationExpiryPayload{
		ReservationID: reservationID,
	})
	if err != nil {
		return "", fmt.Errorf("marshal reservation expiry payload: %w", err)
	}

	task := asynq.NewTask(TypeReservationExpire, payload)
	info, err := c.client.EnqueueContext(ctx, task, asynq.ProcessIn(delay))
	if err != nil {
		return "", fmt.Errorf("enqueue reservation expiry task: %w", err)
	}
	return info.ID, nil
}

// EnqueuePaymentHoldTimeout schedules a payment hold timeout task after the given delay.
func (c *Client) EnqueuePaymentHoldTimeout(ctx context.Context, reservationID string, paymentID string, delay time.Duration) (string, error) {
	payload, err := json.Marshal(PaymentHoldTimeoutPayload{
		ReservationID: reservationID,
		PaymentID:     paymentID,
	})
	if err != nil {
		return "", fmt.Errorf("marshal payment hold timeout payload: %w", err)
	}

	task := asynq.NewTask(TypePaymentHoldTimeout, payload)
	info, err := c.client.EnqueueContext(ctx, task, asynq.ProcessIn(delay))
	if err != nil {
		return "", fmt.Errorf("enqueue payment hold timeout task: %w", err)
	}
	return info.ID, nil
}

// CancelTask cancels a scheduled task by its ID.
func (c *Client) CancelTask(ctx context.Context, taskID string) error {
	if err := c.inspector.DeleteTask("default", taskID); err != nil {
		return fmt.Errorf("cancel task %s: %w", taskID, err)
	}
	return nil
}
