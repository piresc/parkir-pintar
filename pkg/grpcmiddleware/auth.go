package grpcmiddleware

import (
	"context"
	"strings"

	"parkir-pintar/pkg/auth"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AuthUnaryInterceptor returns a grpc.UnaryServerInterceptor that validates
// JWT Bearer tokens from the "authorization" metadata key. On success,
// user_id and role claims are injected into the context. Methods listed in
// publicMethods bypass authentication entirely.
func (i *Interceptors) AuthUnaryInterceptor(publicMethods []string) grpc.UnaryServerInterceptor {
	public := toSet(publicMethods)

	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		if _, ok := public[info.FullMethod]; ok {
			return handler(ctx, req)
		}

		newCtx, err := i.authenticate(ctx)
		if err != nil {
			return nil, err
		}

		return handler(newCtx, req)
	}
}

// authenticate extracts and validates the Bearer token from gRPC metadata,
// returning a new context with user_id and role injected, or a gRPC status
// error on failure.
func (i *Interceptors) authenticate(ctx context.Context) (context.Context, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx, status.Errorf(codes.Unauthenticated, "missing authorization metadata")
	}

	values := md.Get("authorization")
	if len(values) == 0 {
		return ctx, status.Errorf(codes.Unauthenticated, "missing authorization metadata")
	}

	authHeader := values[0]
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ctx, status.Errorf(codes.Unauthenticated, "invalid or expired token")
	}

	tokenString := parts[1]

	claims, err := auth.ValidateToken(tokenString, i.jwtSecret)
	if err != nil {
		return ctx, status.Errorf(codes.Unauthenticated, "invalid or expired token")
	}

	newCtx := context.WithValue(ctx, CtxKeyUserID, claims.UserID)
	newCtx = context.WithValue(newCtx, CtxKeyRole, claims.Role)

	return newCtx, nil
}

// wrappedStream wraps a grpc.ServerStream to override its Context method,
// allowing interceptors to inject values into the stream's context.
type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

// Context returns the wrapped context instead of the original stream context.
func (w *wrappedStream) Context() context.Context {
	return w.ctx
}

// toSet converts a string slice to a map for O(1) lookups.
func toSet(items []string) map[string]struct{} {
	s := make(map[string]struct{}, len(items))
	for _, item := range items {
		s[item] = struct{}{}
	}
	return s
}
