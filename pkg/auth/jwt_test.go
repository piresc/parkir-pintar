package auth

import (
	"testing"
	"time"

	"parkir-pintar/pkg/config"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validJWTConfig() config.JWTConfig {
	return config.JWTConfig{
		Secret:     "test-secret-key-for-unit-tests",
		Expiration: 60,
		Issuer:     "test-issuer",
	}
}

func TestGenerateToken_ShouldReturnSignedToken_WhenValidInputProvided(t *testing.T) {
	cfg := validJWTConfig()

	token, expiresAt, err := GenerateToken("user-123", "admin", cfg)

	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Greater(t, expiresAt, time.Now().Unix())
}

func TestGenerateToken_ShouldReturnError_WhenSecretEmpty(t *testing.T) {
	cfg := validJWTConfig()
	cfg.Secret = ""

	token, expiresAt, err := GenerateToken("user-123", "admin", cfg)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "jwt secret is required")
	assert.Empty(t, token)
	assert.Zero(t, expiresAt)
}

func TestGenerateToken_ShouldReturnError_WhenUserIDEmpty(t *testing.T) {
	cfg := validJWTConfig()

	token, expiresAt, err := GenerateToken("", "admin", cfg)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user ID is required")
	assert.Empty(t, token)
	assert.Zero(t, expiresAt)
}

func TestValidateToken_ShouldReturnClaims_WhenTokenValid(t *testing.T) {
	cfg := validJWTConfig()
	token, _, err := GenerateToken("user-456", "editor", cfg)
	require.NoError(t, err)

	claims, err := ValidateToken(token, cfg.Secret)

	require.NoError(t, err)
	assert.Equal(t, "user-456", claims.UserID)
	assert.Equal(t, "editor", claims.Role)
	assert.Equal(t, "test-issuer", claims.Issuer)
	assert.NotNil(t, claims.ExpiresAt)
	assert.NotNil(t, claims.IssuedAt)
}

func TestGenerateAndValidate_ShouldRoundtrip_WhenValidConfig(t *testing.T) {
	cfg := validJWTConfig()
	userID := "roundtrip-user"
	role := "viewer"

	token, expiresAt, err := GenerateToken(userID, role, cfg)
	require.NoError(t, err)

	claims, err := ValidateToken(token, cfg.Secret)

	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, role, claims.Role)
	assert.Equal(t, cfg.Issuer, claims.Issuer)
	assert.Equal(t, expiresAt, claims.ExpiresAt.Unix())
}

func TestValidateToken_ShouldReturnError_WhenTokenExpired(t *testing.T) {
	cfg := validJWTConfig()
	now := time.Now()
	claims := Claims{
		UserID: "expired-user",
		Role:   "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now.Add(-2 * time.Hour)),
			Issuer:    cfg.Issuer,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(cfg.Secret))
	require.NoError(t, err)

	result, err := ValidateToken(signed, cfg.Secret)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid token")
}

func TestValidateToken_ShouldReturnError_WhenWrongSecret(t *testing.T) {
	cfg := validJWTConfig()
	token, _, err := GenerateToken("user-789", "admin", cfg)
	require.NoError(t, err)

	claims, err := ValidateToken(token, "wrong-secret")

	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestValidateToken_ShouldReturnError_WhenTokenStringEmpty(t *testing.T) {
	claims, err := ValidateToken("", "some-secret")

	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.Contains(t, err.Error(), "token string is required")
}

func TestValidateToken_ShouldReturnError_WhenSecretEmpty(t *testing.T) {
	claims, err := ValidateToken("some.token.string", "")

	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.Contains(t, err.Error(), "secret is required")
}

func TestValidateToken_ShouldReturnError_WhenMalformedToken(t *testing.T) {
	claims, err := ValidateToken("not-a-valid-jwt", "some-secret")

	assert.Error(t, err)
	assert.Nil(t, claims)
}
