package handler

import (
	"context"
	"log/slog"
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"parkir-pintar/pkg/auth"
	"parkir-pintar/pkg/config"
	"parkir-pintar/pkg/middleware"
	"parkir-pintar/pkg/response"

	paymentv1 "parkir-pintar/proto/payment/v1"
	presencev1 "parkir-pintar/proto/presence/v1"
	reservationv1 "parkir-pintar/proto/reservation/v1"
	searchv1 "parkir-pintar/proto/search/v1"
)

type Handler struct {
	reservation reservationv1.ReservationServiceClient
	search      searchv1.SearchServiceClient
	payment     paymentv1.PaymentServiceClient
	presence    presencev1.PresenceServiceClient
	jwtCfg      config.JWTConfig
}

func NewHandler(
	reservation reservationv1.ReservationServiceClient,
	search searchv1.SearchServiceClient,
	payment paymentv1.PaymentServiceClient,
	presence presencev1.PresenceServiceClient,
	jwtCfg config.JWTConfig,
) *Handler {
	return &Handler{
		reservation: reservation,
		search:      search,
		payment:     payment,
		presence:    presence,
		jwtCfg:      jwtCfg,
	}
}

// The JWT middleware guarantees this is always set for authenticated routes.
func getUserID(c *gin.Context) string {
	return c.GetString(middleware.KeyUserID)
}

func (h *Handler) Login(c *gin.Context) {
	var req struct {
		DriverID string `json:"driver_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.DriverID == "" {
		response.Error(c, http.StatusBadRequest, "driver_id is required")
		return
	}

	token, expiresAt, err := auth.GenerateToken(req.DriverID, "driver", h.jwtCfg)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to generate token")
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"token":      token,
		"expires_at": expiresAt,
		"driver_id":  req.DriverID,
	})
}

func contextWithAuth(c *gin.Context) context.Context {
	ctx := c.Request.Context()
	authHeader := c.GetHeader("Authorization")
	userID := getUserID(c)

	pairs := []string{}
	if authHeader != "" {
		pairs = append(pairs, "authorization", authHeader)
	}
	if userID != "" {
		pairs = append(pairs, "x-user-id", userID)
	}

	if len(pairs) > 0 {
		md := metadata.Pairs(pairs...)
		ctx = metadata.NewOutgoingContext(ctx, md)
	}
	return ctx
}

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

	resp, err := h.reservation.CompleteCheckout(contextWithAuth(c), &reservationv1.CompleteCheckoutRequest{
		ReservationId: id,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	response.Success(c, http.StatusOK, resp)
}

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

func (h *Handler) GetPaymentStatus(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "payment id is required")
		return
	}

	resp, err := h.payment.GetPaymentStatus(contextWithAuth(c), &paymentv1.GetPaymentStatusRequest{
		PaymentId: id,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	response.Success(c, http.StatusOK, resp)
}

type streamPresenceRequest struct {
	ReservationID string  `json:"reservation_id" binding:"required"`
	Latitude      float64 `json:"latitude"`
	Longitude     float64 `json:"longitude"`
	Accuracy      float64 `json:"accuracy"`
}

func (h *Handler) StreamPresence(c *gin.Context) {
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

func grpcCodeToHTTP(code codes.Code) int {
	switch code {
	case codes.OK:
		return http.StatusOK
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.FailedPrecondition:
		return http.StatusPreconditionFailed
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout
	default:
		return http.StatusInternalServerError
	}
}

func writeGRPCError(c *gin.Context, err error) {
	st, ok := status.FromError(err)
	if !ok {
		slog.Error("non-gRPC error in gateway", "error", err)
		response.Error(c, http.StatusInternalServerError, "internal server error")
		return
	}
	httpCode := grpcCodeToHTTP(st.Code())
	response.Error(c, httpCode, st.Message())
}
