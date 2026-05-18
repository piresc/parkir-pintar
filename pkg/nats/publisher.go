package nats

import (
	"context"
	"errors"
	"fmt"
)

// ErrNilClient is returned when Publish is called on a Publisher with a nil client.
var ErrNilClient = errors.New("publisher client is nil")

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
	if p.client == nil {
		return ErrNilClient
	}
	_, err := p.client.Publish(ctx, subject, data, msgID)
	if err != nil {
		return fmt.Errorf("publisher: %w", err)
	}
	return nil
}
