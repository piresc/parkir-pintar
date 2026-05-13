# Observability Stack

## Architecture

```
Go Services (OTel SDK)
    ├── traces  ─┐
    ├── metrics ─┼── OTLP gRPC :4319 → Alloy
    └── logs    ─┘
                      ├── traces      → Tempo (storage + query)
                      ├── metrics     → Prometheus (remote_write)
                      ├── span metrics → Prometheus (RED from traces)
                      └── logs        → Loki (storage + query)

Grafana → reads from Tempo + Prometheus + Loki
```

All three signals (traces, metrics, logs) are exported via a single OTLP gRPC connection per service. No Prometheus scraping, no Docker socket log collection — pure push-based telemetry.

## Signals

### Traces
- OTel SDK with `otlp-grpc` exporter → Alloy → Tempo
- W3C TraceContext propagation (traceparent header)
- `otelgrpc` stats handlers for automatic gRPC span creation
- Gateway adds `X-Trace-Id` response header for client correlation

### Metrics
- OTel SDK with OTLP periodic reader (30s interval) → Alloy → Prometheus
- Instruments: HTTP requests, gRPC requests, DB queries, NATS messages, business gauges
- Span metrics (RED): auto-generated from traces by Alloy's spanmetrics connector

### Logs
- `slog` with OTel log bridge (`otelslog`) → OTLP → Alloy → Loki
- Dual output: stdout (JSON) + OTLP (structured)
- Auto-correlated with traces via `trace_id` and `span_id` fields
- Clickable trace links in Grafana (log → trace, trace → log)

## Configuring Services

Set the following environment variables:

```env
TRACING_ENABLED=true
TRACING_EXPORTER=otlp-grpc
TRACING_OTLP_ENDPOINT=monitoring-alloy:4319
```

The `pkg/telemetry` package initializes all three providers (Tracer, Meter, Logger) with a single OTLP endpoint.

### Exporter Options

| Value       | Protocol | Use Case                          |
|-------------|----------|-----------------------------------|
| `otlp-grpc` | gRPC     | Alloy / any gRPC OTLP collector   |
| `otlp`      | HTTP     | Any HTTP OTLP collector           |
| `newrelic`  | HTTP     | New Relic OTLP endpoint           |
| `stdout`    | —        | Local development (pretty-print)  |
| `noop`      | —        | Telemetry disabled                |

## Components

| Component    | Container              | Port  | Purpose                              |
|-------------|------------------------|-------|--------------------------------------|
| Alloy       | monitoring-alloy       | 4319  | OTLP receiver, span metrics, routing |
| Alloy UI    | monitoring-alloy       | 12345 | Pipeline health dashboard            |
| Tempo       | monitoring-tempo       | 3200  | Trace storage and query              |
| Prometheus  | monitoring-prometheus  | 9090  | Metrics storage (TSDB, 30d retention)|
| Loki        | monitoring-loki        | 3100  | Log storage (7d retention)           |
| Grafana     | monitoring-grafana     | 3000  | Visualization and exploration        |
| Alertmanager| monitoring-alertmanager| 9093  | Alert routing and silencing          |

## Access (via Tailscale)

| Dashboard    | URL                          | Credentials          |
|-------------|------------------------------|----------------------|
| Grafana     | `100.79.123.39:3000`         | admin / parkir-pintar-2026 |
| Prometheus  | `100.79.123.39:9090`         | —                    |
| Loki        | `100.79.123.39:3100`         | —                    |
| Alloy       | `100.79.123.39:12345`        | —                    |
| Alertmanager| `100.79.123.39:9093`         | —                    |

## Grafana Datasources

| Name       | Type       | UID          | Correlations                    |
|-----------|------------|--------------|----------------------------------|
| Prometheus | prometheus | `prometheus` | Default, metrics queries         |
| Tempo      | tempo      | `tempo`      | Trace → Logs (Loki), Trace → Metrics |
| Loki       | loki       | `loki`       | Log → Trace (derived field on trace_id) |

## Querying Logs (Loki)

```logql
# All ParkirPintar services
{service_name=~"parkir-pintar-.+"}

# Gateway errors
{service_name="parkir-pintar-gateway"} | json | severity="ERROR"

# Specific trace
{service_name=~"parkir-pintar-.+"} |= "abc123traceId"
```

## Provisioned Dashboards

- **ParkirPintar - Service Overview**: Request rate, error rate, latency P50/P95/P99 by service
- **ParkirPintar - Service Detail**: Per-endpoint breakdown, duration heatmap, infrastructure metrics

## Network Topology (Docker)

All services and monitoring components share the `coolify` network:

- **monitoring-alloy:4319** — OTLP gRPC receiver (traces + metrics + logs)
- **monitoring-alloy:12345** — Alloy UI
- **monitoring-tempo:3200** — Tempo query API
- **monitoring-tempo:4317** — Tempo OTLP ingest (from Alloy)
- **monitoring-prometheus:9090** — Prometheus query + remote_write API
- **monitoring-loki:3100** — Loki push + query API
- **monitoring-grafana:3000** — Grafana dashboards
- **monitoring-alertmanager:9093** — Alert routing

## Running

```bash
cd deploy/monitoring
docker compose -f docker-compose.monitoring.yml up -d
```
