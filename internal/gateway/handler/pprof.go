package handler

import (
	"net/http"
	"net/http/pprof"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// RegisterPprof conditionally registers pprof debug endpoints on the engine.
// Endpoints are only enabled when the ENABLE_PPROF environment variable is
// set to "true" (case-insensitive). This keeps profiling disabled by default
// in production while allowing operators to opt in for debugging.
func RegisterPprof(engine *gin.Engine) {
	if !pprofEnabled() {
		return
	}

	debug := engine.Group("/debug/pprof")
	{
		debug.GET("/", gin.WrapF(pprof.Index))
		debug.GET("/cmdline", gin.WrapF(pprof.Cmdline))
		debug.GET("/profile", gin.WrapF(pprof.Profile))
		debug.GET("/symbol", gin.WrapF(pprof.Symbol))
		debug.POST("/symbol", gin.WrapF(pprof.Symbol))
		debug.GET("/trace", gin.WrapF(pprof.Trace))

		// Named profiles: allocs, block, goroutine, heap, mutex, threadcreate
		debug.GET("/:name", gin.WrapH(http.DefaultServeMux))
	}

	// Register the default pprof handlers on DefaultServeMux so that
	// the named profile wildcard route works correctly.
	// net/http/pprof's init() already registers on DefaultServeMux.
}

// pprofEnabled checks whether the ENABLE_PPROF env var is set to "true".
func pprofEnabled() bool {
	return strings.EqualFold(os.Getenv("ENABLE_PPROF"), "true")
}
