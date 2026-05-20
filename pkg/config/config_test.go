package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func clearEnv(t *testing.T) {
	t.Helper()
	envVars := []string{
		"APP_NAME", "APP_ENV", "APP_DEBUG", "APP_VERSION",
		"SERVER_HOST", "SERVER_PORT", "SERVER_READ_TIMEOUT", "SERVER_WRITE_TIMEOUT",
		"SERVER_SHUTDOWN_TIMEOUT", "SERVER_ALLOWED_ORIGINS",
		"DB_HOST", "DB_PORT", "DB_USERNAME", "DB_PASSWORD", "DB_DATABASE",
		"DB_SSL_MODE", "DB_MAX_CONNS", "DB_IDLE_CONNS", "DB_MAX_LIFETIME",
		"REDIS_HOST", "REDIS_PORT", "REDIS_PASSWORD", "REDIS_DB", "REDIS_POOL_SIZE",
		"JWT_SECRET", "JWT_EXPIRATION", "JWT_ISSUER",
		"TRACING_ENABLED", "TRACING_SERVICE_NAME", "TRACING_SAMPLE_RATE",
		"TRACING_EXCLUDE_PATHS", "TRACING_EXPORTER", "TRACING_OTLP_ENDPOINT",
		"LOG_LEVEL", "LOG_FORMAT",
		"GRPC_SERVER_PORT", "GRPC_MAX_CONN_AGE", "GRPC_DIAL_TIMEOUT",
		"GRPC_KEEPALIVE_TIME", "GRPC_KEEPALIVE_TIMEOUT",
	}
	for _, v := range envVars {
		t.Setenv(v, "")
		os.Unsetenv(v)
	}
}

func TestLoadConfig_ShouldReturnDefaultConfig_WhenNoEnvVarsSet(t *testing.T) {
	clearEnv(t)
	t.Setenv("JWT_SECRET", "test-default-secret")

	cfg, err := LoadConfig("nonexistent")
	require.NoError(t, err)

	assert.Equal(t, "parkir-pintar", cfg.App.Name)
	assert.Equal(t, "local", cfg.App.Environment)
	assert.False(t, cfg.App.Debug)
	assert.Equal(t, "0.0.1", cfg.App.Version)

	assert.Equal(t, "0.0.0.0", cfg.Server.Host)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, 15, cfg.Server.ReadTimeout)
	assert.Equal(t, 15, cfg.Server.WriteTimeout)
	assert.Equal(t, 30, cfg.Server.ShutdownTimeout)
	assert.Equal(t, []string{"http://localhost:3000"}, cfg.Server.AllowedOrigins)

	assert.Equal(t, "localhost", cfg.Database.Host)
	assert.Equal(t, 5432, cfg.Database.Port)
	assert.Equal(t, "", cfg.Database.Username)
	assert.Equal(t, "", cfg.Database.Password)
	assert.Equal(t, "", cfg.Database.Database)
	assert.Equal(t, "disable", cfg.Database.SSLMode)
	assert.Equal(t, 25, cfg.Database.MaxConns)
	assert.Equal(t, 5, cfg.Database.IdleConns)
	assert.Equal(t, 15, cfg.Database.MaxLifetime)

	assert.Equal(t, "localhost", cfg.Redis.Host)
	assert.Equal(t, 6379, cfg.Redis.Port)
	assert.Equal(t, "", cfg.Redis.Password)
	assert.Equal(t, 0, cfg.Redis.DB)
	assert.Equal(t, 10, cfg.Redis.PoolSize)

	assert.Equal(t, "test-default-secret", cfg.JWT.Secret)
	assert.Equal(t, 60, cfg.JWT.Expiration)
	assert.Equal(t, "parkir-pintar", cfg.JWT.Issuer)

	assert.False(t, cfg.Tracing.Enabled)
	assert.Equal(t, "parkir-pintar", cfg.Tracing.ServiceName)
	assert.Equal(t, 1.0, cfg.Tracing.SampleRate)
	assert.Equal(t, []string{"/health", "/health/live", "/health/ready"}, cfg.Tracing.ExcludePaths)
	assert.Equal(t, "noop", cfg.Tracing.Exporter)
	assert.Equal(t, "", cfg.Tracing.OTLPEndpoint)

	assert.Equal(t, "info", cfg.Logger.Level)
	assert.Equal(t, "json", cfg.Logger.Format)

	assert.Equal(t, 9090, cfg.GRPC.Server.Port)
	assert.Equal(t, time.Duration(0), cfg.GRPC.Server.MaxConnAge)
	assert.Equal(t, 5*time.Second, cfg.GRPC.Client.DialTimeout)
	assert.Equal(t, 30*time.Second, cfg.GRPC.Client.KeepAliveTime)
	assert.Equal(t, 10*time.Second, cfg.GRPC.Client.KeepAliveTimeout)
}

