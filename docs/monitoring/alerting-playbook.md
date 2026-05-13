# ParkirPintar Alerting Playbook

This playbook covers response procedures for each alert rule configured in the ParkirPintar monitoring stack. Alert rules are defined in `deploy/monitoring/prometheus/alerts.yml`.

---

## HighErrorRate

**Severity:** Critical

**What it means:**
More than 5% of HTTP requests are returning 5xx status codes for a service over the last 5 minutes.

**Alert expression:**
```promql
(
  sum(rate(http_requests_total{status_code=~"5.."}[5m])) by (job)
  /
  sum(rate(http_requests_total[5m])) by (job)
) > 0.05
```

**Likely causes:**
- Application panic or unhandled error in request handler
- Database connection failure (pool exhausted, PostgreSQL down)
- Downstream service unavailable (gRPC target unreachable)
- Redis connection timeout
- Out of memory (OOM) causing partial failures
- Bad deployment (new code with bugs)

**Investigation steps:**

1. Identify the affected service from the alert label `job`:
   ```promql
   sum(rate(http_requests_total{status_code=~"5..", job="<affected-job>"}[5m])) by (status_code)
   ```

2. Check logs for error details:
   ```logql
   {service_name="parkir-pintar-<service>"} | json | severity="ERROR" | line_format "{{.msg}} {{.error}}"
   ```

3. Look for correlated traces with errors:
   ```traceql
   { resource.service.name = "parkir-pintar-<service>" && status = error && duration > 100ms }
   ```

4. Check if the error rate correlates with a deployment:
   ```bash
   docker logs <container> --since 10m 2>&1 | head -50
   ```

5. Check downstream dependencies:
   ```promql
   sum(rate(traces_spanmetrics_calls_total{status_code="STATUS_CODE_ERROR", service_name="parkir-pintar-<service>"}[5m])) by (span_name)
   ```

**Resolution steps:**

1. If caused by a bad deployment → rollback:
   ```bash
   docker compose -f docker-compose.yml up -d --force-recreate <service>
   ```

2. If database connection issue → check PostgreSQL:
   ```bash
   docker exec parkir-pintar-postgres pg_isready
   docker exec parkir-pintar-postgres psql -U parkir -c "SELECT count(*) FROM pg_stat_activity;"
   ```

3. If Redis issue → check Redis:
   ```bash
   docker exec parkir-pintar-redis redis-cli ping
   docker exec parkir-pintar-redis redis-cli info memory
   ```

4. If OOM → restart the service and investigate memory leak:
   ```bash
   docker restart parkir-pintar-<service>
   ```

5. If downstream service is down → restart it and check its logs.

---

## HighLatency

**Severity:** Warning

**What it means:**
The P95 HTTP request duration exceeds 500ms for a service over the last 5 minutes.

**Alert expression:**
```promql
histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket[5m])) by (le, job)) > 0.5
```

**Likely causes:**
- Slow database queries (missing index, table lock, large result set)
- Redis latency spike (memory pressure, persistence fork)
- Network congestion between services
- CPU saturation on the host
- Garbage collection pauses (large heap)
- Downstream service responding slowly

**Investigation steps:**

1. Identify which endpoints are slow:
   ```promql
   histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket{job="<affected-job>"}[5m])) by (le, handler))
   ```

2. Check database query latency:
   ```promql
   histogram_quantile(0.95, sum(rate(db_query_duration_seconds_bucket{job="<affected-job>"}[5m])) by (le))
   ```

3. Look at slow traces:
   ```traceql
   { resource.service.name = "parkir-pintar-<service>" && duration > 500ms }
   ```

4. Check for GC pressure:
   ```promql
   rate(go_gc_duration_seconds_sum{job=~"parkir-pintar.*"}[5m])
   ```

5. Check host resources:
   ```bash
   docker stats --no-stream
   ```

**Resolution steps:**

1. If slow DB queries → identify and optimize:
   ```bash
   docker exec parkir-pintar-postgres psql -U parkir -c "SELECT * FROM pg_stat_activity WHERE state = 'active' AND query_start < now() - interval '5 seconds';"
   ```

2. If Redis latency → check memory and eviction:
   ```bash
   docker exec parkir-pintar-redis redis-cli info memory
   docker exec parkir-pintar-redis redis-cli slowlog get 10
   ```

3. If CPU saturation → scale horizontally or identify hot path.

