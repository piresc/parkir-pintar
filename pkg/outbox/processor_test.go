package outbox

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRepository implements Repository for testing.
type mockRepository struct {
	mu           sync.Mutex
	messages     []*Message
	createErr    error
	getErr       error
	processedIDs []string
	failedIDs    []string
	markProcErr  error
	markFailErr  error
}

func (m *mockRepository) Create(_ context.Context, _ *sqlx.Tx, msg *Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.createErr != nil {
		return m.createErr
	}
	m.messages = append(m.messages, msg)
	return nil
}

func (m *mockRepository) GetUnprocessed(_ context.Context, limit int) ([]*Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.getErr != nil {
		return nil, m.getErr
	}
	var result []*Message
	for _, msg := range m.messages {
		if msg.Status == StatusPending && len(result) < limit {
			result = append(result, msg)
		}
	}
	return result, nil
}

func (m *mockRepository) MarkProcessed(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.markProcErr != nil {
		return m.markProcErr
	}
	m.processedIDs = append(m.processedIDs, id)
	for _, msg := range m.messages {
		if msg.ID == id {
			msg.Status = StatusProcessed
			now := time.Now()
			msg.ProcessedAt = &now
		}
	}
	return nil
}

func (m *mockRepository) MarkFailed(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.markFailErr != nil {
		return m.markFailErr
	}
	m.failedIDs = append(m.failedIDs, id)
	for _, msg := range m.messages {
		if msg.ID == id {
			msg.RetryCount++
			if msg.RetryCount >= msg.MaxRetries {
				msg.Status = StatusFailed
			}
		}
	}
	return nil
}

// mockNATS implements NATSPublisher for testing.
type mockNATS struct {
	mu        sync.Mutex
	published []publishedMsg
	err       error
}

type publishedMsg struct {
	subject string
	data    []byte
}

func (m *mockNATS) Publish(subject string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.published = append(m.published, publishedMsg{subject: subject, data: data})
	return nil
}

func TestProcessor_ProcessOnce_PublishesMessages(t *testing.T) {
	repo := &mockRepository{
		messages: []*Message{
			{
				ID:            "msg-1",
				AggregateType: "reservation",
				AggregateID:   "res-123",
				EventType:     "reservation.confirmed",
				Payload:       []byte(`{"id":"res-123"}`),
				CreatedAt:     time.Now(),
				RetryCount:    0,
				MaxRetries:    5,
				Status:        StatusPending,
			},
			{
				ID:            "msg-2",
				AggregateType: "reservation",
				AggregateID:   "res-456",
				EventType:     "reservation.cancelled",
				Payload:       []byte(`{"id":"res-456"}`),
				CreatedAt:     time.Now(),
				RetryCount:    0,
				MaxRetries:    5,
				Status:        StatusPending,
			},
		},
	}
	nats := &mockNATS{}

	processor := NewProcessor(repo, nats, DefaultProcessorConfig())
	processor.ProcessOnce()

	require.Len(t, nats.published, 2)
	assert.Equal(t, "reservation.confirmed", nats.published[0].subject)
	assert.Equal(t, []byte(`{"id":"res-123"}`), nats.published[0].data)
	assert.Equal(t, "reservation.cancelled", nats.published[1].subject)
	assert.Equal(t, []byte(`{"id":"res-456"}`), nats.published[1].data)

	assert.Equal(t, []string{"msg-1", "msg-2"}, repo.processedIDs)
}

func TestProcessor_ProcessOnce_NATSFailure_MarksRetry(t *testing.T) {
	repo := &mockRepository{
		messages: []*Message{
			{
				ID:            "msg-1",
				AggregateType: "reservation",
				AggregateID:   "res-123",
				EventType:     "reservation.confirmed",
				Payload:       []byte(`{"id":"res-123"}`),
				CreatedAt:     time.Now(),
				RetryCount:    0,
				MaxRetries:    5,
				Status:        StatusPending,
			},
		},
	}
	nats := &mockNATS{err: errors.New("connection refused")}

	processor := NewProcessor(repo, nats, DefaultProcessorConfig())
	processor.ProcessOnce()

	assert.Empty(t, repo.processedIDs)
	assert.Equal(t, []string{"msg-1"}, repo.failedIDs)
	// Message should still be pending (retry_count=1 < max_retries=5)
	assert.Equal(t, StatusPending, repo.messages[0].Status)
	assert.Equal(t, 1, repo.messages[0].RetryCount)
}

