package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Best practices derived from Go Testify testing knowledge base:
// - Use table-driven tests for multiple test cases
// - Use descriptive test names: Test[Function]_Should[Result]_When[Condition]
// - Use assert for non-fatal checks, require for fatal preconditions
// - Test both happy paths and error cases
// - Keep tests isolated — clear env vars between tests
// - Don't mock the class under test
// - Don't hardcode test data throughout multiple tests

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
		"AUTH_API_KEYS",
		"TRACING_ENABLED", "TRACING_SERVICE_NAME", "TRACING_SAMPLE_RATE",
		"TRACING_EXCLUDE_PATHS", "TRACING_EXPORTER", "TRACING_OTLP_ENDPOINT",
		"NEW_RELIC_LICENSE_KEY", "NEW_RELIC_ENABLED",
		"LOG_LEVEL", "LOG_FORMAT",
		"GRPC_SERVER_PORT", "GRPC_TLS_CERT_PATH", "GRPC_TLS_KEY_PATH",
		"GRPC_MAX_CONN_AGE", "GRPC_DIAL_TIMEOUT", "GRPC_KEEPALIVE_TIME",
		"GRPC_KEEPALIVE_TIMEOUT",
	}
	for _, v := range envVars {
		t.Setenv(v, "")
		os.Unsetenv(v)
	}
}

func TestLoad_ShouldReturnDefaultConfig_WhenNoEnvVarsSet(t *testing.T) {
	clearEnv(t)
	t.Setenv("JWT_SECRET", "test-default-secret")

	cfg, err := Load("")
	require.NoError(t, err)

	// App defaults
	assert.Equal(t, "parkir-pintar", cfg.App.Name)
	assert.Equal(t, "local", cfg.App.Environment)
	assert.False(t, cfg.App.Debug)
	assert.Equal(t, "0.0.1", cfg.App.Version)

	// Server defaults
	assert.Equal(t, "0.0.0.0", cfg.Server.Host)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, 15, cfg.Server.ReadTimeout)
	assert.Equal(t, 15, cfg.Server.WriteTimeout)
	assert.Equal(t, 30, cfg.Server.ShutdownTimeout)
	assert.Equal(t, []string{"http://localhost:3000"}, cfg.Server.AllowedOrigins)

	// Database defaults
	assert.Equal(t, "localhost", cfg.Database.Host)
	assert.Equal(t, 5432, cfg.Database.Port)
	assert.Equal(t, "", cfg.Database.Username)
	assert.Equal(t, "", cfg.Database.Password)
	assert.Equal(t, "", cfg.Database.Database)
	assert.Equal(t, "disable", cfg.Database.SSLMode)
	assert.Equal(t, 25, cfg.Database.MaxConns)
	assert.Equal(t, 5, cfg.Database.IdleConns)
	assert.Equal(t, 15, cfg.Database.MaxLifetime)

	// Redis defaults
	assert.Equal(t, "localhost", cfg.Redis.Host)
	assert.Equal(t, 6379, cfg.Redis.Port)
	assert.Equal(t, "", cfg.Redis.Password)
	assert.Equal(t, 0, cfg.Redis.DB)
	assert.Equal(t, 10, cfg.Redis.PoolSize)

	// JWT defaults
	assert.Equal(t, "test-default-secret", cfg.JWT.Secret)
	assert.Equal(t, 60, cfg.JWT.Expiration)
	assert.Equal(t, "parkir-pintar", cfg.JWT.Issuer)

	// Auth defaults
	assert.NotNil(t, cfg.Auth.APIKeys)
	assert.Empty(t, cfg.Auth.APIKeys)

	// Tracing defaults
	assert.False(t, cfg.Tracing.Enabled)
	assert.Equal(t, "parkir-pintar", cfg.Tracing.ServiceName)
	assert.Equal(t, 1.0, cfg.Tracing.SampleRate)
	assert.Equal(t, []string{"/health", "/health/live", "/health/ready"}, cfg.Tracing.ExcludePaths)
	assert.Equal(t, "noop", cfg.Tracing.Exporter)
	assert.Equal(t, "", cfg.Tracing.OTLPEndpoint)
	assert.Equal(t, "", cfg.Tracing.NewRelic.LicenseKey)
	assert.False(t, cfg.Tracing.NewRelic.Enabled)

	// Logger defaults
	assert.Equal(t, "info", cfg.Logger.Level)
	assert.Equal(t, "json", cfg.Logger.Format)

	// GRPC defaults (Requirements 12.1, 12.2, 12.3)
	assert.Equal(t, 9090, cfg.GRPC.Server.Port)
	assert.Equal(t, "", cfg.GRPC.Server.TLSCertPath)
	assert.Equal(t, "", cfg.GRPC.Server.TLSKeyPath)
	assert.Equal(t, time.Duration(0), cfg.GRPC.Server.MaxConnAge)
	assert.Equal(t, 5*time.Second, cfg.GRPC.Client.DialTimeout)
	assert.Equal(t, 30*time.Second, cfg.GRPC.Client.KeepAliveTime)
	assert.Equal(t, 10*time.Second, cfg.GRPC.Client.KeepAliveTimeout)
}

