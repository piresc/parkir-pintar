package middleware

import (
	"net/http"

	"parkir-pintar/pkg/response"

	"github.com/gin-gonic/gin"
)

// Role constants define the available roles in the system.
const (
	RoleAdmin    = "admin"
	RoleOperator = "operator"
	RoleDriver   = "driver"
)

// RequireRole returns middleware that enforces the user has one of the
// specified roles. All listed roles must match (AND logic). The role is
// read from the gin.Context key set by JWTAuth middleware.
// Returns 403 Forbidden if the role doesn't match.
func (m *Middleware) RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get(KeyRole)
		if !exists || userRole == nil {
			c.Abort()
			response.Error(c, http.StatusForbidden, "missing role claim")
			return
		}

		roleStr, ok := userRole.(string)
		if !ok || roleStr == "" {
			c.Abort()
			response.Error(c, http.StatusForbidden, "invalid role claim")
			return
		}

		for _, required := range roles {
			if roleStr == required {
				c.Next()
				return
			}
		}

		c.Abort()
		response.Error(c, http.StatusForbidden, "insufficient permissions")
	}
}

// RequireAnyRole returns middleware that enforces the user has at least one
// of the specified roles (OR logic). This is functionally equivalent to
// RequireRole since both use OR matching, but is provided for semantic
// clarity when the intent is explicitly "any of these roles".
func (m *Middleware) RequireAnyRole(roles ...string) gin.HandlerFunc {
	return m.RequireRole(roles...)
}
