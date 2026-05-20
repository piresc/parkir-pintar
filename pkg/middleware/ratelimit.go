package middleware

import (
	"net/http"

	"parkir-pintar/pkg/ratelimit"
	"parkir-pintar/pkg/response"

	"github.com/gin-gonic/gin"
)

type RateLimitConfig = ratelimit.Config

func DefaultRateLimitConfig() RateLimitConfig {
	return ratelimit.DefaultConfig()
}

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
