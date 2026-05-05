package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// LogResponse returns middleware that logs response details after the handler
// completes. Logged fields include: transaction ID, MSISDN, HTTP method, URL,
// status code, and response time in milliseconds.
func (m *Middleware) LogResponse() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		m.logger.Info("[API Request]",
			slog.String("transaction_id", c.GetHeader("transactionid")),
			slog.String("msisdn", c.GetHeader("x-msisdn")),
			slog.String("method", c.Request.Method),
			slog.String("url", c.Request.RequestURI),
			slog.Int("status", c.Writer.Status()),
			slog.Int64("time_taken_ms", time.Since(start).Milliseconds()),
			slog.String("client_ip", c.ClientIP()),
		)
	}
}