func TestLoadConfig_ShouldParseEnvVars_WhenAllSet(t *testing.T) {
	clearEnv(t)

	t.Setenv("APP_NAME", "my-service")
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_DEBUG", "true")
	t.Setenv("APP_VERSION", "1.2.3")
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("SERVER_ALLOWED_ORIGINS", "https://example.com, https://app.example.com")
	t.Setenv("DB_HOST", "db.example.com")
	t.Setenv("DB_PORT", "5433")
	t.Setenv("DB_USERNAME", "admin")
	t.Setenv("DB_PASSWORD", "secret")
	t.Setenv("DB_DATABASE", "mydb")
	t.Setenv("DB_SSL_MODE", "require")
	t.Setenv("REDIS_PORT", "6380")
	t.Setenv("REDIS_POOL_SIZE", "20")
	t.Setenv("JWT_SECRET", "my-jwt-secret-that-is-at-least-32-chars-long")
	t.Setenv("JWT_EXPIRATION", "120")
	t.Setenv("TRACING_ENABLED", "true")
	t.Setenv("TRACING_SAMPLE_RATE", "0.5")
	t.Setenv("TRACING_EXPORTER", "otlp")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("LOG_FORMAT", "text")

	cfg, err := LoadConfig("nonexistent")
	require.NoError(t, err)

	assert.Equal(t, "my-service", cfg.App.Name)
	assert.Equal(t, "production", cfg.App.Environment)
	assert.True(t, cfg.App.Debug)
	assert.Equal(t, "1.2.3", cfg.App.Version)
	assert.Equal(t, 9090, cfg.Server.Port)
	assert.Equal(t, []string{"https://example.com", "https://app.example.com"}, cfg.Server.AllowedOrigins)
	assert.Equal(t, "db.example.com", cfg.Database.Host)
	assert.Equal(t, 5433, cfg.Database.Port)
	assert.Equal(t, "admin", cfg.Database.Username)
	assert.Equal(t, "secret", cfg.Database.Password)
	assert.Equal(t, "mydb", cfg.Database.Database)
	assert.Equal(t, "require", cfg.Database.SSLMode)
	assert.Equal(t, 6380, cfg.Redis.Port)
	assert.Equal(t, 20, cfg.Redis.PoolSize)
	assert.Equal(t, "my-jwt-secret-that-is-at-least-32-chars-long", cfg.JWT.Secret)
	assert.Equal(t, 120, cfg.JWT.Expiration)
	assert.True(t, cfg.Tracing.Enabled)
	assert.Equal(t, 0.5, cfg.Tracing.SampleRate)
	assert.Equal(t, "otlp", cfg.Tracing.Exporter)
	assert.Equal(t, "debug", cfg.Logger.Level)
	assert.Equal(t, "text", cfg.Logger.Format)
}

func TestLoadConfig_ShouldReturnError_WhenServerPortInvalid(t *testing.T) {
	tests := []struct {
		name string
		port string
	}{
		{name: "zero port", port: "0"},
		{name: "negative port", port: "-1"},
		{name: "port too high", port: "65536"},
		{name: "port way too high", port: "99999"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			clearEnv(t)
			t.Setenv("SERVER_PORT", tc.port)

			_, err := LoadConfig("nonexistent")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "SERVER_PORT")
		})
	}
}

func TestLoadConfig_ShouldReturnError_WhenTracingSampleRateOutOfRange(t *testing.T) {
	tests := []struct {
		name string
		rate string
	}{
		{name: "negative rate", rate: "-0.1"},
		{name: "rate above 1", rate: "1.1"},
		{name: "rate way above 1", rate: "5.0"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			clearEnv(t)
			t.Setenv("TRACING_SAMPLE_RATE", tc.rate)

			_, err := LoadConfig("nonexistent")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "TRACING_SAMPLE_RATE")
		})
	}
}

func TestLoadConfig_ShouldReturnError_WhenJWTSecretMissing(t *testing.T) {
	clearEnv(t)
	t.Setenv("APP_ENV", "production")

	_, err := LoadConfig("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_SECRET")
}

