package health

import (
	"net/http"
	"runtime"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers health check endpoints on the Gin engine.
// Endpoints:
//   - GET /health       — build info (service_name, version, go_version)
//   - GET /health/live  — liveness probe, always 200 OK
//   - GET /health/ready — readiness probe, runs CheckAll
//   - GET /health/detailed — per-dependency status with durations
func RegisterRoutes(r *gin.Engine, serviceName, version string, svc *Service) {
	healthGroup := r.Group("/health")

	// GET /health — build info
	healthGroup.GET("", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			KeyStatus:      "ok",
			"service_name": serviceName,
			"version":      version,
			"go_version":   runtime.Version(),
		})
	})

	// GET /health/live — liveness probe
	healthGroup.GET("/live", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			KeyStatus: "ok",
		})
	})

	// GET /health/ready — readiness probe
	healthGroup.GET("/ready", func(c *gin.Context) {
		result, err := svc.CheckAll(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				KeyStatus: StatusUnhealthy,
				"error":   err.Error(),
			})
			return
		}

		status, _ := result[KeyStatus].(string)
		if status != StatusHealthy {
			c.JSON(http.StatusServiceUnavailable, result)
			return
		}

		c.JSON(http.StatusOK, result)
	})

	// GET /health/detailed — per-dependency status
	healthGroup.GET("/detailed", func(c *gin.Context) {
		result, err := svc.CheckAll(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				KeyStatus: StatusUnhealthy,
				"error":   err.Error(),
			})
			return
		}

		status, _ := result[KeyStatus].(string)
		if status != StatusHealthy {
			c.JSON(http.StatusServiceUnavailable, result)
			return
		}

		c.JSON(http.StatusOK, result)
	})
}
