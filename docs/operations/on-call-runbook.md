# On-Call Runbook — ParkirPintar

## On-Call Rotation

### Current Setup

- **Rotation**: Single engineer (solo developer project)
- **Schedule**: 24/7 with alert fatigue management
- **Primary contact**: Telegram notifications via Alertmanager webhook
- **Backup**: Alerts auto-escalate if not acknowledged within SLA

### Future Scaling

When team grows:
- Weekly rotation (Monday 09:00 to Monday 09:00)
- Primary + secondary on-call
- Handoff document at rotation change
- No on-call during approved PTO (swap required)

## Alert Response Procedures

### HighErrorRate

**Alert**: Error rate > 5% for 5 minutes on any service

**Steps:**
1. Check Grafana dashboard: `ParkirPintar / Service Overview`
2. Identify which service has elevated errors
3. Check Loki logs for the service:
   ```
   {app="<service-name>"} |= "error" | logfmt | level="error"
   ```
4. Look for patterns: same error repeated? New error type?
5. Check if a recent deployment happened (Watchtower auto-deploy)
6. If deployment-related → rollback (see Rollback Procedures)
7. If infrastructure-related → check dependent services (DB, Redis, NATS)

---

### HighLatency (P95 > 2s)

**Alert**: 95th percentile latency exceeds 2 seconds for 5 minutes

**Steps:**
1. Check Grafana: `ParkirPintar / Latency` panel
2. Identify slow endpoints via Tempo traces:
   ```
   {service.name="<service>"} | duration > 2s
   ```
3. Common causes:
   - Database slow queries → check `pg_stat_statements`
   - Redis timeout → check Redis memory/connections
   - Downstream service slow → check inter-service call traces
4. If DB-related:
   ```sql
   SELECT query, mean_exec_time, calls
   FROM pg_stat_statements
   ORDER BY mean_exec_time DESC
   LIMIT 10;
   ```
5. If Redis-related:
   ```bash
   redis-cli --latency-history
   redis-cli info clients
   ```
6. Temporary mitigation: increase timeouts if cascading

---

### ServiceDown

**Alert**: Service health check failing for > 1 minute

**Steps:**
1. Check Docker container status:
   ```bash
   docker ps -a | grep <service-name>
   docker logs --tail 50 <container-id>
   ```
2. If container is restarting (OOM, panic):
   - Check logs for panic stack trace
   - Check memory usage: `docker stats <container-id>`
   - If OOM: increase memory limit in docker-compose
3. If container is stopped:
   ```bash
   docker-compose up -d <service-name>
   ```
4. If container won't start: check config, env vars, dependent services
5. Verify recovery: check health endpoint responds

---

### DatabaseConnectionPoolExhausted

**Alert**: Available DB connections < 5

**Steps:**
1. Check active connections:
   ```sql
   SELECT count(*), state FROM pg_stat_activity
   WHERE datname = 'parkir_pintar'
   GROUP BY state;
   ```
2. Identify long-running queries:
   ```sql
   SELECT pid, now() - pg_stat_activity.query_start AS duration, query, state
   FROM pg_stat_activity
   WHERE (now() - pg_stat_activity.query_start) > interval '30 seconds'
   AND state != 'idle';
   ```
3. Kill stuck queries if needed:
   ```sql
   SELECT pg_terminate_backend(<pid>);
   ```
4. Check for connection leaks in application logs
5. If recurring: increase pool size or fix connection leak

---

### NATSDisconnected

**Alert**: NATS connection lost for any service

**Steps:**
1. Check NATS server status:
   ```bash
   docker logs parkir-nats --tail 30
   nats server check connection
   ```
2. If NATS is down:
   ```bash
   docker-compose restart nats
   ```
3. After NATS recovery, verify consumers:
   ```bash
   nats stream ls
   nats consumer ls PARKIR_EVENTS
   ```
4. Check for unprocessed messages:
   ```bash
   nats stream info PARKIR_EVENTS
   ```
5. Services should auto-reconnect; if not, restart affected services

---

### RedisMemoryHigh

**Alert**: Redis memory usage > 80%

