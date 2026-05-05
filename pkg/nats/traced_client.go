package nats

import (
	"github.com/nats-io/nats.go/jetstream"

	"parkir-pintar/pkg/tracing"
)

// TracedClient wraps a NATS Client with automatic OTEL tracing.
type TracedClient struct {
	client *Client
	tracer tracing.Tracer
}

// NewTracedClient creates a new traced NATS client.
func NewTracedClient(client *Client, tracer tracing.Tracer) *TracedClient {
	return &TracedClient{
		client: client,
		tracer: tracer,
	}
}

// GetClient returns the underlying NATS client.
func (tc *TracedClient) GetClient() *Client {
	return tc.client
}

// CreateOrUpdateStream delegates to the underlying client.
func (tc *TracedClient) CreateOrUpdateStream(cfg StreamConfig) error {
	return tc.client.CreateOrUpdateStream(cfg)
}

// CreateConsumer delegates to the underlying client.
func (tc *TracedClient) CreateConsumer(cfg ConsumerConfig) error {
	return tc.client.CreateConsumer(cfg)
}

// Publish publishes a message with automatic tracing.
func (tc *TracedClient) Publish(subject string, data []byte) error {
	if !tc.tracer.IsEnabled() {
		return tc.client.Publish(subject, data)
	}

	_, done := tc.tracer.StartMessage(tc.client.ctx, subject, "publish")
	defer done()

	return tc.client.Publish(subject, data)
}

// ConsumeMessages consumes messages with automatic tracing on each message.
func (tc *TracedClient) ConsumeMessages(streamName, consumerName string, handler func(jetstream.Msg) error) error {
	if !tc.tracer.IsEnabled() {
		return tc.client.ConsumeMessages(streamName, consumerName, handler)
	}

	tracedHandler := func(msg jetstream.Msg) error {
		_, done := tc.tracer.StartMessage(tc.client.ctx, msg.Subject(), "consume")
		defer done()
		return handler(msg)
	}

	return tc.client.ConsumeMessages(streamName, consumerName, tracedHandler)
}

// IsConnected returns true if the client is connected to NATS.
func (tc *TracedClient) IsConnected() bool {
	return tc.client.IsConnected()
}

// Close closes the NATS client.
func (tc *TracedClient) Close() {
	tc.client.Close()
}
