package grpcmiddleware

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"pgregory.net/rapid"
)

// Feature: grpc-jwt-pkg-integration, Property 4: Recovery catches any panic value
// **Validates: Requirements 3.1**
//
// For any panic value (string, error, integer, struct), the Recovery interceptor
// SHALL recover the panic and return a gRPC Internal status code without
// crashing the process.
func TestProperty4_RecoveryCatchesAnyPanicValue(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		interceptors := NewInterceptors("", nil, nil, nil)
		interceptor := interceptors.RecoveryUnaryInterceptor()
		info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}

		// Generate a panic value of varying types.
		panicKind := rapid.IntRange(0, 4).Draw(t, "panicKind")

		var panicValue interface{}
		switch panicKind {
		case 0: // string
			panicValue = rapid.String().Draw(t, "panicString")
		case 1: // error
			msg := rapid.String().Draw(t, "errorMsg")
			panicValue = errors.New(msg)
		case 2: // integer
			panicValue = rapid.Int().Draw(t, "panicInt")
		case 3: // struct
			panicValue = struct {
				Code    int
				Message string
			}{
				Code:    rapid.Int().Draw(t, "structCode"),
				Message: rapid.String().Draw(t, "structMsg"),
			}
		case 4: // nil (panic(nil) is valid in Go)
			panicValue = fmt.Errorf("explicit nil-like panic")
		}

		panickingHandler := func(_ context.Context, _ interface{}) (interface{}, error) {
			panic(panicValue)
		}

		resp, err := interceptor(context.Background(), nil, info, panickingHandler)

		// The interceptor must recover and return an error, not crash.
		assert.Nil(t, resp, "response must be nil after panic recovery")
		require.Error(t, err, "interceptor must return an error after panic recovery")

		st, ok := status.FromError(err)
		require.True(t, ok, "error must be a gRPC status")
		assert.Equal(t, codes.Internal, st.Code(), "must return Internal status code")
		assert.Equal(t, "internal server error", st.Message(), "must return generic error message")
	})
}