**Steps:**
1. Check memory breakdown:
   ```bash
   redis-cli info memory
   redis-cli memory doctor
   ```
2. Check largest keys:
   ```bash
   redis-cli --bigkeys
   ```
3. Check TTL compliance (keys should have TTLs):
   ```bash
   redis-cli dbsize
   ```
4. If emergency: flush non-critical caches
   ```bash
   # Only flush session cache, NOT reservation locks
   redis-cli --scan --pattern "cache:*" | xargs redis-cli del
   ```
5. Long-term: review eviction policy, increase maxmemory

---

### DiskSpaceHigh

**Alert**: Disk usage > 85%

**Steps:**
1. Identify large directories:
   ```bash
   du -sh /var/lib/docker/*
   du -sh /var/lib/postgresql/*
   ```
2. Clean Docker resources:
   ```bash
   docker system prune -f
   docker image prune -a --filter "until=168h"
   ```
3. Check PostgreSQL WAL files:
   ```bash
   du -sh /var/lib/postgresql/data/pg_wal/
   ```
4. If WAL bloat: check replication slots, run checkpoint
5. Rotate/compress old log files:
   ```bash
   find /var/log -name "*.log" -mtime +7 -exec gzip {} \;
   ```

## Escalation Paths

```
Level 1: On-call engineer (you)
  ↓ (no progress in 30 min for P0/P1)
Level 2: Team lead / senior engineer
  ↓ (no progress in 1h for P0)
Level 3: CTO / external consultant
```

### When to Escalate

- You don't understand the failure mode
- Fix requires access you don't have
- Customer data may be compromised (always escalate security)
- Multiple services failing simultaneously
- You've been working on it for 30+ minutes with no progress

## Common Troubleshooting Steps

### General Triage Flow

1. **Check Grafana** (`http://monitoring.parkir-pintar.local:3000`)
   - Service Overview dashboard
   - Identify which service/metric is anomalous

2. **Check Loki Logs** (via Grafana Explore)
   ```
   {app=~"reservation-service|billing-service|notification-service"}
     |= "error"
     | logfmt
   ```

3. **Check Tempo Traces** (via Grafana Explore)
   - Filter by service name and error status
   - Look at trace waterfall for slow spans
   - Identify which hop is causing latency

4. **Check Infrastructure**
   ```bash
   docker ps -a                    # Container status
   docker stats                    # Resource usage
   docker-compose logs --tail 20   # Recent logs
   ```

5. **Check Dependencies**
   ```bash
   # PostgreSQL
   pg_isready -h localhost -p 5432

   # Redis
   redis-cli ping

   # NATS
   nats server check connection
   ```

## Service Restart Procedures

### Single Service Restart

```bash
cd /opt/parkir-pintar
docker-compose restart <service-name>

# Verify health
curl -s http://localhost:<port>/healthz | jq .
```

### Full Stack Restart (last resort)

```bash
cd /opt/parkir-pintar

# Graceful shutdown (preserves data)
docker-compose down

# Wait for clean shutdown
sleep 10

# Start infrastructure first
docker-compose up -d postgres redis nats

# Wait for infra readiness
sleep 15

# Start application services
docker-compose up -d reservation-service billing-service notification-service

# Start gateway last
docker-compose up -d api-gateway

# Verify all healthy
docker-compose ps
```

## Database Recovery Steps

### Connection Issues

```bash
# Check if PostgreSQL is running
docker-compose logs postgres --tail 20

# Restart PostgreSQL
docker-compose restart postgres

# Verify connections
pg_isready -h localhost -p 5432
```

### Corrupted Data / Restore from Backup

```bash
# Stop services that write to DB
docker-compose stop reservation-service billing-service

# Restore from latest backup
gunzip < /backups/postgres/parkir_pintar_$(date +%Y%m%d).sql.gz | \
  docker exec -i parkir-postgres psql -U parkir -d parkir_pintar

# Verify data integrity
docker exec parkir-postgres psql -U parkir -d parkir_pintar \
  -c "SELECT count(*) FROM reservations;"

# Restart services
docker-compose start reservation-service billing-service
```

