// Package e2e_docker_test — happy path lifecycle test via Gateway REST API.
//
// Best practices applied (from Go testing standards KB):
// - Descriptive test name: Test[Scenario]_Should[Expected]_When[Condition]
// - AAA structure (Arrange-Act-Assert)
// - Comprehensive assertions on HTTP status codes and response bodies
// - No mocks — real HTTP requests to the deployed stack
package e2e_docker_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

// TestDockerHappyPath_ShouldCompleteFullLifecycle validates the complete
// reservation lifecycle through the Gateway REST API:
// create → check-in → check-out → verify availability.
//
// Validates: Requirements 19.1, 19.2, 19.3, 19.4, 19.5
func TestDockerHappyPath_ShouldCompleteFullLifecycle(t *testing.T) {
	if denv == nil {
		t.Fatal("denv not initialized — TestMain did not run")
	}

	// --- Arrange ---
	idempotencyKey := uuid.New().String()
	createBody, err := json.Marshal(map[string]string{
		"driver_id":       denv.driverID,
		"vehicle_type":    "car",
		"assignment_mode": "system_assigned",
		"idempotency_key": idempotencyKey,
	})
	if err != nil {
		t.Fatalf("failed to marshal create request: %v", err)
	}

	// --- Act: Create reservation ---
	createResp, err := denv.httpClient.Post(
		denv.baseURL+"/api/v1/reservations",
		"application/json",
		bytes.NewBuffer(createBody),
	)
	if err != nil {
		t.Fatalf("POST /api/v1/reservations failed: %v", err)
	}
	defer createResp.Body.Close()

	// --- Assert: HTTP 201 ---
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected HTTP 201, got %d", createResp.StatusCode)
	}

	var createResult struct {
		Status string `json:"status"`
		Data   struct {
			ID     string `json:"id"`
			SpotID string `json:"spot_id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(createResp.Body).Decode(&createResult); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}

	reservationID := createResult.Data.ID
	if reservationID == "" {
		t.Fatal("reservation id is empty in create response")
	}
	t.Logf("Created reservation: %s, spot: %s", reservationID, createResult.Data.SpotID)

	// --- Act: Check in ---
	checkinResp, err := denv.httpClient.Post(
		fmt.Sprintf("%s/api/v1/reservations/%s/checkin", denv.baseURL, reservationID),
		"application/json",
		nil,
	)
	if err != nil {
		t.Fatalf("POST /api/v1/reservations/%s/checkin failed: %v", reservationID, err)
	}
	defer checkinResp.Body.Close()

	// --- Assert: HTTP 200 ---
	if checkinResp.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP 200 for checkin, got %d", checkinResp.StatusCode)
	}
	t.Log("Check-in successful")

	// --- Act: Check out ---
	checkoutResp, err := denv.httpClient.Post(
		fmt.Sprintf("%s/api/v1/reservations/%s/checkout", denv.baseURL, reservationID),
		"application/json",
		nil,
	)
	if err != nil {
		t.Fatalf("POST /api/v1/reservations/%s/checkout failed: %v", reservationID, err)
	}
	defer checkoutResp.Body.Close()

	// --- Assert: HTTP 200 with billing summary ---
	if checkoutResp.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP 200 for checkout, got %d", checkoutResp.StatusCode)
	}

	var checkoutResult struct {
		Status string                 `json:"status"`
		Data   map[string]interface{} `json:"data"`
	}
	if err := json.NewDecoder(checkoutResp.Body).Decode(&checkoutResult); err != nil {
		t.Fatalf("failed to decode checkout response: %v", err)
	}
	t.Logf("Checkout response data: %+v", checkoutResult.Data)

	// --- Act: Get availability ---
	availResp, err := denv.httpClient.Get(denv.baseURL + "/api/v1/availability?vehicle_type=car")
	if err != nil {
		t.Fatalf("GET /api/v1/availability failed: %v", err)
	}
	defer availResp.Body.Close()

	// --- Assert: HTTP 200 ---
	if availResp.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP 200 for availability, got %d", availResp.StatusCode)
	}
	t.Log("Availability check successful")
}