func TestLoadConfig_ShouldReturnError_WhenJWTSecretEmpty(t *testing.T) {
	clearEnv(t)
	t.Setenv("APP_ENV", "local")

	_, err := LoadConfig("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_SECRET")
}

func TestLoadConfig_ShouldParseAllowedOrigins_WhenCommaSeparated(t *testing.T) {
	clearEnv(t)
	t.Setenv("JWT_SECRET", "test-default-secret")
	t.Setenv("SERVER_ALLOWED_ORIGINS", "https://a.com,https://b.com,https://c.com")

	cfg, err := LoadConfig("nonexistent")
	require.NoError(t, err)
	assert.Equal(t, []string{"https://a.com", "https://b.com", "https://c.com"}, cfg.Server.AllowedOrigins)
}

func TestLoadConfig_ShouldReturnError_WhenInvalidIntValues(t *testing.T) {
	clearEnv(t)
	t.Setenv("JWT_SECRET", "test-default-secret")
	t.Setenv("SERVER_PORT", "not-a-number")

	_, err := LoadConfig("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "server.port")
}

func TestLoadConfig_ShouldReturnError_WhenInvalidBoolValues(t *testing.T) {
	clearEnv(t)
	t.Setenv("JWT_SECRET", "test-default-secret")
	t.Setenv("APP_DEBUG", "not-a-bool")

	_, err := LoadConfig("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "app.debug")
}

func TestLoadConfig_ShouldReturnError_WhenInvalidFloatValues(t *testing.T) {
	clearEnv(t)
	t.Setenv("JWT_SECRET", "test-default-secret")
	t.Setenv("TRACING_SAMPLE_RATE", "not-a-float")

	_, err := LoadConfig("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tracing.sample_rate")
}

func TestLoadConfig_ShouldReturnGRPCDefaults_WhenNoGRPCEnvVarsSet(t *testing.T) {
	clearEnv(t)
	t.Setenv("JWT_SECRET", "test-default-secret")

	cfg, err := LoadConfig("nonexistent")
	require.NoError(t, err)

	assert.Equal(t, 9090, cfg.GRPC.Server.Port)
	assert.Equal(t, time.Duration(0), cfg.GRPC.Server.MaxConnAge)

	assert.Equal(t, 5*time.Second, cfg.GRPC.Client.DialTimeout)
	assert.Equal(t, 30*time.Second, cfg.GRPC.Client.KeepAliveTime)
	assert.Equal(t, 10*time.Second, cfg.GRPC.Client.KeepAliveTimeout)
}

func TestLoadConfig_ShouldParseGRPCEnvVars_WhenSet(t *testing.T) {
	clearEnv(t)
	t.Setenv("JWT_SECRET", "test-default-secret")

	t.Setenv("GRPC_SERVER_PORT", "50051")
	t.Setenv("GRPC_MAX_CONN_AGE", "5m")
	t.Setenv("GRPC_DIAL_TIMEOUT", "10s")
	t.Setenv("GRPC_KEEPALIVE_TIME", "1m")
	t.Setenv("GRPC_KEEPALIVE_TIMEOUT", "20s")

	cfg, err := LoadConfig("nonexistent")
	require.NoError(t, err)

	assert.Equal(t, 50051, cfg.GRPC.Server.Port)
	assert.Equal(t, 5*time.Minute, cfg.GRPC.Server.MaxConnAge)

	assert.Equal(t, 10*time.Second, cfg.GRPC.Client.DialTimeout)
	assert.Equal(t, 1*time.Minute, cfg.GRPC.Client.KeepAliveTime)
	assert.Equal(t, 20*time.Second, cfg.GRPC.Client.KeepAliveTimeout)
}

func TestLoadConfig_ShouldReturnError_WhenInvalidGRPCPort(t *testing.T) {
	clearEnv(t)
	t.Setenv("JWT_SECRET", "test-default-secret")
	t.Setenv("GRPC_SERVER_PORT", "not-a-number")

	_, err := LoadConfig("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "grpc.server.port")
}

func TestLoadConfig_ShouldReturnError_WhenInvalidDuration(t *testing.T) {
	clearEnv(t)
	t.Setenv("JWT_SECRET", "test-default-secret")
	t.Setenv("GRPC_DIAL_TIMEOUT", "invalid")

	_, err := LoadConfig("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "grpc.client.dial_timeout")
}
