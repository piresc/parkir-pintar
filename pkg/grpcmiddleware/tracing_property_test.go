package grpcmiddleware

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"testing"

	"parkir-pintar/pkg/tracing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"pgregory.net/rapid"
)

type spyTracer struct {
	*tracing.NoOpTracer
	mu           sync.Mutex
	segmentNames []string
}

func newSpyTracer() *spyTracer {
	noop := tracing.NewNoOpTracer().(*tracing.NoOpTracer)
	return &spyTracer{NoOpTracer: noop}
}

func (s *spyTracer) StartSegment(ctx context.Context, name string) (context.Context, func()) {
	s.mu.Lock()
	s.segmentNames = append(s.segmentNames, name)
	s.mu.Unlock()
	return s.NoOpTracer.StartSegment(ctx, name)
}

func (s *spyTracer) lastSegmentName() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.segmentNames) == 0 {
		return ""
	}
	return s.segmentNames[len(s.segmentNames)-1]
}

// attributes (logged as structured fields since the Tracer interface does not
func TestProperty5_TracingSpanNameAndAttributes(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		service := rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9_.]{0,30}`).Draw(t, "service")
		method := rapid.StringMatching(`[A-Z][a-zA-Z0-9]{0,30}`).Draw(t, "method")
		fullMethod := "/" + service + "/" + method

		spy := newSpyTracer()

		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

		interceptors := NewInterceptors("", logger, spy, nil)
		interceptor := interceptors.TracingUnaryInterceptor()

		info := &grpc.UnaryServerInfo{FullMethod: fullMethod}

		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			return "ok", nil
		}

		_, err := interceptor(context.Background(), nil, info, handler)
		require.NoError(t, err)

		assert.Equal(t, fullMethod, spy.lastSegmentName(),
			"StartSegment must be called with the full gRPC method name")

		logOutput := buf.String()
		require.NotEmpty(t, logOutput, "logger must produce output")

		lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
		require.GreaterOrEqual(t, len(lines), 1, "must have at least one log line")

		var logEntry map[string]interface{}
		err = json.Unmarshal(lines[0], &logEntry)
		require.NoError(t, err, "log output must be valid JSON")

		assert.Equal(t, "grpc", logEntry["rpc.system"], "rpc.system must be grpc")
		assert.Equal(t, service, logEntry["rpc.service"], "rpc.service must match service")
		assert.Equal(t, method, logEntry["rpc.method"], "rpc.method must match method")
	})
}
