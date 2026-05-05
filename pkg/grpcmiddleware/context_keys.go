package grpcmiddleware

import "context"

// contextKey is an unexported type used for context value keys to avoid
// collisions with keys defined in other packages.
type contextKey string

// Context key constants for authenticated user data injected by the Auth
// interceptor.
const (
	CtxKeyUserID contextKey = "user_id"
	CtxKeyRole   contextKey = "role"
)

// UserIDFromContext extracts the user_id value set by the Auth interceptor.
// Returns the user ID and true if present, or an empty string and false otherwise.
func UserIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(CtxKeyUserID).(string)
	return v, ok
}

// RoleFromContext extracts the role value set by the Auth interceptor.
// Returns the role and true if present, or an empty string and false otherwise.
func RoleFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(CtxKeyRole).(string)
	return v, ok
}
