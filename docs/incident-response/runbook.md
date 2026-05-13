# ParkirPintar Incident Response Runbook

## Severity Levels

| Level | Description | Response Time | Resolution Target | Examples |
|-------|-------------|---------------|-------------------|----------|
| **P1** | Critical - Service fully down, data loss risk | 5 min acknowledge, 15 min response | 1 hour | All services unreachable, database corruption, payment system completely down |
| **P2** | High - Major feature broken, significant user impact | 15 min acknowledge, 30 min response | 4 hours | Booking failures, payment processing errors, spot double-booking |
| **P3** | Medium - Degraded performance, partial feature loss | 1 hour acknowledge, 4 hour response | 24 hours | High latency (p95 > 500ms), cache unavailable but degrading gracefully |
| **P4** | Low - Minor issue, cosmetic, no user impact | 4 hours acknowledge, next business day | 1 week | Dashboard rendering issues, non-critical log errors |

---

## On-Call Rotation & Escalation

### Escalation Path

```
L1: On-call Engineer (primary responder)
 └─> L2: Team Lead / Senior Engineer (15 min no response)
      └─> L3: Engineering Manager (30 min no response or P1)
           └─> L4: CTO (data breach, extended outage > 1hr)
```

### On-Call Responsibilities

- Monitor alert channels (Grafana alerts, PagerDuty/Telegram bot)
- Acknowledge incidents within SLA
- Triage and begin investigation
- Escalate if unable to resolve within 30 minutes
- Update status page and stakeholders

### Handoff Protocol

- End-of-shift handoff via shared incident doc
- Ongoing incidents: verbal handoff + written summary
- All context must be in the incident thread, not just in someone's head

---

## Monitoring Dashboards

| Tool | URL | Purpose |
|------|-----|---------|
| **Grafana** | `http://<host>:3000` | Metrics dashboards, alerting |
| **Prometheus** | `http://<host>:9090` | Raw metrics, PromQL queries |
| **Tempo** | `http://<host>:3200` | Distributed traces |
| **Coolify** | `http://<host>:8000` | Container management, deployment status |
| **Traefik Dashboard** | `http://<host>:8080` | Routing, TLS, backend health |

### Key Grafana Dashboards

- **Service Overview** - Request rate, error rate, latency (RED metrics)
- **PostgreSQL** - Connection pool, query duration, replication lag
- **Redis** - Memory usage, hit rate, connected clients
- **NATS** - Message throughput, consumer lag, slow consumers
- **Infrastructure** - CPU, memory, disk, network per container

---

## Common Incidents

### 1. Service Unavailable (502/503)

**Symptoms:** Users see 502/503 errors, Traefik returning bad gateway.

**Diagnosis:**

```bash
# Check container status via Coolify or Docker
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

# Check if Watchtower recently restarted containers
docker logs watchtower --since 10m

# Check Traefik routing
curl -s http://localhost:8080/api/http/services | jq '.[] | {name, status}'

# Check for port conflicts
ss -tlnp | grep -E '(8080|3000|5432|6379|4222)'

# Check container logs for crash loops
docker logs <service-name> --tail 100 --since 5m
```

**Resolution Steps:**

1. Identify which service(s) are down from Traefik dashboard
2. Check if Watchtower triggered an update — roll back if bad image:
   ```bash
   docker pull <registry>/<image>:<previous-tag>
   docker stop <container> && docker rm <container>
   # Redeploy via Coolify with previous version
   ```
3. If OOM killed: check `docker inspect <container>` for memory limits, increase if needed
4. If port conflict: identify conflicting process, stop it or reassign ports
5. If crash loop: check application logs, fix config or code, redeploy

**Rollback:**
```bash
# Via Coolify UI: select previous deployment and redeploy
# Or manually:
docker stop <service>
docker run -d --name <service> --network parkir-net <registry>/<image>:<last-known-good>
```

---

### 2. Database Connection Exhaustion

**Symptoms:** Services returning 500 errors, logs showing `too many connections` or `connection pool timeout`.

**Diagnosis:**

```bash
# Check active connections
docker exec -it postgres psql -U parkir -c "
  SELECT count(*), state, usename, application_name
  FROM pg_stat_activity
  GROUP BY state, usename, application_name
  ORDER BY count DESC;"

# Check max connections setting
docker exec -it postgres psql -U parkir -c "SHOW max_connections;"

# Find long-running queries
docker exec -it postgres psql -U parkir -c "
  SELECT pid, now() - pg_stat_activity.query_start AS duration, query, state
  FROM pg_stat_activity
  WHERE (now() - pg_stat_activity.query_start) > interval '30 seconds'
  AND state != 'idle'
  ORDER BY duration DESC;"

# Check connection pool metrics in Grafana
# Dashboard: PostgreSQL > Connection Pool
```

