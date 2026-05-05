// Package httpclient provides bug condition exploration tests for SSRF private IP blocking.
//
// Best practices applied (from Go testify coding standards KB):
// - Test naming: Test[FunctionName]_Should[ExpectedResult]_When[Condition]
// - AAA pattern: Arrange → Act → Assert
// - testify/assert for assertions
// - Keep tests simple and focused on the behavior being tested
//
// **Validates: Requirements 2.15** (Property 12 from design)
//
// Bug Condition: isPrivateIP(resolvedIP)
// Expected: error returned
// Counterexample on unfixed code: private IP URL accepted
//
// CRITICAL: This test is expected to FAIL on unfixed code.
// DO NOT fix the code or the test when it fails.
package httpclient

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"pgregory.net/rapid"
)

// TestBuildSafeURL_ShouldReject_WhenURLContainsPrivateIP generates URLs
// targeting private IP ranges (127.x, 10.x, 172.16-31.x, 192.168.x, 169.254.x)
// and verifies they are rejected. On unfixed code, validatePathURL only checks
// for "..", "@", and "//" but does not block private IPs.
//
// **Validates: Requirements 2.15**
func TestBuildSafeURL_ShouldReject_WhenURLContainsPrivateIP(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Arrange — generate a private IP address from one of the reserved ranges
		rangeType := rapid.IntRange(0, 4).Draw(t, "rangeType")

		var privateIP string
		switch rangeType {
		case 0: // 127.0.0.0/8 (loopback)
			b := rapid.IntRange(0, 255).Draw(t, "b")
			c := rapid.IntRange(0, 255).Draw(t, "c")
			d := rapid.IntRange(1, 254).Draw(t, "d")
			privateIP = fmt.Sprintf("127.%d.%d.%d", b, c, d)
		case 1: // 10.0.0.0/8
			b := rapid.IntRange(0, 255).Draw(t, "b")
			c := rapid.IntRange(0, 255).Draw(t, "c")
			d := rapid.IntRange(1, 254).Draw(t, "d")
			privateIP = fmt.Sprintf("10.%d.%d.%d", b, c, d)
		case 2: // 172.16.0.0/12
			second := rapid.IntRange(16, 31).Draw(t, "second")
			c := rapid.IntRange(0, 255).Draw(t, "c")
			d := rapid.IntRange(1, 254).Draw(t, "d")
			privateIP = fmt.Sprintf("172.%d.%d.%d", second, c, d)
		case 3: // 192.168.0.0/16
			c := rapid.IntRange(0, 255).Draw(t, "c")
			d := rapid.IntRange(1, 254).Draw(t, "d")
			privateIP = fmt.Sprintf("192.168.%d.%d", c, d)
		case 4: // 169.254.0.0/16 (link-local / AWS metadata)
			c := rapid.IntRange(0, 255).Draw(t, "c")
			d := rapid.IntRange(1, 254).Draw(t, "d")
			privateIP = fmt.Sprintf("169.254.%d.%d", c, d)
		}

		baseURL := fmt.Sprintf("http://%s", privateIP)
		pathURL := "/api/data"

		// Act
		_, err := buildSafeURL(baseURL, pathURL)

		// Assert — should reject URLs targeting private IPs
		assert.Error(t, err,
			"buildSafeURL should reject private IP %s but accepted it (SSRF vulnerability)", privateIP)
	})
}

// TestValidatePathURL_ShouldReject_WhenURLEncodedBypass verifies that
// URL-encoded bypass attempts like %%2e%2e are rejected. On unfixed code,
// only literal ".." is checked, not encoded variants.
//
// **Validates: Requirements 2.15**
func TestValidatePathURL_ShouldReject_WhenURLEncodedBypass(t *testing.T) {
	encodedPaths := []string{
		"%2e%2e/etc/passwd",
		"..%2f..%2fetc/passwd",
		"%2e%2e%2f%2e%2e%2f",
	}

	for _, path := range encodedPaths {
		t.Run(path, func(t *testing.T) {
			err := validatePathURL(path)
			assert.Error(t, err,
				"validatePathURL should reject URL-encoded bypass %q but accepted it", path)
		})
	}
}
