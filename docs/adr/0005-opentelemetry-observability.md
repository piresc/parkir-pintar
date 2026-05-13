# 5. OpenTelemetry Pipeline for Observability

## Status

Accepted

## Context

With a microservices architecture, observability is critical. A single user request (e.g., "find and reserve a spot") traverses multiple services (search → presence → reservation → billing). Without distributed tracing, debugging latency issues or failures across service boundaries is extremely difficult.

Requirements:
- Distributed tracing across all services (correlate requests end-to-end)
- Metrics collection (latency histograms, error rates, saturation)
- Structured logging with trace context propagation
- Vendor-neutral: avoid lock-in to a specific observability vendor
- Self-hosted option for cost control

Alternatives considered:

1. **Vendor-specific SDKs (Datadog, New Relic)** — excellent UX and managed infrastructure, but expensive at scale, vendor lock-in, per-host/per-GB pricing
2. **OpenTelemetry + Grafana stack** — vendor-neutral, full control over data pipeline, open-source backends (Tempo, Mimir, Loki), community-standard instrumentation

## Decision

We will use the **OpenTelemetry** standard for all observability signals (traces, metrics, logs) with **Grafana Alloy** as the collector/agent.

Stack:
- **Instrumentation**: OpenTelemetry SDK (Go) in each service
- **Collection**: Grafana Alloy (OpenTelemetry-compatible collector) deployed as a sidecar or DaemonSet
- **Backends**: Grafana Tempo (traces), Grafana Mimir (metrics), Grafana Loki (logs)
- **Visualization**: Grafana dashboards with trace-to-log and trace-to-metric correlation

All services will propagate W3C Trace Context headers. gRPC interceptors and NATS middleware will automatically inject/extract trace context.

## Consequences

### Positive

- Vendor-neutral: can switch backends (e.g., to Jaeger, or a commercial vendor) without re-instrumenting services
- Full control: own the data pipeline, no per-seat or per-GB vendor costs
- Correlation: traces, metrics, and logs linked by trace ID for fast debugging
- Community standard: OpenTelemetry is the CNCF standard, broad library support
- Cost predictable: infrastructure cost scales with compute, not data volume pricing

### Negative

- Self-managed infrastructure: must operate Tempo, Mimir, Loki, and Alloy (mitigated by Grafana Cloud as a fallback)
- Setup complexity: initial configuration of collectors, pipelines, and dashboards requires effort
- Storage management: trace and metric retention policies must be actively managed
- Learning curve: team needs familiarity with OTel SDK, collector configuration, and PromQL/LogQL
- No built-in alerting UX: requires Grafana Alerting configuration (less polished than Datadog/PagerDuty native)
