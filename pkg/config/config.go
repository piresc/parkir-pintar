package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type Config struct {
	App         AppConfig         `yaml:"app"`
	Server      ServerConfig      `yaml:"server"`
	Database    DatabaseConfig    `yaml:"database"`
	Redis       RedisConfig       `yaml:"redis"`
	JWT         JWTConfig         `yaml:"jwt"`
	Auth        AuthConfig        `yaml:"auth"`
	Tracing     TracingConfig     `yaml:"tracing"`
	Logger      LoggerConfig      `yaml:"logger"`
	GRPC        GRPCConfig        `yaml:"grpc"`
	Reservation ReservationConfig `yaml:"reservation"`
	Asynq       AsynqConfig       `yaml:"asynq"`
	NATS        NATSConfig        `yaml:"nats"`
}

type ReservationConfig struct {
	PaymentTimeoutMinutes int           `yaml:"payment_timeout_minutes"` // default 10 — time allowed to complete payment before reservation fails
	ExpiryTimeoutMinutes  int           `yaml:"expiry_timeout_minutes"`  // default 60 — time allowed to check in after confirmation before expiry
	WorkerPollInterval    time.Duration `yaml:"worker_poll_interval"`    // default 30s — polling interval for legacy fallback workers
}

type AsynqConfig struct {
	Concurrency int `yaml:"concurrency"` // number of concurrent workers (default 10)
}

type NATSConfig struct {
	URL     string `yaml:"url"`     // NATS server URL (default: nats://localhost:4222)
	Enabled bool   `yaml:"enabled"` // Enable NATS messaging (default: false)
}

