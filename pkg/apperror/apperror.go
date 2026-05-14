// Package apperror provides a structured application error type with HTTP status
// mapping and convenience constructors for common error scenarios.
//
// Best practices applied (from Go coding standards KB):
// - Document all exported functions and types with proper Godoc format
// - Use specific error types for different error conditions
// - Implement the error interface for custom error types
// - Use keyed fields in struct literals to prevent breakages during refactors
// - Never ignore errors; always handle them explicitly
// - Provide rich context in error messages
package apperror

import (
	"errors"
	"net/http"
)

// Sentinel errors for common application-level error conditions.
// Use errors.Is() to check for these across service boundaries.
var (
	ErrNotFound        = errors.New("resource not found")
	ErrAlreadyExists   = errors.New("resource already exists")
	ErrUnauthorized    = errors.New("unauthorized access")
	ErrForbidden       = errors.New("forbidden")
	ErrBadRequest      = errors.New("bad request")
	ErrInternal        = errors.New("internal server error")
	ErrTimeout         = errors.New("operation timed out")
	ErrConflict        = errors.New("resource conflict")
	ErrValidation      = errors.New("validation failed")
	ErrServiceUnavail  = errors.New("service unavailable")
	ErrRateLimited     = errors.New("rate limit exceeded")
	ErrDatabaseError   = errors.New("database error")
	ErrCacheError      = errors.New("cache error")
	ErrMessagingError  = errors.New("messaging error")
	ErrExternalService = errors.New("external service error")
	ErrInvalidToken    = errors.New("invalid token")
	ErrTokenExpired    = errors.New("token expired")
)

// AppError represents a structured application error with an error code,
// human-readable message, and associated HTTP status code.
// The HTTPStatus field is excluded from JSON serialization since it is used
// to set the HTTP response status, not included in the response body.
type AppError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"-"`
}

// Error implements the error interface, returning the error message.
func (e *AppError) Error() string {
	return e.Message
}

// New creates a new AppError with the given code, message, and HTTP status.
func New(code, message string, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
	}
}

// Internal creates an AppError for internal server errors (HTTP 500).
func Internal(message string) *AppError {
	return New("INTERNAL_ERROR", message, http.StatusInternalServerError)
}

// BadRequest creates an AppError for bad request errors (HTTP 400).
func BadRequest(message string) *AppError {
	return New("BAD_REQUEST", message, http.StatusBadRequest)
}

// NotFound creates an AppError for not found errors (HTTP 404).
func NotFound(message string) *AppError {
	return New("NOT_FOUND", message, http.StatusNotFound)
}

// Unauthorized creates an AppError for unauthorized errors (HTTP 401).
func Unauthorized(message string) *AppError {
	return New("UNAUTHORIZED", message, http.StatusUnauthorized)
}
