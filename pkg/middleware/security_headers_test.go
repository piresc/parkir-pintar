package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSecurityHeaders_ShouldSetAllHeaders_WhenCalled(t *testing.T) {
	mw := newTestMiddleware()
	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)

	engine.Use(mw.SecurityHeaders())
	engine.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
	assert.Equal(t, "0", w.Header().Get("X-XSS-Protection"))
	assert.Equal(t, "max-age=31536000; includeSubDomains", w.Header().Get("Strict-Transport-Security"))
	assert.Equal(t, "default-src 'self'", w.Header().Get("Content-Security-Policy"))
	assert.Equal(t, "strict-origin-when-cross-origin", w.Header().Get("Referrer-Policy"))
	assert.Equal(t, "camera=(), microphone=(), geolocation=()", w.Header().Get("Permissions-Policy"))
	assert.Equal(t, "no-store", w.Header().Get("Cache-Control"))
}

func TestSecurityHeaders_ShouldNotOverrideBody_WhenCalled(t *testing.T) {
	mw := newTestMiddleware()
	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)

	engine.Use(mw.SecurityHeaders())
	engine.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "hello")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "hello", w.Body.String())
}

func TestSecurityHeaders_ShouldApplyToAllMethods(t *testing.T) {
	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			mw := newTestMiddleware()
			w := httptest.NewRecorder()
			_, engine := gin.CreateTestContext(w)

			engine.Use(mw.SecurityHeaders())
			engine.Handle(method, "/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"ok": true})
			})

			req := httptest.NewRequest(method, "/test", nil)
			engine.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
			assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
		})
	}
}

func TestSecurityHeaders_ShouldBePresent_WhenHandlerErrors(t *testing.T) {
	mw := newTestMiddleware()
	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)

	engine.Use(mw.SecurityHeaders())
	engine.GET("/error", func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "something broke"})
	})

	req := httptest.NewRequest(http.MethodGet, "/error", nil)
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "no-store", w.Header().Get("Cache-Control"))
}