type GRPCServerConfig struct {
	Port            int           `yaml:"port"`
	TLSCertPath     string        `yaml:"tls_cert_path"`
	TLSKeyPath      string        `yaml:"tls_key_path"`
	MaxConnAge      time.Duration `yaml:"max_conn_age"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
	RequestTimeout  time.Duration `yaml:"request_timeout"`
}

type GRPCRateLimitConfig struct {
	RequestsPerSecond int `yaml:"requests_per_second"`
	BurstSize         int `yaml:"burst_size"`
}

type GRPCClientConfig struct {
	DialTimeout      time.Duration `yaml:"dial_timeout"`
	KeepAliveTime    time.Duration `yaml:"keepalive_time"`
	KeepAliveTimeout time.Duration `yaml:"keepalive_timeout"`
}

type GRPCConfig struct {
	Server    GRPCServerConfig    `yaml:"server"`
	Client    GRPCClientConfig    `yaml:"client"`
	RateLimit GRPCRateLimitConfig `yaml:"rate_limit"`
}

type AppConfig struct {
	Name        string `yaml:"name"`
	Environment string `yaml:"environment"` // local, development, staging, production
	Debug       bool   `yaml:"debug"`
	Version     string `yaml:"version"`
}

type ServerConfig struct {
	Host            string   `yaml:"host"`
	Port            int      `yaml:"port"`
	ReadTimeout     int      `yaml:"read_timeout"`  // seconds
	WriteTimeout    int      `yaml:"write_timeout"` // seconds
	ShutdownTimeout int      `yaml:"shutdown_timeout"` // seconds
	AllowedOrigins  []string `yaml:"allowed_origins"`
}

type DatabaseConfig struct {
	Host        string `yaml:"host"`
	Port        int    `yaml:"port"`
	Username    string `yaml:"username"`
	Password    string `yaml:"password"`
	Database    string `yaml:"database"`
	Schema      string `yaml:"schema"`
	SSLMode     string `yaml:"ssl_mode"`
	MaxConns    int    `yaml:"max_conns"`
	IdleConns   int    `yaml:"idle_conns"`
	MaxLifetime int    `yaml:"max_lifetime"` // minutes
}

type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
	PoolSize int    `yaml:"pool_size"`
}

type JWTConfig struct {
	Secret     string `yaml:"secret"`
	Expiration int    `yaml:"expiration"` // minutes
	Issuer     string `yaml:"issuer"`
}

type AuthConfig struct {
	APIKeys map[string]string `yaml:"api_keys"` // service_name -> key
}

type TracingConfig struct {
	Enabled      bool                   `yaml:"enabled"`
	ServiceName  string                 `yaml:"service_name"`
	SampleRate   float64                `yaml:"sample_rate"`
	ExcludePaths []string               `yaml:"exclude_paths"`
	Exporter     string                 `yaml:"exporter"` // "stdout", "otlp", "newrelic", "noop"
	OTLPEndpoint string                 `yaml:"otlp_endpoint"`
	NewRelic     NewRelicExporterConfig `yaml:"new_relic"`
}

type NewRelicExporterConfig struct {
	LicenseKey string `yaml:"license_key"`
	Enabled    bool   `yaml:"enabled"`
}

type LoggerConfig struct {
	Level  string `yaml:"level"`  // debug, info, warn, error
	Format string `yaml:"format"` // json, text
}

func LoadConfig(serviceName string) (*Config, error) {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "local"
	}

	if env == "local" {
		_ = godotenv.Load(".env")
	}

	cfg := &Config{}

	yamlLoaded := false
	yamlPath := filepath.Join("config", fmt.Sprintf("%s.%s.yaml", serviceName, env))
	data, err := os.ReadFile(yamlPath)
	if err == nil {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file %s: %w", yamlPath, err)
		}
		yamlLoaded = true
	}

	// If YAML was not loaded, this populates everything from env (backward compat).
	overlayEnvSecrets(cfg, yamlLoaded)

	if err := validate(cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// Secrets always come from env. Non-secret fields are only applied from env
// if YAML was not loaded (backward compat) or if the env var is explicitly set.
func overlayEnvSecrets(cfg *Config, yamlLoaded bool) {
	// --- Secrets: always overlay from env ---
	cfg.Database.Host = getEnvOverlay("DB_HOST", cfg.Database.Host, "localhost", yamlLoaded)
	cfg.Database.Port = getEnvAsIntOverlay("DB_PORT", cfg.Database.Port, 5432, yamlLoaded)
	cfg.Database.Username = getEnvOverlay("DB_USERNAME", cfg.Database.Username, "", yamlLoaded)
	cfg.Database.Password = getEnvOverlay("DB_PASSWORD", cfg.Database.Password, "", yamlLoaded)
	cfg.Database.Database = getEnvOverlay("DB_DATABASE", cfg.Database.Database, "", yamlLoaded)

	cfg.Redis.Host = getEnvOverlay("REDIS_HOST", cfg.Redis.Host, "localhost", yamlLoaded)
	cfg.Redis.Port = getEnvAsIntOverlay("REDIS_PORT", cfg.Redis.Port, 6379, yamlLoaded)
	cfg.Redis.Password = getEnvOverlay("REDIS_PASSWORD", cfg.Redis.Password, "", yamlLoaded)

	cfg.NATS.URL = getEnvOverlay("NATS_URL", cfg.NATS.URL, "nats://localhost:4222", yamlLoaded)

	cfg.JWT.Secret = getEnvOverlay("JWT_SECRET", cfg.JWT.Secret, "", yamlLoaded)

	cfg.Tracing.OTLPEndpoint = getEnvOverlay("TRACING_OTLP_ENDPOINT", cfg.Tracing.OTLPEndpoint, "", yamlLoaded)
	cfg.Tracing.NewRelic.LicenseKey = getEnvOverlay("NEW_RELIC_LICENSE_KEY", cfg.Tracing.NewRelic.LicenseKey, "", yamlLoaded)

	// Auth API keys — always from env
	if keys := getEnvAsMap("AUTH_API_KEYS"); len(keys) > 0 {
		cfg.Auth.APIKeys = keys
	} else if cfg.Auth.APIKeys == nil {
		cfg.Auth.APIKeys = make(map[string]string)
	}

	// --- Non-secret fields: only overlay if YAML was NOT loaded (full backward compat) ---
	if !yamlLoaded {
		cfg.App.Name = getEnv("APP_NAME", "parkir-pintar")
		cfg.App.Environment = getEnv("APP_ENV", "local")
		cfg.App.Debug = getEnvAsBool("APP_DEBUG", false)
		cfg.App.Version = getEnv("APP_VERSION", "0.0.1")

		cfg.Server.Host = getEnv("SERVER_HOST", "0.0.0.0")
		cfg.Server.Port = getEnvAsInt("SERVER_PORT", 8080)
		cfg.Server.ReadTimeout = getEnvAsInt("SERVER_READ_TIMEOUT", 15)
		cfg.Server.WriteTimeout = getEnvAsInt("SERVER_WRITE_TIMEOUT", 15)
		cfg.Server.ShutdownTimeout = getEnvAsInt("SERVER_SHUTDOWN_TIMEOUT", 30)
		cfg.Server.AllowedOrigins = getEnvAsSlice("SERVER_ALLOWED_ORIGINS", []string{"http://localhost:3000"})

		cfg.Database.Schema = getEnv("DB_SCHEMA", "public")
		cfg.Database.SSLMode = getEnv("DB_SSL_MODE", "disable")
		cfg.Database.MaxConns = getEnvAsInt("DB_MAX_CONNS", 25)
		cfg.Database.IdleConns = getEnvAsInt("DB_IDLE_CONNS", 5)
		cfg.Database.MaxLifetime = getEnvAsInt("DB_MAX_LIFETIME", 15)

		cfg.Redis.DB = getEnvAsInt("REDIS_DB", 0)
		cfg.Redis.PoolSize = getEnvAsInt("REDIS_POOL_SIZE", 10)

		cfg.JWT.Expiration = getEnvAsInt("JWT_EXPIRATION", 60)
		cfg.JWT.Issuer = getEnv("JWT_ISSUER", "parkir-pintar")

		cfg.Tracing.Enabled = getEnvAsBool("TRACING_ENABLED", false)
		cfg.Tracing.ServiceName = getEnv("TRACING_SERVICE_NAME", cfg.App.Name)
		cfg.Tracing.SampleRate = getEnvAsFloat("TRACING_SAMPLE_RATE", 1.0)
		cfg.Tracing.ExcludePaths = getEnvAsSlice("TRACING_EXCLUDE_PATHS", []string{"/health", "/health/live", "/health/ready"})
		cfg.Tracing.Exporter = getEnv("TRACING_EXPORTER", "noop")
		cfg.Tracing.NewRelic.Enabled = getEnvAsBool("NEW_RELIC_ENABLED", false)

		cfg.GRPC.Server.Port = getEnvAsInt("GRPC_SERVER_PORT", 9090)
		cfg.GRPC.Server.TLSCertPath = getEnv("GRPC_TLS_CERT_PATH", "")
		cfg.GRPC.Server.TLSKeyPath = getEnv("GRPC_TLS_KEY_PATH", "")
		cfg.GRPC.Server.MaxConnAge = getEnvAsDuration("GRPC_MAX_CONN_AGE", 0)
		cfg.GRPC.Server.ShutdownTimeout = getEnvAsDuration("GRPC_SHUTDOWN_TIMEOUT", 30*time.Second)
		cfg.GRPC.Server.RequestTimeout = getEnvAsDuration("GRPC_REQUEST_TIMEOUT", 30*time.Second)
		cfg.GRPC.Client.DialTimeout = getEnvAsDuration("GRPC_DIAL_TIMEOUT", 5*time.Second)
		cfg.GRPC.Client.KeepAliveTime = getEnvAsDuration("GRPC_KEEPALIVE_TIME", 30*time.Second)
		cfg.GRPC.Client.KeepAliveTimeout = getEnvAsDuration("GRPC_KEEPALIVE_TIMEOUT", 10*time.Second)
		cfg.GRPC.RateLimit.RequestsPerSecond = getEnvAsInt("GRPC_RATE_LIMIT_RPS", 100)
		cfg.GRPC.RateLimit.BurstSize = getEnvAsInt("GRPC_RATE_LIMIT_BURST", 200)

		cfg.Logger.Level = getEnv("LOG_LEVEL", "info")
		cfg.Logger.Format = getEnv("LOG_FORMAT", "json")

		cfg.Reservation.PaymentTimeoutMinutes = getEnvAsInt("PAYMENT_TIMEOUT_MINUTES", 10)
		cfg.Reservation.ExpiryTimeoutMinutes = getEnvAsInt("RESERVATION_EXPIRY_MINUTES", 60)
		cfg.Reservation.WorkerPollInterval = getEnvAsDuration("WORKER_POLL_INTERVAL", 30*time.Second)

		cfg.Asynq.Concurrency = getEnvAsInt("ASYNQ_CONCURRENCY", 10)

		cfg.NATS.Enabled = getEnvAsBool("NATS_ENABLED", false)
	}
}

// Deprecated: Use LoadConfig(serviceName) for YAML-based configuration.
func Load(envPath string) (*Config, error) {
	env := os.Getenv("APP_ENV")
	if env == "" || env == "local" {
		if envPath != "" {
			_ = godotenv.Load(envPath)
		}
	}

	cfg := &Config{}

	cfg.App.Name = getEnv("APP_NAME", "parkir-pintar")
	cfg.App.Environment = getEnv("APP_ENV", "local")
	cfg.App.Debug = getEnvAsBool("APP_DEBUG", false)
	cfg.App.Version = getEnv("APP_VERSION", "0.0.1")

	cfg.Server.Host = getEnv("SERVER_HOST", "0.0.0.0")
	cfg.Server.Port = getEnvAsInt("SERVER_PORT", 8080)
	cfg.Server.ReadTimeout = getEnvAsInt("SERVER_READ_TIMEOUT", 15)
	cfg.Server.WriteTimeout = getEnvAsInt("SERVER_WRITE_TIMEOUT", 15)
	cfg.Server.ShutdownTimeout = getEnvAsInt("SERVER_SHUTDOWN_TIMEOUT", 30)
	cfg.Server.AllowedOrigins = getEnvAsSlice("SERVER_ALLOWED_ORIGINS", []string{"http://localhost:3000"})

	cfg.Database.Host = getEnv("DB_HOST", "localhost")
	cfg.Database.Port = getEnvAsInt("DB_PORT", 5432)
	cfg.Database.Username = getEnv("DB_USERNAME", "")
	cfg.Database.Password = getEnv("DB_PASSWORD", "")
	cfg.Database.Database = getEnv("DB_DATABASE", "")
	cfg.Database.Schema = getEnv("DB_SCHEMA", "public")
	cfg.Database.SSLMode = getEnv("DB_SSL_MODE", "disable")
	cfg.Database.MaxConns = getEnvAsInt("DB_MAX_CONNS", 25)
	cfg.Database.IdleConns = getEnvAsInt("DB_IDLE_CONNS", 5)
	cfg.Database.MaxLifetime = getEnvAsInt("DB_MAX_LIFETIME", 15)

	cfg.Redis.Host = getEnv("REDIS_HOST", "localhost")
	cfg.Redis.Port = getEnvAsInt("REDIS_PORT", 6379)
	cfg.Redis.Password = getEnv("REDIS_PASSWORD", "")
	cfg.Redis.DB = getEnvAsInt("REDIS_DB", 0)
	cfg.Redis.PoolSize = getEnvAsInt("REDIS_POOL_SIZE", 10)

	cfg.JWT.Secret = getEnv("JWT_SECRET", "")
	cfg.JWT.Expiration = getEnvAsInt("JWT_EXPIRATION", 60)
	cfg.JWT.Issuer = getEnv("JWT_ISSUER", "parkir-pintar")

	cfg.Auth.APIKeys = getEnvAsMap("AUTH_API_KEYS")

	cfg.Tracing.Enabled = getEnvAsBool("TRACING_ENABLED", false)
	cfg.Tracing.ServiceName = getEnv("TRACING_SERVICE_NAME", cfg.App.Name)
	cfg.Tracing.SampleRate = getEnvAsFloat("TRACING_SAMPLE_RATE", 1.0)
	cfg.Tracing.ExcludePaths = getEnvAsSlice("TRACING_EXCLUDE_PATHS", []string{"/health", "/health/live", "/health/ready"})
	cfg.Tracing.Exporter = getEnv("TRACING_EXPORTER", "noop")
	cfg.Tracing.OTLPEndpoint = getEnv("TRACING_OTLP_ENDPOINT", "")
	cfg.Tracing.NewRelic.LicenseKey = getEnv("NEW_RELIC_LICENSE_KEY", "")
	cfg.Tracing.NewRelic.Enabled = getEnvAsBool("NEW_RELIC_ENABLED", false)

	cfg.GRPC.Server.Port = getEnvAsInt("GRPC_SERVER_PORT", 9090)
	cfg.GRPC.Server.TLSCertPath = getEnv("GRPC_TLS_CERT_PATH", "")
	cfg.GRPC.Server.TLSKeyPath = getEnv("GRPC_TLS_KEY_PATH", "")
	cfg.GRPC.Server.MaxConnAge = getEnvAsDuration("GRPC_MAX_CONN_AGE", 0)
	cfg.GRPC.Server.ShutdownTimeout = getEnvAsDuration("GRPC_SHUTDOWN_TIMEOUT", 30*time.Second)
	cfg.GRPC.Server.RequestTimeout = getEnvAsDuration("GRPC_REQUEST_TIMEOUT", 30*time.Second)
	cfg.GRPC.Client.DialTimeout = getEnvAsDuration("GRPC_DIAL_TIMEOUT", 5*time.Second)
	cfg.GRPC.Client.KeepAliveTime = getEnvAsDuration("GRPC_KEEPALIVE_TIME", 30*time.Second)
	cfg.GRPC.Client.KeepAliveTimeout = getEnvAsDuration("GRPC_KEEPALIVE_TIMEOUT", 10*time.Second)
	cfg.GRPC.RateLimit.RequestsPerSecond = getEnvAsInt("GRPC_RATE_LIMIT_RPS", 100)
	cfg.GRPC.RateLimit.BurstSize = getEnvAsInt("GRPC_RATE_LIMIT_BURST", 200)

	cfg.Logger.Level = getEnv("LOG_LEVEL", "info")
	cfg.Logger.Format = getEnv("LOG_FORMAT", "json")

	cfg.Reservation.PaymentTimeoutMinutes = getEnvAsInt("PAYMENT_TIMEOUT_MINUTES", 10)
	cfg.Reservation.ExpiryTimeoutMinutes = getEnvAsInt("RESERVATION_EXPIRY_MINUTES", 60)
	cfg.Reservation.WorkerPollInterval = getEnvAsDuration("WORKER_POLL_INTERVAL", 30*time.Second)

	cfg.Asynq.Concurrency = getEnvAsInt("ASYNQ_CONCURRENCY", 10)

	cfg.NATS.URL = getEnv("NATS_URL", "nats://localhost:4222")
	cfg.NATS.Enabled = getEnvAsBool("NATS_ENABLED", false)

	if err := validate(cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

func validate(cfg *Config) error {
	// Server port must be valid
	if cfg.Server.Port <= 0 || cfg.Server.Port >= 65536 {
		return fmt.Errorf("SERVER_PORT must be between 1 and 65535, got %d", cfg.Server.Port)
	}

	// Tracing sample rate must be between 0.0 and 1.0
	if cfg.Tracing.SampleRate < 0.0 || cfg.Tracing.SampleRate > 1.0 {
		return fmt.Errorf("TRACING_SAMPLE_RATE must be between 0.0 and 1.0, got %f", cfg.Tracing.SampleRate)
	}

	if cfg.JWT.Secret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}

	if cfg.App.Environment != "local" && cfg.App.Environment != "test" && len(cfg.JWT.Secret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters in non-local environments")
	}

	return nil
}

// getEnvOverlay returns the env var value if set, otherwise returns yamlValue if YAML was loaded,
// otherwise returns defaultValue.
func getEnvOverlay(key, yamlValue, defaultValue string, yamlLoaded bool) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	if yamlLoaded && yamlValue != "" {
		return yamlValue
	}
	return defaultValue
}

// getEnvAsIntOverlay returns the env var value if set, otherwise returns yamlValue if YAML was loaded,
// otherwise returns defaultValue.
func getEnvAsIntOverlay(key string, yamlValue, defaultValue int, yamlLoaded bool) int {
	valueStr := os.Getenv(key)
	if valueStr != "" {
		if v, err := strconv.Atoi(valueStr); err == nil {
			return v
		}
	}
	if yamlLoaded && yamlValue != 0 {
		return yamlValue
	}
	return defaultValue
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

//nolint:unparam // defaultValue is always false today but kept for API consistency with other getEnvAs* helpers.
func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func getEnvAsFloat(key string, defaultValue float64) float64 {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return defaultValue
	}
	return value
}

// Returns defaultValue if the env var is unset or cannot be parsed.
func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := time.ParseDuration(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func getEnvAsSlice(key string, defaultValue []string) []string {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	parts := strings.Split(valueStr, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return defaultValue
	}
	return result
}

func getEnvAsMap(key string) map[string]string {
	result := make(map[string]string)
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return result
	}
	pairs := strings.Split(valueStr, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		kv := strings.SplitN(pair, ":", 2)
		if len(kv) == 2 {
			k := strings.TrimSpace(kv[0])
			v := strings.TrimSpace(kv[1])
			if k != "" {
				result[k] = v
			}
		}
	}
	return result
}
