package grpcmiddleware

// contextKey is an unexported type used for context value keys to avoid
type contextKey string

const (
	CtxKeyUserID contextKey = "user_id"
	CtxKeyRole   contextKey = "role"
)
