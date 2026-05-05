package httpclient

import (
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
)

// StubRoundTripper returns stubbed responses in non-production environments.
// It checks environment variables for global and per-MSISDN stub responses.
//
// Global stub env var format:  GET_API_V1_USERS='{"status":"ok"}'
// Per-MSISDN stub env var format: GET_API_V1_USERS_6281234567890='{"status":"ok"}'
type StubRoundTripper struct {
	next   http.RoundTripper
	appEnv string
}

// newStubRoundTripper creates a StubRoundTripper wrapping the given transport.
func newStubRoundTripper(next http.RoundTripper, appEnv string) *StubRoundTripper {
	return &StubRoundTripper{
		next:   next,
		appEnv: appEnv,
	}
}

// RoundTrip implements http.RoundTripper. In non-production environments,
// it checks for stubbed responses before forwarding to the next transport.
func (rt *StubRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if rt.appEnv != "production" {
		if response := rt.getStubbedResponse(req); response != nil {
			return response, nil
		}
	}

	return rt.next.RoundTrip(req)
}

// getStubbedResponse checks env vars for stub data matching the request.
// Checks per-MSISDN stub first, then global stub.
func (rt *StubRoundTripper) getStubbedResponse(req *http.Request) *http.Response {
	urlPath := strings.Split(req.URL.Path, "?")[0]
	urlPath = strings.ReplaceAll(strings.ReplaceAll(urlPath, "/", "_"), "-", "_")

	keyGlobal := strings.ToUpper(req.Method + urlPath)

	// Extract MSISDN from request header
	msisdn := req.Header.Get("X-MSISDN")
	if msisdn == "" {
		msisdn = req.Header.Get("x-msisdn")
	}

	// Check per-MSISDN stub first
	if msisdn != "" {
		keySpecific := keyGlobal + "_" + msisdn
		if stubData := os.Getenv(keySpecific); stubData != "" {
			slog.Info("using per-MSISDN stub response",
				slog.String("url", req.URL.String()),
				slog.String("msisdn", msisdn),
				slog.String("env_key", keySpecific))
			return createStubResponse(stubData)
		}
	}

	// Check global stub
	if stubData := os.Getenv(keyGlobal); stubData != "" {
		slog.Info("using global stub response",
			slog.String("url", req.URL.String()),
			slog.String("env_key", keyGlobal))
		return createStubResponse(stubData)
	}

	return nil
}

// createStubResponse builds an http.Response from stub JSON data.
func createStubResponse(stubData string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(stubData)),
	}
}
