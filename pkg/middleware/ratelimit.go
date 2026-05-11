package middleware

import (
	"net/http"
	"sync"
	"time"

	"parkir-pintar/pkg/response"

	"github.com/gin-gonic/gin"
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

// tokenBucket implements a simple token bucket rate limiter.
type tokenBucket struct {
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens per second
	lastRefill time.Time
}

func newTokenBucket(maxTokens float64, refillRate float64) *tokenBucket {
	return &tokenBucket{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// allow checks if a request is allowed and consumes a token if so.
func (tb *tokenBucket) allow() bool {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens += elapsed * tb.refillRate
	if tb.tokens > tb.maxTokens {
		tb.tokens = tb.maxTokens
	}
	tb.lastRefill = now

	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}
	return false
}

// rateLimitStore is a thread-safe in-memory store of per-key token buckets.
type rateLimitStore struct {
	mu      sync.Mutex
	buckets map[string]*tokenBucket
	cfg     RateLimitConfig
	stopCh  chan struct{}
}

func newRateLimitStore(cfg RateLimitConfig) *rateLimitStore {
	store := &rateLimitStore{
		buckets: make(map[string]*tokenBucket),
		cfg:     cfg,
		stopCh:  make(chan struct{}),
	}

	// Background cleanup of stale entries
	go store.cleanup(cfg.CleanupInterval)

	return store
}

func (s *rateLimitStore) allow(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	bucket, exists := s.buckets[key]
	if !exists {
		bucket = newTokenBucket(float64(s.cfg.BurstSize), float64(s.cfg.RequestsPerSecond))
		s.buckets[key] = bucket
	}

	return bucket.allow()
}

func (s *rateLimitStore) cleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()
			for key, bucket := range s.buckets {
				// Remove buckets that haven't been used for 2x the cleanup interval
				if now.Sub(bucket.lastRefill) > 2*interval {
					delete(s.buckets, key)
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
// token bucket algorithm. Requests exceeding the limit receive HTTP 429.
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
