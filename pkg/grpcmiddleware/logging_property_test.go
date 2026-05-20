package grpcmiddleware

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"pgregory.net/rapid"
)

func TestProperty7_LoggingInterceptorFieldsAndLevel(t *testing.T) {
	grpcCodes := []codes.Code{
		codes.OK,
		codes.Canceled,
		codes.Unknown,
		codes.InvalidArgument,
		codes.DeadlineExceeded,
		codes.NotFound,
		codes.AlreadyExists,
		codes.PermissionDenied,
		codes.ResourceExhausted,
		codes.FailedPrecondition,
		codes.Aborted,
		codes.OutOfRange,
		codes.Unimplemented,
		codes.Internal,
		codes.Unavailable,
		codes.DataLoss,
		codes.Unauthenticated,
	}

	rapid.Check(t, func(t *rapid.T) {
		service := rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9_.]{0,30}`).Draw(t, "service")
		method := rapid.StringMatching(`[A-Z][a-zA-Z0-9]{0,30}`).Draw(t, "method")
		fullMethod := "/" + service + "/" + method

		codeIdx := rapid.IntRange(0, len(grpcCodes)-1).Draw(t, "codeIdx")
		chosenCode := grpcCodes[codeIdx]

		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

		interceptors := NewInterceptors("", logger, nil, nil)
		interceptor := interceptors.LoggingUnaryInterceptor()

		info := &grpc.UnaryServerInfo{FullMethod: fullMethod}

		handler := func(_ context.Context, _ interface{}) (interface{}, error) {
			if chosenCode == codes.OK {
				return "ok", nil
			}
			return nil, status.Errorf(chosenCode, "simulated %s", chosenCode.String())
		}

		_, _ = interceptor(context.Background(), nil, info, handler)

		logOutput := buf.Bytes()
		require.NotEmpty(t, logOutput, "logger must produce output")

		lines := bytes.Split(bytes.TrimSpace(logOutput), []byte("\n"))
		require.Len(t, lines, 1, "exactly one log line expected per call")

		var entry map[string]interface{}
		err := json.Unmarshal(lines[0], &entry)
		require.NoError(t, err, "log output must be valid JSON")

		assert.Equal(t, method, entry["grpc.method"],
			"log entry must contain the gRPC method name")

		assert.Equal(t, chosenCode.String(), entry["grpc.code"],
			"log entry must contain the gRPC status code")

		durationRaw, ok := entry["duration_ms"]
		require.True(t, ok, "log entry must contain duration_ms")
		durationVal, ok := durationRaw.(float64)
		require.True(t, ok, "duration_ms must be a number")
		assert.GreaterOrEqual(t, durationVal, float64(0), "duration_ms must be non-negative")

		level, ok := entry["level"].(string)
		require.True(t, ok, "log entry must contain a level field")

		if chosenCode == codes.OK {
			assert.Equal(t, "INFO", level, "codes.OK must be logged at INFO level")
		} else {
			assert.True(t, level == "INFO" || level == "WARN" || level == "ERROR",
				"non-OK codes must be logged at INFO, WARN, or ERROR level, got: %s", level)
		}
	})
}
