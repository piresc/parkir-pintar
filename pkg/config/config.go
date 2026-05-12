package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config is the root configuration struct loaded from env vars.
type Config struct {
	App         AppConfig
	Server      ServerConfig
	Database    DatabaseConfig
	Redis       RedisConfig
	NATS        NATSConfig
	JWT         JWTConfig
	Auth        AuthConfig
	Tracing     TracingConfig
	Logger      LoggerConfig
	GRPC        GRPCConfig
	Reservation ReservationConfig
}

// ReservationConfig holds reservation service settings.
type ReservationConfig struct {
	PaymentTimeoutMinutes int // default 10
}

// GRPCServerConfig holds gRPC server settings.
type GRPCServerConfig struct {
	Port        int
	TLSCertPath string
	TLSKeyPath  string
	MaxConnAge  time.Duration
}

// GRPCClientConfig holds gRPC client settings.
type GRPCClientConfig struct {
	DialTimeout      time.Duration
	KeepAliveTime    time.Duration
	KeepAliveTimeout time.Duration
}

// GRPCConfig holds gRPC server and client configuration.
type GRPCConfig struct {
	Server GRPCServerConfig
	Client GRPCClientConfig
}

// AppConfig holds application-level settings.
type AppConfig struct {
	Name        string
	Environment string // local, development, staging, production
	Debug       bool
	Version     string
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Host            string
	Port            int
	ReadTimeout     int // seconds
	WriteTimeout    int // seconds
	ShutdownTimeout int // seconds
	AllowedOrigins  []string
}

// DatabaseConfig holds PostgreSQL connection settings.
type DatabaseConfig struct {
	Host        string
	Port        int
	Username    string
	Password    string
	Database    string
	Schema      string
	SSLMode     string
	MaxConns    int
	IdleConns   int
	MaxLifetime int // minutes
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
	PoolSize int
}

// NATSConfig holds NATS connection settings.
type NATSConfig struct {
	URL string
}

// JWTConfig holds JWT token settings.
type JWTConfig struct {
	Secret     string
	Expiration int // minutes
	Issuer     string
}

// AuthConfig holds API key authentication settings.
type AuthConfig struct {
	APIKeys map[string]string // service_name -> key
}

// TracingConfig holds tracing/observability settings.
type TracingConfig struct {
	Enabled      bool
	ServiceName  string
	SampleRate   float64
	ExcludePaths []string
	Exporter     string // "stdout", "otlp", "newrelic", "noop"
	OTLPEndpoint string
	NewRelic     NewRelicExporterConfig
}

// NewRelicExporterConfig holds New Relic OTEL exporter settings.
type NewRelicExporterConfig struct {
	LicenseKey string
	Enabled    bool
}

// LoggerConfig holds structured logging settings.
type LoggerConfig struct {
	Level  string // debug, info, warn, error
	Format string // json, text
}

