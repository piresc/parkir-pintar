package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRequireRole_ShouldAllow_WhenAdminAccessesAdminRoute(t *testing.T) {
	mw := newTestMiddleware()
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(KeyRole, "admin")
		c.Next()
	})
	router.Use(mw.RequireRole(RoleAdmin))
	router.GET("/admin", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/admin", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireRole_ShouldDeny_WhenDriverAccessesAdminRoute(t *testing.T) {
	mw := newTestMiddleware()
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(KeyRole, "driver")
		c.Next()
	})
	router.Use(mw.RequireRole(RoleAdmin))
	router.GET("/admin", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/admin", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	assert.NoError(t, err)
	assert.Contains(t, body["error"], "insufficient permissions")
}

func TestRequireRole_ShouldAllow_WhenAnyRoleMatches(t *testing.T) {
	mw := newTestMiddleware()
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(KeyRole, "operator")
		c.Next()
	})
	router.Use(mw.RequireRole(RoleAdmin, RoleOperator))
	router.GET("/manage", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/manage", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireRole_ShouldDeny_WhenRoleClaimMissing(t *testing.T) {
	mw := newTestMiddleware()
	router := gin.New()
	// No role set in context
	router.Use(mw.RequireRole(RoleAdmin))
	router.GET("/admin", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/admin", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	assert.NoError(t, err)
	assert.Contains(t, body["error"], "missing role claim")
}

func TestRequireAnyRole_ShouldAllow_WhenDriverInDriverOrOperator(t *testing.T) {
	mw := newTestMiddleware()
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(KeyRole, "driver")
		c.Next()
	})
	router.Use(mw.RequireAnyRole(RoleDriver, RoleOperator))
	router.GET("/parking", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/parking", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireRole_ShouldDeny_WhenRoleClaimIsEmptyString(t *testing.T) {
	mw := newTestMiddleware()
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(KeyRole, "")
		c.Next()
	})
	router.Use(mw.RequireRole(RoleAdmin))
	router.GET("/admin", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/admin", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}
