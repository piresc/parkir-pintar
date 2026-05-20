package grpcmiddleware

// contextKey is an unexported type used for context value keys to avoid
// collisions with keys defined in other packages.
type contextKey string

// Context key constants for authenticated user data injected by the Auth
// interceptor.
const (
	CtxKeyUserID contextKey = "user_id"
	CtxKeyRole   contextKey = "role"
)


