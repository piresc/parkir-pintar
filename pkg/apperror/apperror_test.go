package apperror

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppError_ShouldImplementErrorInterface(t *testing.T) {
	appErr := New("TEST_CODE", "test message", http.StatusBadRequest)

	var err error = appErr

	assert.Equal(t, "test message", err.Error())
}

func TestAppError_ShouldReturnMessage_WhenErrorCalled(t *testing.T) {
	appErr := &AppError{
		Code:       "CUSTOM",
		Message:    "custom error message",
		HTTPStatus: http.StatusConflict,
	}

	result := appErr.Error()

	assert.Equal(t, "custom error message", result)
}

func TestNew_ShouldCreateAppError_WhenValidParamsProvided(t *testing.T) {
	appErr := New("VALIDATION_ERROR", "field is required", http.StatusBadRequest)

	assert.Equal(t, "VALIDATION_ERROR", appErr.Code)
	assert.Equal(t, "field is required", appErr.Message)
	assert.Equal(t, http.StatusBadRequest, appErr.HTTPStatus)
}

func TestConvenienceConstructors_ShouldSetCorrectHTTPStatus_WhenCalled(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appErr := tt.constructor("test message")

			assert.Equal(t, tt.expectedCode, appErr.Code)
			assert.Equal(t, "test message", appErr.Message)
			assert.Equal(t, tt.expectedStatus, appErr.HTTPStatus)
			assert.Equal(t, "test message", appErr.Error())
		})
	}
}

func TestSentinelErrors_ShouldBeDistinct_WhenCompared(t *testing.T) {
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

	for i := 0; i < len(sentinels); i++ {
		for j := i + 1; j < len(sentinels); j++ {
			assert.NotEqual(t, sentinels[i].Error(), sentinels[j].Error(),
				"sentinel errors at index %d and %d should be distinct", i, j)
		}
	}
}
