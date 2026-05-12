// Best practices applied from Go testing guidelines:
// - Descriptive test names using ShouldXXX_WhenYYY pattern
// - AAA (Arrange-Act-Assert) structure
// - testify assertions (assert, require)
// - Direct interceptor invocation with mock handlers (no bufconn needed)

package grpcmiddleware

import (
	"context"
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
)

const unitTestSecret = "unit-test-secret-key"

func unitTestJWTConfig() config.JWTConfig {
	return config.JWTConfig{
		Secret:     unitTestSecret,
		Expiration: 60,
		Issuer:     "unit-test-issuer",
	}
}

// ctxCapturingHandler returns a handler that captures the context it receives.
func ctxCapturingHandler() (grpc.UnaryHandler, *context.Context) {
	var captured context.Context
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		captured = ctx
		return "ok", nil
	}
	return handler, &captured
}

func TestAuthUnaryInterceptor_ShouldInjectClaims_WhenValidToken(t *testing.T) {
	// Arrange
	cfg := unitTestJWTConfig()
	token, _, err := auth.GenerateToken("user-123", "admin", cfg)
	require.NoError(t, err)

	interceptors := NewInterceptors(unitTestSecret, nil, nil, nil)
	interceptor := interceptors.AuthUnaryInterceptor(nil)

	md := metadata.Pairs("authorization", "Bearer "+token)
	ctx := metadata.NewIncomingContext(context.Background(), md)
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/GetUser"}
	handler, captured := ctxCapturingHandler()

	// Act
	resp, err := interceptor(ctx, nil, info, handler)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "ok", resp)

	userID, ok := UserIDFromContext(*captured)
	assert.True(t, ok)
	assert.Equal(t, "user-123", userID)

	role, ok := RoleFromContext(*captured)
	assert.True(t, ok)
	assert.Equal(t, "admin", role)
}

func TestAuthUnaryInterceptor_ShouldReturnUnauthenticated_WhenTokenExpired(t *testing.T) {
	// Arrange
	cfg := unitTestJWTConfig()
	now := time.Now()
	claims := auth.Claims{
		UserID: "expired-user",
		Role:   "viewer",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now.Add(-2 * time.Hour)),
			Issuer:    cfg.Issuer,
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString([]byte(cfg.Secret))
	require.NoError(t, err)

	interceptors := NewInterceptors(unitTestSecret, nil, nil, nil)
	interceptor := interceptors.AuthUnaryInterceptor(nil)

	md := metadata.Pairs("authorization", "Bearer "+signed)
	ctx := metadata.NewIncomingContext(context.Background(), md)
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/GetUser"}

	// Act
	_, err = interceptor(ctx, nil, info, noopHandler)

	// Assert
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.Equal(t, "invalid or expired token", st.Message())
}

func TestAuthUnaryInterceptor_ShouldReturnUnauthenticated_WhenMetadataMissing(t *testing.T) {
	// Arrange
	interceptors := NewInterceptors(unitTestSecret, nil, nil, nil)
	interceptor := interceptors.AuthUnaryInterceptor(nil)

	// Context without any metadata.
	ctx := context.Background()
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/GetUser"}

	// Act
	_, err := interceptor(ctx, nil, info, noopHandler)

	// Assert
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.Equal(t, "missing authorization metadata", st.Message())
}

func TestAuthUnaryInterceptor_ShouldReturnUnauthenticated_WhenAuthorizationHeaderEmpty(t *testing.T) {
	// Arrange
	interceptors := NewInterceptors(unitTestSecret, nil, nil, nil)
	interceptor := interceptors.AuthUnaryInterceptor(nil)

	// Metadata present but no authorization key.
	md := metadata.Pairs("other-key", "some-value")
	ctx := metadata.NewIncomingContext(context.Background(), md)
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/GetUser"}

	// Act
	_, err := interceptor(ctx, nil, info, noopHandler)

	// Assert
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.Equal(t, "missing authorization metadata", st.Message())
}