4. If GC pressure → check for memory leaks, consider tuning `GOGC`.

---

## HighGRPCErrorRate

**Severity:** Critical

**What it means:**
More than 5% of gRPC requests are returning non-OK status codes for a service over the last 5 minutes.

**Alert expression:**
```promql
(
  sum(rate(grpc_requests_total{grpc_code!="OK"}[5m])) by (job)
  /
  sum(rate(grpc_requests_total[5m])) by (job)
) > 0.05
```

**Likely causes:**
- Target gRPC service is down or unreachable
- Connection refused (service not listening on expected port)
- Deadline exceeded (timeout too short or service too slow)
- Invalid request payload (schema mismatch after deployment)
- TLS/mTLS certificate issues
- NATS-triggered handler failing (async gRPC calls)

**Investigation steps:**

1. Break down by gRPC status code:
   ```promql
   sum(rate(grpc_requests_total{grpc_code!="OK", job="<affected-job>"}[5m])) by (grpc_code, grpc_method)
   ```

2. Check if the target service is healthy:
   ```bash
   docker exec parkir-pintar-<caller> wget -qO- http://parkir-pintar-<target>:8080/health
   ```

3. Look at error traces:
   ```traceql
   { resource.service.name = "parkir-pintar-<service>" && span.rpc.grpc.status_code != 0 }
   ```

4. Check logs for connection errors:
   ```logql
   {service_name="parkir-pintar-<service>"} | json | msg=~".*grpc.*" | severity="ERROR"
   ```

**Resolution steps:**

1. If target service is down → restart it:
   ```bash
   docker restart parkir-pintar-<target-service>
   ```

2. If `DEADLINE_EXCEEDED` → increase timeout or investigate why target is slow.

3. If `UNAVAILABLE` → check DNS resolution and network connectivity:
   ```bash
   docker exec parkir-pintar-<caller> nslookup parkir-pintar-<target>
   ```

4. If `INVALID_ARGUMENT` → check for proto schema mismatch; may need to rebuild both services.

---

## HighSpanErrorRate

**Severity:** Critical

**What it means:**
More than 5% of traced spans have error status for a service over the last 5 minutes. This is derived from span metrics generated by Alloy's spanmetrics connector.

**Alert expression:**
```promql
(
  sum(rate(traces_spanmetrics_calls_total{status_code="STATUS_CODE_ERROR"}[5m])) by (service_name)
  /
  sum(rate(traces_spanmetrics_calls_total[5m])) by (service_name)
) > 0.05
```

**Likely causes:**
- Application errors being recorded on spans (panics, unhandled errors)
- Failed external calls (DB, Redis, NATS, HTTP, gRPC)
- Business logic errors marked as span errors
- Infrastructure issues causing cascading failures

**Investigation steps:**

1. Identify which operations are failing:
   ```promql
   sum(rate(traces_spanmetrics_calls_total{status_code="STATUS_CODE_ERROR", service_name="<service>"}[5m])) by (span_name)
   ```

2. Find example error traces:
   ```traceql
   { resource.service.name = "parkir-pintar-<service>" && status = error }
   ```

3. Check correlated logs:
   ```logql
   {service_name="parkir-pintar-<service>"} | json | severity="ERROR"
   ```

4. Check if multiple services are affected (cascading failure):
   ```promql
   sum(rate(traces_spanmetrics_calls_total{status_code="STATUS_CODE_ERROR"}[5m])) by (service_name)
   ```

**Resolution steps:**

1. If single operation failing → check that specific handler's dependencies.

2. If cascading across services → identify the root cause service (usually the one that errored first):
   ```logql
   {service_name=~"parkir-pintar-.+"} | json | severity="ERROR" | line_format "{{.ts}} {{.service_name}} {{.msg}}"
   ```

3. If infrastructure issue → check PostgreSQL, Redis, NATS health.

4. Restart affected service if it's in a bad state:
   ```bash
   docker restart parkir-pintar-<service>
   ```

---

## HighSpanLatency

**Severity:** Warning

**What it means:**
The P95 span duration exceeds 500ms for a service over the last 5 minutes. This captures latency across all traced operations (HTTP, gRPC, DB, internal).

**Alert expression:**
```promql
histogram_quantile(0.95, sum(rate(traces_spanmetrics_duration_milliseconds_bucket[5m])) by (le, service_name)) > 500
```

