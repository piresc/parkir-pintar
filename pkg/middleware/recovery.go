package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"parkir-pintar/pkg/response"

	"github.com/gin-gonic/gin"
	traceapi "go.opentelemetry.io/otel/trace"
)

// RecoveryHandler returns middleware that catches panics, logs the stack trace
// at ERROR level, and returns a 500 response with a generic error message and
// request_id. If tracing is active, the panic error is recorded on the
// current transaction.
func (m *Middleware) RecoveryHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()

				err := fmt.Errorf("panic recovered: %v", r)

				m.logger.Error("panic recovered",
					slog.String("error", fmt.Sprintf("%v", r)),
					slog.String("stack", string(stack)),
					slog.String("method", c.Request.Method),
					slog.String("path", c.Request.URL.Path),
				)

				// Record error on tracing transaction if active
				if m.tracer.ShouldTrace(c.Request.URL.Path) {
					// Try to get the transaction from context — the tracing
					// middleware may have started one before the panic.
					// We record the error via a new segment since we don't
					// have direct access to the HTTPTransaction here.
					ctx, end := m.tracer.StartSegment(c.Request.Context(), "panic.recovery")
					// Record the panic error on the tracing span.
					if span := traceapi.SpanFromContext(ctx); span.IsRecording() {
						span.RecordError(err)
					}
					end()
				}

				requestID := c.GetHeader("transactionid")
				if requestID == "" {
					requestID = c.GetHeader("X-Request-ID")
				}

				c.Abort()
				response.ErrorWithRequestID(c, http.StatusInternalServerError,
					"internal server error", requestID)
			}
		}()

		c.Next()
	}
}