**Resolution Steps:**

1. Kill long-running queries if safe:
   ```sql
   SELECT pg_terminate_backend(<pid>);
   ```
2. If a specific service is leaking connections, restart it:
   ```bash
   docker restart <service-name>
   ```
3. Temporarily increase pool size in service config:
   ```yaml
   # In service environment or config
   DB_MAX_OPEN_CONNS: 50  # default is usually 25
   DB_MAX_IDLE_CONNS: 10
   DB_CONN_MAX_LIFETIME: 300s
   ```
4. If systemic, increase PostgreSQL max_connections:
   ```bash
   docker exec -it postgres psql -U parkir -c "ALTER SYSTEM SET max_connections = 200;"
   # Requires restart
   docker restart postgres
   ```

**Prevention:** Ensure all services use connection pooling with proper limits. Set `DB_CONN_MAX_LIFETIME` to prevent stale connections.

---

### 3. Redis Unavailable

**Symptoms:** Increased latency, cache misses, potential booking conflicts.

**Diagnosis:**

```bash
# Check Redis container
docker logs redis --tail 50
docker exec -it redis redis-cli ping

# Check memory usage
docker exec -it redis redis-cli info memory

# Check connected clients
docker exec -it redis redis-cli info clients

# Check if maxmemory reached
docker exec -it redis redis-cli info memory | grep used_memory_human
docker exec -it redis redis-cli config get maxmemory
```

**Resolution Steps:**

1. If Redis is down, restart:
   ```bash
   docker restart redis
   ```
2. If OOM: flush non-critical caches or increase maxmemory:
   ```bash
   docker exec -it redis redis-cli config set maxmemory 512mb
   ```
3. Verify graceful degradation is working:
   - Services should bypass cache and hit DB directly
   - Check service logs for `redis unavailable, falling back to database` messages
4. If distributed locks are affected (spot booking), **pause booking operations** until Redis is back
5. After recovery, warm critical caches if needed

**Graceful Degradation Checklist:**
- [ ] Booking service falls back to DB-level locking
- [ ] Session service falls back to DB sessions
- [ ] Rate limiting disabled (monitor for abuse)
- [ ] Spot availability queries hit DB directly

---

### 4. High Latency (p95 > 500ms)

**Symptoms:** Grafana alerts on latency SLO breach, users reporting slow responses.

**Diagnosis:**

```bash
# Check Tempo for slow traces
# Grafana > Explore > Tempo
# Query: {service.name="parking-service"} | duration > 500ms

# Check Prometheus for latency breakdown
# PromQL: histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))

# Check database query performance
docker exec -it postgres psql -U parkir -c "
  SELECT query, calls, mean_exec_time, total_exec_time
  FROM pg_stat_statements
  ORDER BY mean_exec_time DESC
  LIMIT 10;"

# Check NATS consumer lag
docker exec -it nats nats consumer info <stream> <consumer>

# Check container resource usage
docker stats --no-stream
```

**Resolution Steps:**

1. Identify the slow service from Grafana RED dashboard
2. Find slow traces in Tempo — look for:
   - Slow DB queries → add indexes or optimize query
   - Slow external calls → check payment gateway, third-party APIs
   - NATS message backlog → scale consumers
3. If DB-related:
   ```sql
   -- Check for missing indexes
   EXPLAIN ANALYZE <slow-query>;
   -- Add index if needed
   CREATE INDEX CONCURRENTLY idx_<table>_<column> ON <table>(<column>);
   ```
4. If resource-constrained: scale horizontally via Coolify or increase container limits
5. If NATS backlog: check consumer health, restart slow consumers

---

### 5. Payment Failures

**Symptoms:** Users unable to complete parking payments, payment webhook errors.

**Diagnosis:**

```bash
# Check payment service logs
docker logs payment-service --tail 200 --since 15m | grep -i "error\|fail\|timeout"

# Check payment gateway connectivity
curl -s -o /dev/null -w "%{http_code}" https://<payment-gateway>/health

# Check NATS payment event stream
docker exec -it nats nats stream info PAYMENTS

# Check for stuck/failed payment events
docker exec -it postgres psql -U parkir -c "
  SELECT status, count(*), max(updated_at)
  FROM payments
  WHERE created_at > now() - interval '1 hour'
  GROUP BY status;"
```

**Resolution Steps:**

1. If payment gateway is down:
   - Check gateway status page
   - Enable payment queue mode (events stored in NATS, processed when gateway recovers)
   - Notify users of temporary payment issues
2. If webhook failures:
   - Check Traefik logs for incoming webhook requests
   - Verify webhook URL is accessible externally
   - Replay failed webhooks from gateway dashboard
