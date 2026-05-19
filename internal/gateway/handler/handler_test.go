// Package handler provides tests for the API Gateway REST handlers.
//
// Best practices applied (from Go coding standards KB):
// - Use table-driven tests with descriptive names
// - Use t.Context() for test context (Go 1.24+)
// - Use httptest.NewRecorder and gin.CreateTestContext for HTTP testing
// - Test JWT validation, routing, and gRPC-to-HTTP error mapping
package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"parkir-pintar/pkg/config"
	"parkir-pintar/pkg/middleware"
	"parkir-pintar/pkg/response"

	paymentv1 "parkir-pintar/proto/payment/v1"
	searchv1 "parkir-pintar/proto/search/v1"
)

const testJWTSecret = "test-secret-key-for-gateway-tests"

// generateTestToken creates a valid JWT token for testing.
func generateTestToken(t *testing.T) string {
	t.Helper()
	claims := jwt.MapClaims{
		"user_id": "test-user-123",
		"role":    "driver",
		"exp":     time.Now().Add(1 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(testJWTSecret))
	require.NoError(t, err)
	return signed
}

// setupTestRouter creates a Gin engine with the gateway routes and JWT middleware.
func setupTestRouter(t *testing.T, h *Handler) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := gin.New()

	cfg := &config.Config{}
	mw := middleware.NewMiddleware(cfg, nil, nil)
	h.RegisterRoutes(engine, mw, testJWTSecret)
	return engine
}

func TestJWTValidation_MissingToken(t *testing.T) {
	h := NewHandler(nil, nil, nil, nil, config.JWTConfig{Secret: testJWTSecret, Expiration: 60, Issuer: "parkir-pintar"})
	router := setupTestRouter(t, h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/availability?vehicle_type=car", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp response.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "error", resp.Status)
}

func TestJWTValidation_InvalidToken(t *testing.T) {
	h := NewHandler(nil, nil, nil, nil, config.JWTConfig{Secret: testJWTSecret, Expiration: 60, Issuer: "parkir-pintar"})
	router := setupTestRouter(t, h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/availability?vehicle_type=car", nil)
	req.Header.Set("Authorization", "Bearer invalid-token-string")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGRPCCodeToHTTP_Mapping(t *testing.T) {
	tests := []struct {
		grpcCode codes.Code
		httpCode int
	}{
		{codes.OK, http.StatusOK},
		{codes.InvalidArgument, http.StatusBadRequest},
		{codes.NotFound, http.StatusNotFound},
		{codes.AlreadyExists, http.StatusConflict},
		{codes.PermissionDenied, http.StatusForbidden},
		{codes.Unauthenticated, http.StatusUnauthorized},
		{codes.FailedPrecondition, http.StatusPreconditionFailed},
		{codes.ResourceExhausted, http.StatusTooManyRequests},
		{codes.Unavailable, http.StatusServiceUnavailable},
		{codes.DeadlineExceeded, http.StatusGatewayTimeout},
		{codes.Internal, http.StatusInternalServerError},
		{codes.Unimplemented, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.grpcCode.String(), func(t *testing.T) {
			got := grpcCodeToHTTP(tt.grpcCode)
			assert.Equal(t, tt.httpCode, got)
		})
	}
}

func TestWriteGRPCError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	grpcErr := status.Error(codes.NotFound, "reservation not found")
	writeGRPCError(c, grpcErr)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp response.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "error", resp.Status)
	assert.Equal(t, "reservation not found", resp.Error)
}

func TestCreateReservation_InvalidBody(t *testing.T) {
	h := NewHandler(nil, nil, nil, nil, config.JWTConfig{Secret: testJWTSecret, Expiration: 60, Issuer: "parkir-pintar"})
	router := setupTestRouter(t, h)

	token := generateTestToken(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/reservations", strings.NewReader("not-json"))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetFloorMap_InvalidFloor(t *testing.T) {
	h := NewHandler(nil, nil, nil, nil, config.JWTConfig{Secret: testJWTSecret, Expiration: 60, Issuer: "parkir-pintar"})
	router := setupTestRouter(t, h)

	token := generateTestToken(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/floors/abc", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestRouteRegistration verifies all expected routes are registered.
func TestRouteRegistration(t *testing.T) {
	h := NewHandler(nil, nil, nil, nil, config.JWTConfig{Secret: testJWTSecret, Expiration: 60, Issuer: "parkir-pintar"})
	router := setupTestRouter(t, h)

	routes := router.Routes()
	routeMap := make(map[string]string)
	for _, r := range routes {
		routeMap[r.Method+" "+r.Path] = r.Handler
	}

	expectedRoutes := []string{
		"POST /api/v1/reservations",
		"DELETE /api/v1/reservations/:id",
		"POST /api/v1/reservations/:id/checkin",
		"POST /api/v1/reservations/:id/checkout",
		"GET /api/v1/availability",
		"GET /api/v1/floors/:floor",
		"GET /api/v1/spots/:id",
		"GET /api/v1/payments/:id/status",
	}

	for _, route := range expectedRoutes {
		_, exists := routeMap[route]
		assert.True(t, exists, "route %s should be registered", route)
	}
}

// TestGetAvailability_ValidRequest verifies the route accepts a valid request
// (will fail at gRPC level since client is nil, but validates routing works).
func TestGetAvailability_NilClient(t *testing.T) {
	// With nil search client, calling the handler should panic or return error.
	// We just verify the JWT auth passes and the handler is reached.
	h := NewHandler(nil, &mockSearchClient{}, nil, nil, config.JWTConfig{Secret: testJWTSecret, Expiration: 60, Issuer: "parkir-pintar"})
	router := setupTestRouter(t, h)

	token := generateTestToken(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/availability?vehicle_type=car", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// The mock returns a valid response
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestGetPaymentStatus_NilClient verifies payment status route with mock.
func TestGetPaymentStatus_MockClient(t *testing.T) {
	h := NewHandler(nil, nil, &mockPaymentClient{}, nil, config.JWTConfig{Secret: testJWTSecret, Expiration: 60, Issuer: "parkir-pintar"})
	router := setupTestRouter(t, h)

	token := generateTestToken(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/payments/pay-123/status", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// --- Mock gRPC clients for testing ---

type mockSearchClient struct {
	searchv1.SearchServiceClient
}

func (m *mockSearchClient) GetAvailability(_ context.Context, _ *searchv1.GetAvailabilityRequest, _ ...grpc.CallOption) (*searchv1.AvailabilityResponse, error) {
	return &searchv1.AvailabilityResponse{
		Floors: []*searchv1.FloorAvailability{},
		Total:  &searchv1.AvailabilitySummary{},
	}, nil
}

type mockPaymentClient struct {
	paymentv1.PaymentServiceClient
}

func (m *mockPaymentClient) GetPaymentStatus(_ context.Context, _ *paymentv1.GetPaymentStatusRequest, _ ...grpc.CallOption) (*paymentv1.PaymentResponse, error) {
	return &paymentv1.PaymentResponse{
		Id:     "pay-123",
		Status: "success",
	}, nil
}
