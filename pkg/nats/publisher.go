package nats

import (
	"context"
	"fmt"
)

// Publisher wraps a Client to provide a simple publish interface.
type Publisher struct {
	client *Client
}

// NewPublisher creates a new Publisher backed by the given Client.
func NewPublisher(client *Client) *Publisher {
	return &Publisher{client: client}
}

// Publish sends a message to the given subject with deduplication via msgID.
func (p *Publisher) Publish(ctx context.Context, subject string, data []byte, msgID string) error {
	_, err := p.client.Publish(ctx, subject, data, msgID)
	if err != nil {
		return fmt.Errorf("publisher: %w", err)
	}
	return nil
}
