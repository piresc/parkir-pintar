package middleware

import (
	"net/http"

	"parkir-pintar/pkg/ratelimit"
	"parkir-pintar/pkg/response"

	"github.com/gin-gonic/gin"
)

// RateLimitConfig is an alias for the shared ratelimit.Config.
type RateLimitConfig = ratelimit.Config

// DefaultRateLimitConfig returns sensible defaults: 100 req/s, burst of 200.
func DefaultRateLimitConfig() RateLimitConfig {
	return ratelimit.DefaultConfig()
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
		m.rateStore = ratelimit.NewStore(cfg)
		m.rateStoreCfg = cfg
	}
	m.mu.Unlock()

	return func(c *gin.Context) {
		key := c.ClientIP()

		if !m.rateStore.Allow(key) {
			c.Abort()
			response.Error(c, http.StatusTooManyRequests, "rate limit exceeded")
			return
		}

		c.Next()
	}
}