func TestAuthUnaryInterceptor_ShouldReturnUnauthenticated_WhenMalformedBearer(t *testing.T) {
	// Arrange
	interceptors := NewInterceptors(unitTestSecret, nil, nil, nil)
	interceptor := interceptors.AuthUnaryInterceptor(nil)

	// Malformed: no "Bearer " prefix.
	md := metadata.Pairs("authorization", "Basic some-token")
	ctx := metadata.NewIncomingContext(context.Background(), md)
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/GetUser"}

	// Act
	_, err := interceptor(ctx, nil, info, noopHandler)

	// Assert
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.Equal(t, "invalid or expired token", st.Message())
}

func TestAuthUnaryInterceptor_ShouldReturnUnauthenticated_WhenTokenOnlyBearer(t *testing.T) {
	// Arrange — "Bearer" with no token part.
	interceptors := NewInterceptors(unitTestSecret, nil, nil, nil)
	interceptor := interceptors.AuthUnaryInterceptor(nil)

	md := metadata.Pairs("authorization", "Bearer")
	ctx := metadata.NewIncomingContext(context.Background(), md)
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/GetUser"}

	// Act
	_, err := interceptor(ctx, nil, info, noopHandler)

	// Assert
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.Equal(t, "invalid or expired token", st.Message())
}

func TestAuthUnaryInterceptor_ShouldBypassAuth_WhenPublicMethod(t *testing.T) {
	// Arrange
	publicMethods := []string{"/test.Service/Health", "/test.Service/Ping"}
	interceptors := NewInterceptors(unitTestSecret, nil, nil, nil)
	interceptor := interceptors.AuthUnaryInterceptor(publicMethods)

	// No authorization metadata — should still succeed for public methods.
	ctx := context.Background()
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Health"}

	handlerCalled := false
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		handlerCalled = true
		return "healthy", nil
	}

	// Act
	resp, err := interceptor(ctx, nil, info, handler)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "healthy", resp)
	assert.True(t, handlerCalled)
}

func TestAuthUnaryInterceptor_ShouldRequireAuth_WhenMethodNotPublic(t *testing.T) {
	// Arrange
	publicMethods := []string{"/test.Service/Health"}
	interceptors := NewInterceptors(unitTestSecret, nil, nil, nil)
	interceptor := interceptors.AuthUnaryInterceptor(publicMethods)

	// Non-public method without auth.
	ctx := context.Background()
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/GetUser"}

	// Act
	_, err := interceptor(ctx, nil, info, noopHandler)

	// Assert
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestAuthStreamInterceptor_ShouldReturnUnauthenticated_WhenMetadataMissing(t *testing.T) {
	// Arrange
	interceptors := NewInterceptors(unitTestSecret, nil, nil, nil)
	interceptor := interceptors.AuthStreamInterceptor(nil)

	ctx := context.Background()
	info := &grpc.StreamServerInfo{FullMethod: "/test.Service/StreamData"}
	ss := &mockServerStream{ctx: ctx}

	handlerCalled := false
	handler := func(srv interface{}, stream grpc.ServerStream) error {
		handlerCalled = true
		return nil
	}

	// Act
	err := interceptor(nil, ss, info, handler)

	// Assert
	require.Error(t, err)
	assert.False(t, handlerCalled)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.Equal(t, "missing authorization metadata", st.Message())
}

func TestAuthStreamInterceptor_ShouldBypassAuth_WhenPublicMethod(t *testing.T) {
	// Arrange
	publicMethods := []string{"/test.Service/StreamHealth"}
	interceptors := NewInterceptors(unitTestSecret, nil, nil, nil)
	interceptor := interceptors.AuthStreamInterceptor(publicMethods)

	ctx := context.Background()
	info := &grpc.StreamServerInfo{FullMethod: "/test.Service/StreamHealth"}
	ss := &mockServerStream{ctx: ctx}

	handlerCalled := false
	handler := func(srv interface{}, stream grpc.ServerStream) error {
		handlerCalled = true
		return nil
	}

	// Act
	err := interceptor(nil, ss, info, handler)

	// Assert
	require.NoError(t, err)
	assert.True(t, handlerCalled)
}

// mockServerStream is a minimal grpc.ServerStream implementation for testing.
type mockServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (m *mockServerStream) Context() context.Context {
	return m.ctx
}
