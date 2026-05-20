package config

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

const (
	defaultEnv                 = "local"
	testEnv                    = "test"
	defaultAllowedOrigin       = "http://localhost:3000"
	defaultHealthEndpoint      = "/health"
	defaultHealthLiveEndpoint  = "/health/live"
	defaultHealthReadyEndpoint = "/health/ready"
)

type Config struct {
	App         AppConfig         `yaml:"app" mapstructure:"app"`
	Server      ServerConfig      `yaml:"server" mapstructure:"server"`
	Database    DatabaseConfig    `yaml:"database" mapstructure:"database"`
	Redis       RedisConfig       `yaml:"redis" mapstructure:"redis"`
	JWT         JWTConfig         `yaml:"jwt" mapstructure:"jwt"`
	Auth        AuthConfig        `yaml:"auth" mapstructure:"auth"`
	Tracing     TracingConfig     `yaml:"tracing" mapstructure:"tracing"`
	Logger      LoggerConfig      `yaml:"logger" mapstructure:"logger"`
	GRPC        GRPCConfig        `yaml:"grpc" mapstructure:"grpc"`
	Reservation ReservationConfig `yaml:"reservation" mapstructure:"reservation"`
	Asynq       AsynqConfig       `yaml:"asynq" mapstructure:"asynq"`
	NATS        NATSConfig        `yaml:"nats" mapstructure:"nats"`
}

type ReservationConfig struct {
	PaymentTimeoutMinutes int           `yaml:"payment_timeout_minutes" mapstructure:"payment_timeout_minutes"`
	ExpiryTimeoutMinutes  int           `yaml:"expiry_timeout_minutes" mapstructure:"expiry_timeout_minutes"`
	WorkerPollInterval    time.Duration `yaml:"worker_poll_interval" mapstructure:"worker_poll_interval"`
}

type AsynqConfig struct {
	Concurrency int `yaml:"concurrency" mapstructure:"concurrency"`
}

type NATSConfig struct {
	URL     string `yaml:"url" mapstructure:"url"`
	Enabled bool   `yaml:"enabled" mapstructure:"enabled"`
}

type GRPCServerConfig struct {
	Port            int           `yaml:"port" mapstructure:"port"`
	TLSCertPath     string        `yaml:"tls_cert_path" mapstructure:"tls_cert_path"`
	TLSKeyPath      string        `yaml:"tls_key_path" mapstructure:"tls_key_path"`
	MaxConnAge      time.Duration `yaml:"max_conn_age" mapstructure:"max_conn_age"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout" mapstructure:"shutdown_timeout"`
	RequestTimeout  time.Duration `yaml:"request_timeout" mapstructure:"request_timeout"`
}

type GRPCRateLimitConfig struct {
	RequestsPerSecond int `yaml:"requests_per_second" mapstructure:"requests_per_second"`
	BurstSize         int `yaml:"burst_size" mapstructure:"burst_size"`
}

type GRPCClientConfig struct {
	DialTimeout      time.Duration `yaml:"dial_timeout" mapstructure:"dial_timeout"`
	KeepAliveTime    time.Duration `yaml:"keepalive_time" mapstructure:"keepalive_time"`
	KeepAliveTimeout time.Duration `yaml:"keepalive_timeout" mapstructure:"keepalive_timeout"`
}

type GRPCConfig struct {
	Server    GRPCServerConfig    `yaml:"server" mapstructure:"server"`
	Client    GRPCClientConfig    `yaml:"client" mapstructure:"client"`
	RateLimit GRPCRateLimitConfig `yaml:"rate_limit" mapstructure:"rate_limit"`
}

type AppConfig struct {
	Name        string `yaml:"name" mapstructure:"name"`
	Environment string `yaml:"environment" mapstructure:"environment"`
	Debug       bool   `yaml:"debug" mapstructure:"debug"`
	Version     string `yaml:"version" mapstructure:"version"`
}