func TestProcessor_ProcessOnce_MaxRetriesExceeded(t *testing.T) {
	repo := &mockRepository{
		messages: []*Message{
			{
				ID:            "msg-1",
				AggregateType: "reservation",
				AggregateID:   "res-123",
				EventType:     "reservation.confirmed",
				Payload:       []byte(`{"id":"res-123"}`),
				CreatedAt:     time.Now(),
				RetryCount:    4, // one more failure will hit max_retries=5
				MaxRetries:    5,
				Status:        StatusPending,
			},
		},
	}
	nats := &mockNATS{err: errors.New("connection refused")}

	processor := NewProcessor(repo, nats, DefaultProcessorConfig())
	processor.ProcessOnce()

	assert.Equal(t, StatusFailed, repo.messages[0].Status)
	assert.Equal(t, 5, repo.messages[0].RetryCount)
}

func TestProcessor_ProcessOnce_EmptyQueue(t *testing.T) {
	repo := &mockRepository{}
	nats := &mockNATS{}

	processor := NewProcessor(repo, nats, DefaultProcessorConfig())
	processor.ProcessOnce()

	assert.Empty(t, nats.published)
	assert.Empty(t, repo.processedIDs)
}

func TestProcessor_ProcessOnce_RepoError(t *testing.T) {
	repo := &mockRepository{
		getErr: errors.New("database unavailable"),
	}
	nats := &mockNATS{}

	processor := NewProcessor(repo, nats, DefaultProcessorConfig())
	// Should not panic
	processor.ProcessOnce()

	assert.Empty(t, nats.published)
}

func TestProcessor_StartStop(t *testing.T) {
	repo := &mockRepository{
		messages: []*Message{
			{
				ID:            "msg-1",
				AggregateType: "reservation",
				AggregateID:   "res-123",
				EventType:     "reservation.confirmed",
				Payload:       []byte(`{"id":"res-123"}`),
				CreatedAt:     time.Now(),
				RetryCount:    0,
				MaxRetries:    5,
				Status:        StatusPending,
			},
		},
	}
	nats := &mockNATS{}

	config := ProcessorConfig{
		PollInterval: 50 * time.Millisecond,
		BatchSize:    10,
	}
	processor := NewProcessor(repo, nats, config)

	processor.Start()
	// Give it time to process at least one cycle
	time.Sleep(150 * time.Millisecond)
	processor.Stop()

	nats.mu.Lock()
	defer nats.mu.Unlock()
	assert.NotEmpty(t, nats.published)
	assert.Equal(t, "reservation.confirmed", nats.published[0].subject)
}

func TestProcessor_ProcessOnce_SkipsAlreadyProcessed(t *testing.T) {
	now := time.Now()
	repo := &mockRepository{
		messages: []*Message{
			{
				ID:            "msg-1",
				AggregateType: "reservation",
				AggregateID:   "res-123",
				EventType:     "reservation.confirmed",
				Payload:       []byte(`{"id":"res-123"}`),
				CreatedAt:     now,
				ProcessedAt:   &now,
				RetryCount:    0,
				MaxRetries:    5,
				Status:        StatusProcessed, // already processed
			},
			{
				ID:            "msg-2",
				AggregateType: "reservation",
				AggregateID:   "res-456",
				EventType:     "reservation.cancelled",
				Payload:       []byte(`{"id":"res-456"}`),
				CreatedAt:     now,
				RetryCount:    0,
				MaxRetries:    5,
				Status:        StatusPending,
			},
		},
	}
	nats := &mockNATS{}

	processor := NewProcessor(repo, nats, DefaultProcessorConfig())
	processor.ProcessOnce()

	// Only the pending message should be published
	require.Len(t, nats.published, 1)
	assert.Equal(t, "reservation.cancelled", nats.published[0].subject)
	assert.Equal(t, []string{"msg-2"}, repo.processedIDs)
}
