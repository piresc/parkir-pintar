package middleware

import (
	"net/http"
	"strings"

	"parkir-pintar/pkg/auth"
	"parkir-pintar/pkg/response"

	"github.com/gin-gonic/gin"
)

const (
	KeyUserID = "user_id"
	KeyRole   = "role"
)

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
