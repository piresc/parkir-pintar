// Package response provides standardized JSON response helpers for Gin handlers.
//
// Best practices applied (from Go coding standards KB):
// - Document all exported functions and types with proper Godoc format
// - Use keyed fields in struct literals to prevent breakages during refactors
// - Keep functions focused with single responsibilities
// - Use context.Context-aware patterns (via gin.Context)
//
// Key fix from boilerplate-golang: Error always uses the provided statusCode
// for BOTH the HTTP status AND the JSON body. The original FailedBadRequest /
// FailedInternalServerError functions accepted a statusCode param but ignored
// it for the HTTP status, which was misleading.
package response

import (
	"github.com/gin-gonic/gin"
)

// SuccessResponse is the standard envelope for successful API responses.
type SuccessResponse struct {
	Status string `json:"status"`
	Data   any    `json:"data"`
}

// ErrorResponse is the standard envelope for error API responses.
type ErrorResponse struct {
	Status    string `json:"status"`
	Error     string `json:"error"`
	RequestID string `json:"request_id,omitempty"`
}

// Success writes a JSON success response with the given HTTP status code and data.
// The response body always contains {"status": "success", "data": ...}.
func Success(c *gin.Context, statusCode int, data any) {
	c.JSON(statusCode, SuccessResponse{
		Status: "success",
		Data:   data,
	})
}

// Error writes a JSON error response with the given HTTP status code and message.
// The HTTP status code is used for BOTH the HTTP response status and the JSON body,
// fixing the misleading helpers from the original boilerplate-golang.
func Error(c *gin.Context, statusCode int, err string) {
	c.JSON(statusCode, ErrorResponse{
		Status: "error",
		Error:  err,
	})
}

// ErrorWithRequestID writes a JSON error response that includes a request ID
// for traceability. Useful in recovery middleware and error handlers.
func ErrorWithRequestID(c *gin.Context, statusCode int, err, requestID string) {
	c.JSON(statusCode, ErrorResponse{
		Status:    "error",
		Error:     err,
		RequestID: requestID,
	})
}