type ServerConfig struct {
	Host            string   `yaml:"host" mapstructure:"host"`
	Port            int      `yaml:"port" mapstructure:"port"`
	ReadTimeout     int      `yaml:"read_timeout" mapstructure:"read_timeout"`
	WriteTimeout    int      `yaml:"write_timeout" mapstructure:"write_timeout"`
	ShutdownTimeout int      `yaml:"shutdown_timeout" mapstructure:"shutdown_timeout"`
	AllowedOrigins  []string `yaml:"allowed_origins" mapstructure:"allowed_origins"`
}

type DatabaseConfig struct {
	Host        string `yaml:"host" mapstructure:"host"`
	Port        int    `yaml:"port" mapstructure:"port"`
	Username    string `yaml:"username" mapstructure:"username"`
	Password    string `yaml:"password" mapstructure:"password"`
	Database    string `yaml:"database" mapstructure:"database"`
	Schema      string `yaml:"schema" mapstructure:"schema"`
	SSLMode     string `yaml:"ssl_mode" mapstructure:"ssl_mode"`
	MaxConns    int    `yaml:"max_conns" mapstructure:"max_conns"`
	IdleConns   int    `yaml:"idle_conns" mapstructure:"idle_conns"`
	MaxLifetime int    `yaml:"max_lifetime" mapstructure:"max_lifetime"`
}

type RedisConfig struct {
	Host     string `yaml:"host" mapstructure:"host"`
	Port     int    `yaml:"port" mapstructure:"port"`
	Password string `yaml:"password" mapstructure:"password"`
	DB       int    `yaml:"db" mapstructure:"db"`
	PoolSize int    `yaml:"pool_size" mapstructure:"pool_size"`
}

type JWTConfig struct {
	Secret     string `yaml:"secret" mapstructure:"secret"`
	Expiration int    `yaml:"expiration" mapstructure:"expiration"`
	Issuer     string `yaml:"issuer" mapstructure:"issuer"`
}

type AuthConfig struct {
	APIKeys map[string]string `yaml:"api_keys" mapstructure:"api_keys"`
}

type TracingConfig struct {
	Enabled      bool                   `yaml:"enabled" mapstructure:"enabled"`
	ServiceName  string                 `yaml:"service_name" mapstructure:"service_name"`
	SampleRate   float64                `yaml:"sample_rate" mapstructure:"sample_rate"`
	ExcludePaths []string               `yaml:"exclude_paths" mapstructure:"exclude_paths"`
	Exporter     string                 `yaml:"exporter" mapstructure:"exporter"`
	OTLPEndpoint string                 `yaml:"otlp_endpoint" mapstructure:"otlp_endpoint"`
	NewRelic     NewRelicExporterConfig `yaml:"new_relic" mapstructure:"new_relic"`
}

type NewRelicExporterConfig struct {
	LicenseKey string `yaml:"license_key" mapstructure:"license_key"`
	Enabled    bool   `yaml:"enabled" mapstructure:"enabled"`
}

type LoggerConfig struct {
	Level  string `yaml:"level" mapstructure:"level"`
	Format string `yaml:"format" mapstructure:"format"`
}

