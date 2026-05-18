//go:build e2e_docker

// Package e2e_docker_test — double-booking prevention test via Gateway REST API.
//
// Best practices applied (from Go testing standards KB):
// - Descriptive test name: Test[Scenario]_Should[Expected]_When[Condition]
// - AAA structure (Arrange-Act-Assert)
// - No mocks — real HTTP requests to the deployed stack
package e2e_docker_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"sync"
	"testing"

	"github.com/google/uuid"
)

// TestDockerDoubleBook_ShouldRejectSecond_WhenSameSpot validates that
// two concurrent reservation requests for the same spot result in one
// HTTP 201 and one HTTP 409 via the Gateway REST API.
//
// Validates: Requirements 21.1, 21.2
func TestDockerDoubleBook_ShouldRejectSecond_WhenSameSpot(t *testing.T) {
	if denv == nil {
		t.Fatal("denv not initialized — TestMain did not run")
	}

	// --- Arrange ---
	// We need a known spot ID. First, get availability to find an available spot.
	availResp, err := denv.httpClient.Get(denv.baseURL + "/api/v1/availability?vehicle_type=motorcycle")
	if err != nil {
		t.Fatalf("GET /api/v1/availability failed: %v", err)
	}
	defer availResp.Body.Close()

	if availResp.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP 200 for availability, got %d", availResp.StatusCode)
	}

	// Parse availability response to get a specific spot_id for user_selected mode.
	var availBody struct {
		Spots []struct {
			ID string `json:"id"`
		} `json:"spots"`
	}
	if err := json.NewDecoder(availResp.Body).Decode(&availBody); err != nil || len(availBody.Spots) == 0 {
		t.Logf("Could not parse spot_id from availability response, falling back to system_assigned")
	}

	// Determine spot_id to target — use first available spot if possible.
	spotID := ""
	if len(availBody.Spots) > 0 {
		spotID = availBody.Spots[0].ID
	}

	// Send two concurrent requests targeting the same spot using user_selected mode.
	var wg sync.WaitGroup
	statusCodes := make([]int, 2)

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			reqBody := map[string]string{
				"driver_id":       denv.driverID,
				"vehicle_type":    "motorcycle",
				"idempotency_key": uuid.New().String(),
			}
			if spotID != "" {
				reqBody["assignment_mode"] = "user_selected"
				reqBody["spot_id"] = spotID
			} else {
				reqBody["assignment_mode"] = "system_assigned"
			}

			body, marshalErr := json.Marshal(reqBody)
			if marshalErr != nil {
				t.Errorf("goroutine %d: failed to marshal request: %v", idx, marshalErr)
				return
			}

			resp, postErr := denv.httpClient.Post(
				denv.baseURL+"/api/v1/reservations",
				"application/json",
				bytes.NewBuffer(body),
			)
			if postErr != nil {
				t.Errorf("goroutine %d: POST failed: %v", idx, postErr)
				return
			}
			defer resp.Body.Close()

			statusCodes[idx] = resp.StatusCode
		}(i)
	}
	wg.Wait()

	// --- Assert ---
	// With user_selected mode targeting the same spot, exactly one should succeed
	// and the other should get a conflict (409).
	successCount := 0
	conflictCount := 0
	for _, code := range statusCodes {
		if code == http.StatusCreated {
			successCount++
		}
		if code == http.StatusConflict {
			conflictCount++
		}
	}

	if successCount+conflictCount != 2 {
		t.Fatalf("expected exactly 1 success + 1 conflict, got status codes: %v", statusCodes)
	}
	if successCount != 1 {
		t.Fatalf("expected exactly one HTTP 201, got %d successes in status codes: %v", successCount, statusCodes)
	}
	if conflictCount != 1 {
		t.Fatalf("expected exactly one HTTP 409, got %d conflicts in status codes: %v", conflictCount, statusCodes)
	}

	t.Logf("Double-book status codes: %v (conflict correctly detected)", statusCodes)
}
