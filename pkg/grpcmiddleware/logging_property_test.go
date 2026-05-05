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

// Feature: grpc-jwt-pkg-integration, Property 7: Logging interceptor fields and level
// **Validates: Requirements 6.1, 6.2**
//
// For any gRPC method name and handler result (success or error with any status
// code), the Logging interceptor SHALL produce a log entry containing the method
// name, gRPC status code, and duration in milliseconds, at INFO level for
// codes.OK and ERROR level for all other codes.
func TestProperty7_LoggingInterceptorFieldsAndLevel(t *testing.T) {
	// grpcCodes is the set of gRPC status codes we draw from. codes.OK is
	// included so that both the INFO and ERROR paths are exercised.
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
		// Generate a random gRPC method name.
		service := rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9_.]{0,30}`).Draw(t, "service")
		method := rapid.StringMatching(`[A-Z][a-zA-Z0-9]{0,30}`).Draw(t, "method")
		fullMethod := "/" + service + "/" + method

		// Pick a random gRPC status code.
		codeIdx := rapid.IntRange(0, len(grpcCodes)-1).Draw(t, "codeIdx")
		chosenCode := grpcCodes[codeIdx]

		// Set up a logger that writes JSON to a buffer.
		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

		interceptors := NewInterceptors("", logger, nil, nil)
		interceptor := interceptors.LoggingUnaryInterceptor()

		info := &grpc.UnaryServerInfo{FullMethod: fullMethod}

		// Build a handler that returns nil (success) or a gRPC status error.
		handler := func(_ context.Context, _ interface{}) (interface{}, error) {
			if chosenCode == codes.OK {
				return "ok", nil
			}
			return nil, status.Errorf(chosenCode, "simulated %s", chosenCode.String())
		}

		_, _ = interceptor(context.Background(), nil, info, handler)

		// Parse the JSON log entry.
		logOutput := buf.Bytes()
		require.NotEmpty(t, logOutput, "logger must produce output")

		lines := bytes.Split(bytes.TrimSpace(logOutput), []byte("\n"))
		require.Len(t, lines, 1, "exactly one log line expected per call")

		var entry map[string]interface{}
		err := json.Unmarshal(lines[0], &entry)
		require.NoError(t, err, "log output must be valid JSON")

		// Verify method name is present.
		assert.Equal(t, fullMethod, entry["grpc.method"],
			"log entry must contain the full gRPC method name")

		// Verify gRPC status code is present.
		assert.Equal(t, chosenCode.String(), entry["grpc.code"],
			"log entry must contain the gRPC status code")

		// Verify duration_ms is present and non-negative.
		durationRaw, ok := entry["duration_ms"]
		require.True(t, ok, "log entry must contain duration_ms")
		durationVal, ok := durationRaw.(float64)
		require.True(t, ok, "duration_ms must be a number")
		assert.GreaterOrEqual(t, durationVal, float64(0), "duration_ms must be non-negative")

		// Verify log level: INFO for codes.OK, ERROR for everything else.
		level, ok := entry["level"].(string)
		require.True(t, ok, "log entry must contain a level field")

		if chosenCode == codes.OK {
			assert.Equal(t, "INFO", level, "codes.OK must be logged at INFO level")
		} else {
			assert.Equal(t, "ERROR", level, "non-OK codes must be logged at ERROR level")
		}
	})
}
