# Shared Packages (`pkg/`)

## Philosophy

The `pkg/` layer contains generic infrastructure code that is reusable across all services. These packages provide foundational capabilities (database connectivity, messaging, observability, resilience) without encoding any domain logic.

Key principles:

- **No domain logic** — packages in `pkg/` solve infrastructure problems only. Business rules live in `internal/<service>/usecase/`.
- **Self-contained** — each package has a focused responsibility and minimal API surface.
- **Composable** — packages are designed to be wired together at service startup (e.g., telemetry providers feed into logger, tracer, and metrics).

## Dependency Direction

```
pkg/ packages MUST NOT import internal/
internal/ packages MAY import pkg/
```

All `pkg/` packages depend only on:
- The Go standard library
- Third-party infrastructure libraries (go-redis, sqlx, nats.go, OTel SDK, etc.)
- Other `pkg/` packages (e.g., `pkg/redis` imports `pkg/config` for `RedisConfig`)

This rule is enforced by convention and CI linting. It ensures `pkg/` remains a stable foundation that never couples to service-specific code.

## Package Reference

| Package | Purpose | Key Exports | Dependencies |
|---------|---------|-------------|--------------|
| `apperror` | Application-level error types with HTTP status mapping | `AppError`, `New()`, sentinel errors (`ErrNotFound`, `ErrConflict`, etc.), constructors (`BadRequest()`, `NotFound()`, etc.) | stdlib |
| `asynq` | Background task queue client and server (Redis-backed) | `Client`, `Server`, `NewClient()`, `NewServer()`, `Enqueue()`, `CancelTask()` | hibiken/asynq |
| `auth` | JWT token generation and validation | `Claims`, `GenerateToken()`, `ValidateToken()` | golang-jwt/jwt, `pkg/config` |
| `circuitbreaker` | Circuit breaker for downstream service calls | `CircuitBreaker`, `New()`, `Execute()`, `State()`, `ErrCircuitOpen` | sony/gobreaker |
| `config` | YAML + env var configuration loading with validation | `Config`, `LoadConfig()`, all config structs (`DatabaseConfig`, `RedisConfig`, `JWTConfig`, etc.) | spf13/viper, joho/godotenv |
| `database` | PostgreSQL connection management with pooling | `PostgresClient`, `TracedPostgresClient`, `NewPostgresClient()`, `IsUniqueViolation()` | jmoiron/sqlx, jackc/pgx, `pkg/config`, `pkg/tracing` |
| `grpcclient` | gRPC client dialer with OTel, auth forwarding, and logging | `Dial()`, `ClientConfig` | grpc, otelgrpc, `pkg/tracing` |
| `grpcerror` | Maps `AppError` to gRPC status codes | `MapError()` | grpc/status, `pkg/apperror` |
| `grpcmiddleware` | gRPC server interceptor chain (auth, tracing, logging, rate limit, idempotency, recovery) | `Interceptors`, `NewInterceptors()` | `pkg/ratelimit`, `pkg/redis`, `pkg/tracing` |
| `grpcserver` | gRPC server lifecycle with OTel and graceful shutdown | `GRPCServer`, `New()`, `Start()`, `RegisterService()`, `GracefulStop()` | grpc, otelgrpc |
| `health` | Health check framework with Gin route registration | `Service`, `Checker` interface, `RegisterRoutes()`, `NewPostgresChecker()`, `NewRedisChecker()` | gin, `pkg/database`, `pkg/redis` |
| `idempotency` | Generic idempotency check helper (lookup-before-create) | `Check[T]()`, `Result[T]` | stdlib |
| `logger` | Structured logging with OTel log bridge and trace correlation | `NewLogger()`, `NewLoggerWithProvider()`, attribute helpers (`Err()`, `String()`, etc.) | slog, otelslog, `pkg/config` |
| `metrics` | OTel metric instruments (HTTP, gRPC, DB) with OTLP export | `Metrics`, `NewMetrics()`, `Meter()`, `RecordDBQuery()` | OTel SDK |
| `middleware` | HTTP middleware chain for the gateway (JWT, CORS, rate limit, tracing, recovery, security headers) | `Middleware`, `NewMiddleware()`, `JWTAuth()`, `TracingHandler()`, `CorsHandler()`, `RateLimiter()`, `RecoveryHandler()` | gin, `pkg/config`, `pkg/ratelimit`, `pkg/tracing`, `pkg/auth`, `pkg/response` |
| `nats` | NATS JetStream client with stream/consumer management and publishing | `Client`, `Publisher`, `NewClient()`, `NewPublisher()`, `CreateStreams()`, `StreamConfig`, `ConsumerConfig` | nats-io/nats.go |
| `ratelimit` | In-memory per-key token bucket rate limiter with auto-cleanup | `Store`, `NewStore()`, `Allow()`, `Config`, `DefaultConfig()` | golang.org/x/time/rate |
| `redis` | Redis client wrapper with tracing support | `Client`, `TracedRedisClient`, `NewClient()`, `NewTracedRedisClient()`, geo/set/hash operations | go-redis/v9, `pkg/config`, `pkg/tracing` |
| `redislock` | Distributed locking via Redis | `Locker`, `Lock`, `NewLocker()`, `Acquire()`, `Release()`, `ErrLockUnavailable` | bsm/redislock, `pkg/redis` |
| `response` | Standardized JSON response helpers for Gin | `Success()`, `Error()`, `ErrorWithRequestID()` | gin |
| `retry` | Generic retry-with-backoff helper | `Do()`, `Config`, `DefaultConfig()` | stdlib |
| `server` | HTTP server lifecycle and ordered shutdown manager | `GracefulServer`, `ShutdownManager`, `NewGracefulServer()`, `NewShutdownManager()` | gin |
| `telemetry` | Unified OTel provider initialization (traces + metrics + logs) | `Init()`, `Providers`, `Config` | OTel SDK (trace, metric, log exporters) |
| `tracing` | Tracer interface with OTel implementation and no-op fallback | `Tracer` interface, `NewTracer()`, `NewNoOpTracer()`, `HTTPTransaction` interface, `Config` | OTel SDK |

