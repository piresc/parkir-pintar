// Package subscriber provides NATS event subscribers for the notification
// domain module. It listens to reservation, billing, and payment events
// and logs the payloads (stub implementation).
package subscriber

import (
	"context"
	"log/slog"
)

// NATSSubscriber subscribes to domain events and logs notification payloads.
type NATSSubscriber struct{}

// NewNATSSubscriber creates a new NATSSubscriber.
func NewNATSSubscriber() *NATSSubscriber {
	return &NATSSubscriber{}
}

// Subjects returns the NATS subjects this subscriber listens to.
func (s *NATSSubscriber) Subjects() []string {
	return []string{
		"reservation.*",
		"billing.*",
		"payment.*",
	}
}

// HandleEvent logs the received event subject and payload via slog.
// This is a stub implementation — no actual notifications are sent.
func (s *NATSSubscriber) HandleEvent(_ context.Context, subject string, data []byte) {
	slog.Info("notification subscriber: received event",
		slog.String("subject", subject),
		slog.String("payload", string(data)))
}
