package handler

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

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
}

func NewHandler(
	reservation reservationv1.ReservationServiceClient,
	search searchv1.SearchServiceClient,
	payment paymentv1.PaymentServiceClient,
	presence presencev1.PresenceServiceClient,
) *Handler {
	return &Handler{
		reservation: reservation,
		search:      search,
		payment:     payment,
		presence:    presence,
	}
}

// The JWT middleware guarantees this is always set for authenticated routes.
func getUserID(c *gin.Context) string {
	return c.GetString(middleware.KeyUserID)
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
