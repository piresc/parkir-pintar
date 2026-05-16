package handler

import (
	"net/http"
	"net/http/pprof"
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	"parkir-pintar/pkg/middleware"
	"parkir-pintar/pkg/response"
)

// RegisterPprof conditionally registers pprof debug endpoints on the engine.
// Endpoints are only enabled when the ENABLE_PPROF environment variable is
// set to "true" (case-insensitive). Access requires admin JWT authentication.
func RegisterPprof(engine *gin.Engine, mw *middleware.Middleware, jwtSecret string) {
	if !pprofEnabled() {
		return
	}

	debug := engine.Group("/debug/pprof")
	debug.Use(mw.JWTAuth(jwtSecret))
	debug.Use(requireAdmin())

	debug.GET("/", gin.WrapF(pprof.Index))
	debug.GET("/cmdline", gin.WrapF(pprof.Cmdline))
	debug.GET("/profile", gin.WrapF(pprof.Profile))
	debug.GET("/symbol", gin.WrapF(pprof.Symbol))
	debug.POST("/symbol", gin.WrapF(pprof.Symbol))
	debug.GET("/trace", gin.WrapF(pprof.Trace))

	// Named profiles: allocs, block, goroutine, heap, mutex, threadcreate
	debug.GET("/:name", gin.WrapH(http.DefaultServeMux))

	// Register the default pprof handlers on DefaultServeMux so that
	// the named profile wildcard route works correctly.
	// net/http/pprof's init() already registers on DefaultServeMux.
}

// requireAdmin returns middleware that checks for admin role.
func requireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetString(middleware.KeyRole)
		if role != "admin" {
			c.Abort()
			response.Error(c, http.StatusForbidden, "admin access required")
			return
		}
		c.Next()
	}
}

// pprofEnabled checks whether the ENABLE_PPROF env var is set to "true".
func pprofEnabled() bool {
	return strings.EqualFold(os.Getenv("ENABLE_PPROF"), "true")
}
