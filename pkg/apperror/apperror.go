// - Use keyed fields in struct literals to prevent breakages during refactors
// - Never ignore errors; always handle them explicitly
package apperror

import (
	"errors"
	"net/http"
)

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

// The HTTPStatus field is excluded from JSON serialization since it is used
type AppError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"-"`
}

func (e *AppError) Error() string {
	return e.Message
}

func New(code, message string, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
	}
}

func Internal(message string) *AppError {
	return New("INTERNAL_ERROR", message, http.StatusInternalServerError)
}

func BadRequest(message string) *AppError {
	return New("BAD_REQUEST", message, http.StatusBadRequest)
}

func NotFound(message string) *AppError {
	return New("NOT_FOUND", message, http.StatusNotFound)
}

func Forbidden(message string) *AppError {
	return New("FORBIDDEN", message, http.StatusForbidden)
}

func Conflict(message string) *AppError {
	return New("CONFLICT", message, http.StatusConflict)
}

func ServiceUnavailable(message string) *AppError {
	return New("SERVICE_UNAVAILABLE", message, http.StatusServiceUnavailable)
}

func PaymentFailed(message string) *AppError {
	return New("PAYMENT_FAILED", message, http.StatusPaymentRequired)
}

func UnprocessableEntity(message string) *AppError {
	return New("UNPROCESSABLE_ENTITY", message, http.StatusUnprocessableEntity)
}
