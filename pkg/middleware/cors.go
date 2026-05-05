package middleware

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CorsHandler returns CORS middleware configured with explicit allowed origins.
//
// CRITICAL FIX from boilerplate-golang: Uses AllowOrigins (explicit list)
// instead of AllowAllOrigins: true. The original combined AllowAllOrigins
// with AllowCredentials, which is a security vulnerability (browsers reject
// this combination, and it signals misconfiguration).
//
// AllowCredentials is set to true only for the explicitly listed origins.
func (m *Middleware) CorsHandler(allowedOrigins []string) gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins: allowedOrigins,
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Authorization",
			"X-API-Key",
			"X-Request-ID",
			"transactionid",
			"x-msisdn",
		},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}
