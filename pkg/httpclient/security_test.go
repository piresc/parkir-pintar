// Best practices applied from MCP knowledgebase (Go Testing Guidelines):
// - Descriptive test names using Test[FunctionName]_Should[Result]_When[Condition] pattern
// - AAA (Arrange-Act-Assert) structure
// - Table-driven tests for multiple scenarios
// - Test both success and error scenarios
// - Use testify/assert and testify/require for assertions
// - Tests are fast, isolated, repeatable, clear, and comprehensive
package httpclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePathURL_ShouldReturnNil_WhenPathIsValid(t *testing.T) {
	// Arrange
	validPaths := []string{
		"/api/v1/users",
		"/health",
		"/api/v2/orders/123",
		"api/endpoint",
	}

	for _, path := range validPaths {
		t.Run(path, func(t *testing.T) {
			// Act
			err := validatePathURL(path)

			// Assert
			assert.NoError(t, err)
		})
	}
}

func TestValidatePathURL_ShouldReturnError_WhenPathContainsDoubleDot(t *testing.T) {
	// Arrange
	path := "/api/../etc/passwd"

	// Act
	err := validatePathURL(path)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "..")
}

func TestValidatePathURL_ShouldReturnError_WhenPathContainsAtSign(t *testing.T) {
	// Arrange
	path := "/api/user@evil.com"

	// Act
	err := validatePathURL(path)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "@")
}

func TestValidatePathURL_ShouldReturnError_WhenPathStartsWithDoubleSlash(t *testing.T) {
	// Arrange
	path := "//evil.com/api"

	// Act
	err := validatePathURL(path)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "//")
}

func TestValidatePathURL_ShouldReturnError_WhenSSRFAttackPatterns(t *testing.T) {
	// Table-driven test for SSRF attack patterns
	tests := []struct {
		name    string
		path    string
		wantErr string
	}{
		{
			name:    "directory traversal",
			path:    "/api/../../etc/passwd",
			wantErr: "..",
		},
		{
			name:    "at sign injection",
			path:    "/api@evil.com",
			wantErr: "@",
		},
		{
			name:    "protocol-relative URL",
			path:    "//evil.com/steal-data",
			wantErr: "//",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			err := validatePathURL(tt.path)

			// Assert
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestBuildSafeURL_ShouldReturnURL_WhenValidInputs(t *testing.T) {
	// Arrange
	baseURL := "https://api.example.com"
	pathURL := "/v1/users"

	// Act
	result, err := buildSafeURL(baseURL, pathURL)

	// Assert
	require.NoError(t, err)
	assert.Contains(t, result, "https://api.example.com")
	assert.Contains(t, result, "v1/users")
}

func TestBuildSafeURL_ShouldPrependSlash_WhenPathMissingLeadingSlash(t *testing.T) {
	// Arrange
	baseURL := "https://api.example.com"
	pathURL := "v1/users"

	// Act
	result, err := buildSafeURL(baseURL, pathURL)

	// Assert
	require.NoError(t, err)
	assert.Contains(t, result, "v1/users")
}

func TestBuildSafeURL_ShouldReturnError_WhenBaseURLInvalid(t *testing.T) {
	// Arrange
	baseURL := "://invalid"
	pathURL := "/api/v1"

	// Act
	_, err := buildSafeURL(baseURL, pathURL)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid base URL")
}

func TestBuildSafeURL_ShouldReturnError_WhenPathContainsSSRFPattern(t *testing.T) {
	// Arrange
	baseURL := "https://api.example.com"
	pathURL := "/api/../../../etc/passwd"

	// Act
	_, err := buildSafeURL(baseURL, pathURL)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "..")
}

func TestBuildSafeURL_ShouldJoinPaths_WhenBaseHasExistingPath(t *testing.T) {
	// Arrange
	baseURL := "https://api.example.com/base"
	pathURL := "/v1/endpoint"

	// Act
	result, err := buildSafeURL(baseURL, pathURL)

	// Assert
	require.NoError(t, err)
	assert.Contains(t, result, "base")
	assert.Contains(t, result, "v1/endpoint")
}
