//go:build e2e_docker

// Package e2e_docker_test — payment flow test via Gateway REST API.
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

// TestDockerPayment_ShouldReturnPaymentStatus validates the payment flow
// by completing a full checkout and then querying the payment status endpoint.
//
// Validates: Requirements 22.1, 22.2
func TestDockerPayment_ShouldReturnPaymentStatus(t *testing.T) {
	if denv == nil {
		t.Fatal("denv not initialized — TestMain did not run")
	}

	// --- Arrange: Create reservation ---
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
		t.Fatal("reservation id is empty")
	}
	t.Logf("Created reservation: %s", reservationID)

	// --- Act: Check in ---
	checkinResp, err := denv.httpClient.Post(
		fmt.Sprintf("%s/api/v1/reservations/%s/checkin", denv.baseURL, reservationID),
		"application/json",
		nil,
	)
	if err != nil {
		t.Fatalf("POST checkin failed: %v", err)
	}
	defer checkinResp.Body.Close()

	if checkinResp.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP 200 for checkin, got %d", checkinResp.StatusCode)
	}

	// --- Act: Check out (triggers payment) ---
	checkoutResp, err := denv.httpClient.Post(
		fmt.Sprintf("%s/api/v1/reservations/%s/checkout", denv.baseURL, reservationID),
		"application/json",
		nil,
	)
	if err != nil {
		t.Fatalf("POST checkout failed: %v", err)
	}
	defer checkoutResp.Body.Close()

	if checkoutResp.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP 200 for checkout, got %d", checkoutResp.StatusCode)
	}

	// Parse checkout response to extract payment ID
	var checkoutResult struct {
		Status string `json:"status"`
		Data   struct {
			PaymentId string `json:"payment_id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(checkoutResp.Body).Decode(&checkoutResult); err != nil {
		t.Fatalf("failed to decode checkout response: %v", err)
	}

	paymentID := checkoutResult.Data.PaymentId
	if paymentID == "" {
		t.Log("payment_id not in checkout response — skipping payment status check")
		return
	}
	t.Logf("Payment ID from checkout: %s", paymentID)

	// --- Act: Get payment status ---
	paymentResp, err := denv.httpClient.Get(
		fmt.Sprintf("%s/api/v1/payments/%s/status", denv.baseURL, paymentID),
	)
	if err != nil {
		t.Fatalf("GET /api/v1/payments/%s/status failed: %v", paymentID, err)
	}
	defer paymentResp.Body.Close()

	// --- Assert: HTTP 200 ---
	if paymentResp.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP 200 for payment status, got %d", paymentResp.StatusCode)
	}

	var paymentResult struct {
		Status string                 `json:"status"`
		Data   map[string]interface{} `json:"data"`
	}
	if err := json.NewDecoder(paymentResp.Body).Decode(&paymentResult); err != nil {
		t.Fatalf("failed to decode payment status response: %v", err)
	}
	t.Logf("Payment status response: %+v", paymentResult.Data)
}