func TestLoad_ShouldParseEnvVars_WhenAllSet(t *testing.T) {
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
	t.Setenv("AUTH_API_KEYS", "svc1:key1,svc2:key2")
	t.Setenv("TRACING_ENABLED", "true")
	t.Setenv("TRACING_SAMPLE_RATE", "0.5")
	t.Setenv("TRACING_EXPORTER", "otlp")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("LOG_FORMAT", "text")

	cfg, err := Load("")
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
	assert.Equal(t, map[string]string{"svc1": "key1", "svc2": "key2"}, cfg.Auth.APIKeys)
	assert.True(t, cfg.Tracing.Enabled)
	assert.Equal(t, 0.5, cfg.Tracing.SampleRate)
	assert.Equal(t, "otlp", cfg.Tracing.Exporter)
	assert.Equal(t, "debug", cfg.Logger.Level)
	assert.Equal(t, "text", cfg.Logger.Format)
}

func TestLoad_ShouldReturnError_WhenServerPortInvalid(t *testing.T) {
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

			_, err := Load("")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "SERVER_PORT")
		})
	}
}

func TestLoad_ShouldReturnError_WhenTracingSampleRateOutOfRange(t *testing.T) {
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

			_, err := Load("")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "TRACING_SAMPLE_RATE")
		})
	}
}

func TestLoad_ShouldReturnError_WhenJWTSecretMissingInNonLocal(t *testing.T) {
	clearEnv(t)
	t.Setenv("APP_ENV", "production")

	_, err := Load("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_SECRET")
}

func TestLoad_ShouldReturnError_WhenJWTSecretEmptyInLocalEnv(t *testing.T) {
	clearEnv(t)
	t.Setenv("APP_ENV", "local")

	_, err := Load("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_SECRET")
}

func TestLoad_ShouldLoadDotEnvFile_WhenAppEnvIsLocal(t *testing.T) {
	clearEnv(t)
	t.Setenv("JWT_SECRET", "test-default-secret")

	// Create a temporary .env file
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")
	err := os.WriteFile(envFile, []byte("APP_NAME=from-dotenv\nSERVER_PORT=3000\n"), 0644)
	require.NoError(t, err)

	cfg, err := Load(envFile)
	require.NoError(t, err)
	assert.Equal(t, "from-dotenv", cfg.App.Name)
	assert.Equal(t, 3000, cfg.Server.Port)
}

func TestLoad_ShouldNotLoadDotEnvFile_WhenAppEnvIsProduction(t *testing.T) {
	clearEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_SECRET", "prod-secret-that-is-at-least-32-chars-long")

	// Create a .env file that would override APP_NAME
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")
	err := os.WriteFile(envFile, []byte("APP_NAME=should-not-load\n"), 0644)
	require.NoError(t, err)

	cfg, err := Load(envFile)
	require.NoError(t, err)
	// APP_NAME should be the default, not from .env
	assert.Equal(t, "parkir-pintar", cfg.App.Name)
}

func TestLoad_ShouldParseAllowedOrigins_WhenCommaSeparated(t *testing.T) {
	clearEnv(t)
	t.Setenv("JWT_SECRET", "test-default-secret")
	t.Setenv("SERVER_ALLOWED_ORIGINS", "https://a.com,https://b.com,https://c.com")

	cfg, err := Load("")
	require.NoError(t, err)
	assert.Equal(t, []string{"https://a.com", "https://b.com", "https://c.com"}, cfg.Server.AllowedOrigins)
}

func TestLoad_ShouldParseAPIKeys_WhenCommaSeparatedKeyValuePairs(t *testing.T) {
	clearEnv(t)
	t.Setenv("JWT_SECRET", "test-default-secret")
	t.Setenv("AUTH_API_KEYS", "service-a:key-a, service-b:key-b")

	cfg, err := Load("")
	require.NoError(t, err)
	assert.Equal(t, "key-a", cfg.Auth.APIKeys["service-a"])
	assert.Equal(t, "key-b", cfg.Auth.APIKeys["service-b"])
}

