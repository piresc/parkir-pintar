package asynq

import (
	"context"
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

// Enqueue enqueues a task with the given type, payload, and delay.
func (c *Client) Enqueue(ctx context.Context, taskType string, payload []byte, delay time.Duration, opts ...asynq.Option) (string, error) {
	allOpts := make([]asynq.Option, 0, 1+len(opts))
	allOpts = append(allOpts, asynq.ProcessIn(delay))
	allOpts = append(allOpts, opts...)
	task := asynq.NewTask(taskType, payload)
	info, err := c.client.EnqueueContext(ctx, task, allOpts...)
	if err != nil {
		return "", fmt.Errorf("enqueue task %s: %w", taskType, err)
	}
	return info.ID, nil
}

// CancelTask removes a pending task by ID.
func (c *Client) CancelTask(_ context.Context, taskID string) error {
	if err := c.inspector.DeleteTask("default", taskID); err != nil {
		return fmt.Errorf("cancel task %s: %w", taskID, err)
	}
	return nil
}
