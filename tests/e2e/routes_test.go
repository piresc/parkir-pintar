// Package e2e_test — API routes validation test.
//
// Best practices applied (from Go testify testing standards):
// - Use require for assertions that must pass to continue (fail-fast)
// - Use assert for non-critical checks
// - Follow AAA (Arrange-Act-Assert) structure
// - Use descriptive test names: Test[Scenario]_Should[Expected]_When[Condition]
package e2e_test

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	gwhandler "parkir-pintar/internal/gateway/handler"
	"parkir-pintar/pkg/config"
	"parkir-pintar/pkg/middleware"
)

// TestRoutes_ShouldRegisterAllPRDRoutes verifies that the gateway handler
// registers all 9 routes defined in PRD Section 14.
//
// Validates: Requirements 17.1, 17.2, 17.3, 17.4, 17.5, 17.6, 17.7, 17.8, 17.9
func TestRoutes_ShouldRegisterAllPRDRoutes(t *testing.T) {
	// Arrange — Create a Gin engine with routes registered
	gin.SetMode(gin.TestMode)
	engine := gin.New()

	h := gwhandler.NewHandler(nil, nil, nil, nil)
	cfg := &config.Config{}
	mw := middleware.NewMiddleware(cfg, nil, nil)
	h.RegisterRoutes(engine, mw, "test-secret")

	// Act — Introspect registered routes
	routes := engine.Routes()
	routeMap := make(map[string]bool)
	for _, r := range routes {
		routeMap[r.Method+" "+r.Path] = true
	}

	// Assert — All 9 PRD routes exist
	expectedRoutes := []struct {
		route string
		desc  string
	}{
		{"POST /api/v1/reservations", "CreateReservation"},
		{"DELETE /api/v1/reservations/:id", "CancelReservation"},
		{"POST /api/v1/reservations/:id/checkin", "CheckIn"},
		{"POST /api/v1/reservations/:id/checkout", "CheckOut"},
		{"GET /api/v1/availability", "GetAvailability"},
		{"GET /api/v1/floors/:floor", "GetFloorMap"},
		{"GET /api/v1/spots/:id", "GetSpotDetails"},
		{"POST /api/v1/presence/stream", "StreamLocation"},
		{"GET /api/v1/payments/:id/status", "GetPaymentStatus"},
	}

	for _, er := range expectedRoutes {
		assert.True(t, routeMap[er.route],
			"route %s (%s) should be registered", er.route, er.desc)
	}

	// Verify we found at least 9 routes (may have more from middleware)
	assert.GreaterOrEqual(t, len(routes), 9,
		"gateway should register at least 9 routes")
}
