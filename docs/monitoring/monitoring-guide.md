# ParkirPintar Monitoring Operations Guide

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         ParkirPintar Services                            │
├──────────┬──────────┬──────────┬──────────┬──────────┬────────┬────────┤
│ Gateway  │Reservation│ Payment │ Billing  │ Presence │Notific.│ Search │
│ :8080    │  :8081   │  :8082  │  :8083   │  :8084   │ :8085  │ :8086  │
└────┬─────┴────┬─────┴────┬────┴────┬─────┴────┬─────┴───┬────┴───┬────┘
     │          │          │         │          │         │        │
     └──────────┴──────────┴────┬────┴──────────┴─────────┴────────┘
                                │
                    OTel SDK (traces + metrics + logs)
                    OTLP gRPC push
                                │
                                ▼
                 ┌──────────────────────────────┐
                 │   Grafana Alloy (:4319)      │
                 │   - OTLP receiver            │
                 │   - Batch processor          │
                 │   - Span metrics connector   │
                 └──────┬───────┬───────┬───────┘
                        │       │       │
              ┌─────────┘       │       └─────────┐
              ▼                 ▼                  ▼
   ┌──────────────────┐ ┌─────────────┐ ┌──────────────────┐
   │ Prometheus (:9090)│ │ Tempo (:3200)│ │  Loki (:3100)    │
   │ - Metrics TSDB   │ │ - Trace store│ │  - Log store     │
   │ - Alert rules    │ │ - 30d retain │ │  - 7d retention  │
   │ - 30d retention  │ └──────┬───────┘ └────────┬─────────┘
   └────────┬─────────┘        │                   │
            │                  │                   │
            ▼                  ▼                   ▼
   ┌──────────────────────────────────────────────────────────┐
   │              Grafana (:3000)                              │
   │  - Dashboards (provisioned)                              │
   │  - Explore (Prometheus, Tempo, Loki)                     │
   │  - Alerting (via Prometheus → Alertmanager)              │
   └──────────────────────────────────────────────────────────┘
            │
            ▼
   ┌──────────────────┐
   │ Alertmanager     │
   │ (:9093)          │
   │ - Telegram notif │
   └──────────────────┘
```

All telemetry is **push-based** via OTLP gRPC. No Prometheus scraping or Docker log collection needed.

## Accessing Dashboards

All monitoring UIs are accessible via Tailscale at the server IP `100.79.123.39`:

| Service       | URL                          | Credentials              |
|---------------|------------------------------|--------------------------|
| Grafana       | `http://100.79.123.39:3000`  | admin / parkir-pintar-2026 |
| Prometheus    | `http://100.79.123.39:9090`  | —                        |
| Loki          | `http://100.79.123.39:3100`  | —                        |
| Tempo         | `http://100.79.123.39:3200`  | —                        |
| Alloy UI      | `http://100.79.123.39:12345` | —                        |
| Alertmanager  | `http://100.79.123.39:9093`  | —                        |

### Quick Start

1. Connect to Tailscale VPN
2. Open Grafana at `http://100.79.123.39:3000`
3. Navigate to **Dashboards → ParkirPintar - Overview** for system health
4. Use **Explore** for ad-hoc queries against Prometheus, Tempo, or Loki

## Key Metrics to Watch

### Service Health (RED)

| Metric | PromQL | Healthy Threshold |
|--------|--------|-------------------|
| Request Rate | `sum(rate(traces_spanmetrics_calls_total{span_kind="SPAN_KIND_SERVER"}[5m])) by (service_name)` | > 0 (traffic flowing) |
| Error Rate | `sum(rate(traces_spanmetrics_calls_total{status_code="STATUS_CODE_ERROR"}[5m])) by (service_name) / sum(rate(traces_spanmetrics_calls_total[5m])) by (service_name)` | < 5% |
| Latency P95 | `histogram_quantile(0.95, sum(rate(traces_spanmetrics_duration_milliseconds_bucket[5m])) by (le, service_name))` | < 500ms |

### Infrastructure

| Metric | PromQL | Watch For |
|--------|--------|-----------|
| DB Connection Pool | `db_pool_active_connections` | Approaching `db_pool_max_connections` |
| DB Query Latency | `histogram_quantile(0.95, sum(rate(db_query_duration_seconds_bucket[5m])) by (le, job))` | > 200ms |
| NATS Consumer Lag | `rate(nats_messages_published_total[5m]) - rate(nats_messages_consumed_total[5m])` | > 100 msg/s |
| Redis Memory | `redis_memory_used_bytes` | Approaching `redis_memory_max_bytes` |
| Process Memory | `process_resident_memory_bytes{job=~"parkir-pintar.*"}` | Steady growth (leak) |
| Goroutines | `go_goroutines{job=~"parkir-pintar.*"}` | Unbounded growth |

