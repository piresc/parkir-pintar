package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func clearEnv(t *testing.T) {
	t.Helper()
	envVars := []string{
		"APP_ENV",
		"DB_HOST", "DB_PORT", "DB_USERNAME", "DB_PASSWORD", "DB_DATABASE", "DB_SCHEMA", "DB_SSL_MODE",
		"REDIS_HOST", "REDIS_PORT", "REDIS_PASSWORD",
		"JWT_SECRET",
		"NATS_URL",
		"TRACING_OTLP_ENDPOINT",
	}
	for _, v := range envVars {
		t.Setenv(v, "")
		os.Unsetenv(v)
	}
}

func TestLoadConfig_ShouldLoadFromYAML(t *testing.T) {
	clearEnv(t)
	t.Setenv("JWT_SECRET", "test-jwt-secret")
	t.Chdir("../../") // repo root so config/ is accessible

	cfg, err := LoadConfig("gateway")
	require.NoError(t, err)

	assert.Equal(t, "parkir-pintar-gateway", cfg.App.Name)
	assert.Equal(t, "local", cfg.App.Environment)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, "debug", cfg.Logger.Level)
}

func TestLoadConfig_ShouldOverrideSecretsFromEnv(t *testing.T) {
	clearEnv(t)
	t.Setenv("JWT_SECRET", "env-secret-value")
	t.Setenv("DB_USERNAME", "env-user")
	t.Setenv("DB_PASSWORD", "env-pass")
	t.Chdir("../../")

	cfg, err := LoadConfig("reservation")
	require.NoError(t, err)

	assert.Equal(t, "env-secret-value", cfg.JWT.Secret)
	assert.Equal(t, "env-user", cfg.Database.Username)
	assert.Equal(t, "env-pass", cfg.Database.Password)
}

func TestLoadConfig_ShouldReturnError_WhenYAMLMissing(t *testing.T) {
	clearEnv(t)
	t.Setenv("JWT_SECRET", "test-secret")

	_, err := LoadConfig("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read config file")
}

func TestLoadConfig_ShouldReturnError_WhenJWTSecretMissing(t *testing.T) {
	clearEnv(t)
	t.Setenv("APP_ENV", "staging") // staging won't load .env
	t.Chdir("../../")

	_, err := LoadConfig("gateway")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_SECRET")
}

func TestLoadConfig_ShouldReturnError_WhenJWTSecretTooShortInStaging(t *testing.T) {
	clearEnv(t)
	t.Setenv("APP_ENV", "staging")
	t.Setenv("JWT_SECRET", "short")
	t.Chdir("../../")

	_, err := LoadConfig("gateway")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_SECRET must be at least 32 characters")
}

func TestLoadConfig_ShouldValidateServerPort(t *testing.T) {
	clearEnv(t)
	t.Setenv("JWT_SECRET", "test-secret")
	t.Chdir("../../")

	cfg, err := LoadConfig("gateway")
	require.NoError(t, err)
	assert.Equal(t, 8080, cfg.Server.Port)
}
