// Package middleware tests.
//
// Best practices applied (from Go coding standards KB / Go testify patterns):
// - Use table-driven tests for multiple scenarios
// - Use descriptive test names: Test[Function]_Should[Result]_When[Condition]
// - Use gin.TestMode and httptest for HTTP middleware testing
// - Use testify/assert for assertions
// - Test both success and failure scenarios
// - Keep mocks simple and focused on the behavior being tested
// - Clean up resources properly with defer
// - Mock external dependencies only
package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"parkir-pintar/pkg/config"
	"parkir-pintar/pkg/response"
	"parkir-pintar/pkg/tracing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// newTestMiddleware creates a Middleware with sensible test defaults.
func newTestMiddleware() *Middleware {
	cfg := &config.Config{
		App: config.AppConfig{
			Name:        "test-app",
			Environment: "local",
		},
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	tracer := tracing.NewNoOpTracer()
	return NewMiddleware(cfg, logger, tracer)
}

// --- 5.1 NewMiddleware ---

func TestNewMiddleware_ShouldReturnInstance_WhenValidDependencies(t *testing.T) {
	mw := newTestMiddleware()
	assert.NotNil(t, mw)
	assert.NotNil(t, mw.config)
	assert.NotNil(t, mw.logger)
	assert.NotNil(t, mw.tracer)
}

func TestNewMiddleware_ShouldUseDefaults_WhenNilLoggerAndTracer(t *testing.T) {
	mw := NewMiddleware(&config.Config{}, nil, nil)
	assert.NotNil(t, mw)
	assert.NotNil(t, mw.logger)
	assert.NotNil(t, mw.tracer)
}


// --- 5.2 SetContextValues ---

func TestSetContextValues_ShouldStoreHeaders_WhenPresent(t *testing.T) {
	mw := newTestMiddleware()
	w := httptest.NewRecorder()
	c, engine := gin.CreateTestContext(w)

	engine.Use(mw.SetContextValues())
	engine.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"txid":    c.GetString(KeyTransactionID),
			"msisdn":  c.GetString(KeyMsisdn),
			"app":     c.GetString(KeyAppVersion),
			"os":      c.GetString(KeyOSVersion),
			"device":  c.GetString(KeyDeviceID),
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("transactionid", "TX123")
	req.Header.Set("x-msisdn", "6281234567890")
	req.Header.Set("appversion", "1.0.0")
	req.Header.Set("osversion", "Android 14")
	req.Header.Set("x-device", "device-abc")
	c.Request = req

	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "TX123", body["txid"])
	assert.Equal(t, "6281234567890", body["msisdn"])
	assert.Equal(t, "1.0.0", body["app"])
	assert.Equal(t, "Android 14", body["os"])
	assert.Equal(t, "device-abc", body["device"])
}

func TestSetContextValues_ShouldFallbackAppVersion_WhenMobileHeader(t *testing.T) {
	mw := newTestMiddleware()
	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)

	engine.Use(mw.SetContextValues())
	engine.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"app": c.GetString(KeyAppVersion)})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("mytelkomsel-mobile-app-version", "2.0.0")

	engine.ServeHTTP(w, req)

	var body map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "2.0.0", body["app"])
}

// --- 5.3 NormalizeMsisdn ---

func TestNormalizeMsisdn_ShouldNormalize_WhenVariousFormats(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "empty_defaults_to_00000", input: "", expected: "00000"},
		{name: "plus_prefix_stripped", input: "+6281234567890", expected: "6281234567890"},
		{name: "zero_prefix_to_62", input: "081234567890", expected: "6281234567890"},
		{name: "already_normalized", input: "6281234567890", expected: "6281234567890"},
		{name: "short_number", input: "123", expected: "123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := newTestMiddleware()
			w := httptest.NewRecorder()
			_, engine := gin.CreateTestContext(w)

			engine.Use(mw.NormalizeMsisdn())
			engine.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{
					"header":  c.GetHeader("x-msisdn"),
					"context": c.GetString(KeyMsisdn),
				})
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.input != "" {
				req.Header.Set("x-msisdn", tt.input)
			}

			engine.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var body map[string]string
			err := json.Unmarshal(w.Body.Bytes(), &body)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, body["header"])
			assert.Equal(t, tt.expected, body["context"])
		})
	}
}

