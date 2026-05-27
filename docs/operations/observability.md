# Observability

## Architecture

ParkirPintar uses OpenTelemetry as the unified instrumentation layer, exporting traces, metrics, and logs via OTLP gRPC to a Grafana-based backend stack.

```
┌──────────────────────────────────────────────────────────────┐
│  Application Services                                        │
│  (gateway, reservation, billing, payment, search, presence,  │
│   analytics)                                                 │
│                                                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │ OTel Traces │  │ OTel Metrics│  │ OTel Logs   │         │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘         │
└─────────┼────────────────┼────────────────┼─────────────────┘
          │                │                │
          │    OTLP gRPC   │                │
          ▼                ▼                ▼
┌─────────────────────────────────────────────────────────────┐
│  Grafana Alloy (collector)                                   │
│  Receives OTLP → fans out to backends                        │
└────────┬─────────────────┬────────────────┬─────────────────┘
         │                 │                │
         ▼                 ▼                ▼
┌────────────┐   ┌──────────────┐   ┌────────────┐
│   Tempo    │   │  Prometheus  │   │    Loki    │
│  (traces)  │   │  (metrics)   │   │   (logs)   │
└────────────┘   └──────────────┘   └────────────┘
         │                 │                │
         └────────────┬────┘────────────────┘
                      ▼
              ┌──────────────┐
              │   Grafana    │
              │ (dashboards) │
              └──────────────┘
```

## Telemetry Initialization

Each service initializes the full OpenTelemetry pipeline via `pkg/telemetry.Init()`:

```go
providers, err := telemetry.Init(ctx, telemetry.Config{
    ServiceName:     "parkir-pintar-gateway",
    OTLPEndpoint:    "alloy:4317",       // gRPC endpoint (staging: alloy:4317, prod: alloy.pirescer-monitoring.svc.cluster.local:4317)
    TraceSampleRate: 1.0,                 // 100% in staging
    MetricInterval:  15 * time.Second,    // push interval
})
```

This creates three SDK providers:
- **TracerProvider** — batched span export with parent-based sampling
- **MeterProvider** — periodic reader pushing every 15 seconds
- **LoggerProvider** — batched log record export

If `OTLPEndpoint` is empty (local dev), all providers are created with no-op exporters — zero overhead, no network calls.

## Grafana Stack Components

| Component | Role | Port |
|-----------|------|------|
| **Alloy** | OpenTelemetry collector; receives OTLP, writes to backends | 4317 (gRPC), 4318 (HTTP) |
| **Tempo** | Distributed trace storage and query | 3200 |
| **Prometheus** | Metrics storage, recording rules, alerting rules | 9090 |
| **Loki** | Log aggregation and query | 3100 |
| **Grafana** | Unified dashboard UI | 3000 |
| **Alertmanager** | Alert routing (Slack, PagerDuty) | 9093 |

Infrastructure exporters (postgres-exporter, redis-exporter, nats-exporter) scrape data store metrics into Prometheus.

## Custom Metrics

### HTTP Metrics (Gateway)

Collected via Gin middleware (`pkg/metrics.HTTPMiddleware()`):

| Metric | Type | Labels |
|--------|------|--------|
| `http_requests_total` | Counter | method, path, status_code |
| `http_request_duration_seconds` | Histogram | method, path, status_code |
| `http_response_size_bytes` | Histogram | method, path |

Path normalization replaces UUIDs and numeric IDs with `:id` to prevent cardinality explosion.

### gRPC Metrics (All Backend Services)

Collected via unary server interceptor (`pkg/metrics.GRPCUnaryInterceptor()`):

| Metric | Type | Labels |
|--------|------|--------|
| `grpc_requests_total` | Counter | method, grpc_code |
| `grpc_request_duration_seconds` | Histogram | method, grpc_code |

### Database Metrics

| Metric | Type | Labels |
|--------|------|--------|
| `db_query_duration_seconds` | Histogram | db.operation, db.sql.table |

