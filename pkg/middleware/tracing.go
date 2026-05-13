package middleware

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
)

// traceFromContext extracts the trace ID from the OTel span in context.
// Returns empty string if no valid trace ID is present.
func traceFromContext(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return ""
	}
	traceID := span.SpanContext().TraceID()
	if !traceID.IsValid() {
		return ""
	}
	return traceID.String()
}

// TracingHandler returns middleware that integrates with the Tracer interface.
// For each request whose path should be traced (tracer.ShouldTrace), it starts
// an HTTP transaction, sets the transaction name to "METHOD /path", records
// standard attributes, and ends the transaction after the handler completes.
//
// If the tracer reports that the path should not be traced (e.g. health
// endpoints), the middleware is a no-op pass-through.
func (m *Middleware) TracingHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		if !m.tracer.ShouldTrace(path) {
			c.Next()
			return
		}

		ctx, txn := m.tracer.StartHTTPRequest(c.Request)
		c.Request = c.Request.WithContext(ctx)

		// Expose trace ID as response header for debugging.
		if span := traceFromContext(ctx); span != "" {
			c.Header("X-Trace-Id", span)
		}

		// Set transaction name: "GET /api/v1/users"
		txn.SetName(fmt.Sprintf("%s %s", c.Request.Method, c.FullPath()))

		// Record request attributes
		txn.AddAttribute("http.method", c.Request.Method)
		txn.AddAttribute("http.url", c.Request.RequestURI)
		txn.AddAttribute("http.route", c.FullPath())
		txn.AddAttribute("http.user_agent", c.Request.UserAgent())

		defer func() {
			// Record response status
			txn.AddAttribute("http.status_code", fmt.Sprintf("%d", c.Writer.Status()))

			// Record errors (5xx)
			if c.Writer.Status() >= http.StatusInternalServerError {
				for _, err := range c.Errors {
					txn.AddError(err.Err)
				}
			}

			txn.End()
		}()

		c.Next()
	}
}
