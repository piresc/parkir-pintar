package events

import (
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubjectConstants_ShouldHaveExpectedValues(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"ReservationSearchSpotUpdated", SubjectReservationSearchSpotUpdated, "reservation.search.spot-updated"},
		{"ReservationAnalyticsCreated", SubjectReservationAnalyticsCreated, "reservation.analytics.created"},
		{"ReservationAnalyticsConfirmed", SubjectReservationAnalyticsConfirmed, "reservation.analytics.confirmed"},
		{"ReservationAnalyticsCheckedIn", SubjectReservationAnalyticsCheckedIn, "reservation.analytics.checked-in"},
		{"ReservationAnalyticsCompleted", SubjectReservationAnalyticsCompleted, "reservation.analytics.completed"},
		{"ReservationAnalyticsCancelled", SubjectReservationAnalyticsCancelled, "reservation.analytics.cancelled"},
		{"ReservationAnalyticsExpired", SubjectReservationAnalyticsExpired, "reservation.analytics.expired"},
		{"ReservationAnalyticsFailed", SubjectReservationAnalyticsFailed, "reservation.analytics.failed"},
		{"PaymentReservationSuccess", SubjectPaymentReservationSuccess, "payment.reservation.success"},
		{"PaymentReservationFailed", SubjectPaymentReservationFailed, "payment.reservation.failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.constant)
		})
	}
}

func TestStreamNameConstants_ShouldHaveExpectedValues(t *testing.T) {
	assert.Equal(t, "RESERVATION_SEARCH", StreamReservationSearch)
	assert.Equal(t, "RESERVATION_ANALYTICS", StreamReservationAnalytics)
	assert.Equal(t, "PAYMENT_RESERVATION", StreamPaymentReservation)
}

func TestConsumerNameConstants_ShouldHaveExpectedValues(t *testing.T) {
	assert.Equal(t, "search-spot-consumer", ConsumerSearchSpot)
	assert.Equal(t, "analytics-consumer", ConsumerAnalytics)
	assert.Equal(t, "reservation-payment-consumer", ConsumerReservationPayment)
}

func TestDefaultStreamConfigs_ShouldReturnThreeStreams(t *testing.T) {
	configs := DefaultStreamConfigs()
	require.Len(t, configs, 3)
}

func TestDefaultStreamConfigs_ShouldHaveCorrectReservationSearchConfig(t *testing.T) {
	configs := DefaultStreamConfigs()

	var found bool
	for _, cfg := range configs {
		if cfg.Name == StreamReservationSearch {
			found = true
			assert.Equal(t, []string{"reservation.search.*"}, cfg.Subjects)
			assert.Equal(t, jetstream.InterestPolicy, cfg.Retention)
			assert.Equal(t, jetstream.FileStorage, cfg.Storage)
			assert.Equal(t, 24*time.Hour, cfg.MaxAge)
		}
	}
	assert.True(t, found, "expected to find RESERVATION_SEARCH stream config")
}

func TestDefaultStreamConfigs_ShouldHaveCorrectReservationAnalyticsConfig(t *testing.T) {
	configs := DefaultStreamConfigs()

	var found bool
	for _, cfg := range configs {
		if cfg.Name == StreamReservationAnalytics {
			found = true
			assert.Equal(t, []string{"reservation.analytics.*"}, cfg.Subjects)
			assert.Equal(t, jetstream.LimitsPolicy, cfg.Retention)
			assert.Equal(t, jetstream.FileStorage, cfg.Storage)
			assert.Equal(t, 7*24*time.Hour, cfg.MaxAge)
		}
	}
	assert.True(t, found, "expected to find RESERVATION_ANALYTICS stream config")
}

func TestDefaultStreamConfigs_ShouldHaveCorrectPaymentReservationConfig(t *testing.T) {
	configs := DefaultStreamConfigs()

	var found bool
	for _, cfg := range configs {
		if cfg.Name == StreamPaymentReservation {
			found = true
			assert.Equal(t, []string{"payment.reservation.*"}, cfg.Subjects)
			assert.Equal(t, jetstream.InterestPolicy, cfg.Retention)
			assert.Equal(t, jetstream.FileStorage, cfg.Storage)
			assert.Equal(t, 24*time.Hour, cfg.MaxAge)
		}
	}
	assert.True(t, found, "expected to find PAYMENT_RESERVATION stream config")
}

func TestDefaultConsumerConfigs_ShouldReturnThreeConsumers(t *testing.T) {
	configs := DefaultConsumerConfigs()
	require.Len(t, configs, 3)
}

func TestDefaultConsumerConfigs_ShouldHaveCorrectSearchSpotConfig(t *testing.T) {
	configs := DefaultConsumerConfigs()

	cfg, ok := configs[ConsumerSearchSpot]
	require.True(t, ok, "expected ConsumerSearchSpot in configs")

	assert.Equal(t, StreamReservationSearch, cfg.Stream)
	assert.Equal(t, ConsumerSearchSpot, cfg.Name)
	assert.Equal(t, "reservation.search.*", cfg.FilterSubject)
	assert.Equal(t, jetstream.AckExplicitPolicy, cfg.AckPolicy)
	assert.Equal(t, 30*time.Second, cfg.AckWait)
	assert.Equal(t, 5, cfg.MaxDeliver)
	assert.Equal(t, jetstream.DeliverNewPolicy, cfg.DeliverPolicy)
}

func TestDefaultConsumerConfigs_ShouldHaveCorrectAnalyticsConfig(t *testing.T) {
	configs := DefaultConsumerConfigs()

	cfg, ok := configs[ConsumerAnalytics]
	require.True(t, ok, "expected ConsumerAnalytics in configs")

	assert.Equal(t, StreamReservationAnalytics, cfg.Stream)
	assert.Equal(t, ConsumerAnalytics, cfg.Name)
	assert.Equal(t, "reservation.analytics.*", cfg.FilterSubject)
	assert.Equal(t, jetstream.AckExplicitPolicy, cfg.AckPolicy)
	assert.Equal(t, 30*time.Second, cfg.AckWait)
	assert.Equal(t, 5, cfg.MaxDeliver)
	assert.Equal(t, jetstream.DeliverNewPolicy, cfg.DeliverPolicy)
}

func TestDefaultConsumerConfigs_ShouldHaveCorrectReservationPaymentConfig(t *testing.T) {
	configs := DefaultConsumerConfigs()

	cfg, ok := configs[ConsumerReservationPayment]
	require.True(t, ok, "expected ConsumerReservationPayment in configs")

	assert.Equal(t, StreamPaymentReservation, cfg.Stream)
	assert.Equal(t, ConsumerReservationPayment, cfg.Name)
	assert.Equal(t, "payment.reservation.*", cfg.FilterSubject)
	assert.Equal(t, jetstream.AckExplicitPolicy, cfg.AckPolicy)
	assert.Equal(t, 30*time.Second, cfg.AckWait)
	assert.Equal(t, 5, cfg.MaxDeliver)
	assert.Equal(t, jetstream.DeliverNewPolicy, cfg.DeliverPolicy)
}
