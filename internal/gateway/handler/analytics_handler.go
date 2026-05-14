// Package handler — analytics REST endpoints for the API Gateway.
// These endpoints are DB-backed (not gRPC) and expose reporting data
// from the analytics usecase layer.
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"parkir-pintar/internal/analytics/usecase"
	"parkir-pintar/pkg/middleware"
	"parkir-pintar/pkg/response"
)

// AnalyticsHandler provides REST endpoints for analytics reporting.
type AnalyticsHandler struct {
	uc usecase.Usecase
}

// NewAnalyticsHandler creates a new AnalyticsHandler with the given usecase.
func NewAnalyticsHandler(uc usecase.Usecase) *AnalyticsHandler {
	return &AnalyticsHandler{uc: uc}
}

// RegisterRoutes registers analytics REST routes on the Gin engine with JWT auth.
func (ah *AnalyticsHandler) RegisterRoutes(engine *gin.Engine, mw *middleware.Middleware, jwtSecret string) {
	api := engine.Group("/api/v1/analytics")
	api.Use(mw.JWTAuth(jwtSecret))

	api.GET("/peak-hours", ah.GetPeakHours)
	api.GET("/occupancy", ah.GetOccupancy)
}

// GetPeakHours handles GET /api/v1/analytics/peak-hours.
// Returns peak hour statistics from the last 30 days of reservation data.
func (ah *AnalyticsHandler) GetPeakHours(c *gin.Context) {
	stats, err := ah.uc.GetPeakHours(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, http.StatusOK, stats)
}

// GetOccupancy handles GET /api/v1/analytics/occupancy.
// Returns daily occupancy/usage patterns summarized over the last 30 days.
func (ah *AnalyticsHandler) GetOccupancy(c *gin.Context) {
	patterns, err := ah.uc.GetUsagePatterns(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, http.StatusOK, patterns)
}
