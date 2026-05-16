// Package ratelimit provides a shared per-key rate limiter store backed by
// golang.org/x/time/rate. Used by both HTTP (gin) and gRPC middleware.
package ratelimit

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Config holds configuration for the rate limiter.
type Config struct {
	// RequestsPerSecond is the token refill rate per second per key.
	RequestsPerSecond int
	// BurstSize is the maximum token bucket capacity (max burst).
	BurstSize int
	// CleanupInterval is how often stale per-key entries are removed.
	CleanupInterval time.Duration
}

// DefaultConfig returns sensible defaults: 100 req/s, burst of 200.
func DefaultConfig() Config {
	return Config{
		RequestsPerSecond: 100,
		BurstSize:         200,
		CleanupInterval:   5 * time.Minute,
	}
}

// entry holds a per-client rate limiter and its last access time.
type entry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// Store is a thread-safe in-memory store of per-key rate limiters.
type Store struct {
	mu       sync.Mutex
	limiters map[string]*entry
	cfg      Config
	stopCh   chan struct{}
}

// NewStore creates a rate limit store with background cleanup.
func NewStore(cfg Config) *Store {
	s := &Store{
		limiters: make(map[string]*entry),
		cfg:      cfg,
		stopCh:   make(chan struct{}),
	}

	if cfg.CleanupInterval > 0 {
		go s.cleanup(cfg.CleanupInterval)
	}

	return s
}

// Allow checks whether a request identified by key is allowed.
func (s *Store) Allow(key string) bool {
	s.mu.Lock()
	e, exists := s.limiters[key]
	if !exists {
		e = &entry{
			limiter:  rate.NewLimiter(rate.Limit(s.cfg.RequestsPerSecond), s.cfg.BurstSize),
			lastSeen: time.Now(),
		}
		s.limiters[key] = e
	} else {
		e.lastSeen = time.Now()
	}
	s.mu.Unlock()

	return e.limiter.Allow()
}

func (s *Store) cleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()
			for key, e := range s.limiters {
				if now.Sub(e.lastSeen) > 2*interval {
					delete(s.limiters, key)
				}
			}
			s.mu.Unlock()
		case <-s.stopCh:
			return
		}
	}
}

// Stop terminates the background cleanup goroutine.
func (s *Store) Stop() {
	close(s.stopCh)
}
