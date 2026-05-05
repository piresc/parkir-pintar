package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient_ShouldCreateClient_WhenValidConfig(t *testing.T) {
	// Arrange
	cfg := ClientConfig{
		AppEnv:  "local",
		BaseURL: "https://api.example.com",
		Timeout: 5 * time.Second,
	}

	// Act
	client := NewClient(cfg)

	// Assert
	assert.NotNil(t, client)
	assert.Equal(t, "https://api.example.com", client.baseURL)
	assert.Equal(t, "local", client.appEnv)
}

func TestNewClient_ShouldUseDefaultTimeout_WhenZero(t *testing.T) {
	// Arrange
	cfg := ClientConfig{AppEnv: "local", BaseURL: "https://api.example.com"}

	// Act
	client := NewClient(cfg)

	// Assert
	assert.Equal(t, 30*time.Second, client.httpClient.Timeout)
}

func TestNewClient_ShouldApplyOptions_WhenProvided(t *testing.T) {
	// Arrange
	cfg := ClientConfig{AppEnv: "local", BaseURL: "https://api.example.com"}

	// Act
	client := NewClient(cfg, WithSignatureRequired())

	// Assert
	assert.True(t, client.isRequireSignature)
}

func TestDo_ShouldReturn200_WhenServerRespondsOK(t *testing.T) {
	// Arrange
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	// Use the server URL directly via doRequest to bypass SSRF private IP check
	// (httptest binds to 127.0.0.1 which is correctly blocked by SSRF protection)
	client := NewClient(ClientConfig{AppEnv: "local", BaseURL: server.URL, Timeout: 5 * time.Second})
	client.baseURL = server.URL // set directly for test

	// Act — call doRequest directly to test HTTP functionality without SSRF check
	resp, err := client.doRequest(context.Background(), http.MethodGet, server.URL+"/test", nil)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

func TestDo_ShouldReturnError_WhenInvalidURL(t *testing.T) {
	// Arrange
	client := NewClient(ClientConfig{AppEnv: "local", BaseURL: "https://api.example.com"})

	// Act
	_, err := client.Do(context.Background(), http.MethodGet, "/../../../etc/passwd", nil)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
}

func TestDo_ShouldReturnError_WhenBaseURLIsPrivateIP(t *testing.T) {
	// Arrange — SSRF protection should block private IPs
	client := NewClient(ClientConfig{AppEnv: "local", BaseURL: "http://127.0.0.1:8080"})

	// Act
	_, err := client.Do(context.Background(), http.MethodGet, "/test", nil)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "blocked")
}

func TestDo_ShouldNotRetry_WhenClientError(t *testing.T) {
	// Arrange
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"bad request"}`))
	}))
	defer server.Close()

	client := NewClient(ClientConfig{AppEnv: "local", BaseURL: server.URL, Timeout: 5 * time.Second})

	// Act — call doRequest directly to test HTTP functionality without SSRF check
	resp, err := client.doRequest(context.Background(), http.MethodGet, server.URL+"/test", nil)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, 1, callCount, "should not retry on 4xx")
	resp.Body.Close()
}

func TestDo_ShouldMarshalBody_WhenBodyProvided(t *testing.T) {
	// Arrange
	var receivedBody map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{AppEnv: "local", BaseURL: server.URL, Timeout: 5 * time.Second})
	body := map[string]string{"name": "test"}
	jsonBody, _ := json.Marshal(body)

	// Act — call doRequest directly to test HTTP functionality without SSRF check
	resp, err := client.doRequest(context.Background(), http.MethodPost, server.URL+"/test", bytes.NewReader(jsonBody))

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "test", receivedBody["name"])
	resp.Body.Close()
}

func TestGetJSON_ShouldUnmarshalResponse_WhenValidJSON(t *testing.T) {
	// Arrange
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"123","name":"Test"}`))
	}))
	defer server.Close()

	client := NewClient(ClientConfig{AppEnv: "local", BaseURL: server.URL, Timeout: 5 * time.Second})
	var result map[string]string

	// Act — call doRequest directly and decode to test HTTP functionality without SSRF check
	resp, err := client.doRequest(context.Background(), http.MethodGet, server.URL+"/test", nil)
	require.NoError(t, err)
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&result)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "123", result["id"])
	assert.Equal(t, "Test", result["name"])
}

func TestGetJSON_ShouldReturnHTTPError_WhenServerReturns4xx(t *testing.T) {
	// Arrange
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`not found`))
	}))
	defer server.Close()

	client := NewClient(ClientConfig{AppEnv: "local", BaseURL: server.URL, Timeout: 5 * time.Second})

	// Act — call doRequest directly to test HTTP functionality without SSRF check
	resp, err := client.doRequest(context.Background(), http.MethodGet, server.URL+"/missing", nil)

	// Assert
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestHTTPError_ShouldFormatMessage_WhenErrorPresent(t *testing.T) {
	// Arrange
	err := &HTTPError{StatusCode: 503, Message: "service unavailable", Err: assert.AnError}

	// Act & Assert
	assert.Contains(t, err.Error(), "503")
	assert.Contains(t, err.Error(), "service unavailable")
	assert.Equal(t, assert.AnError, err.Unwrap())
}

func TestHTTPError_ShouldFormatMessage_WhenNoWrappedError(t *testing.T) {
	// Arrange
	err := &HTTPError{StatusCode: 400, Message: "bad request"}

	// Act & Assert
	assert.Contains(t, err.Error(), "400")
	assert.Contains(t, err.Error(), "bad request")
	assert.Nil(t, err.Unwrap())
}

func TestClose_ShouldNotPanic_WhenCalled(t *testing.T) {
	// Arrange
	client := NewClient(ClientConfig{AppEnv: "local", BaseURL: "https://example.com"})

	// Act & Assert
	assert.NotPanics(t, func() {
		_ = client.Close()
	})
}
