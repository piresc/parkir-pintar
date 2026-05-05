// Best practices applied from Go testing guidelines:
// - Descriptive test names using Test[FunctionName]_Should[Expected]_When[Condition] pattern
// - AAA (Arrange-Act-Assert) structure
// - testify assertions (assert, require) for clear failure messages
// - Tests verify behavior, not implementation details
// - No mocks needed — testing real JWT generation and validation

package testhelpers

import (
	"testing"

	"parkir-pintar/pkg/auth"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateTestJWT_ShouldReturnNonEmptyToken_WhenValidInputProvided(t *testing.T) {
	// Arrange
	userID := "driver-001"
	role := "driver"
	secret := "test-secret-key"

	// Act
	token := GenerateTestJWT(userID, role, secret)

	// Assert
	assert.NotEmpty(t, token, "generated token should not be empty")
}

func TestGenerateTestJWT_ShouldReturnValidatableToken_WhenValidInputProvided(t *testing.T) {
	// Arrange
	userID := "driver-002"
	role := "admin"
	secret := "test-secret-key"

	// Act
	token := GenerateTestJWT(userID, role, secret)

	// Assert
	claims, err := auth.ValidateToken(token, secret)
	require.NoError(t, err, "token should be valid and parseable")
	assert.NotNil(t, claims, "claims should not be nil")
}

func TestGenerateTestJWT_ShouldContainCorrectClaims_WhenValidInputProvided(t *testing.T) {
	// Arrange
	userID := "driver-003"
	role := "operator"
	secret := "test-secret-key"

	// Act
	token := GenerateTestJWT(userID, role, secret)

	// Assert
	claims, err := auth.ValidateToken(token, secret)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID, "userID claim should match input")
	assert.Equal(t, role, claims.Role, "role claim should match input")
	assert.Equal(t, "parkir-pintar", claims.Issuer, "issuer should be parkir-pintar")
}
