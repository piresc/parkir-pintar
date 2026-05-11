package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"parkir-pintar/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
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

		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(secret), nil
		}, jwt.WithValidMethods([]string{"HS256"}))
		if err != nil || !token.Valid {
			c.Abort()
			response.Error(c, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.Abort()
			response.Error(c, http.StatusUnauthorized, "invalid token claims")
			return
		}

		// Extract user_id and role from claims
		if userID, exists := claims["user_id"]; exists {
			c.Set(KeyUserID, userID)
		}
		if role, exists := claims["role"]; exists {
			c.Set(KeyRole, role)
		}

		c.Next()
	}
}

// APIKeyAuth returns middleware that validates the X-API-Key header against
// a map of expected service→key pairs. Returns 401 on missing or invalid keys.
func (m *Middleware) APIKeyAuth(expectedKeys map[string]string) gin.HandlerFunc {
	// Build a reverse lookup: key → service name for O(1) validation.
	validKeys := make(map[string]string, len(expectedKeys))
	for svc, key := range expectedKeys {
		validKeys[key] = svc
	}

	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			c.Abort()
			response.Error(c, http.StatusUnauthorized, "missing API key")
			return
		}

		if _, ok := validKeys[apiKey]; !ok {
			c.Abort()
			response.Error(c, http.StatusUnauthorized, "invalid API key")
			return
		}

		c.Next()
	}
}
