package handler

import (
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"parkir-pintar/pkg/response"

	searchv1 "parkir-pintar/proto/search/v1"
)

func (h *Handler) GetAvailability(c *gin.Context) {
	vehicleType := c.Query("vehicle_type")

	resp, err := h.search.GetAvailability(contextWithAuth(c), &searchv1.GetAvailabilityRequest{
		VehicleType: vehicleType,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	response.Success(c, http.StatusOK, resp)
}

func (h *Handler) GetFloorMap(c *gin.Context) {
	floorStr := c.Param("floor")
	floor, err := strconv.Atoi(floorStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid floor number")
		return
	}
	if floor < 0 || floor > math.MaxInt32 {
		response.Error(c, http.StatusBadRequest, "floor number out of range")
		return
	}

	resp, grpcErr := h.search.GetFloorMap(contextWithAuth(c), &searchv1.GetFloorMapRequest{
		FloorNumber: int32(floor),
	})
	if grpcErr != nil {
		writeGRPCError(c, grpcErr)
		return
	}

	response.Success(c, http.StatusOK, resp)
}

func (h *Handler) GetSpotDetails(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "spot id is required")
		return
	}

	resp, err := h.search.GetSpotDetails(contextWithAuth(c), &searchv1.GetSpotDetailsRequest{
		SpotId: id,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	response.Success(c, http.StatusOK, resp)
}
