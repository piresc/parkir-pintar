// Package httpclient provides preservation property tests for public URL SSRF pass-through.
//
// Best practices applied (from Go testify coding standards KB):
// - Test naming: Test[FunctionName]_Should[ExpectedResult]_When[Condition]
// - AAA pattern: Arrange → Act → Assert
// - testify/assert and testify/require for assertions
// - Keep tests simple and focused on the behavior being tested
//
// **Validates: Requirements 3.11** (Preservation Property 14 from design)
//
// Non-bug condition: NOT isPrivateIP(resolvedIP)
// These tests verify that valid public URL paths pass through buildSafeURL
// correctly on unfixed code. They must PASS on unfixed code.
package httpclient

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"pgregory.net/rapid"
)

// TestBuildSafeURL_ShouldConstructCorrectURL_WhenPublicPath verifies that
// buildSafeURL constructs correct URLs for valid public URL paths.
// Non-bug condition: NOT isPrivateIP(resolvedIP).
//
// **Validates: Requirements 3.11**
func TestBuildSafeURL_ShouldConstructCorrectURL_WhenPublicPath(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Arrange — generate valid public URL paths
		segment := rapid.StringMatching(`[a-z0-9]{1,10}`).Draw(t, "segment")
		pathURL := "/" + segment

		baseURL := "https://api.example.com"

		// Act
		result, err := buildSafeURL(baseURL, pathURL)

		// Assert — should construct URL without error
		require.NoError(t, err, "valid public path %q should not produce error", pathURL)
		assert.Contains(t, result, segment, "result URL should contain the path segment")
		assert.True(t, strings.HasPrefix(result, "https://api.example.com/"),
			"result URL should start with base URL")
	})
}

// TestValidatePathURL_ShouldAccept_WhenCleanPath verifies that validatePathURL
// accepts clean paths without traversal or injection characters.
// Non-bug condition: clean path without "..", "@", or "//".
//
// **Validates: Requirements 3.11**
func TestValidatePathURL_ShouldAccept_WhenCleanPath(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Arrange — generate clean path segments
		segment := rapid.StringMatching(`/[a-z0-9]{1,20}`).Draw(t, "path")

		// Act
		err := validatePathURL(segment)

		// Assert — clean paths should be accepted
		assert.NoError(t, err, "clean path %q should be accepted", segment)
	})
}

// TestBuildSafeURL_ShouldPreserveBaseURL_WhenPathProvided verifies that
// the base URL is preserved in the output.
// Non-bug condition: valid public path.
//
// **Validates: Requirements 3.11**
func TestBuildSafeURL_ShouldPreserveBaseURL_WhenPathProvided(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Arrange
		host := rapid.SampledFrom([]string{
			"https://api.example.com",
			"https://service.internal.io",
			"http://gateway.test.com",
		}).Draw(t, "baseURL")
		segment := rapid.StringMatching(`[a-z]{1,8}`).Draw(t, "segment")
		pathURL := "/" + segment

		// Act
		result, err := buildSafeURL(host, pathURL)

		// Assert
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(result, host),
			"result %q should start with base URL %q", result, host)
	})
}
