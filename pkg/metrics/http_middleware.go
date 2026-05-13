package metrics

import (
	"fmt"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
)

// responseWriter wraps gin.ResponseWriter to capture status code and response size.
type responseWriter struct {
	gin.ResponseWriter
	statusCode int
	size       int
}

func newResponseWriter(w gin.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     200,
	}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.size += n
	return n, err
}

// uuidPattern matches UUID-like path segments.
var uuidPattern = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)

// numericPattern matches purely numeric path segments.
var numericPattern = regexp.MustCompile(`^[0-9]+$`)

// normalizePath replaces dynamic path segments (UUIDs, numeric IDs) with :id
// to reduce metric cardinality.
func normalizePath(path string) string {
	// Replace UUIDs first.
	normalized := uuidPattern.ReplaceAllString(path, ":id")

	// Replace remaining numeric-only segments.
	// Split by / and check each segment.
	parts := splitPath(normalized)
	for i, part := range parts {
		if numericPattern.MatchString(part) {
			parts[i] = ":id"
		}
	}
	return joinPath(parts)
}

func splitPath(path string) []string {
	var parts []string
	current := ""
	for _, c := range path {
		if c == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
			parts = append(parts, "/")
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

func joinPath(parts []string) string {
	result := ""
	for _, p := range parts {
		result += p
	}
	return result
}

// HTTPMiddleware returns a Gin middleware that records HTTP request metrics.
func (m *Metrics) HTTPMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		rw := newResponseWriter(c.Writer)
		c.Writer = rw

		c.Next()

		duration := time.Since(start).Seconds()
		method := c.Request.Method

		// Use the matched route pattern if available; fall back to normalized path.
		path := c.FullPath()
		if path == "" {
			path = normalizePath(c.Request.URL.Path)
		}

		statusCode := fmt.Sprintf("%d", rw.statusCode)

		attrs := otelmetric.WithAttributes(
			attribute.String("method", method),
			attribute.String("path", path),
			attribute.String("status_code", statusCode),
		)

		m.HTTPRequestsTotal.Add(c.Request.Context(), 1, attrs)
		m.HTTPRequestDuration.Record(c.Request.Context(), duration, attrs)

		sizeAttrs := otelmetric.WithAttributes(
			attribute.String("method", method),
			attribute.String("path", path),
		)
		m.HTTPResponseSize.Record(c.Request.Context(), int64(rw.size), sizeAttrs)
	}
}
