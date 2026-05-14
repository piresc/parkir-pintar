package nats

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Client wraps a NATS connection and JetStream context for publishing and consuming.
type Client struct {
	conn *nats.Conn
	js   jetstream.JetStream

	mu        sync.RWMutex
	streams   map[string]jetstream.Stream
	consumers map[string]jetstream.Consumer
}

// NewClient connects to NATS and initializes a JetStream client.
func NewClient(url string, opts ...nats.Option) (*Client, error) {
	nc, err := nats.Connect(url, opts...)
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("jetstream new: %w", err)
	}

	slog.Info("nats connected", "url", url)

	return &Client{
		conn:      nc,
		js:        js,
		streams:   make(map[string]jetstream.Stream),
		consumers: make(map[string]jetstream.Consumer),
	}, nil
}

// JetStream returns the underlying JetStream interface.
func (c *Client) JetStream() jetstream.JetStream {
	return c.js
}

// Conn returns the underlying NATS connection.
func (c *Client) Conn() *nats.Conn {
	return c.conn
}

// CreateStream creates or updates a stream with the given config.
func (c *Client) CreateStream(ctx context.Context, cfg jetstream.StreamConfig) (jetstream.Stream, error) {
	stream, err := c.js.CreateOrUpdateStream(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create stream %s: %w", cfg.Name, err)
	}

	c.mu.Lock()
	c.streams[cfg.Name] = stream
	c.mu.Unlock()

	slog.Info("stream created", "name", cfg.Name, "subjects", cfg.Subjects)
	return stream, nil
}

// CreateConsumer creates or updates a durable consumer on the given stream.
func (c *Client) CreateConsumer(ctx context.Context, streamName string, cfg jetstream.ConsumerConfig) (jetstream.Consumer, error) {
	c.mu.RLock()
	stream, ok := c.streams[streamName]
	c.mu.RUnlock()

	if !ok {
		// Try to get the stream from the server
		var err error
		stream, err = c.js.Stream(ctx, streamName)
		if err != nil {
			return nil, fmt.Errorf("get stream %s: %w", streamName, err)
		}
		c.mu.Lock()
		c.streams[streamName] = stream
		c.mu.Unlock()
	}

	consumer, err := stream.CreateOrUpdateConsumer(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create consumer %s on stream %s: %w", cfg.Name, streamName, err)
	}

	c.mu.Lock()
	c.consumers[cfg.Name] = consumer
	c.mu.Unlock()

	slog.Info("consumer created", "name", cfg.Name, "stream", streamName)
	return consumer, nil
}

// Publish publishes a message to a JetStream subject with deduplication via msgID.
func (c *Client) Publish(ctx context.Context, subject string, data []byte, msgID string) (*jetstream.PubAck, error) {
	opts := []jetstream.PublishOpt{}
	if msgID != "" {
		opts = append(opts, jetstream.WithMsgID(msgID))
	}

	ack, err := c.js.Publish(ctx, subject, data, opts...)
	if err != nil {
		return nil, fmt.Errorf("publish to %s: %w", subject, err)
	}

	return ack, nil
}

// Consume starts consuming messages from a named consumer with the given handler.
// It returns a ConsumeContext that can be used to stop consumption.
func (c *Client) Consume(consumerName string, handler func(jetstream.Msg)) (jetstream.ConsumeContext, error) {
	c.mu.RLock()
	consumer, ok := c.consumers[consumerName]
	c.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("consumer %s not found, create it first", consumerName)
	}

	cc, err := consumer.Consume(handler)
	if err != nil {
		return nil, fmt.Errorf("consume %s: %w", consumerName, err)
	}

	slog.Info("consuming started", "consumer", consumerName)
	return cc, nil
}

// Close drains the connection and closes it gracefully.
func (c *Client) Close(ctx context.Context) {
	if c.conn != nil {
		_ = c.conn.FlushWithContext(ctx)
		c.conn.Close()
		slog.Info("nats connection closed")
	}
}
