package middleware

import (
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


