// Package handler — analytics REST endpoints for the API Gateway.
// These endpoints transcode REST requests to gRPC calls to the analytics service.
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"parkir-pintar/pkg/middleware"
	"parkir-pintar/pkg/response"
	analyticsv1 "parkir-pintar/proto/analytics/v1"
)

// AnalyticsHandler provides REST endpoints for analytics reporting via gRPC.
type AnalyticsHandler struct {
	client analyticsv1.AnalyticsServiceClient
}

// NewAnalyticsHandler creates a new AnalyticsHandler with the given gRPC client.
func NewAnalyticsHandler(client analyticsv1.AnalyticsServiceClient) *AnalyticsHandler {
	return &AnalyticsHandler{client: client}
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
	resp, err := ah.client.GetPeakHours(contextWithAuth(c), &analyticsv1.GetPeakHoursRequest{})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	response.Success(c, http.StatusOK, resp)
}

// GetOccupancy handles GET /api/v1/analytics/occupancy.
// Returns daily occupancy/usage patterns summarized over the last 30 days.
func (ah *AnalyticsHandler) GetOccupancy(c *gin.Context) {
	resp, err := ah.client.GetUsagePatterns(contextWithAuth(c), &analyticsv1.GetUsagePatternsRequest{})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	response.Success(c, http.StatusOK, resp)
}