// LoadConfig loads configuration using Viper. YAML files provide all non-secret
// config; environment variables provide secrets and can override any YAML value.
func LoadConfig(serviceName string) (*Config, error) {
	env := getEnv("APP_ENV", defaultEnv)

	// In local/test dev, load .env for secrets
	if env == defaultEnv || env == testEnv {
		_ = godotenv.Load(".env")
		_ = godotenv.Load("config/.env")
	}

	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Load YAML config file: config/<service>.<env>.yaml
	v.SetConfigName(fmt.Sprintf("%s.%s", serviceName, env))
	v.SetConfigType("yaml")
	v.AddConfigPath("config/")
	v.AddConfigPath(".")

	if err := v.ReadInConfig(); err != nil {
		var configNotFound viper.ConfigFileNotFoundError
		if !errors.As(err, &configNotFound) {
			return nil, fmt.Errorf("read config file: %w", err)
		}
		// Config file not found is OK — fall back to env + defaults
	}

	// Bind env vars explicitly (secrets + legacy env var names)
	bindEnvVars(v)

	// Allow env vars to override any config with underscore-separated keys
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	var cfg Config
	if err := v.Unmarshal(&cfg, decoderConfigOption()); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	// Handle AUTH_API_KEYS from env (comma-separated key:value pairs)
	if keys := parseEnvAsMap("AUTH_API_KEYS"); len(keys) > 0 {
		cfg.Auth.APIKeys = keys
	} else if cfg.Auth.APIKeys == nil {
		cfg.Auth.APIKeys = make(map[string]string)
	}

	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

// Load loads configuration purely from environment variables (deprecated).
// Use LoadConfig(serviceName) for YAML-based configuration.
//
// Deprecated: Use LoadConfig instead.
func Load(envPath string) (*Config, error) {
	env := getEnv("APP_ENV", defaultEnv)

	if env == defaultEnv || env == testEnv {
		if envPath != "" {
			_ = godotenv.Load(envPath)
		}
	}

	v := viper.New()

	// Set defaults
	setDefaults(v)

	// No YAML file for legacy Load — purely env-based
	// Bind all env vars
	bindEnvVars(v)

	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	var cfg Config
	if err := v.Unmarshal(&cfg, decoderConfigOption()); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	// Handle AUTH_API_KEYS from env (comma-separated key:value pairs)
	if keys := parseEnvAsMap("AUTH_API_KEYS"); len(keys) > 0 {
		cfg.Auth.APIKeys = keys
	} else if cfg.Auth.APIKeys == nil {
		cfg.Auth.APIKeys = make(map[string]string)
	}

	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

// setDefaults sets sensible defaults so services work without a YAML file.
func setDefaults(v *viper.Viper) {
	// App
	v.SetDefault("app.name", "parkir-pintar")
	v.SetDefault("app.environment", defaultEnv)
	v.SetDefault("app.debug", false)
	v.SetDefault("app.version", "0.0.1")

	// Server
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", 15)
	v.SetDefault("server.write_timeout", 15)
	v.SetDefault("server.shutdown_timeout", 30)
	v.SetDefault("server.allowed_origins", []string{defaultAllowedOrigin})

	// Database
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.username", "")
	v.SetDefault("database.password", "")
	v.SetDefault("database.database", "")
	v.SetDefault("database.schema", "public")
	v.SetDefault("database.ssl_mode", "disable")
	v.SetDefault("database.max_conns", 25)
	v.SetDefault("database.idle_conns", 5)
	v.SetDefault("database.max_lifetime", 15)

	// Redis
	v.SetDefault("redis.host", "localhost")
	v.SetDefault("redis.port", 6379)
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.pool_size", 10)

	// JWT
	v.SetDefault("jwt.secret", "")
	v.SetDefault("jwt.expiration", 60)
	v.SetDefault("jwt.issuer", "parkir-pintar")

	// Tracing
	v.SetDefault("tracing.enabled", false)
	v.SetDefault("tracing.service_name", "parkir-pintar")
	v.SetDefault("tracing.sample_rate", 1.0)
	v.SetDefault("tracing.exclude_paths", []string{defaultHealthEndpoint, defaultHealthLiveEndpoint, defaultHealthReadyEndpoint})
	v.SetDefault("tracing.exporter", "noop")
	v.SetDefault("tracing.otlp_endpoint", "")
	v.SetDefault("tracing.new_relic.license_key", "")
	v.SetDefault("tracing.new_relic.enabled", false)

	// Logger
	v.SetDefault("logger.level", "info")
	v.SetDefault("logger.format", "json")

	// GRPC
	v.SetDefault("grpc.server.port", 9090)
	v.SetDefault("grpc.server.tls_cert_path", "")
	v.SetDefault("grpc.server.tls_key_path", "")
	v.SetDefault("grpc.server.max_conn_age", time.Duration(0))
	v.SetDefault("grpc.server.shutdown_timeout", 30*time.Second)
	v.SetDefault("grpc.server.request_timeout", 30*time.Second)
	v.SetDefault("grpc.client.dial_timeout", 5*time.Second)
	v.SetDefault("grpc.client.keepalive_time", 30*time.Second)
	v.SetDefault("grpc.client.keepalive_timeout", 10*time.Second)
	v.SetDefault("grpc.rate_limit.requests_per_second", 100)
	v.SetDefault("grpc.rate_limit.burst_size", 200)

	// Reservation
	v.SetDefault("reservation.payment_timeout_minutes", 10)
	v.SetDefault("reservation.expiry_timeout_minutes", 60)
	v.SetDefault("reservation.worker_poll_interval", 30*time.Second)

	// Asynq
	v.SetDefault("asynq.concurrency", 10)

	// NATS
	v.SetDefault("nats.url", "nats://localhost:4222")
	v.SetDefault("nats.enabled", false)
}

// bindEnvVars binds environment variables to Viper keys.
// This handles the mapping between legacy env var names (e.g. DB_HOST)
// and the Viper config path (e.g. database.host).
func bindEnvVars(v *viper.Viper) {
	// App
	_ = v.BindEnv("app.name", "APP_NAME")
	_ = v.BindEnv("app.environment", "APP_ENV")
	_ = v.BindEnv("app.debug", "APP_DEBUG")
	_ = v.BindEnv("app.version", "APP_VERSION")

	// Server
	_ = v.BindEnv("server.host", "SERVER_HOST")
	_ = v.BindEnv("server.port", "SERVER_PORT")
	_ = v.BindEnv("server.read_timeout", "SERVER_READ_TIMEOUT")
	_ = v.BindEnv("server.write_timeout", "SERVER_WRITE_TIMEOUT")
	_ = v.BindEnv("server.shutdown_timeout", "SERVER_SHUTDOWN_TIMEOUT")
	_ = v.BindEnv("server.allowed_origins", "SERVER_ALLOWED_ORIGINS")

	// Database (secrets + non-secrets)
	_ = v.BindEnv("database.host", "DB_HOST")
	_ = v.BindEnv("database.port", "DB_PORT")
	_ = v.BindEnv("database.username", "DB_USERNAME")
	_ = v.BindEnv("database.password", "DB_PASSWORD")
	_ = v.BindEnv("database.database", "DB_DATABASE")
	_ = v.BindEnv("database.schema", "DB_SCHEMA")
	_ = v.BindEnv("database.ssl_mode", "DB_SSL_MODE")
	_ = v.BindEnv("database.max_conns", "DB_MAX_CONNS")
	_ = v.BindEnv("database.idle_conns", "DB_IDLE_CONNS")
	_ = v.BindEnv("database.max_lifetime", "DB_MAX_LIFETIME")

	// Redis
	_ = v.BindEnv("redis.host", "REDIS_HOST")
	_ = v.BindEnv("redis.port", "REDIS_PORT")
	_ = v.BindEnv("redis.password", "REDIS_PASSWORD")
	_ = v.BindEnv("redis.db", "REDIS_DB")
	_ = v.BindEnv("redis.pool_size", "REDIS_POOL_SIZE")

	// JWT
	_ = v.BindEnv("jwt.secret", "JWT_SECRET")
	_ = v.BindEnv("jwt.expiration", "JWT_EXPIRATION")
	_ = v.BindEnv("jwt.issuer", "JWT_ISSUER")

	// Tracing
	_ = v.BindEnv("tracing.enabled", "TRACING_ENABLED")
	_ = v.BindEnv("tracing.service_name", "TRACING_SERVICE_NAME")
	_ = v.BindEnv("tracing.sample_rate", "TRACING_SAMPLE_RATE")
	_ = v.BindEnv("tracing.exclude_paths", "TRACING_EXCLUDE_PATHS")
	_ = v.BindEnv("tracing.exporter", "TRACING_EXPORTER")
	_ = v.BindEnv("tracing.otlp_endpoint", "TRACING_OTLP_ENDPOINT")
	_ = v.BindEnv("tracing.new_relic.license_key", "NEW_RELIC_LICENSE_KEY")
	_ = v.BindEnv("tracing.new_relic.enabled", "NEW_RELIC_ENABLED")

	// Logger
	_ = v.BindEnv("logger.level", "LOG_LEVEL")
	_ = v.BindEnv("logger.format", "LOG_FORMAT")

	// GRPC
	_ = v.BindEnv("grpc.server.port", "GRPC_SERVER_PORT")
	_ = v.BindEnv("grpc.server.tls_cert_path", "GRPC_TLS_CERT_PATH")
	_ = v.BindEnv("grpc.server.tls_key_path", "GRPC_TLS_KEY_PATH")
	_ = v.BindEnv("grpc.server.max_conn_age", "GRPC_MAX_CONN_AGE")
	_ = v.BindEnv("grpc.server.shutdown_timeout", "GRPC_SHUTDOWN_TIMEOUT")
	_ = v.BindEnv("grpc.server.request_timeout", "GRPC_REQUEST_TIMEOUT")
	_ = v.BindEnv("grpc.client.dial_timeout", "GRPC_DIAL_TIMEOUT")
	_ = v.BindEnv("grpc.client.keepalive_time", "GRPC_KEEPALIVE_TIME")
	_ = v.BindEnv("grpc.client.keepalive_timeout", "GRPC_KEEPALIVE_TIMEOUT")
	_ = v.BindEnv("grpc.rate_limit.requests_per_second", "GRPC_RATE_LIMIT_RPS")
	_ = v.BindEnv("grpc.rate_limit.burst_size", "GRPC_RATE_LIMIT_BURST")

	// Reservation
	_ = v.BindEnv("reservation.payment_timeout_minutes", "PAYMENT_TIMEOUT_MINUTES")
	_ = v.BindEnv("reservation.expiry_timeout_minutes", "RESERVATION_EXPIRY_MINUTES")
	_ = v.BindEnv("reservation.worker_poll_interval", "WORKER_POLL_INTERVAL")

	// Asynq
	_ = v.BindEnv("asynq.concurrency", "ASYNQ_CONCURRENCY")

	// NATS
	_ = v.BindEnv("nats.url", "NATS_URL")
	_ = v.BindEnv("nats.enabled", "NATS_ENABLED")
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

	if cfg.App.Environment != defaultEnv && cfg.App.Environment != testEnv && len(cfg.JWT.Secret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters in non-local environments")
	}

	return nil
}

// getEnv returns the value of an environment variable or a default.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// parseEnvAsMap parses a comma-separated key:value env var into a map.
func parseEnvAsMap(key string) map[string]string {
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

// decoderConfigOption returns a viper.DecoderConfigOption that adds a custom
// decode hook for trimming whitespace in string slices (e.g. "a, b" → ["a","b"]).
func decoderConfigOption() viper.DecoderConfigOption {
	return func(dc *mapstructure.DecoderConfig) {
		dc.DecodeHook = mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
			trimStringSliceHookFunc(),
		)
	}
}

// trimStringSliceHookFunc trims whitespace from each element in a []string.
func trimStringSliceHookFunc() mapstructure.DecodeHookFunc {
	return func(from reflect.Type, to reflect.Type, data interface{}) (interface{}, error) {
		if to != reflect.TypeOf([]string{}) {
			return data, nil
		}

		switch v := data.(type) {
		case []string:
			trimmed := make([]string, 0, len(v))
			for _, s := range v {
				s = strings.TrimSpace(s)
				if s != "" {
					trimmed = append(trimmed, s)
				}
			}
			return trimmed, nil
		case []interface{}:
			trimmed := make([]string, 0, len(v))
			for _, item := range v {
				if s, ok := item.(string); ok {
					s = strings.TrimSpace(s)
					if s != "" {
						trimmed = append(trimmed, s)
					}
				}
			}
			return trimmed, nil
		}
		return data, nil
	}
}
