package middleware

import "github.com/gin-gonic/gin"

// SecurityHeaders returns middleware that sets recommended HTTP security headers
// on every response. These headers mitigate common web vulnerabilities:
//
//   - X-Content-Type-Options: prevents MIME-type sniffing
//   - X-Frame-Options: prevents clickjacking
//   - X-XSS-Protection: disabled (modern approach relies on CSP)
//   - Strict-Transport-Security: enforces HTTPS
//   - Content-Security-Policy: restricts resource loading
//   - Referrer-Policy: controls referrer information leakage
//   - Permissions-Policy: restricts browser feature access
//   - Cache-Control: prevents caching of API responses
func (m *Middleware) SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()

		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("X-XSS-Protection", "0")
		h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		h.Set("Content-Security-Policy", "default-src 'self'")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		h.Set("Cache-Control", "no-store")

		c.Next()
	}
}
