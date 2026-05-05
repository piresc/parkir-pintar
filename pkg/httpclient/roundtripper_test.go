package httpclient

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStubRoundTripper_ShouldPassThrough_WhenProductionEnv(t *testing.T) {
	// Arrange
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"real":"response"}`))
	}))
	defer server.Close()

	rt := newStubRoundTripper(http.DefaultTransport, "production")
	req, _ := http.NewRequest(http.MethodGet, server.URL+"/api/test", nil)

	// Act
	resp, err := rt.RoundTrip(req)

	// Assert
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	assert.Contains(t, string(body), "real")
}

func TestStubRoundTripper_ShouldReturnStub_WhenGlobalEnvSet(t *testing.T) {
	// Arrange — path /api/test → _api_test → key GET_API_TEST
	os.Setenv("GET_API_TEST", `{"stubbed":"true"}`)
	defer os.Unsetenv("GET_API_TEST")

	rt := newStubRoundTripper(http.DefaultTransport, "local")
	req, _ := http.NewRequest(http.MethodGet, "https://api.example.com/api/test", nil)

	// Act
	resp, err := rt.RoundTrip(req)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	assert.Contains(t, string(body), "stubbed")
}

func TestStubRoundTripper_ShouldPreferMSISDNStub_WhenBothExist(t *testing.T) {
	// Arrange
	os.Setenv("GET_API_TEST", `{"source":"global"}`)
	os.Setenv("GET_API_TEST_6281234567890", `{"source":"msisdn"}`)
	defer os.Unsetenv("GET_API_TEST")
	defer os.Unsetenv("GET_API_TEST_6281234567890")

	rt := newStubRoundTripper(http.DefaultTransport, "local")
	req, _ := http.NewRequest(http.MethodGet, "https://api.example.com/api/test", nil)
	req.Header.Set("x-msisdn", "6281234567890")

	// Act
	resp, err := rt.RoundTrip(req)

	// Assert
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	assert.Contains(t, string(body), "msisdn")
}

func TestStubRoundTripper_ShouldPassThrough_WhenNoStubEnvSet(t *testing.T) {
	// Arrange
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"real":"data"}`))
	}))
	defer server.Close()

	rt := newStubRoundTripper(http.DefaultTransport, "local")
	req, _ := http.NewRequest(http.MethodGet, server.URL+"/unique/path", nil)

	// Act
	resp, err := rt.RoundTrip(req)

	// Assert
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	assert.Contains(t, string(body), "real")
}

func TestCreateStubResponse_ShouldReturnValidResponse(t *testing.T) {
	// Act
	resp := createStubResponse(`{"test":"value"}`)

	// Assert
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	assert.Equal(t, `{"test":"value"}`, string(body))
}
