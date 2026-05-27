# Configuration System

## Design Principles

1. **YAML-first:** All non-secret configuration lives in versioned YAML files. No magic defaults hidden in code.
2. **Secrets via environment variables only:** Database credentials, JWT keys, and API tokens are never committed.
3. **Fail-fast on invalid config:** Missing required values or invalid ranges cause a hard startup error with a descriptive message.
4. **Single loader for all services:** Every service uses the same `pkg/config.LoadConfig()` function.

## Config Loader

The configuration system is built on [Viper](https://github.com/spf13/viper) with explicit env-var binding for secrets.

```go
// Usage in any service's main.go:
cfg, err := config.LoadConfig("gateway")  // loads config/<env>/gateway.yaml
if err != nil {
    log.Fatalf("config: %v", err)  // hard crash — no partial startup
}
```

### Load Sequence

```
1. Determine environment: APP_ENV env var (default: "local")
2. If local/test: load config/.env via godotenv
3. Read YAML: config/<env>/<service>.yaml
4. Bind secrets: map specific env vars → config paths
5. Unmarshal into typed Config struct
6. Validate: port ranges, sample rates, JWT secret length
7. Return *Config or error (no nil config, no partial state)
```

## Environment Hierarchy

| Environment | YAML Path | Secrets Source | Behavior |
|-------------|-----------|----------------|----------|
| `local` | `config/local/<service>.yaml` | `config/.env` file (auto-loaded) | Debug enabled, noop tracing |
| `test` | `config/test/<service>.yaml` | `config/.env` file (auto-loaded) | Minimal config for unit tests |
| `staging` | `config/staging/<service>.yaml` | Coolify env vars | Full tracing, production-like |
| `production` | `config/staging/<service>.yaml` | K8s Secrets + ConfigMaps (AWS EKS) | Strict validation, optimized pools |

The `APP_ENV` environment variable selects which directory to load from. In staging/production, the `.env` file is not loaded — all secrets must come from the container runtime environment.

## Secret Binding

Only these environment variables override YAML values:

```
DB_HOST          → database.host
DB_PORT          → database.port
DB_USERNAME      → database.username
DB_PASSWORD      → database.password
DB_DATABASE      → database.database
DB_SCHEMA        → database.schema
DB_SSL_MODE      → database.ssl_mode
REDIS_HOST       → redis.host
REDIS_PORT       → redis.port
REDIS_PASSWORD   → redis.password
JWT_SECRET       → jwt.secret
NATS_URL         → nats.url
TRACING_OTLP_ENDPOINT → tracing.otlp_endpoint
```

This is an explicit allowlist. Arbitrary env vars do not override YAML keys — only the bindings above are honored.

## Config Structure

The unified `Config` struct covers all services. Each service uses only the sections relevant to it.

```yaml
app:
  name: parkir-pintar-<service>
  environment: local
  debug: true
  version: "0.1.0"

server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: 30        # seconds
  write_timeout: 30       # seconds
  shutdown_timeout: 30    # seconds
  allowed_origins:
    - "http://localhost:3000"

database:
  host: localhost
  port: 5432
  database: parkir_pintar
  schema: public
  ssl_mode: disable
  max_conns: 25
  idle_conns: 5
  max_lifetime: 15        # minutes

redis:
  db: 0
  pool_size: 10

jwt:
  expiration: 60          # minutes
  issuer: parkir-pintar

tracing:
  enabled: false
  service_name: parkir-pintar-<service>
  sample_rate: 1.0
  exclude_paths:
    - /health
    - /health/live
    - /health/ready
  exporter: noop          # noop | stdout | otlp | otlp-grpc
  otlp_endpoint: ""       # set via TRACING_OTLP_ENDPOINT in staging/prod

logger:
  level: debug            # debug | info | warn | error
  format: json            # json | text

grpc:
  server:
    port: 9090
    shutdown_timeout: 30s
    request_timeout: 30s
  client:
    dial_timeout: 5s
    keepalive_time: 30s
    keepalive_timeout: 10s
  rate_limit:
    requests_per_second: 100
    burst_size: 200

nats:
  enabled: false
  # url set via NATS_URL env var

reservation:              # reservation service only
  payment_timeout_minutes: 10
  expiry_timeout_minutes: 60
  worker_poll_interval: 30s

asynq:                    # reservation service only
  concurrency: 10
```

## Per-Service Differences

| Service | Notable Config |
|---------|---------------|
| **Gateway** | HTTP server on port 8080, CORS origins, gRPC client settings for downstream calls |
| **Search** | Redis DB 1 (isolated from other services), higher rate limits (200 rps / 400 burst) |
| **Reservation** | Payment timeout, expiry timeout, Asynq worker concurrency, NATS enabled |
| **Billing** | Minimal — gRPC server only, no Redis, no NATS |
| **Payment** | NATS enabled for event publishing |
| **Presence** | Redis for real-time state, no NATS |
| **Analytics** | NATS consumer, gRPC on port 9095 (non-standard) |

## Validation Rules

The config loader enforces these constraints at startup:

| Rule | Failure Mode |
|------|-------------|
| `server.port` must be 1–65535 | Fatal error |
| `tracing.sample_rate` must be 0.0–1.0 | Fatal error |
| `JWT_SECRET` must be set | Fatal error |
| `JWT_SECRET` must be ≥ 32 chars (non-local envs) | Fatal error |
| YAML file must exist at expected path | Fatal error |

If any validation fails, the service prints the error and exits immediately. There is no fallback to defaults or partial startup.

## Adding New Configuration

1. Add the field to the appropriate struct in `pkg/config/config.go`
2. Add the YAML key to all environment-specific config files (`config/local/<service>.yaml`, etc.)
3. If the value is a secret, add an env-var binding in `bindSecrets()`
4. If the value needs validation, add a check in `validate()`
5. Update `config/.env.example` if a new secret is introduced
