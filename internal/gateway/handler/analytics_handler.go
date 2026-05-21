package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"parkir-pintar/pkg/middleware"
	"parkir-pintar/pkg/response"
	analyticsv1 "parkir-pintar/proto/analytics/v1"
)

type AnalyticsHandler struct {
	client analyticsv1.AnalyticsServiceClient
}

func NewAnalyticsHandler(client analyticsv1.AnalyticsServiceClient) *AnalyticsHandler {
	return &AnalyticsHandler{client: client}
}

func (ah *AnalyticsHandler) RegisterRoutes(engine *gin.Engine, mw *middleware.Middleware, jwtSecret string) {
	api := engine.Group("/api/v1/analytics")
	api.Use(mw.JWTAuth(jwtSecret))

	api.GET("/peak-hours", ah.GetPeakHours)
	api.GET("/occupancy", ah.GetOccupancy)
	api.GET("/predictions", ah.GetPredictions)
}

func (ah *AnalyticsHandler) GetPeakHours(c *gin.Context) {
	resp, err := ah.client.GetPeakHours(contextWithAuth(c), &analyticsv1.GetPeakHoursRequest{})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	response.Success(c, http.StatusOK, resp)
}

func (ah *AnalyticsHandler) GetOccupancy(c *gin.Context) {
	resp, err := ah.client.GetUsagePatterns(contextWithAuth(c), &analyticsv1.GetUsagePatternsRequest{})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	response.Success(c, http.StatusOK, resp)
}

func (ah *AnalyticsHandler) GetPredictions(c *gin.Context) {
	var horizonMinutes int32
	if h := c.Query("horizon_minutes"); h != "" {
		if v, err := strconv.Atoi(h); err == nil {
			horizonMinutes = int32(v)
		}
	}

	resp, err := ah.client.PredictResources(contextWithAuth(c), &analyticsv1.PredictResourcesRequest{
		HorizonMinutes: horizonMinutes,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	response.Success(c, http.StatusOK, resp)
}
