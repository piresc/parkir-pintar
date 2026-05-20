// Package retry provides a generic retry-with-backoff helper.
package retry

import (
	"context"
	"time"
)

// Config holds retry configuration.
type Config struct {
	Attempts int
	Backoffs []time.Duration
}

// DefaultConfig returns a standard retry config.
func DefaultConfig() Config {
	return Config{
		Attempts: 3,
		Backoffs: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond, 400 * time.Millisecond},
	}
}

// Do executes fn with retries. Returns the last error if all attempts fail.
func Do(ctx context.Context, cfg Config, fn func(ctx context.Context) error) error {
	var lastErr error
	for i := 0; i < cfg.Attempts; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		lastErr = fn(ctx)
		if lastErr == nil {
			return nil
		}
		if i < len(cfg.Backoffs) {
			select {
			case <-time.After(cfg.Backoffs[i]):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	return lastErr
}
