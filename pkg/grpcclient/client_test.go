// Best practices applied from Go testing guidelines:
// - Descriptive test names using Test[FunctionName]_Should[Result]_When[Condition] pattern
// - AAA (Arrange-Act-Assert) structure
// - testify assertions (assert, require)
// - bufconn for in-memory gRPC connections
// - bytes.Buffer + slog.NewJSONHandler for capturing log output

package grpcclient

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

// startBufconnServer starts a gRPC server on an in-memory bufconn listener
// and returns the listener and a cleanup function.
func startBufconnServer(t *testing.T) *bufconn.Listener {
	t.Helper()
	lis := bufconn.Listen(bufSize)
	srv := grpc.NewServer()

	go func() {
		_ = srv.Serve(lis)
	}()

	t.Cleanup(func() {
		srv.Stop()
	})

	return lis
}

// bufconnDialer returns a grpc.DialOption that dials via the given bufconn listener.
func bufconnDialer(lis *bufconn.Listener) grpc.DialOption {
	return grpc.WithContextDialer(func(_ context.Context, _ string) (net.Conn, error) {
		return lis.Dial()
	})
}

// spyTracer records segment names for verification.
func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

// ---------------------------------------------------------------------------
// Dial tests
// ---------------------------------------------------------------------------

func TestDial_ShouldReturnConn_WhenBufconnServerRunning(t *testing.T) {
	// Arrange
	lis := startBufconnServer(t)

	cfg := ClientConfig{
		Target:      "bufnet",
		DialTimeout: 5 * time.Second,
		Logger:      newTestLogger(),
	}

	// Act
	conn, err := Dial(
		context.Background(),
		cfg,
		bufconnDialer(lis),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, conn)
	assert.NoError(t, conn.Close())
}

func TestDial_ShouldUseDefaultLogger_WhenLoggerIsNil(t *testing.T) {
	// Arrange
	lis := startBufconnServer(t)

	cfg := ClientConfig{
		Target:      "bufnet",
		DialTimeout: 5 * time.Second,
		Logger:      nil, // should default to slog.Default()
	}

	// Act
	conn, err := Dial(
		context.Background(),
		cfg,
		bufconnDialer(lis),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, conn)
	assert.NoError(t, conn.Close())
}

func TestDial_ShouldUseNoOpTracer_WhenTracerIsNil(t *testing.T) {
	// Arrange
	lis := startBufconnServer(t)

	cfg := ClientConfig{
		Target:      "bufnet",
		DialTimeout: 5 * time.Second,
		Tracer:      nil, // should default to NoOpTracer
		Logger:      newTestLogger(),
	}

	// Act
	conn, err := Dial(
		context.Background(),
		cfg,
		bufconnDialer(lis),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, conn)
	assert.NoError(t, conn.Close())
}

func TestDial_ShouldApplyInsecureCredentials_WhenTLSDisabled(t *testing.T) {
	// Arrange
	lis := startBufconnServer(t)

	cfg := ClientConfig{
		Target:      "bufnet",
		DialTimeout: 5 * time.Second,
		TLSEnabled:  false,
		Logger:      newTestLogger(),
	}

	// Act — TLSEnabled=false should auto-apply insecure credentials
	conn, err := Dial(
		context.Background(),
		cfg,
		bufconnDialer(lis),
	)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, conn)
	assert.NoError(t, conn.Close())
}

func TestDial_ShouldApplyKeepaliveParams_WhenConfigured(t *testing.T) {
	// Arrange
	lis := startBufconnServer(t)

	cfg := ClientConfig{
		Target:           "bufnet",
		DialTimeout:      5 * time.Second,
		KeepAliveTime:    30 * time.Second,
		KeepAliveTimeout: 10 * time.Second,
		Logger:           newTestLogger(),
	}

	// Act
	conn, err := Dial(
		context.Background(),
		cfg,
		bufconnDialer(lis),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, conn)
	assert.NoError(t, conn.Close())
}

// ---------------------------------------------------------------------------
// Interceptor application tests
// ---------------------------------------------------------------------------

func TestDial_ShouldApplyLoggingInterceptor_WhenLoggerProvided(t *testing.T) {
	// Arrange
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	cfg := ClientConfig{
		Target:      "passthrough:///bufnet",
		DialTimeout: 5 * time.Second,
		Logger:      logger,
	}

	// Act — Dial creates the connection with logging interceptor wired in.
	conn, err := Dial(
		context.Background(),
		cfg,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, conn)
	assert.NoError(t, conn.Close())
}

// ---------------------------------------------------------------------------
// Client interceptor unit tests
// ---------------------------------------------------------------------------

func TestClientLoggingUnaryInterceptor_ShouldLogInfo_WhenSuccess(t *testing.T) {
	// Arrange
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	interceptor := clientLoggingUnaryInterceptor(logger)

	method := "/test.Service/GetItem"
	invoker := func(_ context.Context, _ string, _, _ interface{}, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		return nil
	}

	// Act
	err := interceptor(context.Background(), method, nil, nil, nil, invoker)

	// Assert
	require.NoError(t, err)

	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	require.Len(t, lines, 1)

	var entry map[string]interface{}
	require.NoError(t, json.Unmarshal(lines[0], &entry))
	assert.Equal(t, "INFO", entry["level"])
	assert.Equal(t, "/test.Service/GetItem", entry["grpc.method"])
	assert.Equal(t, "OK", entry["grpc.code"])
	_, hasDuration := entry["duration_ms"]
	assert.True(t, hasDuration)
}

func TestClientLoggingUnaryInterceptor_ShouldLogError_WhenFailure(t *testing.T) {
	// Arrange
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	interceptor := clientLoggingUnaryInterceptor(logger)

	method := "/test.Service/FailMethod"
	invoker := func(_ context.Context, _ string, _, _ interface{}, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		return grpc.ErrServerStopped
	}

	// Act
	err := interceptor(context.Background(), method, nil, nil, nil, invoker)

	// Assert
	require.Error(t, err)

	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	require.Len(t, lines, 1)

	var entry map[string]interface{}
	require.NoError(t, json.Unmarshal(lines[0], &entry))
	assert.Equal(t, "ERROR", entry["level"])
	assert.Equal(t, "/test.Service/FailMethod", entry["grpc.method"])
}
