# Observability Stack

## Architecture

```
Services → Alloy (span metrics) → Tempo (traces) + Prometheus (metrics) → Grafana (visualization)
```

Each service exports OpenTelemetry traces via OTLP gRPC to Grafana Alloy. Alloy processes spans to generate RED metrics (Rate, Errors, Duration), then forwards:

- **Traces** → Tempo for storage and querying
- **Span metrics** → Prometheus for alerting and dashboards
- **Visualization** → Grafana connects to both Tempo and Prometheus

## Configuring Services

Set the following environment variables for each service:

```env
TRACING_ENABLED=true
TRACING_EXPORTER=otlp-grpc
TRACING_OTLP_ENDPOINT=monitoring-alloy:4319
```

Within the Docker `coolify` network, Alloy is reachable at `monitoring-alloy:4319` (gRPC OTLP receiver).

### Exporter Options

| Value       | Protocol | Use Case                          |
|-------------|----------|-----------------------------------|
| `otlp-grpc` | gRPC     | Alloy / any gRPC OTLP collector   |
| `otlp`      | HTTP     | Any HTTP OTLP collector           |
| `newrelic`  | HTTP     | New Relic OTLP endpoint           |
| `stdout`    | —        | Local development (pretty-print)  |
| `noop`      | —        | Tracing disabled                  |

## Metrics HTTP Ports

Each service exposes a `/metrics` endpoint for Prometheus scraping:

| Service       | Metrics Port |
|---------------|-------------|
| gateway       | 8082        |
| reservation   | 8091        |
| search        | 8092        |
| billing       | 8093        |
| payment       | 8094        |
| presence      | 8095        |
| notification  | 8096        |

Alloy scrapes these ports to collect application-level metrics alongside the span-derived metrics.

## Network Topology (Docker)

All services and monitoring components run on the `coolify` Docker network:

- **monitoring-alloy:4319** — OTLP gRPC receiver (traces from services)
- **monitoring-alloy:12345** — Alloy UI
- **monitoring-tempo:3200** — Tempo query API
- **monitoring-prometheus:9090** — Prometheus query API
- **monitoring-grafana:3000** — Grafana dashboards
