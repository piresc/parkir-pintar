//go:build e2e_docker

// Package e2e_docker_test — middleware validation tests via Gateway REST API.
//
// Best practices applied (from Go testing standards KB):
// - Descriptive test name: Test[Scenario]_Should[Expected]_When[Condition]
// - AAA structure (Arrange-Act-Assert)
// - No mocks — real HTTP requests to the deployed stack
// - Tests cover auth, invalid tokens, not-found, and bad-request scenarios
package e2e_docker_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

// TestDockerMiddleware_ShouldEnforceAuth validates that the Gateway middleware
// correctly enforces JWT authentication and returns proper error codes.
//
// Validates: Requirements 23.1, 23.2, 23.3, 23.4
func TestDockerMiddleware_ShouldEnforceAuth(t *testing.T) {
	if denv == nil {
		t.Fatal("denv not initialized — TestMain did not run")
	}

	t.Run("NoJWT_ShouldReturn401", func(t *testing.T) {
		// --- Arrange: Use a plain HTTP client without JWT ---
		plainClient := &http.Client{}

		// --- Act ---
		resp, err := plainClient.Get(denv.baseURL + "/api/v1/availability")
		if err != nil {
			t.Fatalf("GET /api/v1/availability without JWT failed: %v", err)
		}
		defer resp.Body.Close()

		// --- Assert: HTTP 401 ---
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected HTTP 401 without JWT, got %d", resp.StatusCode)
		}
		t.Log("No JWT → 401 confirmed")
	})

	t.Run("InvalidJWT_ShouldReturn401", func(t *testing.T) {
		// --- Arrange: Use a client with an invalid JWT ---
		invalidClient := &http.Client{
			Transport: &invalidTokenTransport{token: "invalid.jwt.token"},
		}

		// --- Act ---
		resp, err := invalidClient.Get(denv.baseURL + "/api/v1/availability")
		if err != nil {
			t.Fatalf("GET /api/v1/availability with invalid JWT failed: %v", err)
		}
		defer resp.Body.Close()

		// --- Assert: HTTP 401 ---
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected HTTP 401 with invalid JWT, got %d", resp.StatusCode)
		}
		t.Log("Invalid JWT → 401 confirmed")
	})

	t.Run("NonExistentReservation_ShouldReturn404", func(t *testing.T) {
		// --- Arrange: Use authenticated client with a fake reservation ID ---
		fakeID := "00000000-0000-0000-0000-000000000000"

		// --- Act ---
		resp, err := denv.httpClient.Post(
			denv.baseURL+"/api/v1/reservations/"+fakeID+"/checkin",
			"application/json",
			nil,
		)
		if err != nil {
			t.Fatalf("POST checkin for non-existent reservation failed: %v", err)
		}
		defer resp.Body.Close()

		// --- Assert: HTTP 404 ---
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("expected HTTP 404 for non-existent reservation, got %d", resp.StatusCode)
		}
		t.Log("Non-existent reservation → 404 confirmed")
	})

	t.Run("MissingRequiredFields_ShouldReturn400", func(t *testing.T) {
		// --- Arrange: Send empty JSON body to create reservation ---
		emptyBody, err := json.Marshal(map[string]string{})
		if err != nil {
			t.Fatalf("failed to marshal empty body: %v", err)
		}

		// --- Act ---
		resp, err := denv.httpClient.Post(
			denv.baseURL+"/api/v1/reservations",
			"application/json",
			bytes.NewBuffer(emptyBody),
		)
		if err != nil {
			t.Fatalf("POST /api/v1/reservations with empty body failed: %v", err)
		}
		defer resp.Body.Close()

		// --- Assert: HTTP 400 ---
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected HTTP 400 for missing fields, got %d", resp.StatusCode)
		}
		t.Log("Missing required fields → 400 confirmed")
	})
}

// invalidTokenTransport is a custom RoundTripper that injects an invalid
// Authorization header into every outgoing HTTP request.
type invalidTokenTransport struct {
	token string
}

// RoundTrip adds the invalid Bearer token to the request.
func (t *invalidTokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.token)
	return http.DefaultTransport.RoundTrip(req)
}
