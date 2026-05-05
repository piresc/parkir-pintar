// Package handler provides bug condition exploration tests for gateway StreamLocation correct RPC.
//
// Best practices applied (from Go testify coding standards KB):
// - Test naming: Test[FunctionName]_Should[ExpectedResult]_When[Condition]
// - AAA pattern: Arrange → Act → Assert
// - testify/assert and testify/require for assertions
// - Mock at interface boundaries rather than concrete implementations
// - Keep mocks simple and focused on the behavior being tested
//
// **Validates: Requirements 2.8** (Property 5 from design)
//
// Bug Condition: input.hasLocationData
// Expected: DetectArrival called with lat/lng/accuracy fields
// Counterexample on unfixed code: GetPresence called instead, location data ignored
//
// CRITICAL: This test is expected to FAIL on unfixed code.
// DO NOT fix the code or the test when it fails.
package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	presencev1 "parkir-pintar/proto/presence/v1"

	"pgregory.net/rapid"
)

// spyPresenceClient tracks which RPC methods are called.
type spyPresenceClient struct {
	presencev1.PresenceServiceClient
	detectArrivalCalled atomic.Int64
	getPresenceCalled   atomic.Int64
	lastArrivalReq      *presencev1.DetectArrivalRequest
}

func (s *spyPresenceClient) DetectArrival(ctx context.Context, in *presencev1.DetectArrivalRequest, opts ...grpc.CallOption) (*presencev1.ArrivalResponse, error) {
	s.detectArrivalCalled.Add(1)
	s.lastArrivalReq = in
	return &presencev1.ArrivalResponse{
		Arrived:       true,
		ReservationId: in.GetReservationId(),
		DetectedAt:    timestamppb.Now(),
	}, nil
}

func (s *spyPresenceClient) GetPresence(ctx context.Context, in *presencev1.GetPresenceRequest, opts ...grpc.CallOption) (*presencev1.PresenceResponse, error) {
	s.getPresenceCalled.Add(1)
	return &presencev1.PresenceResponse{
		ReservationId: in.GetReservationId(),
	}, nil
}

// TestStreamLocation_ShouldCallDetectArrival_WhenLocationDataProvided verifies
// that the gateway StreamLocation handler calls DetectArrival (not GetPresence)
// when location data is provided. On unfixed code, GetPresence is called instead.
//
// **Validates: Requirements 2.8**
func TestStreamLocation_ShouldCallDetectArrival_WhenLocationDataProvided(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Arrange
		lat := rapid.Float64Range(-90, 90).Draw(t, "latitude")
		lng := rapid.Float64Range(-180, 180).Draw(t, "longitude")
		accuracy := rapid.Float64Range(1, 100).Draw(t, "accuracy")
		reservationID := rapid.StringMatching(`[a-z0-9]{8}`).Draw(t, "reservationID")

		spy := &spyPresenceClient{}
		h := NewHandler(nil, nil, nil, spy)

		gin.SetMode(gin.TestMode)
		engine := gin.New()
		engine.POST("/api/v1/presence/stream", h.StreamLocation)

		body := map[string]interface{}{
			"reservation_id": reservationID,
			"latitude":       lat,
			"longitude":      lng,
			"accuracy":       accuracy,
		}
		bodyBytes, err := json.Marshal(body)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/presence/stream", strings.NewReader(string(bodyBytes)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Act
		engine.ServeHTTP(w, req)

		// Assert — DetectArrival should be called, NOT GetPresence
		assert.Equal(t, int64(1), spy.detectArrivalCalled.Load(),
			"DetectArrival should be called exactly once")
		assert.Equal(t, int64(0), spy.getPresenceCalled.Load(),
			"GetPresence should NOT be called — location data should go to DetectArrival")

		// Verify location fields were passed through
		if spy.lastArrivalReq != nil {
			assert.Equal(t, reservationID, spy.lastArrivalReq.GetReservationId())
			assert.InDelta(t, lat, spy.lastArrivalReq.GetLatitude(), 0.0001)
			assert.InDelta(t, lng, spy.lastArrivalReq.GetLongitude(), 0.0001)
		}
	})
}
