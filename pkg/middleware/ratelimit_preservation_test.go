// Package middleware provides preservation property tests for rate limit allow pass-through.
//
// Best practices applied (from Go testify coding standards KB):
// - Test naming: Test[FunctionName]_Should[ExpectedResult]_When[Condition]
// - AAA pattern: Arrange → Act → Assert
// - testify/assert for assertions
// - Keep tests simple and focused on the behavior being tested
//
// **Validates: Requirements 3.12** (Preservation Property 14 from design)
//
// Non-bug condition: requestRate <= limit
// These tests verify that requests within configured rate limits pass through
// to the handler on unfixed code. They must PASS on unfixed code.
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"parkir-pintar/pkg/config"

	"pgregory.net/rapid"
)

// TestRateLimiter_ShouldAllowRequest_WhenWithinLimit verifies that requests
// within the configured rate limit pass through to the handler.
// Non-bug condition: requestRate <= limit.
//
// **Validates: Requirements 3.12**
func TestRateLimiter_ShouldAllowRequest_WhenWithinLimit(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Arrange — use a generous rate limit so single requests always pass
		burstSize := rapid.IntRange(10, 200).Draw(t, "burstSize")
		rps := rapid.IntRange(10, 100).Draw(t, "rps")

		cfg := RateLimitConfig{
			RequestsPerSecond: rps,
			BurstSize:         burstSize,
			CleanupInterval:   5 * time.Minute,
		}

		mw := NewMiddleware(&config.Config{}, nil, nil)

		gin.SetMode(gin.TestMode)
		engine := gin.New()

		handlerCalled := false
		engine.Use(mw.RateLimiter(cfg))
		engine.GET("/test", func(c *gin.Context) {
			handlerCalled = true
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.0.2.1:12345"
		w := httptest.NewRecorder()

		// Act — single request, well within limit
		engine.ServeHTTP(w, req)

		// Assert — request should pass through
		assert.Equal(t, http.StatusOK, w.Code, "request within rate limit should pass through")
		assert.True(t, handlerCalled, "handler should have been called")
	})
}

// TestRateLimiter_ShouldAllowMultipleRequests_WhenWithinBurst verifies that
// multiple requests within the burst size all pass through.
// Non-bug condition: requestRate <= limit.
//
// **Validates: Requirements 3.12**
func TestRateLimiter_ShouldAllowMultipleRequests_WhenWithinBurst(t *testing.T) {
	// Arrange — burst of 50, send 5 requests
	cfg := RateLimitConfig{
		RequestsPerSecond: 100,
		BurstSize:         50,
		CleanupInterval:   5 * time.Minute,
	}

	mw := NewMiddleware(&config.Config{}, nil, nil)

	gin.SetMode(gin.TestMode)
	engine := gin.New()

	callCount := 0
	engine.Use(mw.RateLimiter(cfg))
	engine.GET("/test", func(c *gin.Context) {
		callCount++
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Act — send 5 requests (well within burst of 50)
	for range 5 {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.0.2.1:12345"
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	// Assert
	assert.Equal(t, 5, callCount, "all 5 requests should have passed through")
}
