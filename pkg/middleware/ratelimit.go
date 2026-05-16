package middleware

import (
	"net/http"
	"sync"
	"time"

	"parkir-pintar/pkg/response"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimitConfig holds configuration for the rate limiter.
type RateLimitConfig struct {
	// RequestsPerSecond is the maximum number of requests allowed per second per key.
	RequestsPerSecond int
	// BurstSize is the maximum burst size (token bucket capacity).
	BurstSize int
	// CleanupInterval is how often expired entries are removed from the store.
	CleanupInterval time.Duration
}

// DefaultRateLimitConfig returns sensible defaults: 100 req/s, burst of 200.
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		RequestsPerSecond: 100,
		BurstSize:         200,
		CleanupInterval:   5 * time.Minute,
	}
}

// rateLimitEntry holds a per-client rate limiter and its last access time.
type rateLimitEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// rateLimitStore is a thread-safe in-memory store of per-key rate limiters.
// Powered by golang.org/x/time/rate.
type rateLimitStore struct {
	mu       sync.Mutex
	limiters map[string]*rateLimitEntry
	cfg      RateLimitConfig
	stopCh   chan struct{}
}

func newRateLimitStore(cfg RateLimitConfig) *rateLimitStore {
	store := &rateLimitStore{
		limiters: make(map[string]*rateLimitEntry),
		cfg:      cfg,
		stopCh:   make(chan struct{}),
	}

	// Background cleanup of stale entries
	if cfg.CleanupInterval > 0 {
		go store.cleanup(cfg.CleanupInterval)
	}

	return store
}

func (s *rateLimitStore) allow(key string) bool {
	s.mu.Lock()
	entry, exists := s.limiters[key]
	if !exists {
		entry = &rateLimitEntry{
			limiter:  rate.NewLimiter(rate.Limit(s.cfg.RequestsPerSecond), s.cfg.BurstSize),
			lastSeen: time.Now(),
		}
		s.limiters[key] = entry
	} else {
		entry.lastSeen = time.Now()
	}
	s.mu.Unlock()

	return entry.limiter.Allow()
}

func (s *rateLimitStore) cleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()
			for key, entry := range s.limiters {
				// Remove limiters that haven't been used for 2x the cleanup interval
				if now.Sub(entry.lastSeen) > 2*interval {
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
func (s *rateLimitStore) Stop() {
	close(s.stopCh)
}

// RateLimiter returns middleware that enforces per-IP rate limiting using a
// token bucket algorithm (golang.org/x/time/rate). Requests exceeding the
// limit receive HTTP 429.
//
// The key is derived from the client IP (c.ClientIP()), which respects
// X-Forwarded-For and X-Real-IP headers when behind a reverse proxy.
func (m *Middleware) RateLimiter(cfg RateLimitConfig) gin.HandlerFunc {
	m.mu.Lock()
	if m.rateStore == nil {
		m.rateStore = newRateLimitStore(cfg)
		m.rateStoreCfg = cfg
	}
	m.mu.Unlock()

	return func(c *gin.Context) {
		key := c.ClientIP()

		if !m.rateStore.allow(key) {
			c.Abort()
			response.Error(c, http.StatusTooManyRequests, "rate limit exceeded")
			return
		}

		c.Next()
	}
}
