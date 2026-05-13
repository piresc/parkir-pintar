package outbox

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// NATSPublisher defines the interface for publishing messages to NATS.
type NATSPublisher interface {
	Publish(subject string, data []byte) error
}

// ProcessorConfig holds configuration for the OutboxProcessor.
type ProcessorConfig struct {
	// PollInterval is how often the processor checks for unprocessed messages.
	PollInterval time.Duration
	// BatchSize is the maximum number of messages to process per poll cycle.
	BatchSize int
}

// DefaultProcessorConfig returns sensible defaults for the processor.
func DefaultProcessorConfig() ProcessorConfig {
	return ProcessorConfig{
		PollInterval: 1 * time.Second,
		BatchSize:    100,
	}
}

// Processor polls the outbox table for unprocessed messages and publishes
// them to NATS with retry logic. It runs as a background goroutine.
type Processor struct {
	repo   Repository
	nats   NATSPublisher
	config ProcessorConfig
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewProcessor creates a new outbox Processor.
func NewProcessor(repo Repository, nats NATSPublisher, config ProcessorConfig) *Processor {
	return &Processor{
		repo:   repo,
		nats:   nats,
		config: config,
		stopCh: make(chan struct{}),
	}
}

// Start begins the background polling loop.
func (p *Processor) Start() {
	p.wg.Add(1)
	go p.run()
}

// Stop signals the processor to stop and waits for it to finish.
func (p *Processor) Stop() {
	close(p.stopCh)
	p.wg.Wait()
}

func (p *Processor) run() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopCh:
			return
		case <-ticker.C:
			p.processBatch()
		}
	}
}

func (p *Processor) processBatch() {
	ctx := context.Background()

	messages, err := p.repo.GetUnprocessed(ctx, p.config.BatchSize)
	if err != nil {
		slog.Error("outbox processor: failed to get unprocessed messages", slog.Any("error", err))
		return
	}

	for _, msg := range messages {
		select {
		case <-p.stopCh:
			return
		default:
			p.processMessage(ctx, msg)
		}
	}
}

func (p *Processor) processMessage(ctx context.Context, msg *Message) {
	if err := p.nats.Publish(msg.EventType, msg.Payload); err != nil {
		slog.Error("outbox processor: failed to publish message",
			slog.String("id", msg.ID),
			slog.String("event_type", msg.EventType),
			slog.Int("retry_count", msg.RetryCount),
			slog.Any("error", err),
		)

		if markErr := p.repo.MarkFailed(ctx, msg.ID); markErr != nil {
			slog.Error("outbox processor: failed to mark message as failed",
				slog.String("id", msg.ID),
				slog.Any("error", markErr),
			)
		}
		return
	}

	if err := p.repo.MarkProcessed(ctx, msg.ID); err != nil {
		slog.Error("outbox processor: failed to mark message as processed",
			slog.String("id", msg.ID),
			slog.Any("error", err),
		)
	}
}

// ProcessOnce runs a single processing cycle. Useful for testing.
func (p *Processor) ProcessOnce() {
	p.processBatch()
}
