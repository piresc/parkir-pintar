package nats

import (
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStreamConfig_ToJetStreamConfig_ShouldConvertCorrectly(t *testing.T) {
	sc := StreamConfig{
		Name:      "TEST_STREAM",
		Subjects:  []string{"test.subject.*"},
		Retention: jetstream.InterestPolicy,
		Storage:   jetstream.MemoryStorage,
		MaxAge:    2 * time.Hour,
	}

	jsCfg := sc.ToJetStreamConfig()

	assert.Equal(t, "TEST_STREAM", jsCfg.Name)
	assert.Equal(t, []string{"test.subject.*"}, jsCfg.Subjects)
	assert.Equal(t, jetstream.InterestPolicy, jsCfg.Retention)
	assert.Equal(t, jetstream.MemoryStorage, jsCfg.Storage)
	assert.Equal(t, 2*time.Hour, jsCfg.MaxAge)
}

func TestConsumerConfig_ToJetStreamConfig_ShouldConvertCorrectly(t *testing.T) {
	cc := ConsumerConfig{
		Stream:        "TEST_STREAM",
		Name:          "test-consumer",
		FilterSubject: "test.subject.*",
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       15 * time.Second,
		MaxDeliver:    3,
		DeliverPolicy: jetstream.DeliverAllPolicy,
	}

	jsCfg := cc.ToJetStreamConfig()

	assert.Equal(t, "test-consumer", jsCfg.Name)
	assert.Equal(t, "test-consumer", jsCfg.Durable)
	assert.Equal(t, "test.subject.*", jsCfg.FilterSubject)
	assert.Equal(t, jetstream.AckExplicitPolicy, jsCfg.AckPolicy)
	assert.Equal(t, 15*time.Second, jsCfg.AckWait)
	assert.Equal(t, 3, jsCfg.MaxDeliver)
	assert.Equal(t, jetstream.DeliverAllPolicy, jsCfg.DeliverPolicy)
}

func TestNewPublisher_ShouldReturnNonNil(t *testing.T) {
	// We can't create a real Client without a NATS connection,
	publisher := NewPublisher(nil)
	require.NotNil(t, publisher)
}

func TestClient_Consume_ShouldReturnError_WhenConsumerNotFound(t *testing.T) {
	client := &Client{
		streams:   make(map[string]jetstream.Stream),
		consumers: make(map[string]jetstream.Consumer),
	}

	_, err := client.Consume("nonexistent-consumer", func(msg jetstream.Msg) {})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "consumer nonexistent-consumer not found")
}