Recorded via `Metrics.RecordDBQuery()` in repository implementations.

### Business Metrics

Domain-specific packages use `Metrics.Meter()` to register additional instruments on the shared meter provider. This ensures all metrics flow through the same OTLP pipeline without requiring separate exporters.

## Distributed Tracing

### Tracer Abstraction

The `pkg/tracing.Tracer` interface provides span creation for common operations:

| Method | Span Kind | Use Case |
|--------|-----------|----------|
| `StartHTTPRequest` | Server | Incoming HTTP requests |
| `StartExternalCall` | Client | Outbound HTTP calls |
| `StartMessage` | Producer/Consumer | NATS publish/subscribe |
| `StartDatabase` | Client | SQL queries |
| `StartSegment` | Internal | Business logic spans |

Context propagation uses W3C TraceContext + Baggage via `InjectContext` / `ExtractContext`.

### Exporter Options

Configured via `tracing.exporter` in YAML:

| Value | Transport | Use Case |
|-------|-----------|----------|
| `noop` | None | Local development (default) |
| `stdout` | Stdout | Debugging trace output |
| `otlp` | HTTP | Alternative collector endpoint |
| `otlp-grpc` | gRPC | Production (Alloy at port 4317) |

### Path Exclusion

Health check paths (`/health`, `/health/live`, `/health/ready`) are excluded from tracing to reduce noise. Configured per-service in YAML under `tracing.exclude_paths`.

### Sampling

Parent-based sampling with configurable ratio:
- **Staging:** 1.0 (100% — full visibility for debugging)
- **Production:** Tunable via `tracing.sample_rate` (recommended 0.1–0.5 depending on traffic)

## Structured Logging

### Implementation

Built on Go's `log/slog` with two output paths:

1. **JSON to stdout** — collected by container runtime, forwarded to Loki
2. **OTLP bridge** — logs sent directly to Alloy via OpenTelemetry Log SDK

The dual-output fanout handler ensures logs are available both in container logs (for `docker logs`) and in Loki (for correlation with traces).

### Trace Correlation

Every log record automatically includes `trace_id` and `span_id` when emitted within an active span context. This enables clicking from a log line in Grafana directly to the corresponding trace in Tempo.

```json
{
  "time": "2025-01-15T10:30:00Z",
  "level": "ERROR",
  "source": {"function": "CreateReservation", "file": "handler.go", "line": 42},
  "msg": "payment timeout exceeded",
  "trace_id": "abc123...",
  "span_id": "def456...",
  "reservation_id": "uuid-here"
}
```

### Log Levels

| Level | Usage |
|-------|-------|
| `debug` | Detailed flow information (local/staging only) |
| `info` | Normal operations, request lifecycle |
| `warn` | Recoverable issues, degraded performance |
| `error` | Failed operations requiring attention |

Production services run at `info` level. Debug logging is enabled per-service via YAML config.

## Adding New Instrumentation

### New Metric

```go
// In your domain package:
func NewMyMetrics(m *metrics.Metrics) (*MyMetrics, error) {
    meter := m.Meter()
    counter, err := meter.Int64Counter("my_domain_events_total",
        otelmetric.WithDescription("Total domain events processed"),
    )
    if err != nil {
        return nil, err
    }
    return &MyMetrics{events: counter}, nil
}
```

### New Trace Span

```go
// In your handler/service:
ctx, end := tracer.StartDatabase(ctx, "SELECT", "reservations")
defer end()
// ... do database work
```

### New Structured Log Field

```go
logger.InfoContext(ctx, "reservation created",
    slog.String("reservation_id", id),
    slog.Int("slot_count", count),
    slog.Duration("processing_time", elapsed),
)
// trace_id and span_id are injected automatically from ctx
```

## Related Documentation

- [SLO/SLI Definitions](./slo-sli.md) — service level objectives, alerting rules, and Prometheus recording rules
- [Runtime Profiling (pprof)](./pprof.md) — CPU, memory, and goroutine profiling for performance diagnosis
