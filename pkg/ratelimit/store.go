package ratelimit

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type Config struct {
	RequestsPerSecond int
	BurstSize int
	CleanupInterval time.Duration
}

func DefaultConfig() Config {
	return Config{
		RequestsPerSecond: 100,
		BurstSize:         200,
		CleanupInterval:   5 * time.Minute,
	}
}

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

func (s *Store) Stop() {
	close(s.stopCh)
}
