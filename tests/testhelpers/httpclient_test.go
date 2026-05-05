// Best practices applied from Go testing guidelines:
// - Descriptive test names using Test[FunctionName]_Should[Expected]_When[Condition] pattern
// - AAA (Arrange-Act-Assert) structure
// - testify assertions (assert, require) for clear failure messages
// - net/http/httptest for HTTP server simulation
// - Tests verify behavior, not implementation details
// - No mocks needed — testing real HTTP client and health polling

package testhelpers

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAuthenticatedClient_ShouldAddAuthorizationHeader_WhenRequestIsMade(t *testing.T) {
	// Arrange
	token := "test-jwt-token-abc123"
	var receivedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewAuthenticatedClient(token)

	// Act
	resp, err := client.Get(server.URL)

	// Assert
	require.NoError(t, err, "request should succeed")
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "Bearer test-jwt-token-abc123", receivedAuth, "Authorization header should contain Bearer token")
}

func TestNewAuthenticatedClient_ShouldAddHeaderToEveryRequest_WhenMultipleRequestsMade(t *testing.T) {
	// Arrange
	token := "multi-request-token"
	var callCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer multi-request-token" {
			atomic.AddInt32(&callCount, 1)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewAuthenticatedClient(token)

	// Act
	for i := 0; i < 3; i++ {
		resp, err := client.Get(server.URL)
		require.NoError(t, err)
		resp.Body.Close()
	}

	// Assert
	assert.Equal(t, int32(3), atomic.LoadInt32(&callCount), "all 3 requests should have the Authorization header")
}

func TestWaitForHealth_ShouldReturnNil_WhenServerReturns200(t *testing.T) {
	// Arrange
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Act
	err := WaitForHealth(server.URL, 5*time.Second)

	// Assert
	assert.NoError(t, err, "should succeed when server returns 200")
}

func TestWaitForHealth_ShouldReturnError_WhenServerIsUnavailable(t *testing.T) {
	// Arrange — use a URL that nothing is listening on
	url := "http://127.0.0.1:19999"

	// Act
	err := WaitForHealth(url, 2*time.Second)

	// Assert
	assert.Error(t, err, "should return error when server is unreachable")
	assert.Contains(t, err.Error(), "timed out", "error should indicate timeout")
}

func TestWaitForHealth_ShouldSucceed_WhenServerBecomesHealthyBeforeTimeout(t *testing.T) {
	// Arrange — server returns 503 initially, then 200 after a few calls
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attempts, 1)
		if count < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Act
	err := WaitForHealth(server.URL, 10*time.Second)

	// Assert
	assert.NoError(t, err, "should succeed once server becomes healthy")
	assert.GreaterOrEqual(t, atomic.LoadInt32(&attempts), int32(3), "should have polled multiple times")
}
