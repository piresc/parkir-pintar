// Package testhelpers provides shared utilities for both E2E test layers.
package testhelpers

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// authTransport is a custom RoundTripper that injects an Authorization header
// into every outgoing HTTP request.
type authTransport struct {
	token     string
	transport http.RoundTripper
}

// RoundTrip adds the Bearer token to the request and delegates to the wrapped transport.
func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.token)
	return t.transport.RoundTrip(req)
}

// NewAuthenticatedClient returns an *http.Client that injects
// Authorization: Bearer <token> on every request via a custom RoundTripper
// wrapping http.DefaultTransport.
func NewAuthenticatedClient(token string) *http.Client {
	return &http.Client{
		Transport: &authTransport{
			token:     token,
			transport: http.DefaultTransport,
		},
	}
}

// WaitForHealth polls the given URL until it returns HTTP 200 or the timeout
// is exceeded. It polls every 1 second. Returns nil on success, or an error
// if the timeout is reached without a healthy response.
func WaitForHealth(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}

	for time.Now().Before(deadline) {
		req, reqErr := http.NewRequestWithContext(context.Background(), http.MethodGet, url, http.NoBody)
		if reqErr != nil {
			return fmt.Errorf("WaitForHealth: failed to create request: %w", reqErr)
		}
		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("WaitForHealth: timed out waiting for %s to return HTTP 200 after %s", url, timeout)
}
