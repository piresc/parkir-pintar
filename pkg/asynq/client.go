package asynq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
)

type Client struct {
	client    *asynq.Client
	inspector *asynq.Inspector
}

func NewClient(redisAddr, redisPassword string) *Client {
	redisOpt := asynq.RedisClientOpt{Addr: redisAddr, Password: redisPassword}
	return &Client{
		client:    asynq.NewClient(redisOpt),
		inspector: asynq.NewInspector(redisOpt),
	}
}

func (c *Client) Close() error {
	return c.client.Close()
}

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

func (c *Client) EnqueuePaymentHoldTimeout(ctx context.Context, reservationID string, paymentID string, delay time.Duration) (string, error) {
	payload, err := json.Marshal(PaymentHoldTimeoutPayload{
		ReservationID: reservationID,
		PaymentID:     paymentID,
	})
	if err != nil {
		return "", fmt.Errorf("marshal payment hold timeout payload: %w", err)
	}

	taskID := fmt.Sprintf("payment-hold:%s", reservationID)
	task := asynq.NewTask(TypePaymentHoldTimeout, payload)
	info, err := c.client.EnqueueContext(ctx, task, asynq.ProcessIn(delay), asynq.TaskID(taskID))
	if err != nil {
		return "", fmt.Errorf("enqueue payment hold timeout task: %w", err)
	}
	return info.ID, nil
}

func (c *Client) CancelTask(ctx context.Context, taskID string) error {
	if err := c.inspector.DeleteTask("default", taskID); err != nil {
		return fmt.Errorf("cancel task %s: %w", taskID, err)
	}
	return nil
}