## Usage Examples

### config — Loading service configuration

```go
cfg, err := config.LoadConfig("reservation")
if err != nil {
    log.Fatalf("load config: %v", err)
}
// cfg.Database, cfg.Redis, cfg.JWT, cfg.GRPC, etc. are all populated
```

Configuration is loaded from `config/<APP_ENV>/<service>.yaml`. Secrets (DB password, JWT secret, Redis password) are injected via environment variables.

### telemetry — Unified observability initialization

```go
providers, err := telemetry.Init(ctx, telemetry.Config{
    ServiceName:     "reservation",
    OTLPEndpoint:    cfg.Tracing.OTLPEndpoint,
    TraceSampleRate: cfg.Tracing.SampleRate,
    MetricInterval:  15 * time.Second,
})
if err != nil {
    log.Fatalf("init telemetry: %v", err)
}
defer providers.Shutdown(ctx)
```

Returns `TracerProvider`, `MeterProvider`, and `LoggerProvider` — all exporting to a single OTLP endpoint.

### database — PostgreSQL with connection pooling

```go
db, err := database.NewPostgresClient(cfg.Database)
if err != nil {
    log.Fatalf("connect postgres: %v", err)
}
defer db.Close()

// Use with tracing
tracedDB := database.NewTracedPostgresClient(db, tracer)
sqlxDB := tracedDB.GetDB()
```

### nats — JetStream messaging

```go
nc, err := nats.NewClient(cfg.NATS.URL)
if err != nil {
    log.Fatalf("connect nats: %v", err)
}
defer nc.Close(ctx)

// Create streams
err = nats.CreateStreams(ctx, nc, []nats.StreamConfig{
    {Name: "RESERVATIONS", Subjects: []string{"reservation.>"}, MaxAge: 72 * time.Hour},
})

// Publish events
publisher := nats.NewPublisher(nc)
err = publisher.Publish(ctx, "reservation.created", payload, reservationID)
```

### redis — Caching and geo queries

```go
rc, err := redis.NewClient(cfg.Redis)
if err != nil {
    log.Fatalf("connect redis: %v", err)
}
defer rc.Close()

// With tracing
traced := redis.NewTracedRedisClient(rc, tracer)
err = traced.Set(ctx, "spot:123:status", "available", 5*time.Minute)
```

### grpcserver — Service server with middleware

```go
interceptors := grpcmiddleware.NewInterceptors(cfg.JWT.Secret, logger, tracer, redisClient)

srv := grpcserver.New(logger, cfg.GRPC.Server.Port, cfg.GRPC.Server.ShutdownTimeout,
    grpc.ChainUnaryInterceptor(
        interceptors.RecoveryUnary(),
        interceptors.TracingUnary(),
        interceptors.LoggingUnary(),
        interceptors.RateLimitUnary(cfg.GRPC.RateLimit.RequestsPerSecond, cfg.GRPC.RateLimit.BurstSize),
        interceptors.AuthUnary(),
    ),
)

pb.RegisterReservationServiceServer(srv.Server(), handler)
srv.Start()
```

### grpcmiddleware — Interceptor composition

```go
interceptors := grpcmiddleware.NewInterceptors(jwtSecret, logger, tracer, redisClient)

// Each interceptor is available as a unary/stream pair:
// - RecoveryUnary() / RecoveryStream()
// - TracingUnary() / TracingStream()
// - LoggingUnary() / LoggingStream()
// - RateLimitUnary() / RateLimitStream()
// - AuthUnary() / AuthStream()
// - IdempotencyUnary() (write RPCs only)
```

## Composition Patterns

### Full observability stack

The telemetry, logger, tracing, and metrics packages compose to provide complete observability:

```
telemetry.Init()
    ├── TracerProvider → tracing.NewTracer() → spans with context propagation
    ├── MeterProvider  → metrics.NewMetrics() → HTTP/gRPC/DB histograms & counters
    └── LoggerProvider → logger.NewLoggerWithProvider() → structured logs with trace correlation
```

Each service initializes this stack once at startup. The providers are passed down to infrastructure clients (`TracedPostgresClient`, `TracedRedisClient`) and middleware (`grpcmiddleware.Interceptors`, `middleware.Middleware`) for automatic instrumentation.

### Resilience composition

```
circuitbreaker.New() → wraps downstream gRPC calls
redislock.NewLocker() → prevents double-booking (uses pkg/redis)
retry.Do()           → retries transient failures with backoff
idempotency.Check()  → prevents duplicate creates at the usecase layer
```

### Server lifecycle

```
server.NewGracefulServer()   → HTTP server (gateway)
grpcserver.New()             → gRPC server (internal services)
server.NewShutdownManager()  → ordered cleanup of all resources
```

The `ShutdownManager` ensures resources are released in reverse-dependency order (server stops first, then DB/Redis/NATS connections close).
