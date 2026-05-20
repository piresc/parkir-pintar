package nats

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

const (
	SubjectPatternReservationSearch    = "reservation.search.*"
	SubjectPatternReservationAnalytics = "reservation.analytics.*"
	SubjectPatternPaymentReservation   = "payment.reservation.*"

	defaultStreamMaxAge = 7 * 24 * time.Hour
)

type StreamConfig struct {
	Name      string
	Subjects  []string
	Retention jetstream.RetentionPolicy
	Storage   jetstream.StorageType
	MaxAge    time.Duration
}

type ConsumerConfig struct {
	Stream        string
	Name          string
	FilterSubject string
	AckPolicy     jetstream.AckPolicy
	AckWait       time.Duration
	MaxDeliver    int
	DeliverPolicy jetstream.DeliverPolicy
}

func DefaultStreamConfigs() []StreamConfig {
	return []StreamConfig{
		{
			Name:      StreamReservationSearch,
			Subjects:  []string{SubjectPatternReservationSearch},
			Retention: jetstream.InterestPolicy,
			Storage:   jetstream.FileStorage,
			MaxAge:    24 * time.Hour,
		},
		{
			Name:      StreamReservationAnalytics,
			Subjects:  []string{SubjectPatternReservationAnalytics},
			Retention: jetstream.LimitsPolicy,
			Storage:   jetstream.FileStorage,
			MaxAge:    defaultStreamMaxAge,
		},
		{
			Name:      StreamPaymentReservation,
			Subjects:  []string{SubjectPatternPaymentReservation},
			Retention: jetstream.InterestPolicy,
			Storage:   jetstream.FileStorage,
			MaxAge:    24 * time.Hour,
		},
	}
}

func DefaultConsumerConfigs() map[string]ConsumerConfig {
	return map[string]ConsumerConfig{
		ConsumerSearchSpot: {
			Stream:        StreamReservationSearch,
			Name:          ConsumerSearchSpot,
			FilterSubject: SubjectPatternReservationSearch,
			AckPolicy:     jetstream.AckExplicitPolicy,
			AckWait:       30 * time.Second,
			MaxDeliver:    5,
			DeliverPolicy: jetstream.DeliverNewPolicy,
		},
		ConsumerAnalytics: {
			Stream:        StreamReservationAnalytics,
			Name:          ConsumerAnalytics,
			FilterSubject: SubjectPatternReservationAnalytics,
			AckPolicy:     jetstream.AckExplicitPolicy,
			AckWait:       30 * time.Second,
			MaxDeliver:    5,
			DeliverPolicy: jetstream.DeliverNewPolicy,
		},
		ConsumerReservationPayment: {
			Stream:        StreamPaymentReservation,
			Name:          ConsumerReservationPayment,
			FilterSubject: SubjectPatternPaymentReservation,
			AckPolicy:     jetstream.AckExplicitPolicy,
			AckWait:       30 * time.Second,
			MaxDeliver:    5,
			DeliverPolicy: jetstream.DeliverNewPolicy,
		},
	}
}

func (sc StreamConfig) toJetStreamConfig() jetstream.StreamConfig {
	return jetstream.StreamConfig{
		Name:      sc.Name,
		Subjects:  sc.Subjects,
		Retention: sc.Retention,
		Storage:   sc.Storage,
		MaxAge:    sc.MaxAge,
	}
}

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

func CreateStreams(ctx context.Context, client *Client) error {
	for _, cfg := range DefaultStreamConfigs() {
		if _, err := client.CreateStream(ctx, cfg.toJetStreamConfig()); err != nil {
			return fmt.Errorf("create stream %s: %w", cfg.Name, err)
		}
	}
	return nil
}

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
