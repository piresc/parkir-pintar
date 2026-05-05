// Package apperror tests
//
// Best practices applied (from Go testing standards KB):
// - Use descriptive names: Test[FunctionName]_Should[ExpectedResult]_When[Condition]
// - Follow AAA (Arrange-Act-Assert) pattern
// - Table-driven tests for multiple scenarios
// - Use testify assertions for clear failure messages
// - Tests are fast, isolated, repeatable, and clear
// - Test both success and error/edge cases
// - Verify error interface implementation
package apperror

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- Error interface tests ---

func TestAppError_ShouldImplementErrorInterface(t *testing.T) {
	// Arrange
	appErr := New("TEST_CODE", "test message", http.StatusBadRequest)

	// Act
	var err error = appErr

	// Assert
	assert.Equal(t, "test message", err.Error())
}

func TestAppError_ShouldReturnMessage_WhenErrorCalled(t *testing.T) {
	// Arrange
	appErr := &AppError{
		Code:       "CUSTOM",
		Message:    "custom error message",
		HTTPStatus: http.StatusConflict,
	}

	// Act
	result := appErr.Error()

	// Assert
	assert.Equal(t, "custom error message", result)
}

// --- New constructor tests ---

func TestNew_ShouldCreateAppError_WhenValidParamsProvided(t *testing.T) {
	// Act
	appErr := New("VALIDATION_ERROR", "field is required", http.StatusBadRequest)

	// Assert
	assert.Equal(t, "VALIDATION_ERROR", appErr.Code)
	assert.Equal(t, "field is required", appErr.Message)
	assert.Equal(t, http.StatusBadRequest, appErr.HTTPStatus)
}

// --- Convenience constructor tests (table-driven) ---

func TestConvenienceConstructors_ShouldSetCorrectHTTPStatus_WhenCalled(t *testing.T) {
	// Arrange — table-driven test for all convenience constructors
	tests := []struct {
		name           string
		constructor    func(string) *AppError
		expectedCode   string
		expectedStatus int
	}{
		{
			name:           "Internal returns 500",
			constructor:    Internal,
			expectedCode:   "INTERNAL_ERROR",
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "BadRequest returns 400",
			constructor:    BadRequest,
			expectedCode:   "BAD_REQUEST",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "NotFound returns 404",
			constructor:    NotFound,
			expectedCode:   "NOT_FOUND",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Unauthorized returns 401",
			constructor:    Unauthorized,
			expectedCode:   "UNAUTHORIZED",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			appErr := tt.constructor("test message")

			// Assert
			assert.Equal(t, tt.expectedCode, appErr.Code)
			assert.Equal(t, "test message", appErr.Message)
			assert.Equal(t, tt.expectedStatus, appErr.HTTPStatus)
			assert.Equal(t, "test message", appErr.Error())
		})
	}
}

// --- Sentinel error tests ---

func TestSentinelErrors_ShouldBeDistinct_WhenCompared(t *testing.T) {
	// Verify all sentinel errors are unique and non-nil
	sentinels := []error{
		ErrNotFound,
		ErrAlreadyExists,
		ErrUnauthorized,
		ErrForbidden,
		ErrBadRequest,
		ErrInternal,
		ErrTimeout,
		ErrConflict,
		ErrValidation,
		ErrServiceUnavail,
		ErrRateLimited,
		ErrDatabaseError,
		ErrCacheError,
		ErrMessagingError,
		ErrExternalService,
		ErrInvalidToken,
		ErrTokenExpired,
	}

	for i, err := range sentinels {
		assert.NotNil(t, err, "sentinel error at index %d should not be nil", i)
		assert.NotEmpty(t, err.Error(), "sentinel error at index %d should have a message", i)
	}

	// Verify no two sentinel errors are the same
	for i := 0; i < len(sentinels); i++ {
		for j := i + 1; j < len(sentinels); j++ {
			assert.NotEqual(t, sentinels[i].Error(), sentinels[j].Error(),
				"sentinel errors at index %d and %d should be distinct", i, j)
		}
	}
}
