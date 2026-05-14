package nats

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// StreamConfig holds the configuration for creating a stream.
type StreamConfig struct {
	Name      string
	Subjects  []string
	Retention jetstream.RetentionPolicy
	Storage   jetstream.StorageType
	MaxAge    time.Duration
}

// ConsumerConfig holds the configuration for creating a consumer.
type ConsumerConfig struct {
	Stream        string
	Name          string
	FilterSubject string
	AckPolicy     jetstream.AckPolicy
	AckWait       time.Duration
	MaxDeliver    int
	DeliverPolicy jetstream.DeliverPolicy
}

// DefaultStreamConfigs returns the default stream configurations for parkir-pintar.
func DefaultStreamConfigs() []StreamConfig {
	return []StreamConfig{
		{
			Name:      StreamReservationSearch,
			Subjects:  []string{"reservation.search.*"},
			Retention: jetstream.InterestPolicy,
			Storage:   jetstream.FileStorage,
			MaxAge:    24 * time.Hour,
		},
		{
			Name:      StreamReservationAnalytics,
			Subjects:  []string{"reservation.analytics.*"},
			Retention: jetstream.LimitsPolicy,
			Storage:   jetstream.FileStorage,
			MaxAge:    7 * 24 * time.Hour,
		},
		{
			Name:      StreamPaymentReservation,
			Subjects:  []string{"payment.reservation.*"},
			Retention: jetstream.InterestPolicy,
			Storage:   jetstream.FileStorage,
			MaxAge:    24 * time.Hour,
		},
	}
}

// DefaultConsumerConfigs returns the default consumer configurations keyed by consumer name.
// The map key is the consumer name, and the value contains the stream and config details.
func DefaultConsumerConfigs() map[string]ConsumerConfig {
	return map[string]ConsumerConfig{
		ConsumerSearchSpot: {
			Stream:        StreamReservationSearch,
			Name:          ConsumerSearchSpot,
			FilterSubject: "reservation.search.*",
			AckPolicy:     jetstream.AckExplicitPolicy,
			AckWait:       30 * time.Second,
			MaxDeliver:    5,
			DeliverPolicy: jetstream.DeliverNewPolicy,
		},
		ConsumerAnalytics: {
			Stream:        StreamReservationAnalytics,
			Name:          ConsumerAnalytics,
			FilterSubject: "reservation.analytics.*",
			AckPolicy:     jetstream.AckExplicitPolicy,
			AckWait:       30 * time.Second,
			MaxDeliver:    5,
			DeliverPolicy: jetstream.DeliverNewPolicy,
		},
		ConsumerReservationPayment: {
			Stream:        StreamPaymentReservation,
			Name:          ConsumerReservationPayment,
			FilterSubject: "payment.reservation.*",
			AckPolicy:     jetstream.AckExplicitPolicy,
			AckWait:       30 * time.Second,
			MaxDeliver:    5,
			DeliverPolicy: jetstream.DeliverNewPolicy,
		},
	}
}

// toJetStreamConfig converts our StreamConfig to a jetstream.StreamConfig.
func (sc StreamConfig) toJetStreamConfig() jetstream.StreamConfig {
	return jetstream.StreamConfig{
		Name:      sc.Name,
		Subjects:  sc.Subjects,
		Retention: sc.Retention,
		Storage:   sc.Storage,
		MaxAge:    sc.MaxAge,
	}
}

// toJetStreamConfig converts our ConsumerConfig to a jetstream.ConsumerConfig.
func (cc ConsumerConfig) toJetStreamConfig() jetstream.ConsumerConfig {
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

// CreateStreams creates all default streams using the client.
func CreateStreams(ctx context.Context, client *Client) error {
	for _, cfg := range DefaultStreamConfigs() {
		if _, err := client.CreateStream(ctx, cfg.toJetStreamConfig()); err != nil {
			return fmt.Errorf("create stream %s: %w", cfg.Name, err)
		}
	}
	return nil
}

// CreateConsumersForService creates consumers relevant to the given service.
// Service names map to consumers:
//   - "search": ConsumerSearchSpot
//   - "analytics": ConsumerAnalytics
//   - "reservation": ConsumerReservationPayment
func CreateConsumersForService(ctx context.Context, client *Client, serviceName string) error {
	serviceConsumers := map[string][]string{
		"search":      {ConsumerSearchSpot},
		"analytics":   {ConsumerAnalytics},
		"reservation": {ConsumerReservationPayment},
	}

	consumers, ok := serviceConsumers[serviceName]
	if !ok {
		slog.Warn("no consumers configured for service", "service", serviceName)
		return nil
	}

	allConfigs := DefaultConsumerConfigs()
	for _, name := range consumers {
		cfg, exists := allConfigs[name]
		if !exists {
			return fmt.Errorf("consumer config not found: %s", name)
		}

		if _, err := client.CreateConsumer(ctx, cfg.Stream, cfg.toJetStreamConfig()); err != nil {
			return fmt.Errorf("create consumer %s: %w", name, err)
		}
	}

	return nil
}