**Likely causes:**
- Slow database queries
- Network latency to downstream services
- Lock contention (database row locks, mutex)
- Large payload serialization/deserialization
- External API calls timing out
- Resource exhaustion (CPU, memory, file descriptors)

**Investigation steps:**

1. Find the slowest operations:
   ```promql
   histogram_quantile(0.95, sum(rate(traces_spanmetrics_duration_milliseconds_bucket{service_name="<service>"}[5m])) by (le, span_name))
   ```

2. Look at slow traces in Tempo:
   ```traceql
   { resource.service.name = "parkir-pintar-<service>" && duration > 500ms }
   ```

3. Check if DB queries are the bottleneck:
   ```promql
   histogram_quantile(0.95, sum(rate(db_query_duration_seconds_bucket{job=~".*<service>.*"}[5m])) by (le))
   ```

4. Check goroutine count for contention:
   ```promql
   go_goroutines{job=~".*<service>.*"}
   ```

**Resolution steps:**

1. If DB queries are slow → analyze query plans, add indexes:
   ```bash
   docker exec parkir-pintar-postgres psql -U parkir -c "EXPLAIN ANALYZE <slow-query>;"
   ```

2. If network latency → check inter-container connectivity.

3. If resource exhaustion → scale the service or optimize hot paths.

4. If lock contention → review concurrent access patterns in code.

---

## NoTrafficDetected

**Severity:** Warning

**What it means:**
A service has received zero server spans for 10 minutes. The service may be down, crashed, or unable to send telemetry.

**Alert expression:**
```promql
sum(rate(traces_spanmetrics_calls_total{span_kind="SPAN_KIND_SERVER"}[10m])) by (service_name) == 0
```

**Likely causes:**
- Service container crashed or was stopped
- Service is running but OTel SDK failed to initialize
- Network partition between service and Alloy
- Alloy is down or not accepting connections
- Service has no incoming traffic (legitimate during off-hours for some services)

**Investigation steps:**

1. Check if the container is running:
   ```bash
   docker ps | grep parkir-pintar-<service>
   ```

2. Check container logs:
   ```bash
   docker logs parkir-pintar-<service> --tail 50
   ```

3. Check if Alloy is receiving any data:
   ```bash
   curl -s http://100.79.123.39:12345/metrics | grep otelcol_receiver_accepted_spans
   ```

4. Verify network connectivity:
   ```bash
   docker exec parkir-pintar-<service> wget -qO- http://monitoring-alloy:4319 || echo "Cannot reach Alloy"
   ```

5. Check if it's a legitimate no-traffic period (e.g., search service during maintenance).

**Resolution steps:**

1. If container is stopped/crashed → restart:
   ```bash
   docker compose up -d <service>
   ```

2. If OTel initialization failed → check environment variables:
   ```bash
   docker inspect parkir-pintar-<service> | jq '.[0].Config.Env' | grep -i trac
   ```

3. If Alloy is down → restart Alloy:
   ```bash
   cd deploy/monitoring
   docker compose -f docker-compose.monitoring.yml restart alloy
   ```

4. If network issue → check Docker network:
   ```bash
   docker network inspect coolify
   ```

---

## NATSConsumerLag

**Severity:** Warning

**What it means:**
The rate of messages being published exceeds the rate of messages being consumed by more than 100 msg/s for 5 minutes. Consumers are falling behind.

**Alert expression:**
```promql
rate(nats_messages_published_total[5m]) - rate(nats_messages_consumed_total[5m]) > 100
```

**Likely causes:**
- Consumer service is slow (processing bottleneck)
- Consumer service crashed and isn't processing messages
- Sudden spike in published messages (burst traffic)
- Consumer blocked on downstream dependency (DB, external API)
- NATS JetStream consumer acknowledgment issues

**Investigation steps:**

1. Identify which consumer is lagging:
   ```promql
   rate(nats_messages_published_total[5m]) - rate(nats_messages_consumed_total[5m])
   ```

2. Check consumer service health:
   ```bash
   docker logs parkir-pintar-<consumer-service> --tail 100
   ```

3. Look for processing errors:
   ```logql
   {service_name="parkir-pintar-<service>"} | json | msg=~".*nats.*|.*consume.*|.*subscribe.*"
   ```

4. Check if the consumer is stuck on a slow operation:
   ```traceql
   { resource.service.name = "parkir-pintar-<service>" && name =~ "nats.*" && duration > 1s }
   ```