// --- 5.4 GenerateTransactionID ---

func TestGenerateTransactionID_ShouldGenerate_WhenCalled(t *testing.T) {
	mw := newTestMiddleware()
	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)

	engine.Use(mw.GenerateTransactionID())
	engine.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"txid": c.GetHeader("transactionid"),
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("x-msisdn", "6281234567890")

	engine.ServeHTTP(w, req)

	var body map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)

	txID := body["txid"]
	assert.NotEmpty(t, txID)
	// Should start with "L" for local environment
	assert.True(t, strings.HasPrefix(txID, "L"), "expected prefix L for local env, got: %s", txID)
	// Should end with last 5 digits of MSISDN
	assert.True(t, strings.HasSuffix(txID, "67890"), "expected suffix 67890, got: %s", txID)
}

func TestGenerateTransactionID_ShouldPreserveOld_WhenExisting(t *testing.T) {
	mw := newTestMiddleware()
	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)

	engine.Use(mw.GenerateTransactionID())
	engine.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"old": c.GetHeader("oldtransactionid"),
			"new": c.GetHeader("transactionid"),
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("transactionid", "OLD-TX-123")
	req.Header.Set("x-msisdn", "6281234567890")

	engine.ServeHTTP(w, req)

	var body map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "OLD-TX-123", body["old"])
	assert.NotEqual(t, "OLD-TX-123", body["new"])
}

func TestEnvAppID_ShouldReturnCorrectPrefix_WhenDifferentEnvironments(t *testing.T) {
	tests := []struct {
		env      string
		expected string
	}{
		{"local", "L"},
		{"development", "D"},
		{"staging", "S"},
		{"production", "P"},
		{"unknown", "L"},
	}

	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			cfg := &config.Config{App: config.AppConfig{Environment: tt.env}}
			assert.Equal(t, tt.expected, envAppID(cfg))
		})
	}
}

func TestEnvAppID_ShouldReturnL_WhenNilConfig(t *testing.T) {
	assert.Equal(t, "L", envAppID(nil))
}


// --- 5.5 LogResponse ---

func TestLogResponse_ShouldNotPanic_WhenCalled(t *testing.T) {
	mw := newTestMiddleware()
	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)

	engine.Use(mw.LogResponse())
	engine.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("transactionid", "TX-LOG-TEST")
	req.Header.Set("x-msisdn", "6281234567890")

	assert.NotPanics(t, func() {
		engine.ServeHTTP(w, req)
	})
	assert.Equal(t, http.StatusOK, w.Code)
}

// --- 5.6 CorsHandler ---

func TestCorsHandler_ShouldSetHeaders_WhenAllowedOrigin(t *testing.T) {
	mw := newTestMiddleware()
	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)

	engine.Use(mw.CorsHandler([]string{"http://localhost:3000"}))
	engine.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")

	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "http://localhost:3000", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
}

func TestCorsHandler_ShouldReject_WhenDisallowedOrigin(t *testing.T) {
	mw := newTestMiddleware()
	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)

	engine.Use(mw.CorsHandler([]string{"http://localhost:3000"}))
	engine.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://evil.com")

	engine.ServeHTTP(w, req)

	// gin-contrib/cors returns 403 for disallowed origins
	assert.Equal(t, http.StatusForbidden, w.Code)
	// Should NOT set Access-Control-Allow-Origin for disallowed origin
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCorsHandler_ShouldReturn204_WhenPreflightRequest(t *testing.T) {
	mw := newTestMiddleware()
	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)

	engine.Use(mw.CorsHandler([]string{"http://localhost:3000"}))
	engine.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")

	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestCorsHandler_ShouldNeverSetWildcard_WhenCredentialsEnabled(t *testing.T) {
	mw := newTestMiddleware()
	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)

	engine.Use(mw.CorsHandler([]string{"http://localhost:3000"}))
	engine.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")

	engine.ServeHTTP(w, req)

	// Must never be "*" when credentials are allowed
	assert.NotEqual(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
}

// --- 5.7 TracingHandler ---

func TestTracingHandler_ShouldPassThrough_WhenTracingDisabled(t *testing.T) {
	mw := newTestMiddleware() // uses NoOpTracer which returns false for ShouldTrace
	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)

	engine.Use(mw.TracingHandler())
	engine.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// --- 5.8 RecoveryHandler ---