### Business Metrics

| Metric | PromQL | Notes |
|--------|--------|-------|
| Active Reservations | `sum(active_reservations)` | Gauge of in-progress reservations |
| HTTP 4xx Rate | `sum(rate(http_requests_total{status_code=~"4.."}[5m])) by (job)` | Client errors, may indicate API misuse |
| HTTP 5xx Rate | `sum(rate(http_requests_total{status_code=~"5.."}[5m])) by (job)` | Server errors, needs investigation |

## Alert Response Procedures

Active alerts are routed through Prometheus → Alertmanager → Telegram.

1. **Check Grafana Alerting** — `http://100.79.123.39:3000/alerting/list`
2. **Identify the alert** — see [Alerting Playbook](./alerting-playbook.md) for per-alert response
3. **Investigate** — use Explore with correlated traces/logs
4. **Resolve or escalate** — follow runbook steps
5. **Silence if needed** — Alertmanager UI at `:9093/#/silences`

## Adding New Metrics / Traces / Logs

### Adding a New Metric

1. In your service code, use the OTel Meter from `pkg/telemetry`:

```go
meter := otel.Meter("parkir-pintar-myservice")
counter, _ := meter.Int64Counter("my_custom_counter",
    metric.WithDescription("Description of what this counts"),
    metric.WithUnit("{count}"),
)
counter.Add(ctx, 1, metric.WithAttributes(
    attribute.String("key", "value"),
))
```

2. Metrics are automatically pushed via OTLP → Alloy → Prometheus
3. Query in Grafana with: `my_custom_counter`

### Adding a New Trace Span

```go
tracer := otel.Tracer("parkir-pintar-myservice")
ctx, span := tracer.Start(ctx, "operation-name",
    trace.WithAttributes(attribute.String("key", "value")),
)
defer span.End()
```

### Adding Structured Logs

```go
slog.InfoContext(ctx, "operation completed",
    "user_id", userID,
    "duration_ms", elapsed.Milliseconds(),
)
```

Logs are automatically correlated with the active trace via `trace_id` and `span_id`.

### Metric Naming Conventions

- Use snake_case: `reservation_created_total`, `db_query_duration_seconds`
- Counters end with `_total`
- Histograms end with `_seconds` or `_bytes` (unit suffix)
- Gauges use present tense: `active_reservations`, `db_pool_active_connections`

## Creating Custom Dashboards

### From Grafana UI

1. Go to **Dashboards → New → New Dashboard**
2. Add panels using the Prometheus datasource (UID: `prometheus`)
3. Use Tempo datasource for trace panels
4. Use Loki datasource for log panels
5. Save and optionally export as JSON

### From JSON (Provisioned)

1. Create a JSON file in `deploy/monitoring/grafana/provisioning/dashboards/`
2. Follow the schema of existing dashboards (see `overview.json`)
3. Set a unique `uid` field
4. Restart Grafana or wait for auto-reload:

```bash
cd deploy/monitoring
docker compose -f docker-compose.monitoring.yml restart grafana
```

### Dashboard Best Practices

- Use template variables for service filtering
- Include links between dashboards (overview → detail)
- Set meaningful thresholds on panels
- Use `$__rate_interval` instead of hardcoded `[5m]` for rate queries
- Add panel descriptions for on-call context

## Troubleshooting Monitoring Stack

### No Metrics Appearing

1. **Check Alloy is receiving data:**
   ```bash
   curl http://100.79.123.39:12345/metrics | grep otelcol_receiver_accepted
   ```
   If `otelcol_receiver_accepted_spans` is 0, services aren't sending telemetry.

2. **Check service OTLP config:**
   ```bash
   docker logs <service-container> 2>&1 | grep -i "otlp\|telemetry\|tracing"
   ```
   Ensure `TRACING_ENABLED=true` and `TRACING_OTLP_ENDPOINT=monitoring-alloy:4319`.

3. **Check Alloy → Prometheus connectivity:**
   ```bash
   docker logs monitoring-alloy 2>&1 | grep -i "error\|failed"
   ```

