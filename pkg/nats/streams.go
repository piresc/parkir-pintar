package nats

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// StreamConfig is a simplified stream configuration.
type StreamConfig struct {
	Name      string
	Subjects  []string
	Retention jetstream.RetentionPolicy
	Storage   jetstream.StorageType
	MaxAge    time.Duration
}

// ConsumerConfig is a simplified consumer configuration.
type ConsumerConfig struct {
	Stream        string
	Name          string
	FilterSubject string
	AckPolicy     jetstream.AckPolicy
	AckWait       time.Duration
	MaxDeliver    int
	DeliverPolicy jetstream.DeliverPolicy
}

// ToJetStreamConfig converts StreamConfig to a jetstream.StreamConfig.
func (sc StreamConfig) ToJetStreamConfig() jetstream.StreamConfig {
	return jetstream.StreamConfig{
		Name:      sc.Name,
		Subjects:  sc.Subjects,
		Retention: sc.Retention,
		Storage:   sc.Storage,
		MaxAge:    sc.MaxAge,
	}
}

// ToJetStreamConfig converts ConsumerConfig to a jetstream.ConsumerConfig.
func (cc ConsumerConfig) ToJetStreamConfig() jetstream.ConsumerConfig {
	return jetstream.ConsumerConfig{
		Name:          cc.Name,
		Durable:       cc.Name,
		FilterSubject: cc.FilterSubject,
		AckPolicy:     cc.AckPolicy,
		AckWait:       cc.AckWait,
		MaxDeliver:    cc.MaxDeliver,
		DeliverPolicy: cc.DeliverPolicy,
	}
}

// CreateStreams creates the given streams on the NATS client.
func CreateStreams(ctx context.Context, client *Client, configs []StreamConfig) error {
	for _, cfg := range configs {
		if _, err := client.CreateStream(ctx, cfg.ToJetStreamConfig()); err != nil {
			return fmt.Errorf("create stream %s: %w", cfg.Name, err)
		}
	}
	return nil
}