func TestLoad_ShouldReturnEmptyAPIKeys_WhenEnvVarNotSet(t *testing.T) {
	clearEnv(t)
	t.Setenv("JWT_SECRET", "test-default-secret")

	cfg, err := Load("")
	require.NoError(t, err)
	assert.NotNil(t, cfg.Auth.APIKeys)
	assert.Len(t, cfg.Auth.APIKeys, 0)
}

func TestLoad_ShouldUseDefaultsForInvalidIntValues(t *testing.T) {
	clearEnv(t)
	t.Setenv("JWT_SECRET", "test-default-secret")
	t.Setenv("SERVER_PORT", "not-a-number")

	cfg, err := Load("")
	require.NoError(t, err)
	// Falls back to default 8080
	assert.Equal(t, 8080, cfg.Server.Port)
}

func TestLoad_ShouldUseDefaultsForInvalidBoolValues(t *testing.T) {
	clearEnv(t)
	t.Setenv("JWT_SECRET", "test-default-secret")
	t.Setenv("APP_DEBUG", "not-a-bool")

	cfg, err := Load("")
	require.NoError(t, err)
	assert.False(t, cfg.App.Debug)
}

func TestLoad_ShouldUseDefaultsForInvalidFloatValues(t *testing.T) {
	clearEnv(t)
	t.Setenv("JWT_SECRET", "test-default-secret")
	t.Setenv("TRACING_SAMPLE_RATE", "not-a-float")

	cfg, err := Load("")
	require.NoError(t, err)
	assert.Equal(t, 1.0, cfg.Tracing.SampleRate)
}

// --- GRPCConfig unit tests (Task 1.3) ---
// Requirements: 12.1, 12.2, 12.3, 12.5

func TestLoad_ShouldReturnGRPCDefaults_WhenNoGRPCEnvVarsSet(t *testing.T) {
	clearEnv(t)
	t.Setenv("JWT_SECRET", "test-default-secret")

	cfg, err := Load("")
	require.NoError(t, err)

	// Server defaults (Requirement 12.1)
	assert.Equal(t, 9090, cfg.GRPC.Server.Port)
	assert.Equal(t, "", cfg.GRPC.Server.TLSCertPath)
	assert.Equal(t, "", cfg.GRPC.Server.TLSKeyPath)
	assert.Equal(t, time.Duration(0), cfg.GRPC.Server.MaxConnAge)

	// Client defaults (Requirement 12.2, 12.3)
	assert.Equal(t, 5*time.Second, cfg.GRPC.Client.DialTimeout)
	assert.Equal(t, 30*time.Second, cfg.GRPC.Client.KeepAliveTime)
	assert.Equal(t, 10*time.Second, cfg.GRPC.Client.KeepAliveTimeout)
}

func TestLoad_ShouldParseAllGRPCEnvVars_WhenAllSet(t *testing.T) {
	clearEnv(t)
	t.Setenv("JWT_SECRET", "test-default-secret")

	t.Setenv("GRPC_SERVER_PORT", "50051")
	t.Setenv("GRPC_TLS_CERT_PATH", "/certs/server.crt")
	t.Setenv("GRPC_TLS_KEY_PATH", "/certs/server.key")
	t.Setenv("GRPC_MAX_CONN_AGE", "5m")
	t.Setenv("GRPC_DIAL_TIMEOUT", "10s")
	t.Setenv("GRPC_KEEPALIVE_TIME", "1m")
	t.Setenv("GRPC_KEEPALIVE_TIMEOUT", "20s")

	cfg, err := Load("")
	require.NoError(t, err)

	// Server (Requirement 12.1)
	assert.Equal(t, 50051, cfg.GRPC.Server.Port)
	assert.Equal(t, "/certs/server.crt", cfg.GRPC.Server.TLSCertPath)
	assert.Equal(t, "/certs/server.key", cfg.GRPC.Server.TLSKeyPath)
	assert.Equal(t, 5*time.Minute, cfg.GRPC.Server.MaxConnAge)

	// Client (Requirement 12.2)
	assert.Equal(t, 10*time.Second, cfg.GRPC.Client.DialTimeout)
	assert.Equal(t, 1*time.Minute, cfg.GRPC.Client.KeepAliveTime)
	assert.Equal(t, 20*time.Second, cfg.GRPC.Client.KeepAliveTimeout)
}

