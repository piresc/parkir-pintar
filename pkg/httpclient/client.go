// Package httpclient provides a unified HTTP client combining retry logic,
// distributed tracing, SSRF protection, connection pooling, TLS 1.2 minimum,
// stub support for non-production, and structured error responses.
package httpclient

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"parkir-pintar/pkg/tracing"
)

// ClientConfig holds configuration for creating a new HTTP client.
type ClientConfig struct {
	AppEnv             string
	APIKey             string
	SecretKey          string
	BaseURL            string
	Timeout            time.Duration
	Tracer             tracing.Tracer
	IsRequireSignature bool
	StubEnabled        bool
}

// Client is a unified HTTP client with retry, tracing, and SSRF protection.
type Client struct {
	httpClient         *http.Client
	apiKey             string
	secretKey          string
	baseURL            string
	tracer             tracing.Tracer
	appEnv             string
	isRequireSignature bool
	logger             *slog.Logger
}

// ClientOption is a functional option for configuring the Client.
type ClientOption func(*Client)

// WithSignatureRequired enables signature requirement on requests.
func WithSignatureRequired() ClientOption {
	return func(c *Client) {
		c.isRequireSignature = true
	}
}

// NewClient creates a new HTTP client with connection pooling, TLS 1.2 minimum,
// optional stub support, and configurable timeout.
func NewClient(cfg ClientConfig, opts ...ClientOption) *Client {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	if cfg.AppEnv == "local" {
		// #nosec G402 -- InsecureSkipVerify only enabled in local development environment
		tlsConfig.InsecureSkipVerify = true
	}

	baseTransport := &http.Transport{
		TLSClientConfig:     tlsConfig,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}

	var transport http.RoundTripper = baseTransport
	if cfg.StubEnabled {
		transport = newStubRoundTripper(baseTransport, cfg.AppEnv)
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	c := &Client{
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
		apiKey:             cfg.APIKey,
		secretKey:          cfg.SecretKey,
		baseURL:            cfg.BaseURL,
		tracer:             cfg.Tracer,
		appEnv:             cfg.AppEnv,
		isRequireSignature: cfg.IsRequireSignature,
		logger:             slog.Default(),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Do executes an HTTP request with retry logic, tracing, and SSRF protection.
// Retries up to 3 attempts with exponential backoff (100ms, 200ms, 400ms).
// Does not retry on 4xx client errors.
func (c *Client) Do(ctx context.Context, method, endpoint string, body interface{}) (*http.Response, error) {
	safeURL, err := buildSafeURL(c.baseURL, endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	var lastErr error
	backoffs := []time.Duration{100 * time.Millisecond, 200 * time.Millisecond, 400 * time.Millisecond}

	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoffs[attempt-1]):
			}

			// Reset body reader for retry
			if body != nil {
				jsonBody, _ := json.Marshal(body)
				bodyReader = bytes.NewReader(jsonBody)
			}
		}

		resp, err := c.doRequest(ctx, method, safeURL, bodyReader)
		if err != nil {
			lastErr = err
			// Don't retry on context cancellation
			if errors.Is(err, context.Canceled) {
				return nil, err
			}
			continue
		}

		// Don't retry on 4xx client errors
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			return resp, nil
		}

		// Don't retry on success (2xx/3xx)
		if resp.StatusCode < 500 {
			return resp, nil
		}

		// Close body before retry on 5xx
		if err := resp.Body.Close(); err != nil {
			c.logger.Warn("failed to close response body on retry", slog.String("error", err.Error()))
		}
		lastErr = fmt.Errorf("server error: status %d", resp.StatusCode)
	}

	return nil, c.classifyError(lastErr)
}

// doRequest performs a single HTTP request with tracing and header injection.
func (c *Client) doRequest(ctx context.Context, method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Inject API key
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}

	// Propagate request ID
	if requestID := ctx.Value("request_id"); requestID != nil {
		if id, ok := requestID.(string); ok && id != "" {
			req.Header.Set("X-Request-ID", id)
		}
	}

	// Inject tracing headers
	if c.tracer != nil && c.tracer.IsEnabled() {
		c.tracer.InjectContext(ctx, req.Header)
	}

	// Start external call span
	var endSpan func()
	if c.tracer != nil && c.tracer.IsEnabled() {
		ctx, endSpan = c.tracer.StartExternalCall(ctx, req.URL.Host, method)
		req = req.WithContext(ctx)
		defer endSpan()
	}

	// #nosec G107 -- URL is validated via validatePathURL and buildSafeURL
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// classifyError maps errors to structured HTTP status codes.
// timeout → 504, connection errors → 503, parse errors → 502.
func (c *Client) classifyError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return &HTTPError{StatusCode: http.StatusGatewayTimeout, Message: "gateway timeout", Err: err}
	}

	var netErr *net.OpError
	if errors.As(err, &netErr) {
		return &HTTPError{StatusCode: http.StatusServiceUnavailable, Message: "service unavailable", Err: err}
	}

	errMsg := strings.ToLower(err.Error())
	if strings.Contains(errMsg, "connection refused") || strings.Contains(errMsg, "connection reset") {
		return &HTTPError{StatusCode: http.StatusServiceUnavailable, Message: "service unavailable", Err: err}
	}

	return &HTTPError{StatusCode: http.StatusBadGateway, Message: "bad gateway", Err: err}
}

// GetJSON performs a GET request and unmarshals the JSON response into result.
func (c *Client) GetJSON(ctx context.Context, endpoint string, result interface{}) error {
	resp, err := c.Do(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			c.logger.Warn("failed to close response body", slog.String("error", closeErr.Error()))
		}
	}()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return &HTTPError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return &HTTPError{StatusCode: http.StatusBadGateway, Message: "failed to parse response", Err: err}
	}

	return nil
}

// PostJSON performs a POST request with a JSON body and unmarshals the response into result.
func (c *Client) PostJSON(ctx context.Context, endpoint string, body, result interface{}) error {
	resp, err := c.Do(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			c.logger.Warn("failed to close response body", slog.String("error", closeErr.Error()))
		}
	}()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return &HTTPError{
			StatusCode: resp.StatusCode,
			Message:    string(respBody),
		}
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return &HTTPError{StatusCode: http.StatusBadGateway, Message: "failed to parse response", Err: err}
		}
	}

	return nil
}

// Close releases resources held by the HTTP client.
func (c *Client) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}

// HTTPError represents a structured HTTP error with status code.
type HTTPError struct {
	StatusCode int
	Message    string
	Err        error
}

func (e *HTTPError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("http error %d: %s: %v", e.StatusCode, e.Message, e.Err)
	}
	return fmt.Sprintf("http error %d: %s", e.StatusCode, e.Message)
}

func (e *HTTPError) Unwrap() error {
	return e.Err
}
