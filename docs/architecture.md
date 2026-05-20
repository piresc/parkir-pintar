# Architecture

## Overview

ParkirPintar uses a microservices architecture with 7 Go services communicating via gRPC over HTTP/2. The system follows clean architecture principles within each service.

## Clean Architecture Layers (per service)

```
gRPC Request → Handler → Usecase → Repository → Database
```

- **Handler**: Binds gRPC input, delegates to usecase, returns protobuf responses.
- **Usecase**: Business logic. Depends on repository interface, not concrete implementation.
- **Repository**: Data access via sqlx with parameterized queries.
- **Model**: Domain structs, request/response types, sentinel errors.

## Service Communication

### Synchronous (gRPC over HTTP/2)

```
Client → Gateway (REST) → [Search | Reservation | Payment | Presence] (gRPC)
Reservation → Billing (gRPC)
Reservation → Payment (gRPC)
```

### Asynchronous (NATS JetStream)

```
Reservation → NATS → Search (spot status sync)
Reservation → NATS → Analytics (lifecycle events)
Payment → NATS → Reservation (payment results)
```

## Gateway Middleware Chain (order matters)

1. `RecoveryHandler` — catches panics, returns 500
2. `CorsHandler` — explicit origins only (no wildcard + credentials)
3. `RateLimiter` — per-IP token bucket rate limiting
4. `TracingHandler` — OTEL span creation
5. `JWTAuth` — JWT token validation (on API routes)

## gRPC Middleware Chain (internal services)

1. `Recovery` — catches panics
2. `Tracing` — OTEL span propagation
3. `Logging` — structured request/response logging
4. `RateLimit` — per-method rate limiting
5. `Auth` — service-to-service authentication
6. `Idempotency` — atomic SETNX deduplication (on write RPCs)

## Infrastructure

- **PostgreSQL** via sqlx with connection pooling (max 25 conns)
- **Redis** via go-redis with pool size configuration (10 conns)
- **NATS JetStream** for inter-service messaging with auto-reconnect
- **OpenTelemetry** for all observability signals (traces, metrics, logs) via OTLP gRPC

## Observability (Full OTel)

All three signals exported via a single OTLP gRPC connection per service:

```
Go Service (OTel SDK)
    ├── traces  → Alloy → Tempo
    ├── metrics → Alloy → Prometheus
    └── logs    → Alloy → Loki
                           ↓
                        Grafana (unified view)
```

| Signal  | SDK                          | Exporter         | Backend    |
|---------|------------------------------|------------------|------------|
| Traces  | `go.opentelemetry.io/otel`   | otlptracegrpc    | Tempo      |
| Metrics | OTel SDK + periodic reader   | otlpmetricgrpc   | Prometheus |
| Logs    | slog + otelslog bridge       | otlploggrpc      | Loki       |

Key packages:
- `pkg/telemetry` — unified init (TracerProvider + MeterProvider + LoggerProvider)
- `pkg/tracing` — tracer abstraction with span helpers
- `pkg/metrics` — OTel metric instruments (HTTP, gRPC, DB, NATS, business)
- `pkg/logger` — slog with dual output (stdout + OTLP) and trace correlation

See `deploy/coolify/README.md` for full stack details.

## Error Handling

Sentinel errors are defined at the domain level (`model/errors.go`) and at the
application level (`pkg/apperror/`). Use `errors.Is()` for checking and
`fmt.Errorf("%w", err)` for wrapping.

## Health Checks

- `GET /health` — build info
- `GET /health/live` — liveness (always 200)
- `GET /health/ready` — readiness (checks Postgres, Redis, NATS)
- `GET /health/detailed` — per-dependency status with durations

## Resilience Patterns

| Pattern | Implementation | Purpose |
|---------|---------------|---------|
| Circuit Breaker | `pkg/circuitbreaker/` | Fail fast when downstream is unhealthy |
| Retry with Backoff | `pkg/httpclient/` | Handle transient failures |
| Distributed Lock | `pkg/redislock/` | Prevent double-booking |
| Idempotency | `pkg/grpcmiddleware/idempotency.go` | Prevent duplicate operations |
| Graceful Degradation | Non-critical calls logged, not failed | Core flows survive non-core failures |
| Singleflight | `internal/search/usecase/` | Prevent cache stampedes |
| Context-Aware Retry | `internal/payment/usecase/` | Respect cancellation during retries |

## Data Flow: Reservation Lifecycle

```
1. CreateReservation
   → Redis lock → DB check → DB insert (transaction) → Billing.StartBilling → NATS event

2. CheckIn
   → DB lock → DB update (transaction) → Billing notification → NATS event

3. CheckOut
   → DB lock → DB update (transaction) → Billing.CalculateFee → Billing.GenerateInvoice → Payment.ProcessPayment → NATS event

4. ExpireReservation (background worker, every 30s)
   → DB scan → DB update (transaction) → NATS event
   → Booking fee (already charged) is the only cost forfeited

5. CancelReservation
   → DB lock → DB update (transaction) → NATS event
   → Booking fee (already charged) is non-refundable
```
