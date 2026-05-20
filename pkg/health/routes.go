package health

import (
	"net/http"
	"runtime"

	"github.com/gin-gonic/gin"
)

// - GET /health/live  — liveness probe, always 200 OK
func RegisterRoutes(r *gin.Engine, serviceName, version string, svc *Service) {
	healthGroup := r.Group("/health")

	healthGroup.GET("", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			KeyStatus:      "ok",
			"service_name": serviceName,
			"version":      version,
			"go_version":   runtime.Version(),
		})
	})

	healthGroup.GET("/live", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			KeyStatus: "ok",
		})
	})

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
