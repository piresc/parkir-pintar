// Package handler provides preservation property tests for non-StreamLocation gateway endpoints.
//
// Best practices applied (from Go testify coding standards KB):
// - Test naming: Test[FunctionName]_Should[ExpectedResult]_When[Condition]
// - AAA pattern: Arrange → Act → Assert
// - testify/assert for assertions
// - Mock at interface boundaries rather than concrete implementations
// - Keep mocks simple and focused on the behavior being tested
//
// **Validates: Requirements 3.5** (Preservation Property 14 from design)
//
// Non-bug condition: endpoint != StreamLocation
// These tests verify that all other gateway REST endpoints continue to transcode
// REST→gRPC correctly on unfixed code. They must PASS on unfixed code.
package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	paymentv1 "parkir-pintar/proto/payment/v1"
	reservationv1 "parkir-pintar/proto/reservation/v1"
	searchv1 "parkir-pintar/proto/search/v1"

	"pgregory.net/rapid"
)

// --- Mock gRPC clients for preservation tests ---

type preservationReservationClient struct {
	reservationv1.ReservationServiceClient
	lastCreateReq  *reservationv1.CreateReservationRequest
	lastCancelID   string
	lastCheckInID  string
	lastCheckOutID string
}

func (m *preservationReservationClient) CreateReservation(_ context.Context, in *reservationv1.CreateReservationRequest, _ ...grpc.CallOption) (*reservationv1.ReservationResponse, error) {
	m.lastCreateReq = in
	return &reservationv1.ReservationResponse{Id: "res-123", Status: "confirmed"}, nil
}

func (m *preservationReservationClient) CancelReservation(_ context.Context, in *reservationv1.CancelReservationRequest, _ ...grpc.CallOption) (*reservationv1.ReservationResponse, error) {
	m.lastCancelID = in.GetReservationId()
	return &reservationv1.ReservationResponse{Id: in.GetReservationId(), Status: "cancelled"}, nil
}

func (m *preservationReservationClient) CheckIn(_ context.Context, in *reservationv1.CheckInRequest, _ ...grpc.CallOption) (*reservationv1.ReservationResponse, error) {
	m.lastCheckInID = in.GetReservationId()
	return &reservationv1.ReservationResponse{Id: in.GetReservationId(), Status: "checked_in"}, nil
}

func (m *preservationReservationClient) CheckOut(_ context.Context, in *reservationv1.CheckOutRequest, _ ...grpc.CallOption) (*reservationv1.CheckOutResponse, error) {
	m.lastCheckOutID = in.GetReservationId()
	return &reservationv1.CheckOutResponse{}, nil
}

type preservationSearchClient struct {
	searchv1.SearchServiceClient
	lastAvailVehicleType string
	lastFloorNumber      int32
	lastSpotID           string
}

func (m *preservationSearchClient) GetAvailability(_ context.Context, in *searchv1.GetAvailabilityRequest, _ ...grpc.CallOption) (*searchv1.AvailabilityResponse, error) {
	m.lastAvailVehicleType = in.GetVehicleType()
	return &searchv1.AvailabilityResponse{
		Floors: []*searchv1.FloorAvailability{},
		Total:  &searchv1.AvailabilitySummary{},
	}, nil
}

func (m *preservationSearchClient) GetFloorMap(_ context.Context, in *searchv1.GetFloorMapRequest, _ ...grpc.CallOption) (*searchv1.FloorMapResponse, error) {
	m.lastFloorNumber = in.GetFloorNumber()
	return &searchv1.FloorMapResponse{Spots: []*searchv1.SpotInfo{}}, nil
}

func (m *preservationSearchClient) GetSpotDetails(_ context.Context, in *searchv1.GetSpotDetailsRequest, _ ...grpc.CallOption) (*searchv1.SpotDetailsResponse, error) {
	m.lastSpotID = in.GetSpotId()
	return &searchv1.SpotDetailsResponse{Id: in.GetSpotId(), Status: "available"}, nil
}

type preservationPaymentClient struct {
	paymentv1.PaymentServiceClient
	lastPaymentID string
}

func (m *preservationPaymentClient) GetPaymentStatus(_ context.Context, in *paymentv1.GetPaymentStatusRequest, _ ...grpc.CallOption) (*paymentv1.PaymentResponse, error) {
	m.lastPaymentID = in.GetPaymentId()
	return &paymentv1.PaymentResponse{Id: in.GetPaymentId(), Status: "success"}, nil
}