5. Check NATS server health:
   ```bash
   docker exec parkir-pintar-nats nats-server --signal ldm  # or check via monitoring endpoint
   ```

**Resolution steps:**

1. If consumer crashed → restart:
   ```bash
   docker restart parkir-pintar-<consumer-service>
   ```

2. If processing is slow → check downstream dependencies (DB, Redis).

3. If burst traffic → the lag should resolve once the burst subsides. Monitor.

4. If persistent lag → consider scaling consumers or optimizing message processing:
   - Batch processing instead of one-at-a-time
   - Increase consumer concurrency
   - Optimize the handler logic

5. If NATS itself is unhealthy → restart NATS (caution: may lose in-flight messages):
   ```bash
   docker restart parkir-pintar-nats
   ```

---

## DatabaseSlowQueries

**Severity:** Warning

**What it means:**
The P95 database query duration exceeds 200ms for a service over the last 5 minutes.

**Alert expression:**
```promql
histogram_quantile(0.95, sum(rate(db_query_duration_seconds_bucket[5m])) by (le, job)) > 0.2
```

**Likely causes:**
- Missing database index on frequently queried columns
- Table lock contention (long-running transactions)
- Large sequential scans (table grew beyond expected size)
- PostgreSQL autovacuum running
- Connection pool exhaustion (queries queuing)
- Complex joins or subqueries without optimization
- Disk I/O saturation on the database host

**Investigation steps:**

1. Identify which service has slow queries:
   ```promql
   histogram_quantile(0.95, sum(rate(db_query_duration_seconds_bucket[5m])) by (le, job))
   ```

2. Check active queries in PostgreSQL:
   ```bash
   docker exec parkir-pintar-postgres psql -U parkir -c "
     SELECT pid, now() - pg_stat_activity.query_start AS duration, query, state
     FROM pg_stat_activity
     WHERE state != 'idle'
     ORDER BY duration DESC
     LIMIT 10;
   "
   ```

3. Check for lock contention:
   ```bash
   docker exec parkir-pintar-postgres psql -U parkir -c "
     SELECT blocked_locks.pid AS blocked_pid,
            blocking_locks.pid AS blocking_pid,
            blocked_activity.query AS blocked_query
     FROM pg_catalog.pg_locks blocked_locks
     JOIN pg_catalog.pg_stat_activity blocked_activity ON blocked_activity.pid = blocked_locks.pid
     JOIN pg_catalog.pg_locks blocking_locks ON blocking_locks.locktype = blocked_locks.locktype
     WHERE NOT blocked_locks.granted;
   "
   ```

4. Check connection pool usage:
   ```promql
   db_pool_active_connections{job=~".*<service>.*"} / db_pool_max_connections{job=~".*<service>.*"}
   ```

5. Look at slow query traces:
   ```traceql
   { resource.service.name = "parkir-pintar-<service>" && name =~ "db.*" && duration > 200ms }
   ```

**Resolution steps:**

1. If missing index → identify the query and add an index:
   ```bash
   docker exec parkir-pintar-postgres psql -U parkir -c "
     SELECT schemaname, relname, seq_scan, idx_scan
     FROM pg_stat_user_tables
     ORDER BY seq_scan DESC
     LIMIT 10;
   "
   ```

2. If lock contention → terminate blocking queries:
   ```bash
   docker exec parkir-pintar-postgres psql -U parkir -c "SELECT pg_terminate_backend(<blocking_pid>);"
   ```

3. If connection pool exhausted → increase pool size in service config or investigate connection leaks.

4. If autovacuum → let it complete; consider scheduling during low-traffic periods.

5. If disk I/O → check host disk usage and consider moving to faster storage.

---

## General Escalation

If an alert cannot be resolved within 15 minutes:

1. Check if multiple alerts are firing (indicates systemic issue)
2. Review recent deployments: `docker ps --format "{{.Names}} {{.Status}}"`
3. Check host resources: `docker stats --no-stream`
4. Escalate to the team via Telegram group
5. Consider rolling back the last deployment if it correlates with alert onset

## Silencing Alerts

During planned maintenance or known issues:

1. Go to Alertmanager: `http://100.79.123.39:9093/#/silences`
2. Click **New Silence**
3. Add matchers (e.g., `alertname=NoTrafficDetected`, `service_name=parkir-pintar-search`)
4. Set duration and add a comment explaining why
5. **Always set an expiry** — never silence indefinitely
