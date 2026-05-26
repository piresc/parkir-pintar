# SLO/SLI Definitions — ParkirPintar

## Service Level Indicators (SLIs)

### Availability SLI
- **Definition:** Proportion of successful HTTP responses (non-5xx) from the gateway.
- **Measurement:** `1 - (http_requests_total{status=~"5.."} / http_requests_total)`
- **Data source:** Prometheus via Alloy span metrics.

### Latency SLI
- **Definition:** Proportion of requests served within the latency threshold.
- **Measurement:** `histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))`
- **Data source:** Prometheus histogram from OTel instrumentation.

### Error Rate SLI
- **Definition:** Proportion of gRPC calls that return OK status.
- **Measurement:** `1 - (grpc_server_handled_total{grpc_code!="OK"} / grpc_server_handled_total)`
- **Data source:** Prometheus via gRPC middleware metrics.

---

## Service Level Objectives (SLOs)

| Service      | SLI              | Target | Window  | Error Budget |
|--------------|------------------|--------|---------|--------------|
| Gateway      | Availability     | 99.5%  | 30 days | 3.6 hours    |
| Gateway      | Latency (p95)    | < 500ms| 30 days | —            |
| Gateway      | Latency (p99)    | < 2s   | 30 days | —            |
| Search       | Latency (p95)    | < 100ms| 30 days | —            |
| Search       | Cache Hit Rate   | > 90%  | 30 days | —            |
| Reservation  | Availability     | 99.9%  | 30 days | 43 minutes   |
| Reservation  | Latency (p95)    | < 300ms| 30 days | —            |
| Billing      | Availability     | 99.9%  | 30 days | 43 minutes   |
| Payment      | Availability     | 99.95% | 30 days | 21 minutes   |
| Payment      | Latency (p95)    | < 1s   | 30 days | —            |

---

## Alerting Rules

### Critical (PagerDuty / immediate)

```yaml
# Error budget burn rate > 14.4x (2% budget consumed in 1 hour)
- alert: HighErrorBudgetBurn
  expr: |
    (
      sum(rate(http_requests_total{status=~"5.."}[1h]))
      / sum(rate(http_requests_total[1h]))
    ) > (14.4 * 0.005)
  for: 2m
  labels:
    severity: critical
  annotations:
    summary: "Error budget burning too fast ({{ $value | humanizePercentage }})"

# Service completely down
- alert: ServiceDown
  expr: up{job=~"parkir-.*"} == 0
  for: 1m
  labels:
    severity: critical
```

### Warning (Slack / business hours)

```yaml
# Latency SLO breach
- alert: HighP95Latency
  expr: |
    histogram_quantile(0.95,
      sum(rate(http_request_duration_seconds_bucket{job="parkir-gateway"}[5m])) by (le)
    ) > 0.5
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "Gateway p95 latency above 500ms ({{ $value | humanizeDuration }})"

# Cache hit rate dropping
- alert: LowCacheHitRate
  expr: |
    (
      sum(rate(redis_cache_hits_total{service="search"}[5m]))
      / (sum(rate(redis_cache_hits_total{service="search"}[5m])) + sum(rate(redis_cache_misses_total{service="search"}[5m])))
    ) < 0.9
  for: 10m
  labels:
    severity: warning
  annotations:
    summary: "Search cache hit rate below 90% ({{ $value | humanizePercentage }})"
```

---

## Prometheus Recording Rules

```yaml
groups:
  - name: parkir_pintar_slo
    interval: 30s
    rules:
      # Gateway availability (rolling 30d)
      - record: slo:gateway_availability:ratio_rate5m
        expr: |
          1 - (
            sum(rate(http_requests_total{job="parkir-gateway",status=~"5.."}[5m]))
            / sum(rate(http_requests_total{job="parkir-gateway"}[5m]))
          )

      # Reservation service error rate
      - record: slo:reservation_error_rate:ratio_rate5m
        expr: |
          sum(rate(grpc_server_handled_total{grpc_service="reservation.ReservationService",grpc_code!="OK"}[5m]))
          / sum(rate(grpc_server_handled_total{grpc_service="reservation.ReservationService"}[5m]))

      # Search latency p95
      - record: slo:search_latency_p95:seconds
        expr: |
          histogram_quantile(0.95,
            sum(rate(http_request_duration_seconds_bucket{job="parkir-search"}[5m])) by (le)
          )
```

---

## Grafana Dashboard Panels

The SLO dashboard should include:
1. **Error Budget Remaining** — gauge showing % of 30-day budget left.
2. **Availability over time** — time series of `slo:gateway_availability:ratio_rate5m`.
3. **Latency heatmap** — request duration distribution by service.
4. **Burn rate** — multi-window burn rate (1h, 6h, 3d) for alerting context.
5. **Cache hit rate** — search service Redis hit/miss ratio.

---

## Review Cadence

- **Weekly:** Review error budget consumption in team standup.
- **Monthly:** SLO review meeting — adjust targets if consistently over/under.
- **Quarterly:** Re-evaluate SLI definitions and add new SLOs as services mature.
