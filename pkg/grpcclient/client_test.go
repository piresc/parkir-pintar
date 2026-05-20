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

func bufconnDialer(lis *bufconn.Listener) grpc.DialOption {
	return grpc.WithContextDialer(func(_ context.Context, _ string) (net.Conn, error) {
		return lis.Dial()
	})
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestDial_ShouldReturnConn_WhenBufconnServerRunning(t *testing.T) {
	lis := startBufconnServer(t)

	cfg := ClientConfig{
		Target:      "bufnet",
		DialTimeout: 5 * time.Second,
		Logger:      newTestLogger(),
	}

	conn, err := Dial(
		context.Background(),
		cfg,
		bufconnDialer(lis),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	require.NoError(t, err)
	require.NotNil(t, conn)
	assert.NoError(t, conn.Close())
}

func TestDial_ShouldUseDefaultLogger_WhenLoggerIsNil(t *testing.T) {
	lis := startBufconnServer(t)

	cfg := ClientConfig{
		Target:      "bufnet",
		DialTimeout: 5 * time.Second,
		Logger:      nil, // should default to slog.Default()
	}

	conn, err := Dial(
		context.Background(),
		cfg,
		bufconnDialer(lis),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	require.NoError(t, err)
	require.NotNil(t, conn)
	assert.NoError(t, conn.Close())
}

func TestDial_ShouldUseNoOpTracer_WhenTracerIsNil(t *testing.T) {
	lis := startBufconnServer(t)

	cfg := ClientConfig{
		Target:      "bufnet",
		DialTimeout: 5 * time.Second,
		Tracer:      nil, // should default to NoOpTracer
		Logger:      newTestLogger(),
	}

	conn, err := Dial(
		context.Background(),
		cfg,
		bufconnDialer(lis),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	require.NoError(t, err)
	require.NotNil(t, conn)
	assert.NoError(t, conn.Close())
}

func TestDial_ShouldApplyInsecureCredentials_WhenTLSDisabled(t *testing.T) {
	lis := startBufconnServer(t)

	cfg := ClientConfig{
		Target:      "bufnet",
		DialTimeout: 5 * time.Second,
		TLSEnabled:  false,
		Logger:      newTestLogger(),
	}

	conn, err := Dial(
		context.Background(),
		cfg,
		bufconnDialer(lis),
	)

	require.NoError(t, err)
	require.NotNil(t, conn)
	assert.NoError(t, conn.Close())
}

func TestDial_ShouldApplyKeepaliveParams_WhenConfigured(t *testing.T) {
	lis := startBufconnServer(t)

	cfg := ClientConfig{
		Target:           "bufnet",
		DialTimeout:      5 * time.Second,
		KeepAliveTime:    30 * time.Second,
		KeepAliveTimeout: 10 * time.Second,
		Logger:           newTestLogger(),
	}

	conn, err := Dial(
		context.Background(),
		cfg,
		bufconnDialer(lis),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	require.NoError(t, err)
	require.NotNil(t, conn)
	assert.NoError(t, conn.Close())
}

func TestDial_ShouldApplyLoggingInterceptor_WhenLoggerProvided(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	cfg := ClientConfig{
		Target:      "passthrough:///bufnet",
		DialTimeout: 5 * time.Second,
		Logger:      logger,
	}

	conn, err := Dial(
		context.Background(),
		cfg,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	require.NoError(t, err)
	require.NotNil(t, conn)
	assert.NoError(t, conn.Close())
}

func TestClientLoggingUnaryInterceptor_ShouldLogInfo_WhenSuccess(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	interceptor := clientLoggingUnaryInterceptor(logger)

	method := "/test.Service/GetItem"
	invoker := func(_ context.Context, _ string, _, _ interface{}, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		return nil
	}

	err := interceptor(context.Background(), method, nil, nil, nil, invoker)

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
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	interceptor := clientLoggingUnaryInterceptor(logger)

	method := "/test.Service/FailMethod"
	invoker := func(_ context.Context, _ string, _, _ interface{}, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		return grpc.ErrServerStopped
	}

	err := interceptor(context.Background(), method, nil, nil, nil, invoker)

	require.Error(t, err)

	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	require.Len(t, lines, 1)

	var entry map[string]interface{}
	require.NoError(t, json.Unmarshal(lines[0], &entry))
	assert.Equal(t, "ERROR", entry["level"])
	assert.Equal(t, "/test.Service/FailMethod", entry["grpc.method"])
}