func TestRecoveryHandler_ShouldReturn500_WhenPanicOccurs(t *testing.T) {
	mw := newTestMiddleware()
	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)

	engine.Use(mw.RecoveryHandler())
	engine.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var body response.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "error", body.Status)
	assert.Equal(t, "internal server error", body.Error)
}

func TestRecoveryHandler_ShouldIncludeRequestID_WhenHeaderPresent(t *testing.T) {
	mw := newTestMiddleware()
	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)

	engine.Use(mw.RecoveryHandler())
	engine.GET("/panic", func(c *gin.Context) {
		panic("boom")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	req.Header.Set("transactionid", "TX-PANIC-123")
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var body response.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "TX-PANIC-123", body.RequestID)
}

func TestRecoveryHandler_ShouldNotInterfere_WhenNoPanic(t *testing.T) {
	mw := newTestMiddleware()
	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)

	engine.Use(mw.RecoveryHandler())
	engine.GET("/ok", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}


// --- 5.9 JWTAuth ---

func createTestJWT(secret string, claims jwt.MapClaims) string {
	if claims["exp"] == nil {
		claims["exp"] = time.Now().Add(time.Hour).Unix()
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(secret))
	return tokenString
}

func TestJWTAuth_ShouldSetClaims_WhenValidToken(t *testing.T) {
	mw := newTestMiddleware()
	secret := "test-secret-key"
	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)

	engine.Use(mw.JWTAuth(secret))
	engine.GET("/protected", func(c *gin.Context) {
		userID, _ := c.Get(KeyUserID)
		role, _ := c.Get(KeyRole)
		c.JSON(http.StatusOK, gin.H{
			"user_id": userID,
			"role":    role,
		})
	})

	token := createTestJWT(secret, jwt.MapClaims{
		"user_id": "user-123",
		"role":    "admin",
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "user-123", body["user_id"])
	assert.Equal(t, "admin", body["role"])
}

