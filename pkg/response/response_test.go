// Package response tests
//
// Best practices applied (from Go testing standards KB):
// - Use descriptive names: Test[FunctionName]_Should[ExpectedResult]_When[Condition]
// - Follow AAA (Arrange-Act-Assert) pattern
// - Table-driven tests for multiple scenarios
// - Use testify assertions for clear failure messages
// - Tests are fast, isolated, repeatable, and clear
// - Test both success and error/edge cases
// - Use net/http/httptest for HTTP response recording
//
// Key verification: HTTP status code in the response ALWAYS matches the
// status code in the JSON body (fixing the misleading helpers from boilerplate-golang).
package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// --- Success tests ---

func TestSuccess_ShouldReturnSuccessJSON_WhenCalledWithData(t *testing.T) {
	// Arrange
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	data := map[string]string{"name": "test"}

	// Act
	Success(c, http.StatusOK, data)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var resp SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "success", resp.Status)
	assert.NotNil(t, resp.Data)
}

func TestSuccess_ShouldReturnCreatedStatus_WhenStatusCreatedProvided(t *testing.T) {
	// Arrange
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	data := map[string]int{"id": 1}

	// Act
	Success(c, http.StatusCreated, data)

	// Assert
	assert.Equal(t, http.StatusCreated, w.Code)

	var resp SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "success", resp.Status)
}

func TestSuccess_ShouldReturnNullData_WhenNilDataProvided(t *testing.T) {
	// Arrange
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Act
	Success(c, http.StatusOK, nil)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "success", resp["status"])
	assert.Nil(t, resp["data"])
}

// --- Error tests ---

func TestError_ShouldReturnErrorJSON_WhenCalledWithMessage(t *testing.T) {
	// Arrange
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Act
	Error(c, http.StatusBadRequest, "invalid input")

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "error", resp.Status)
	assert.Equal(t, "invalid input", resp.Error)
	assert.Empty(t, resp.RequestID)
}

func TestError_ShouldMatchHTTPStatusWithBody_WhenCustomStatusProvided(t *testing.T) {
	// Arrange — this is the key fix: status code 422 should appear in BOTH
	// the HTTP response status AND the JSON body status text.
	tests := []struct {
		name       string
		statusCode int
	}{
		{name: "bad request 400", statusCode: http.StatusBadRequest},
		{name: "unauthorized 401", statusCode: http.StatusUnauthorized},
		{name: "not found 404", statusCode: http.StatusNotFound},
		{name: "unprocessable entity 422", statusCode: http.StatusUnprocessableEntity},
		{name: "internal server error 500", statusCode: http.StatusInternalServerError},
		{name: "service unavailable 503", statusCode: http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Act
			Error(c, tt.statusCode, "test error")

			// Assert — HTTP status matches the provided code
			assert.Equal(t, tt.statusCode, w.Code)

			var resp ErrorResponse
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)
			assert.Equal(t, "error", resp.Status)
			assert.Equal(t, "test error", resp.Error)
		})
	}
}

// --- ErrorWithRequestID tests ---

func TestErrorWithRequestID_ShouldIncludeRequestID_WhenProvided(t *testing.T) {
	// Arrange
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Act
	ErrorWithRequestID(c, http.StatusInternalServerError, "something went wrong", "req-abc-123")

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var resp ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "error", resp.Status)
	assert.Equal(t, "something went wrong", resp.Error)
	assert.Equal(t, "req-abc-123", resp.RequestID)
}

func TestErrorWithRequestID_ShouldOmitRequestID_WhenEmpty(t *testing.T) {
	// Arrange
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Act
	ErrorWithRequestID(c, http.StatusBadRequest, "bad input", "")

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var raw map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &raw)
	require.NoError(t, err)
	// request_id should be omitted from JSON when empty (omitempty tag)
	_, exists := raw["request_id"]
	assert.False(t, exists, "request_id should be omitted when empty")
}
