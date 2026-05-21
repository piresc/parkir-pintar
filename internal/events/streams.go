package events

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	pkgnats "parkir-pintar/pkg/nats"
)

// Stream names.
const (
	StreamReservationSearch    = "RESERVATION_SEARCH"
	StreamReservationAnalytics = "RESERVATION_ANALYTICS"
	StreamPaymentReservation   = "PAYMENT_RESERVATION"
)

// Consumer names.
const (
	ConsumerSearchSpot         = "search-spot-consumer"
	ConsumerAnalytics          = "analytics-consumer"
	ConsumerReservationPayment = "reservation-payment-consumer"
)

// Subject patterns for stream subscriptions.
const (
	SubjectPatternReservationSearch    = "reservation.search.*"
	SubjectPatternReservationAnalytics = "reservation.analytics.*"
	SubjectPatternPaymentReservation   = "payment.reservation.*"
)

const defaultStreamMaxAge = 7 * 24 * time.Hour

// DefaultStreamConfigs returns the domain-specific stream configurations.
func DefaultStreamConfigs() []pkgnats.StreamConfig {
	return []pkgnats.StreamConfig{
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

// DefaultConsumerConfigs returns the domain-specific consumer configurations.
func DefaultConsumerConfigs() map[string]pkgnats.ConsumerConfig {
	return map[string]pkgnats.ConsumerConfig{
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

// CreateConsumersForService creates the NATS consumers required by the given service.
func CreateConsumersForService(ctx context.Context, client *pkgnats.Client, serviceName string) error {
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

		if _, err := client.CreateConsumer(ctx, cfg.Stream, cfg.ToJetStreamConfig()); err != nil {
			return fmt.Errorf("create consumer %s: %w", name, err)
		}
	}

	return nil
}
