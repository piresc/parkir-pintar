// Package handler provides REST handlers for the API Gateway service.
// It transcodes REST requests to gRPC calls to downstream microservices
// and maps gRPC status codes to HTTP status codes in responses.
package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"parkir-pintar/pkg/middleware"
	"parkir-pintar/pkg/response"

	paymentv1 "parkir-pintar/proto/payment/v1"
	reservationv1 "parkir-pintar/proto/reservation/v1"
	searchv1 "parkir-pintar/proto/search/v1"
)

// Handler holds gRPC client connections for downstream services and
// provides REST endpoint handlers that transcode to gRPC.
type Handler struct {
	reservation reservationv1.ReservationServiceClient
	search      searchv1.SearchServiceClient
	payment     paymentv1.PaymentServiceClient
}

// NewHandler creates a new gateway Handler with gRPC clients for each service.
func NewHandler(
	reservation reservationv1.ReservationServiceClient,
	search searchv1.SearchServiceClient,
	payment paymentv1.PaymentServiceClient,
) *Handler {
	return &Handler{
		reservation: reservation,
		search:      search,
		payment:     payment,
	}
}

// RegisterRoutes registers all REST API routes on the Gin engine with JWT auth.
func (h *Handler) RegisterRoutes(engine *gin.Engine, mw *middleware.Middleware, jwtSecret string) {
	api := engine.Group("/api/v1")
	api.Use(mw.JWTAuth(jwtSecret))

	// Reservation routes
	api.POST("/reservations", h.CreateReservation)
	api.GET("/reservations", h.ListByDriver)
	api.GET("/reservations/:id", h.GetReservation)
	api.DELETE("/reservations/:id", h.CancelReservation)
	api.POST("/reservations/:id/checkin", h.CheckIn)
	api.POST("/reservations/:id/checkout", h.CheckOut)
	api.POST("/reservations/:id/confirm", h.ConfirmReservation)
	api.POST("/reservations/:id/complete", h.CompleteCheckout)

	// Search routes
	api.GET("/availability", h.GetAvailability)
	api.GET("/floors/:floor", h.GetFloorMap)
	api.GET("/spots/:id", h.GetSpotDetails)

	// Payment routes
	api.GET("/payments/:id/status", h.GetPaymentStatus)
}

// contextWithAuth extracts the Authorization header from the Gin context
// and attaches it as gRPC metadata so downstream services can authenticate.
func contextWithAuth(c *gin.Context) context.Context {
	authHeader := c.GetHeader("Authorization")
	ctx := c.Request.Context()
	if authHeader != "" {
		md := metadata.Pairs("authorization", authHeader)
		ctx = metadata.NewOutgoingContext(ctx, md)
	}
	return ctx
}

// CreateReservation transcodes POST /api/v1/reservations to ReservationService.CreateReservation.
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

// GetReservation transcodes GET /api/v1/reservations/:id to ReservationService.GetReservation.
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

// ListByDriver transcodes GET /api/v1/reservations?driver_id=xxx&status=yyy to ReservationService.ListByDriver.
func (h *Handler) ListByDriver(c *gin.Context) {
	driverID := c.Query("driver_id")
	if driverID == "" {
		response.Error(c, http.StatusBadRequest, "driver_id query parameter is required")
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

// CancelReservation transcodes DELETE /api/v1/reservations/:id to ReservationService.CancelReservation.
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

// CheckIn transcodes POST /api/v1/reservations/:id/checkin to ReservationService.CheckIn.
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

// CheckOut transcodes POST /api/v1/reservations/:id/checkout to ReservationService.CheckOut.
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

// ConfirmReservation transcodes POST /api/v1/reservations/:id/confirm to ReservationService.ConfirmReservation.
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

// CompleteCheckout transcodes POST /api/v1/reservations/:id/complete to ReservationService.CompleteCheckout.
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

// GetAvailability transcodes GET /api/v1/availability to SearchService.GetAvailability.
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

// GetFloorMap transcodes GET /api/v1/floors/:floor to SearchService.GetFloorMap.
func (h *Handler) GetFloorMap(c *gin.Context) {
	floorStr := c.Param("floor")
	floor, err := strconv.Atoi(floorStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid floor number")
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

// GetSpotDetails transcodes GET /api/v1/spots/:id to SearchService.GetSpotDetails.
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

// GetPaymentStatus transcodes GET /api/v1/payments/:id/status to PaymentService.GetPaymentStatus.
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

// grpcCodeToHTTP maps gRPC status codes to HTTP status codes.
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

// writeGRPCError extracts the gRPC status from an error and writes the
// corresponding HTTP error response using pkg/response.
func writeGRPCError(c *gin.Context, err error) {
	st, ok := status.FromError(err)
	if !ok {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	httpCode := grpcCodeToHTTP(st.Code())
	response.Error(c, httpCode, st.Message())
}
