package grpcmiddleware

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"parkir-pintar/pkg/auth"
	"parkir-pintar/pkg/config"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"pgregory.net/rapid"
)

const testSecret = "test-secret-for-property-tests"

func testJWTConfig() config.JWTConfig {
	return config.JWTConfig{
		Secret:     testSecret,
		Expiration: 60,
		Issuer:     "test-issuer",
	}
}

// noopHandler is a grpc.UnaryHandler that captures the context and returns nil.
func noopHandler(ctx context.Context, _ interface{}) (interface{}, error) {
	return ctx, nil
}

// Feature: grpc-jwt-pkg-integration, Property 1: Auth JWT round-trip
// **Validates: Requirements 2.1, 2.2**
//
// For any valid user_id and role strings, generating a JWT token with those
// claims and passing it through the Auth interceptor SHALL produce a context
// containing the same user_id and role values.
func TestProperty1_AuthJWTRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate non-empty user_id and role strings (GenerateToken requires non-empty userID).
		userID := rapid.StringMatching(`[a-zA-Z0-9_-]{1,64}`).Draw(t, "userID")
		role := rapid.StringMatching(`[a-zA-Z0-9_-]{0,32}`).Draw(t, "role")

		cfg := testJWTConfig()
		token, _, err := auth.GenerateToken(userID, role, cfg)
		require.NoError(t, err, "GenerateToken must succeed for valid inputs")

		interceptors := NewInterceptors(testSecret, nil, nil, nil)
		interceptor := interceptors.AuthUnaryInterceptor(nil)

		// Build incoming context with Bearer token in authorization metadata.
		md := metadata.Pairs("authorization", "Bearer "+token)
		ctx := metadata.NewIncomingContext(context.Background(), md)

		info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}

		resp, err := interceptor(ctx, nil, info, noopHandler)
		require.NoError(t, err, "interceptor must not return error for valid token")

		// The handler receives the enriched context as the response.
		enrichedCtx, ok := resp.(context.Context)
		require.True(t, ok, "handler must return context")

		gotUserID, ok := UserIDFromContext(enrichedCtx)
		assert.True(t, ok, "user_id must be present in context")
		assert.Equal(t, userID, gotUserID, "user_id must round-trip")

		gotRole, ok := RoleFromContext(enrichedCtx)
		assert.True(t, ok, "role must be present in context")
		assert.Equal(t, role, gotRole, "role must round-trip")
	})
}

// Feature: grpc-jwt-pkg-integration, Property 2: Invalid tokens are rejected
// **Validates: Requirements 2.4**
//
// For any random non-JWT string or expired JWT token, the Auth interceptor
// SHALL return a gRPC Unauthenticated status code with the message
// "invalid or expired token".
func TestProperty2_InvalidTokensAreRejected(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		interceptors := NewInterceptors(testSecret, nil, nil, nil)
		interceptor := interceptors.AuthUnaryInterceptor(nil)
		info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}

		// Choose between a random non-JWT string or an expired JWT token.
		useExpired := rapid.Bool().Draw(t, "useExpired")

		var tokenStr string
		if useExpired {
			// Create an expired JWT token.
			cfg := testJWTConfig()
			now := time.Now()
			claims := auth.Claims{
				UserID: "expired-user",
				Role:   "admin",
				RegisteredClaims: jwt.RegisteredClaims{
					ExpiresAt: jwt.NewNumericDate(now.Add(-1 * time.Hour)),
					IssuedAt:  jwt.NewNumericDate(now.Add(-2 * time.Hour)),
					Issuer:    cfg.Issuer,
				},
			}
			tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			signed, err := tok.SignedString([]byte(cfg.Secret))
			require.NoError(t, err)
			tokenStr = signed
		} else {
			// Generate a random non-JWT string.
			tokenStr = rapid.StringMatching(`[a-zA-Z0-9!@#$%^&*]{1,128}`).Draw(t, "randomToken")
		}

		md := metadata.Pairs("authorization", "Bearer "+tokenStr)
		ctx := metadata.NewIncomingContext(context.Background(), md)

		_, err := interceptor(ctx, nil, info, noopHandler)
		require.Error(t, err, "interceptor must reject invalid/expired tokens")

		st, ok := status.FromError(err)
		require.True(t, ok, "error must be a gRPC status")
		assert.Equal(t, codes.Unauthenticated, st.Code(), "must return Unauthenticated")
		assert.Equal(t, "invalid or expired token", st.Message(), "must return correct message")
	})
}

// Feature: grpc-jwt-pkg-integration, Property 3: Auth public method bypass
// **Validates: Requirements 2.5**
//
// For any gRPC method name and any set of public method names, the Auth
// interceptor SHALL skip authentication if and only if the method name is
// in the public methods set.
func TestProperty3_AuthPublicMethodBypass(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a set of public method names.
		numPublic := rapid.IntRange(1, 5).Draw(t, "numPublic")
		publicMethods := make([]string, numPublic)
		for i := 0; i < numPublic; i++ {
			svc := rapid.StringMatching(`[A-Z][a-zA-Z]{2,15}`).Draw(t, fmt.Sprintf("svc_%d", i))
			method := rapid.StringMatching(`[A-Z][a-zA-Z]{2,15}`).Draw(t, fmt.Sprintf("method_%d", i))
			publicMethods[i] = "/" + svc + "/" + method
		}

		// Decide whether the target method is in the public set or not.
		isPublic := rapid.Bool().Draw(t, "isPublic")

		var targetMethod string
		if isPublic {
			idx := rapid.IntRange(0, len(publicMethods)-1).Draw(t, "publicIdx")
			targetMethod = publicMethods[idx]
		} else {
			// Generate a method name that is NOT in the public set.
			for {
				svc := rapid.StringMatching(`[A-Z][a-zA-Z]{2,15}`).Draw(t, "privateSvc")
				method := rapid.StringMatching(`[A-Z][a-zA-Z]{2,15}`).Draw(t, "privateMethod")
				targetMethod = "/" + svc + "/" + method
				found := false
				for _, pm := range publicMethods {
					if pm == targetMethod {
						found = true
						break
					}
				}
				if !found {
					break
				}
			}
		}

		interceptors := NewInterceptors(testSecret, nil, nil, nil)
		interceptor := interceptors.AuthUnaryInterceptor(publicMethods)
		info := &grpc.UnaryServerInfo{FullMethod: targetMethod}

		// Use a context WITHOUT any authorization metadata.
		ctx := context.Background()

		handlerCalled := false
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			handlerCalled = true
			return "ok", nil
		}

		_, err := interceptor(ctx, nil, info, handler)

		if isPublic {
			// Public methods bypass auth — handler should be called, no error.
			assert.NoError(t, err, "public method must bypass auth")
			assert.True(t, handlerCalled, "handler must be called for public methods")
		} else {
			// Non-public methods without auth metadata must fail.
			assert.Error(t, err, "non-public method without auth must fail")
			assert.False(t, handlerCalled, "handler must NOT be called for non-public methods without auth")

			st, ok := status.FromError(err)
			require.True(t, ok, "error must be a gRPC status")
			assert.Equal(t, codes.Unauthenticated, st.Code())
			assert.True(t,
				strings.Contains(st.Message(), "missing authorization metadata") ||
					strings.Contains(st.Message(), "invalid or expired token"),
				"must return appropriate unauthenticated message, got: %s", st.Message())
		}
	})
}