func TestJWTAuth_ShouldReturn401_WhenMissingHeader(t *testing.T) {
	mw := newTestMiddleware()
	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)

	engine.Use(mw.JWTAuth("secret"))
	engine.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuth_ShouldReturn401_WhenInvalidFormat(t *testing.T) {
	mw := newTestMiddleware()
	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)

	engine.Use(mw.JWTAuth("secret"))
	engine.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "InvalidFormat")
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuth_ShouldReturn401_WhenExpiredToken(t *testing.T) {
	mw := newTestMiddleware()
	secret := "test-secret"
	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)

	engine.Use(mw.JWTAuth(secret))
	engine.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	token := createTestJWT(secret, jwt.MapClaims{
		"user_id": "user-123",
		"exp":     time.Now().Add(-time.Hour).Unix(), // expired
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuth_ShouldReturn401_WhenWrongSecret(t *testing.T) {
	mw := newTestMiddleware()
	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)

	engine.Use(mw.JWTAuth("correct-secret"))
	engine.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	token := createTestJWT("wrong-secret", jwt.MapClaims{
		"user_id": "user-123",
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuth_ShouldReturn401_WhenAlgorithmIsNotHS256(t *testing.T) {
	mw := newTestMiddleware()
	secret := "test-secret-key"
	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)

	engine.Use(mw.JWTAuth(secret))
	engine.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	token := jwt.NewWithClaims(jwt.SigningMethodHS384, jwt.MapClaims{
		"user_id": "user-123",
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString([]byte(secret))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// --- 5.9 APIKeyAuth ---

func TestAPIKeyAuth_ShouldPass_WhenValidKey(t *testing.T) {
	mw := newTestMiddleware()
	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)

	keys := map[string]string{"service-a": "key-abc-123"}
	engine.Use(mw.APIKeyAuth(keys))
	engine.GET("/api", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/api", nil)
	req.Header.Set("X-API-Key", "key-abc-123")
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAPIKeyAuth_ShouldReturn401_WhenMissingKey(t *testing.T) {
	mw := newTestMiddleware()
	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)

	keys := map[string]string{"service-a": "key-abc-123"}
	engine.Use(mw.APIKeyAuth(keys))
	engine.GET("/api", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/api", nil)
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAPIKeyAuth_ShouldReturn401_WhenInvalidKey(t *testing.T) {
	mw := newTestMiddleware()
	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)

	keys := map[string]string{"service-a": "key-abc-123"}
	engine.Use(mw.APIKeyAuth(keys))
	engine.GET("/api", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/api", nil)
	req.Header.Set("X-API-Key", "wrong-key")
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// --- Integration: Full middleware chain ---

func TestMiddlewareChain_ShouldWorkTogether_WhenAllApplied(t *testing.T) {
	mw := newTestMiddleware()
	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)

	engine.Use(mw.RecoveryHandler())
	engine.Use(mw.NormalizeMsisdn())
	engine.Use(mw.GenerateTransactionID())
	engine.Use(mw.SetContextValues())
	engine.Use(mw.TracingHandler())
	engine.Use(mw.LogResponse())

	engine.GET("/chain", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"msisdn": c.GetString(KeyMsisdn),
			"txid":   c.GetString(KeyTransactionID),
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/chain", nil)
	req.Header.Set("x-msisdn", "081234567890")

	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	// MSISDN should be normalized
	assert.Equal(t, "6281234567890", body["msisdn"])
	// Transaction ID should be generated
	assert.NotEmpty(t, body["txid"])
}




// --- 5.10 RateLimiter ---

func TestRateLimiter_ShouldAllow_WhenUnderLimit(t *testing.T) {
	mw := newTestMiddleware()
	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)

	cfg := RateLimitConfig{
		RequestsPerSecond: 100,
		BurstSize:         100,
		CleanupInterval:   5 * time.Minute,
	}
	engine.Use(mw.RateLimiter(cfg))
	engine.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRateLimiter_ShouldReturn429_WhenOverLimit(t *testing.T) {
	mw := newTestMiddleware()
	_, engine := gin.CreateTestContext(httptest.NewRecorder())

	cfg := RateLimitConfig{
		RequestsPerSecond: 1,
		BurstSize:         2,
		CleanupInterval:   5 * time.Minute,
	}
	engine.Use(mw.RateLimiter(cfg))
	engine.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// Exhaust the burst
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		engine.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	// Next request should be rate limited
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	var body response.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "error", body.Status)
	assert.Equal(t, "rate limit exceeded", body.Error)
}

func TestRateLimiter_ShouldTrackPerIP_WhenDifferentClients(t *testing.T) {
	mw := newTestMiddleware()
	_, engine := gin.CreateTestContext(httptest.NewRecorder())

	cfg := RateLimitConfig{
		RequestsPerSecond: 1,
		BurstSize:         1,
		CleanupInterval:   5 * time.Minute,
	}
	engine.Use(mw.RateLimiter(cfg))
	engine.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// First IP uses its token
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.RemoteAddr = "10.0.0.1:12345"
	engine.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// Second IP should still have its own bucket
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.RemoteAddr = "10.0.0.2:12345"
	engine.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
}

func TestDefaultRateLimitConfig_ShouldReturnSensibleDefaults(t *testing.T) {
	cfg := DefaultRateLimitConfig()
	assert.Equal(t, 100, cfg.RequestsPerSecond)
	assert.Equal(t, 200, cfg.BurstSize)
	assert.Equal(t, 5*time.Minute, cfg.CleanupInterval)
}

func TestRateLimiter_ShouldReuseStore_WhenCalledMultipleTimes(t *testing.T) {
	mw := newTestMiddleware()

	cfg := RateLimitConfig{
		RequestsPerSecond: 100,
		BurstSize:         100,
		CleanupInterval:   5 * time.Minute,
	}

	mw.RateLimiter(cfg)
	mw.RateLimiter(cfg)

	assert.NotNil(t, mw.rateStore)
}

func TestMiddleware_ShouldCleanupStore_WhenShutdown(t *testing.T) {
	mw := newTestMiddleware()

	cfg := RateLimitConfig{
		RequestsPerSecond: 100,
		BurstSize:         100,
		CleanupInterval:   5 * time.Minute,
	}

	mw.RateLimiter(cfg)
	assert.NotNil(t, mw.rateStore)

	mw.Shutdown()
	assert.NotNil(t, mw.rateStore)
}
