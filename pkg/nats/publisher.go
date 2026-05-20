package nats

import (
	"context"
	"errors"
	"fmt"
)

var ErrNilClient = errors.New("publisher client is nil")

type Publisher struct {
	client *Client
}

func NewPublisher(client *Client) *Publisher {
	return &Publisher{client: client}
}

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
