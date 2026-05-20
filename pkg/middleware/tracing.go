package middleware

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
)

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

func (m *Middleware) TracingHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		if !m.tracer.ShouldTrace(path) {
			c.Next()
			return
		}

		ctx, txn := m.tracer.StartHTTPRequest(c.Request)
		c.Request = c.Request.WithContext(ctx)

		if span := traceFromContext(ctx); span != "" {
			c.Header("X-Trace-Id", span)
		}

		txn.SetName(fmt.Sprintf("%s %s", c.Request.Method, c.FullPath()))

		txn.AddAttribute("http.method", c.Request.Method)
		txn.AddAttribute("http.url", c.Request.RequestURI)
		txn.AddAttribute("http.route", c.FullPath())
		txn.AddAttribute("http.user_agent", c.Request.UserAgent())

		defer func() {
			txn.AddAttribute("http.status_code", fmt.Sprintf("%d", c.Writer.Status()))

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