// TestCreateReservation_ShouldTranscodeCorrectly_WhenValidRequest verifies
// that CreateReservation correctly transcodes REST→gRPC fields.
// Non-bug condition: endpoint != StreamLocation.
//
// **Validates: Requirements 3.5**
func TestCreateReservation_ShouldTranscodeCorrectly_WhenValidRequest(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Arrange
		driverID := rapid.StringMatching(`[a-z0-9]{8}`).Draw(t, "driverID")
		vehicleType := rapid.SampledFrom([]string{"car", "motorcycle"}).Draw(t, "vehicleType")
		idempotencyKey := rapid.StringMatching(`[a-z0-9]{16}`).Draw(t, "idempotencyKey")

		resSpy := &preservationReservationClient{}
		h := NewHandler(resSpy, nil, nil)

		gin.SetMode(gin.TestMode)
		engine := gin.New()
		engine.POST("/api/v1/reservations", h.CreateReservation)

		body, _ := json.Marshal(map[string]string{
			"driver_id":       driverID,
			"vehicle_type":    vehicleType,
			"assignment_mode": "auto",
			"idempotency_key": idempotencyKey,
		})

		req := httptest.NewRequest(http.MethodPost, "/api/v1/reservations", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Act
		engine.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusCreated, w.Code)
		require.NotNil(t, resSpy.lastCreateReq)
		assert.Equal(t, driverID, resSpy.lastCreateReq.GetDriverId())
		assert.Equal(t, vehicleType, resSpy.lastCreateReq.GetVehicleType())
		assert.Equal(t, idempotencyKey, resSpy.lastCreateReq.GetIdempotencyKey())
	})
}

// TestCancelReservation_ShouldPassID_WhenCalled verifies CancelReservation
// correctly passes the reservation ID to the gRPC client.
// Non-bug condition: endpoint != StreamLocation.
//
// **Validates: Requirements 3.5**
func TestCancelReservation_ShouldPassID_WhenCalled(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Arrange
		resID := rapid.StringMatching(`[a-z0-9]{8}`).Draw(t, "resID")

		resSpy := &preservationReservationClient{}
		h := NewHandler(resSpy, nil, nil)

		gin.SetMode(gin.TestMode)
		engine := gin.New()
		engine.DELETE("/api/v1/reservations/:id", h.CancelReservation)

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/reservations/"+resID, nil)
		w := httptest.NewRecorder()

		// Act
		engine.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, resID, resSpy.lastCancelID)
	})
}

// TestCheckIn_ShouldPassID_WhenCalled verifies CheckIn correctly passes
// the reservation ID to the gRPC client.
// Non-bug condition: endpoint != StreamLocation.
//
// **Validates: Requirements 3.5**
func TestCheckIn_ShouldPassID_WhenCalled(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Arrange
		resID := rapid.StringMatching(`[a-z0-9]{8}`).Draw(t, "resID")

		resSpy := &preservationReservationClient{}
		h := NewHandler(resSpy, nil, nil)

		gin.SetMode(gin.TestMode)
		engine := gin.New()
		engine.POST("/api/v1/reservations/:id/checkin", h.CheckIn)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/reservations/"+resID+"/checkin", nil)
		w := httptest.NewRecorder()

		// Act
		engine.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, resID, resSpy.lastCheckInID)
	})
}

// TestCheckOut_ShouldPassID_WhenCalled verifies CheckOut correctly passes
// the reservation ID to the gRPC client.
// Non-bug condition: endpoint != StreamLocation.
//
// **Validates: Requirements 3.5**
func TestCheckOut_ShouldPassID_WhenCalled(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Arrange
		resID := rapid.StringMatching(`[a-z0-9]{8}`).Draw(t, "resID")

		resSpy := &preservationReservationClient{}
		h := NewHandler(resSpy, nil, nil)

		gin.SetMode(gin.TestMode)
		engine := gin.New()
		engine.POST("/api/v1/reservations/:id/checkout", h.CheckOut)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/reservations/"+resID+"/checkout", nil)
		w := httptest.NewRecorder()

		// Act
		engine.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, resID, resSpy.lastCheckOutID)
	})
}

// TestGetAvailability_ShouldPassVehicleType_WhenCalled verifies GetAvailability
// correctly passes the vehicle_type query param to the gRPC client.
// Non-bug condition: endpoint != StreamLocation.
//
// **Validates: Requirements 3.5**
func TestGetAvailability_ShouldPassVehicleType_WhenCalled(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Arrange
		vehicleType := rapid.SampledFrom([]string{"car", "motorcycle"}).Draw(t, "vehicleType")

		searchSpy := &preservationSearchClient{}
		h := NewHandler(nil, searchSpy, nil)

		gin.SetMode(gin.TestMode)
		engine := gin.New()
		engine.GET("/api/v1/availability", h.GetAvailability)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/availability?vehicle_type="+vehicleType, nil)
		w := httptest.NewRecorder()

		// Act
		engine.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, vehicleType, searchSpy.lastAvailVehicleType)
	})
}

// TestGetPaymentStatus_ShouldPassPaymentID_WhenCalled verifies GetPaymentStatus
// correctly passes the payment ID to the gRPC client.
// Non-bug condition: endpoint != StreamLocation.
//
// **Validates: Requirements 3.5**
func TestGetPaymentStatus_ShouldPassPaymentID_WhenCalled(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Arrange
		paymentID := rapid.StringMatching(`[a-z0-9]{8}`).Draw(t, "paymentID")

		paySpy := &preservationPaymentClient{}
		h := NewHandler(nil, nil, paySpy)

		gin.SetMode(gin.TestMode)
		engine := gin.New()
		engine.GET("/api/v1/payments/:id/status", h.GetPaymentStatus)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/payments/"+paymentID+"/status", nil)
		w := httptest.NewRecorder()

		// Act
		engine.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, paymentID, paySpy.lastPaymentID)
	})
}
