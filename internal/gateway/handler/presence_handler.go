package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"parkir-pintar/pkg/response"

	presencev1 "parkir-pintar/proto/presence/v1"
)

type streamPresenceRequest struct {
	ReservationID string  `json:"reservation_id" binding:"required"`
	Latitude      float64 `json:"latitude"`
	Longitude     float64 `json:"longitude"`
	Accuracy      float64 `json:"accuracy"`
}

func (h *Handler) StreamPresence(c *gin.Context) {
	if h.presence == nil {
		response.Error(c, http.StatusServiceUnavailable, "presence service unavailable")
		return
	}

	var req streamPresenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	driverID := getUserID(c)

	resp, err := h.presence.VerifyPresence(contextWithAuth(c), &presencev1.VerifyPresenceRequest{
		DriverId:      driverID,
		ReservationId: req.ReservationID,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	response.Success(c, http.StatusOK, map[string]interface{}{
		"is_geofenced": resp.GetVerified(),
		"message":      resp.GetMessage(),
	})
}