3. If retry logic is failing:
   ```bash
   # Check dead letter queue
   docker exec -it nats nats stream info PAYMENTS_DLQ
   # Replay failed messages
   docker exec -it nats nats consumer replay PAYMENTS <consumer>
   ```
4. If idempotency issues: check for duplicate payment records, reconcile manually

**Critical:** Never retry payments without idempotency keys. Always verify payment state with gateway before retrying.

---

### 6. Spot Double-Booking

**Symptoms:** Two vehicles assigned to the same parking spot, user complaints.

**Diagnosis:**

```bash
# Check for actual double-bookings
docker exec -it postgres psql -U parkir -c "
  SELECT spot_id, count(*), array_agg(booking_id)
  FROM bookings
  WHERE status = 'active'
  GROUP BY spot_id
  HAVING count(*) > 1;"

# Check Redis distributed lock status
docker exec -it redis redis-cli keys "lock:spot:*"

# Check booking service logs for lock failures
docker logs booking-service --tail 200 | grep -i "lock\|acquire\|conflict"

# Check Redis connectivity from booking service
docker exec -it booking-service wget -qO- http://redis:6379/ping 2>&1 || echo "Redis unreachable from service"
```

**Resolution Steps:**

1. **Immediate:** Manually resolve the conflict:
   ```sql
   -- Identify the later booking and cancel it
   UPDATE bookings SET status = 'cancelled', cancelled_reason = 'double_booking_incident'
   WHERE booking_id = '<later-booking-id>';
   ```
2. Notify affected user, offer alternative spot + compensation
3. Investigate root cause:
   - Redis was unavailable during booking → lock not acquired
   - Lock TTL too short → booking took longer than lock duration
   - Race condition in code → review distributed lock implementation
4. If Redis connectivity issue: fix network, restart service
5. If lock TTL issue: increase TTL in booking service config:
   ```yaml
   BOOKING_LOCK_TTL: 30s  # increase from default
   BOOKING_LOCK_RETRY: 3
   ```
6. Verify fix: run concurrent booking test against same spot

**Prevention:** Ensure DB-level unique constraint exists as a safety net:
```sql
CREATE UNIQUE INDEX idx_active_spot_booking ON bookings(spot_id) WHERE status = 'active';
```

---

## Communication Templates

### Initial Incident Notification

```
🚨 [P{level}] Incident: {title}

Status: Investigating
Impact: {brief impact description}
Start time: {HH:MM UTC}
Responder: {name}

We are aware of the issue and actively investigating. Updates will follow every {15/30} minutes.
```

### Status Update

```
🔄 [P{level}] Update: {title}

Status: {Investigating | Identified | Mitigating | Resolved}
Duration: {X} minutes
Update: {what we know, what we're doing}

Next update in {X} minutes.
```

### Resolution Notification

```
✅ [P{level}] Resolved: {title}

Duration: {total duration}
Resolution: {brief description of fix}
Impact: {users/transactions affected}

A post-mortem will be conducted within 48 hours.
```

### Internal Escalation Message

```
⬆️ Escalating: {title}

Current status: {summary}
What's been tried: {actions taken}
Why escalating: {reason - timeout, expertise needed, business decision}
Relevant links: {dashboard URL, logs, trace ID}
```

---

## Quick Reference

### Service Map

```
Internet → Traefik (reverse proxy)
  ├── API Gateway → Booking Service → PostgreSQL, Redis, NATS
  ├── API Gateway → Payment Service → Payment Gateway, NATS
  ├── API Gateway → Parking Service → PostgreSQL, Redis
  └── API Gateway → User Service → PostgreSQL, Redis
```

### Key Ports

| Service | Port |
|---------|------|
| Traefik (HTTP) | 80 |
| Traefik (HTTPS) | 443 |
| Traefik Dashboard | 8080 |
| Grafana | 3000 |
| Prometheus | 9090 |
| Tempo | 3200 |
| PostgreSQL | 5432 |
| Redis | 6379 |
| NATS | 4222 |
| NATS Monitor | 8222 |
| Coolify | 8000 |

### Useful Commands Cheat Sheet

```bash
# Restart all services
docker compose -f docker-compose.prod.yml restart

# View all container resource usage
docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}"

# Follow logs for a service
docker logs -f <service> --since 5m

# Check Traefik routing
curl http://localhost:8080/api/http/routers | jq

# Quick PostgreSQL health check
docker exec postgres pg_isready -U parkir

# Quick Redis health check
docker exec redis redis-cli ping

# Check NATS server status
docker exec nats nats server info

# Export OTel traces for a specific trace ID
curl "http://localhost:3200/api/traces/<trace-id>" | jq
```
