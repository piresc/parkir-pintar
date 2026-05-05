// Package nats provides a NATS JetStream client for inter-service messaging
// with stream/consumer management, auto-reconnect, and a traced wrapper.
package nats

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// StreamConfig defines configuration for a JetStream stream.
type StreamConfig struct {
	Name      string
	Subjects  []string
	Retention jetstream.RetentionPolicy
	Storage   jetstream.StorageType
	Replicas  int
	MaxAge    time.Duration
	MaxBytes  int64
	MaxMsgs   int64
	Discard   jetstream.DiscardPolicy
}

// ConsumerConfig defines configuration for a JetStream consumer.
type ConsumerConfig struct {
	StreamName    string
	ConsumerName  string
	FilterSubject string
	DeliverPolicy jetstream.DeliverPolicy
	AckPolicy     jetstream.AckPolicy
	AckWait       time.Duration
	MaxDeliver    int
	ReplayPolicy  jetstream.ReplayPolicy
	MaxAckPending int
}

// Client represents a JetStream-enabled NATS client.
type Client struct {
	conn       *nats.Conn
	js         jetstream.JetStream
	ctx        context.Context
	mu         sync.RWMutex
	streams    map[string]jetstream.Stream
	consumers  map[string]jetstream.Consumer
	cancelFunc context.CancelFunc
}

// NewClient creates a new JetStream-enabled NATS client with auto-reconnect.
// MaxReconnects(-1) for unlimited retries, ReconnectWait(2s), ReconnectBufSize(5MB).
func NewClient(url string) (*Client, error) {
	opts := []nats.Option{
		nats.ReconnectWait(2 * time.Second),
		nats.MaxReconnects(-1),
		nats.ReconnectBufSize(5 * 1024 * 1024), // 5MB buffer
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			if err != nil {
				slog.Error("NATS disconnected", slog.String("error", err.Error()))
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			slog.Info("NATS reconnected", slog.String("url", nc.ConnectedUrl()))
		}),
		nats.ClosedHandler(func(_ *nats.Conn) {
			slog.Info("NATS connection closed")
		}),
	}

	conn, err := nats.Connect(url, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS server: %w", err)
	}

	js, err := jetstream.New(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Client{
		conn:       conn,
		js:         js,
		ctx:        ctx,
		streams:    make(map[string]jetstream.Stream),
		consumers:  make(map[string]jetstream.Consumer),
		cancelFunc: cancel,
	}, nil
}

// CreateOrUpdateStream creates or updates a JetStream stream.
func (c *Client) CreateOrUpdateStream(cfg StreamConfig) error {
	streamCfg := jetstream.StreamConfig{
		Name:       cfg.Name,
		Subjects:   cfg.Subjects,
		Retention:  cfg.Retention,
		Storage:    cfg.Storage,
		Replicas:   cfg.Replicas,
		MaxAge:     cfg.MaxAge,
		MaxBytes:   cfg.MaxBytes,
		MaxMsgs:    cfg.MaxMsgs,
		Discard:    cfg.Discard,
		NoAck:      false,
		Duplicates: 5 * time.Minute,
	}

	stream, err := c.js.CreateOrUpdateStream(c.ctx, streamCfg)
	if err != nil {
		return fmt.Errorf("failed to create/update stream %s: %w", cfg.Name, err)
	}

	c.mu.Lock()
	c.streams[cfg.Name] = stream
	c.mu.Unlock()
	return nil
}

// CreateConsumer creates a durable consumer for a stream.
func (c *Client) CreateConsumer(cfg ConsumerConfig) error {
	c.mu.RLock()
	stream, exists := c.streams[cfg.StreamName]
	c.mu.RUnlock()
	if !exists {
		return fmt.Errorf("stream %s not found", cfg.StreamName)
	}

	consumerCfg := jetstream.ConsumerConfig{
		Name:          cfg.ConsumerName,
		DeliverPolicy: cfg.DeliverPolicy,
		AckPolicy:     cfg.AckPolicy,
		AckWait:       cfg.AckWait,
		MaxDeliver:    cfg.MaxDeliver,
		FilterSubject: cfg.FilterSubject,
		ReplayPolicy:  cfg.ReplayPolicy,
		MaxAckPending: cfg.MaxAckPending,
	}

	consumer, err := stream.CreateOrUpdateConsumer(c.ctx, consumerCfg)
	if err != nil {
		return fmt.Errorf("failed to create consumer %s: %w", cfg.ConsumerName, err)
	}

	consumerKey := fmt.Sprintf("%s:%s", cfg.StreamName, cfg.ConsumerName)
	c.mu.Lock()
	c.consumers[consumerKey] = consumer
	c.mu.Unlock()
	return nil
}

// Publish publishes a message to JetStream with delivery guarantees (ack).
func (c *Client) Publish(subject string, data []byte) error {
	ctx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
	defer cancel()

	_, err := c.js.Publish(ctx, subject, data)
	if err != nil {
		return fmt.Errorf("failed to publish message to %s: %w", subject, err)
	}
	return nil
}

// ConsumeMessages consumes messages from a JetStream consumer with ack/nak.
func (c *Client) ConsumeMessages(streamName, consumerName string, handler func(jetstream.Msg) error) error {
	consumerKey := fmt.Sprintf("%s:%s", streamName, consumerName)
	c.mu.RLock()
	consumer, exists := c.consumers[consumerKey]
	c.mu.RUnlock()
	if !exists {
		return fmt.Errorf("consumer %s not found", consumerKey)
	}

	consumeCtx, err := consumer.Consume(func(msg jetstream.Msg) {
		if err := handler(msg); err != nil {
			slog.Error("error processing message",
				slog.String("consumer", consumerKey),
				slog.String("subject", msg.Subject()),
				slog.String("error", err.Error()))
			if nakErr := msg.Nak(); nakErr != nil {
				slog.Error("failed to NAK message", slog.String("error", nakErr.Error()))
			}
			return
		}

		if ackErr := msg.Ack(); ackErr != nil {
			slog.Error("failed to ACK message",
				slog.String("consumer", consumerKey),
				slog.String("error", ackErr.Error()))
		}
	})
	if err != nil {
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	// Stop consuming when client context is cancelled
	go func() {
		<-c.ctx.Done()
		consumeCtx.Stop()
	}()

	return nil
}

// IsConnected returns true if the client is connected to NATS.
func (c *Client) IsConnected() bool {
	return c.conn != nil && c.conn.IsConnected()
}

// Close cancels consumers and closes the NATS connection.
func (c *Client) Close() {
	if c.cancelFunc != nil {
		c.cancelFunc()
	}
	if c.conn != nil {
		c.conn.Close()
	}
	c.mu.Lock()
	c.streams = make(map[string]jetstream.Stream)
	c.consumers = make(map[string]jetstream.Consumer)
	c.mu.Unlock()
}