// Load reads .env in local mode, then populates Config from os env vars.
// envPath is the path to the .env file (used only when APP_ENV is "local" or empty).
func Load(envPath string) (*Config, error) {
	env := os.Getenv("APP_ENV")
	if env == "" || env == "local" {
		if envPath != "" {
			// Best-effort load; ignore error if file doesn't exist in local mode
			_ = godotenv.Load(envPath)
		}
	}

	cfg := &Config{}

	// App
	cfg.App.Name = getEnv("APP_NAME", "parkir-pintar")
	cfg.App.Environment = getEnv("APP_ENV", "local")
	cfg.App.Debug = getEnvAsBool("APP_DEBUG", false)
	cfg.App.Version = getEnv("APP_VERSION", "0.0.1")

	// Server
	cfg.Server.Host = getEnv("SERVER_HOST", "0.0.0.0")
	cfg.Server.Port = getEnvAsInt("SERVER_PORT", 8080)
	cfg.Server.ReadTimeout = getEnvAsInt("SERVER_READ_TIMEOUT", 15)
	cfg.Server.WriteTimeout = getEnvAsInt("SERVER_WRITE_TIMEOUT", 15)
	cfg.Server.ShutdownTimeout = getEnvAsInt("SERVER_SHUTDOWN_TIMEOUT", 30)
	cfg.Server.AllowedOrigins = getEnvAsSlice("SERVER_ALLOWED_ORIGINS", []string{"http://localhost:3000"})

	// Database
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

	// Redis
	cfg.Redis.Host = getEnv("REDIS_HOST", "localhost")
	cfg.Redis.Port = getEnvAsInt("REDIS_PORT", 6379)
	cfg.Redis.Password = getEnv("REDIS_PASSWORD", "")
	cfg.Redis.DB = getEnvAsInt("REDIS_DB", 0)
	cfg.Redis.PoolSize = getEnvAsInt("REDIS_POOL_SIZE", 10)

	// NATS
	cfg.NATS.URL = getEnv("NATS_URL", "nats://localhost:4222")

	// JWT — no default for Secret (security)
	cfg.JWT.Secret = getEnv("JWT_SECRET", "")
	cfg.JWT.Expiration = getEnvAsInt("JWT_EXPIRATION", 60)
	cfg.JWT.Issuer = getEnv("JWT_ISSUER", "parkir-pintar")

	// Auth — API keys parsed from comma-separated "service:key" pairs
	cfg.Auth.APIKeys = getEnvAsMap("AUTH_API_KEYS")

	// Tracing
	cfg.Tracing.Enabled = getEnvAsBool("TRACING_ENABLED", false)
	cfg.Tracing.ServiceName = getEnv("TRACING_SERVICE_NAME", cfg.App.Name)
	cfg.Tracing.SampleRate = getEnvAsFloat("TRACING_SAMPLE_RATE", 1.0)
	cfg.Tracing.ExcludePaths = getEnvAsSlice("TRACING_EXCLUDE_PATHS", []string{"/health", "/health/live", "/health/ready"})
	cfg.Tracing.Exporter = getEnv("TRACING_EXPORTER", "noop")
	cfg.Tracing.OTLPEndpoint = getEnv("TRACING_OTLP_ENDPOINT", "")
	cfg.Tracing.NewRelic.LicenseKey = getEnv("NEW_RELIC_LICENSE_KEY", "")
	cfg.Tracing.NewRelic.Enabled = getEnvAsBool("NEW_RELIC_ENABLED", false)

	// GRPC
	cfg.GRPC.Server.Port = getEnvAsInt("GRPC_SERVER_PORT", 9090)
	cfg.GRPC.Server.TLSCertPath = getEnv("GRPC_TLS_CERT_PATH", "")
	cfg.GRPC.Server.TLSKeyPath = getEnv("GRPC_TLS_KEY_PATH", "")
	cfg.GRPC.Server.MaxConnAge = getEnvAsDuration("GRPC_MAX_CONN_AGE", 0)
	cfg.GRPC.Client.DialTimeout = getEnvAsDuration("GRPC_DIAL_TIMEOUT", 5*time.Second)
	cfg.GRPC.Client.KeepAliveTime = getEnvAsDuration("GRPC_KEEPALIVE_TIME", 30*time.Second)
	cfg.GRPC.Client.KeepAliveTimeout = getEnvAsDuration("GRPC_KEEPALIVE_TIMEOUT", 10*time.Second)

	// Logger
	cfg.Logger.Level = getEnv("LOG_LEVEL", "info")
	cfg.Logger.Format = getEnv("LOG_FORMAT", "json")

	// Reservation
	cfg.Reservation.PaymentTimeoutMinutes = getEnvAsInt("PAYMENT_TIMEOUT_MINUTES", 10)

	if err := validate(cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// validate checks required fields and value constraints.
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

	return nil
}

// --- helper functions ---

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

// getEnvAsDuration parses an env var as a time.Duration string (e.g. "5s", "30s", "1m").
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

// getEnvAsSlice parses a comma-separated env var into a string slice.
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

// getEnvAsMap parses a comma-separated "key:value" env var into a map.
// Example: "service1:key1,service2:key2"
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
