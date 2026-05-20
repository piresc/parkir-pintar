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

	debug.GET("/:name", gin.WrapH(http.DefaultServeMux))

}

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

func pprofEnabled() bool {
	return strings.EqualFold(os.Getenv("ENABLE_PPROF"), "true")
}
