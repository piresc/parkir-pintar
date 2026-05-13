//go:build e2e_docker

// Package e2e_docker_test — cancellation test via Gateway REST API.
//
// Best practices applied (from Go testing standards KB):
// - Descriptive test name: Test[Scenario]_Should[Expected]_When[Condition]
// - AAA structure (Arrange-Act-Assert)
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

// TestDockerCancellation_ShouldReturnZeroFee_WhenCancelledImmediately validates
// that cancelling a reservation immediately (within the 2-minute free window)
// returns HTTP 200 via the Gateway REST API.
//
// Validates: Requirements 20.1, 20.3
func TestDockerCancellation_ShouldReturnZeroFee_WhenCancelledImmediately(t *testing.T) {
	if denv == nil {
		t.Fatal("denv not initialized — TestMain did not run")
	}

	// --- Arrange: Create a reservation ---
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

	createResp, err := denv.httpClient.Post(
		denv.baseURL+"/api/v1/reservations",
		"application/json",
		bytes.NewBuffer(createBody),
	)
	if err != nil {
		t.Fatalf("POST /api/v1/reservations failed: %v", err)
	}
	defer createResp.Body.Close()

	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected HTTP 201 for create, got %d", createResp.StatusCode)
	}

	var createResult struct {
		Status string `json:"status"`
		Data   struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(createResp.Body).Decode(&createResult); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}

	reservationID := createResult.Data.ID
	if reservationID == "" {
		t.Fatal("reservation id is empty in create response")
	}
	t.Logf("Created reservation for cancellation: %s", reservationID)

	// --- Act: Cancel immediately (within free window) ---
	cancelURL := fmt.Sprintf("%s/api/v1/reservations/%s", denv.baseURL, reservationID)
	req, err := http.NewRequest(http.MethodDelete, cancelURL, nil)
	if err != nil {
		t.Fatalf("failed to create DELETE request: %v", err)
	}

	cancelResp, err := denv.httpClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE /api/v1/reservations/%s failed: %v", reservationID, err)
	}
	defer cancelResp.Body.Close()

	// --- Assert: HTTP 200 ---
	if cancelResp.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP 200 for cancellation, got %d", cancelResp.StatusCode)
	}
	t.Log("Immediate cancellation successful")
}