func TestLoad_ShouldParsePartialGRPCEnvVars_WhenSomeSet(t *testing.T) {
	clearEnv(t)
	t.Setenv("JWT_SECRET", "test-default-secret")

	// Only set port and dial timeout, leave the rest as defaults
	t.Setenv("GRPC_SERVER_PORT", "8443")
	t.Setenv("GRPC_DIAL_TIMEOUT", "15s")

	cfg, err := Load("")
	require.NoError(t, err)

	// Set values
	assert.Equal(t, 8443, cfg.GRPC.Server.Port)
	assert.Equal(t, 15*time.Second, cfg.GRPC.Client.DialTimeout)

	// Defaults for unset vars
	assert.Equal(t, "", cfg.GRPC.Server.TLSCertPath)
	assert.Equal(t, "", cfg.GRPC.Server.TLSKeyPath)
	assert.Equal(t, time.Duration(0), cfg.GRPC.Server.MaxConnAge)
	assert.Equal(t, 30*time.Second, cfg.GRPC.Client.KeepAliveTime)
	assert.Equal(t, 10*time.Second, cfg.GRPC.Client.KeepAliveTimeout)
}

func TestLoad_ShouldIndicateTLSEnabled_WhenBothCertAndKeyPathsProvided(t *testing.T) {
	clearEnv(t)
	t.Setenv("JWT_SECRET", "test-default-secret")

	t.Setenv("GRPC_TLS_CERT_PATH", "/certs/server.crt")
	t.Setenv("GRPC_TLS_KEY_PATH", "/certs/server.key")

	cfg, err := Load("")
	require.NoError(t, err)

	// Requirement 12.5: TLS enabled when both paths are present
	assert.NotEmpty(t, cfg.GRPC.Server.TLSCertPath)
	assert.NotEmpty(t, cfg.GRPC.Server.TLSKeyPath)
	assert.Equal(t, "/certs/server.crt", cfg.GRPC.Server.TLSCertPath)
	assert.Equal(t, "/certs/server.key", cfg.GRPC.Server.TLSKeyPath)
}

func TestLoad_ShouldIndicateNoTLS_WhenOnlyCertPathProvided(t *testing.T) {
	clearEnv(t)
	t.Setenv("JWT_SECRET", "test-default-secret")

	t.Setenv("GRPC_TLS_CERT_PATH", "/certs/server.crt")
	// GRPC_TLS_KEY_PATH not set

	cfg, err := Load("")
	require.NoError(t, err)

	// Requirement 12.5: TLS not usable when only cert path is set
	assert.NotEmpty(t, cfg.GRPC.Server.TLSCertPath)
	assert.Empty(t, cfg.GRPC.Server.TLSKeyPath)
}

func TestLoad_ShouldIndicateNoTLS_WhenOnlyKeyPathProvided(t *testing.T) {
	clearEnv(t)
	t.Setenv("JWT_SECRET", "test-default-secret")

	t.Setenv("GRPC_TLS_KEY_PATH", "/certs/server.key")
	// GRPC_TLS_CERT_PATH not set

	cfg, err := Load("")
	require.NoError(t, err)

	// Requirement 12.5: TLS not usable when only key path is set
	assert.Empty(t, cfg.GRPC.Server.TLSCertPath)
	assert.NotEmpty(t, cfg.GRPC.Server.TLSKeyPath)
}

func TestLoad_ShouldIndicateNoTLS_WhenNeitherCertNorKeyPathProvided(t *testing.T) {
	clearEnv(t)
	t.Setenv("JWT_SECRET", "test-default-secret")

	cfg, err := Load("")
	require.NoError(t, err)

	// Requirement 12.5: No TLS when both paths are empty
	assert.Empty(t, cfg.GRPC.Server.TLSCertPath)
	assert.Empty(t, cfg.GRPC.Server.TLSKeyPath)
}

func TestLoad_ShouldUseDefaultGRPCPort_WhenInvalidPortProvided(t *testing.T) {
	clearEnv(t)
	t.Setenv("JWT_SECRET", "test-default-secret")

	t.Setenv("GRPC_SERVER_PORT", "not-a-number")

	cfg, err := Load("")
	require.NoError(t, err)

	// Falls back to default 9090
	assert.Equal(t, 9090, cfg.GRPC.Server.Port)
}

func TestLoad_ShouldUseDefaultDuration_WhenInvalidDurationProvided(t *testing.T) {
	clearEnv(t)
	t.Setenv("JWT_SECRET", "test-default-secret")

	t.Setenv("GRPC_DIAL_TIMEOUT", "invalid")
	t.Setenv("GRPC_KEEPALIVE_TIME", "bad-value")
	t.Setenv("GRPC_KEEPALIVE_TIMEOUT", "xyz")

	cfg, err := Load("")
	require.NoError(t, err)

	// Falls back to defaults
	assert.Equal(t, 5*time.Second, cfg.GRPC.Client.DialTimeout)
	assert.Equal(t, 30*time.Second, cfg.GRPC.Client.KeepAliveTime)
	assert.Equal(t, 10*time.Second, cfg.GRPC.Client.KeepAliveTimeout)
}
