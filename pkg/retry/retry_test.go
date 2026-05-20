package retry_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"parkir-pintar/pkg/retry"
)

func TestDo_SucceedsOnFirstAttempt(t *testing.T) {
	calls := 0
	err := retry.Do(context.Background(), retry.DefaultConfig(), func(_ context.Context) error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestDo_SucceedsOnRetry(t *testing.T) {
	calls := 0
	err := retry.Do(context.Background(), retry.DefaultConfig(), func(_ context.Context) error {
		calls++
		if calls < 3 {
			return errors.New("transient")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if calls != 3 {
		t.Fatalf("expected 3 calls, got %d", calls)
	}
}

func TestDo_ExhaustsAttempts(t *testing.T) {
	cfg := retry.Config{
		Attempts: 3,
		Backoffs: []time.Duration{1 * time.Millisecond, 1 * time.Millisecond, 1 * time.Millisecond},
	}
	calls := 0
	sentinel := errors.New("persistent error")
	err := retry.Do(context.Background(), cfg, func(_ context.Context) error {
		calls++
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error, got %v", err)
	}
	if calls != 3 {
		t.Fatalf("expected 3 calls, got %d", calls)
	}
}

func TestDo_RespectsContextCancellation(t *testing.T) {
	cfg := retry.Config{
		Attempts: 3,
		Backoffs: []time.Duration{5 * time.Second, 5 * time.Second},
	}
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	err := retry.Do(ctx, cfg, func(_ context.Context) error {
		calls++
		cancel() // cancel after first attempt
		return errors.New("fail")
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestDo_ReturnsEarlyIfContextAlreadyCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	calls := 0
	err := retry.Do(ctx, retry.DefaultConfig(), func(_ context.Context) error {
		calls++
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if calls != 0 {
		t.Fatalf("expected 0 calls, got %d", calls)
	}
}