### Migration Rollback

```bash
# Check current migration version
docker exec parkir-postgres psql -U parkir -d parkir_pintar \
  -c "SELECT version, dirty FROM schema_migrations;"

# If dirty, force version
migrate -path ./migrations -database "postgres://..." force <last-good-version>

# Rollback one step
migrate -path ./migrations -database "postgres://..." down 1
```

## NATS Recovery Steps

### NATS Server Won't Start

```bash
# Check logs
docker-compose logs nats --tail 50

# If JetStream data corrupted, reset (CAUTION: loses undelivered messages)
docker-compose stop nats
rm -rf ./data/nats/jetstream  # Only if confirmed corrupt
docker-compose start nats

# Recreate streams and consumers
nats stream add PARKIR_EVENTS --config ./config/nats-streams.json
```

### Messages Stuck / Consumer Lag

```bash
# Check consumer info
nats consumer info PARKIR_EVENTS <consumer-name>

# If consumer is stuck, reset sequence
nats consumer sub PARKIR_EVENTS <consumer-name> --reset

# Or delete and recreate consumer (reprocesses from stream start)
nats consumer rm PARKIR_EVENTS <consumer-name>
nats consumer add PARKIR_EVENTS <consumer-name> --config ./config/nats-consumers.json
```

## Redis Recovery Steps

### Redis Won't Start

```bash
# Check logs
docker-compose logs redis --tail 20

# Common: AOF file corrupted
docker exec parkir-redis redis-check-aof --fix /data/appendonly.aof

# Restart
docker-compose restart redis
```

### Redis Data Loss (Cache Miss Storm)

```bash
# Redis is running but empty (after restart without persistence)
# Services will repopulate cache on demand — monitor for:
# - Increased DB load (cache miss → DB query)
# - Temporary latency spike

# Pre-warm critical caches if needed:
# (Application handles this via cache-aside pattern)
# Monitor Grafana for cache hit rate recovery
```

### Redis Memory Emergency

```bash
# Check what's using memory
redis-cli info memory
redis-cli --bigkeys

# Emergency: flush only cache namespace (preserve locks/sessions)
redis-cli --scan --pattern "cache:spots:*" | xargs redis-cli del
redis-cli --scan --pattern "cache:billing:*" | xargs redis-cli del

# DO NOT flush reservation locks:
# redis-cli --scan --pattern "lock:reservation:*"  ← KEEP THESE
```

## Rollback Procedures

### Watchtower Auto-Deploy Rollback

ParkirPintar uses Watchtower for automatic container updates. To rollback:

```bash
# 1. Stop Watchtower to prevent re-deploying
docker-compose stop watchtower

# 2. Find previous image tag
docker images | grep parkir-pintar

# 3. Update docker-compose.yml to pin previous version
# Change: image: ghcr.io/parkir-pintar/reservation-service:latest
# To:     image: ghcr.io/parkir-pintar/reservation-service:sha-abc1234

# 4. Pull and restart with pinned version
docker-compose up -d <service-name>

# 5. Verify service is healthy
curl -s http://localhost:<port>/healthz

# 6. Once stable, investigate the bad deployment
# Check the failing image:
docker run --rm ghcr.io/parkir-pintar/<service>:latest --version

# 7. After fix is deployed, unpin and restart Watchtower
# Revert docker-compose.yml to :latest
docker-compose start watchtower
```

### Git-Based Rollback (if building locally)

```bash
# Find last known good commit
git log --oneline -10

# Checkout and rebuild
git checkout <good-commit>
docker-compose build <service-name>
docker-compose up -d <service-name>

# Return to main after fix
git checkout main
```

## Post-Incident Checklist

After resolving any P0 or P1 incident:

- [ ] Verify all services healthy (check Grafana)
- [ ] Confirm no data loss (spot-check DB records)
- [ ] Check NATS for unprocessed messages
- [ ] Review error rate has returned to baseline
- [ ] Write incident timeline (within 24h)
- [ ] Schedule post-mortem (within 48h)
- [ ] Create action items to prevent recurrence
- [ ] Update this runbook if new procedure was discovered