4. **Check Prometheus is receiving remote writes:**
   ```bash
   curl http://100.79.123.39:9090/api/v1/status/tsdb | jq .data.headStats
   ```

### No Traces in Tempo

1. Verify Alloy is forwarding to Tempo:
   ```bash
   docker logs monitoring-alloy 2>&1 | grep tempo
   ```

2. Check Tempo health:
   ```bash
   curl http://100.79.123.39:3200/ready
   ```

3. Query Tempo directly:
   ```bash
   curl "http://100.79.123.39:3200/api/search?q={}" | jq .
   ```

### No Logs in Loki

1. Check Loki health:
   ```bash
   curl http://100.79.123.39:3100/ready
   ```

2. Verify labels exist:
   ```bash
   curl http://100.79.123.39:3100/loki/api/v1/labels | jq .
   ```

3. Check Alloy log pipeline:
   ```bash
   curl http://100.79.123.39:12345/metrics | grep loki_write
   ```

### Grafana Datasource Errors

1. Go to **Configuration → Data Sources** in Grafana
2. Click each datasource and hit **Test**
3. Verify container networking:
   ```bash
   docker exec monitoring-grafana wget -qO- http://prometheus:9090/-/healthy
   docker exec monitoring-grafana wget -qO- http://tempo:3200/ready
   docker exec monitoring-grafana wget -qO- http://loki:3100/ready
   ```

### High Memory / Disk Usage

- **Prometheus:** Check retention (`--storage.tsdb.retention.time=30d`). Reduce if disk is full.
- **Loki:** Default 7d retention. Check `/loki/data` volume.
- **Tempo:** Check `/var/tempo` volume. Traces are large; consider reducing retention.

## Log Query Examples (LogQL)

```logql
# All logs from ParkirPintar services
{service_name=~"parkir-pintar-.+"}

# Gateway errors only
{service_name="parkir-pintar-gateway"} | json | severity="ERROR"

# Reservation service warnings and errors
{service_name="parkir-pintar-reservation"} | json | severity=~"WARN|ERROR"

# Logs containing a specific trace ID
{service_name=~"parkir-pintar-.+"} |= "abc123def456"

# Payment failures with structured field extraction
{service_name="parkir-pintar-payment"} | json | status="failed" | line_format "{{.user_id}} - {{.error}}"

# Rate of error logs per service (metric query)
sum(rate({service_name=~"parkir-pintar-.+"} | json | severity="ERROR" [5m])) by (service_name)

# Logs around a specific time window with context
{service_name="parkir-pintar-billing"} | json | ts >= "2026-01-15T10:00:00Z" and ts <= "2026-01-15T10:05:00Z"

# Search for slow database queries in logs
{service_name=~"parkir-pintar-.+"} | json | msg=~".*slow.*query.*"

# Count unique users hitting errors
sum(count_over_time({service_name="parkir-pintar-gateway"} | json | severity="ERROR" | keep user_id [1h])) by (user_id)
```

## Trace Query Examples (TraceQL)

```traceql
# Find traces with errors
{ status = error }

# Gateway traces taking more than 1 second
{ resource.service.name = "parkir-pintar-gateway" && duration > 1s }

# Reservation creation spans
{ resource.service.name = "parkir-pintar-reservation" && name = "CreateReservation" }

# All spans for a specific HTTP endpoint
{ span.http.route = "/api/v1/reservations" }

# gRPC calls with non-OK status
{ span.rpc.grpc.status_code != 0 }

# Find slow database queries
{ name =~ "db.*" && duration > 200ms }

# Traces involving payment service with errors
{ resource.service.name = "parkir-pintar-payment" && status = error }

# Cross-service traces (gateway → reservation)
{ resource.service.name = "parkir-pintar-gateway" } >> { resource.service.name = "parkir-pintar-reservation" }

# Find traces where notification delivery failed
{ resource.service.name = "parkir-pintar-notification" && span.notification.status = "failed" }

# Traces with high fan-out (many child spans)
{ resource.service.name = "parkir-pintar-search" && duration > 500ms } | count() > 10
```

## Useful Links

- [Grafana Dashboards](http://100.79.123.39:3000/dashboards)
- [Prometheus Targets](http://100.79.123.39:9090/targets)
- [Alertmanager](http://100.79.123.39:9093)
- [Alloy Pipeline UI](http://100.79.123.39:12345)
- [Alerting Playbook](./alerting-playbook.md)
