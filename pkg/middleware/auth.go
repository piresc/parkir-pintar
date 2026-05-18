package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"parkir-pintar/pkg/auth"
	"parkir-pintar/pkg/response"

	"github.com/gin-gonic/gin"
)

// Context keys for authenticated user data.
const (
	KeyUserID = "user_id"
	KeyRole   = "role"
)

// JWTAuth returns middleware that validates a JWT Bearer token from the
// Authorization header. On success, user_id and role claims are set in
// the gin.Context. Returns 401 on missing, malformed, or invalid tokens.
func (m *Middleware) JWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Abort()
			response.Error(c, http.StatusUnauthorized, "missing authorization header")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.Abort()
			response.Error(c, http.StatusUnauthorized, "invalid authorization format")
			return
		}

		tokenString := parts[1]

		claims, err := auth.ValidateToken(tokenString, secret)
		if err != nil {
			c.Abort()
			response.Error(c, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		// Extract user_id and role from validated claims
		if claims.UserID == "" {
			c.Abort()
			response.Error(c, http.StatusUnauthorized, "token missing user identity")
			return
		}
		c.Set(KeyUserID, claims.UserID)
		if claims.Role != "" {
			c.Set(KeyRole, claims.Role)
		}

		c.Next()
	}
}

// APIKeyAuth returns middleware that validates the X-API-Key header against
// a map of expected service→key pairs using constant-time comparison to
// prevent timing attacks. Returns 401 on missing or invalid keys.
func (m *Middleware) APIKeyAuth(expectedKeys map[string]string) gin.HandlerFunc {
	type keyEntry struct {
		service string
		key     []byte
	}
	entries := make([]keyEntry, 0, len(expectedKeys))
	for svc, key := range expectedKeys {
		entries = append(entries, keyEntry{service: svc, key: []byte(key)})
	}

	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			c.Abort()
			response.Error(c, http.StatusUnauthorized, "missing API key")
			return
		}

		apiKeyBytes := []byte(apiKey)
		valid := false
		for _, entry := range entries {
			if subtle.ConstantTimeEq(int32(len(apiKeyBytes)), int32(len(entry.key))) == 1 &&
				subtle.ConstantTimeCompare(apiKeyBytes, entry.key) == 1 {
				valid = true
				break
			}
		}

		if !valid {
			c.Abort()
			response.Error(c, http.StatusUnauthorized, "invalid API key")
			return
		}

		c.Next()
	}
}
