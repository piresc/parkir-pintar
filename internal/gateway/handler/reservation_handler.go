package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"parkir-pintar/pkg/response"

	reservationv1 "parkir-pintar/proto/reservation/v1"
)

func (h *Handler) CreateReservation(c *gin.Context) {
	var req struct {
		DriverID       string `json:"driver_id"`
		VehicleType    string `json:"vehicle_type"`
		AssignmentMode string `json:"assignment_mode"`
		SpotID         string `json:"spot_id,omitempty"`
		IdempotencyKey string `json:"idempotency_key"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	// Enforce ownership: driver_id must match authenticated user (fail closed)
	uid := getUserID(c)
	if uid == "" {
		response.Error(c, http.StatusUnauthorized, "user identity not found")
		return
	}
	if req.DriverID != uid {
		response.Error(c, http.StatusForbidden, "cannot create reservation for another driver")
		return
	}

	resp, err := h.reservation.CreateReservation(contextWithAuth(c), &reservationv1.CreateReservationRequest{
		DriverId:       req.DriverID,
		VehicleType:    req.VehicleType,
		AssignmentMode: req.AssignmentMode,
		SpotId:         req.SpotID,
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	response.Success(c, http.StatusCreated, resp)
}

func (h *Handler) GetReservation(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "reservation id is required")
		return
	}

	resp, err := h.reservation.GetReservation(contextWithAuth(c), &reservationv1.GetReservationRequest{
		ReservationId: id,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	response.Success(c, http.StatusOK, resp)
}

func (h *Handler) ListByDriver(c *gin.Context) {
	driverID := getUserID(c)
	if driverID == "" {
		response.Error(c, http.StatusUnauthorized, "user identity not found")
		return
	}
	statusFilter := c.Query("status")

	resp, err := h.reservation.ListByDriver(contextWithAuth(c), &reservationv1.ListByDriverRequest{
		DriverId: driverID,
		Status:   statusFilter,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	response.Success(c, http.StatusOK, resp)
}

func (h *Handler) CancelReservation(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "reservation id is required")
		return
	}

	userID := getUserID(c)
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "user identity not found")
		return
	}

	// Fetch reservation to verify ownership
	getResp, err := h.reservation.GetReservation(contextWithAuth(c), &reservationv1.GetReservationRequest{
		ReservationId: id,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}
	if getResp.GetDriverId() != userID {
		response.Error(c, http.StatusForbidden, "not authorized to modify this reservation")
		return
	}

	resp, err := h.reservation.CancelReservation(contextWithAuth(c), &reservationv1.CancelReservationRequest{
		ReservationId: id,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	response.Success(c, http.StatusOK, resp)
}

func (h *Handler) CheckIn(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "reservation id is required")
		return
	}

	userID := getUserID(c)
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "user identity not found")
		return
	}

	// Fetch reservation to verify ownership
	getResp, err := h.reservation.GetReservation(contextWithAuth(c), &reservationv1.GetReservationRequest{
		ReservationId: id,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}
	if getResp.GetDriverId() != userID {
		response.Error(c, http.StatusForbidden, "not authorized to modify this reservation")
		return
	}

	resp, err := h.reservation.CheckIn(contextWithAuth(c), &reservationv1.CheckInRequest{
		ReservationId: id,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	response.Success(c, http.StatusOK, resp)
}

func (h *Handler) CheckOut(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "reservation id is required")
		return
	}

	userID := getUserID(c)
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "user identity not found")
		return
	}

	// Fetch reservation to verify ownership
	getResp, err := h.reservation.GetReservation(contextWithAuth(c), &reservationv1.GetReservationRequest{
		ReservationId: id,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}
	if getResp.GetDriverId() != userID {
		response.Error(c, http.StatusForbidden, "not authorized to modify this reservation")
		return
	}

	resp, err := h.reservation.CheckOut(contextWithAuth(c), &reservationv1.CheckOutRequest{
		ReservationId: id,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	response.Success(c, http.StatusOK, resp)
}

func (h *Handler) ConfirmReservation(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "reservation id is required")
		return
	}

	userID := getUserID(c)
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "user identity not found")
		return
	}

	// Fetch reservation to verify ownership
	getResp, err := h.reservation.GetReservation(contextWithAuth(c), &reservationv1.GetReservationRequest{
		ReservationId: id,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}
	if getResp.GetDriverId() != userID {
		response.Error(c, http.StatusForbidden, "not authorized to modify this reservation")
		return
	}

	resp, err := h.reservation.ConfirmReservation(contextWithAuth(c), &reservationv1.ConfirmReservationRequest{
		ReservationId: id,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	response.Success(c, http.StatusOK, resp)
}

func (h *Handler) CompleteCheckout(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "reservation id is required")
		return
	}

	userID := getUserID(c)
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "user identity not found")
		return
	}

	// Fetch reservation to verify ownership
	getResp, err := h.reservation.GetReservation(contextWithAuth(c), &reservationv1.GetReservationRequest{
		ReservationId: id,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}
	if getResp.GetDriverId() != userID {
		response.Error(c, http.StatusForbidden, "not authorized to modify this reservation")
		return
	}

	resp, err := h.reservation.CompleteCheckout(contextWithAuth(c), &reservationv1.CompleteCheckoutRequest{
		ReservationId: id,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	response.Success(c, http.StatusOK, resp)
}
