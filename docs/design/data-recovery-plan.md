# Data Recovery Plan — ParkirPintar

## Overview

This document defines backup strategies, recovery objectives, and disaster recovery procedures for all ParkirPintar data stores.

## Data Stores Inventory

| Store | Purpose | Data Criticality | Backup Method |
|-------|---------|-----------------|---------------|
| PostgreSQL | Reservations, users, billing, spots | Critical | pg_dump daily |
| Redis | Cache, session, distributed locks | Medium | RDB snapshots hourly |
| NATS JetStream | Event streams, async messages | High | File-based stream storage + replay |
| Local filesystem | Configs, logs | Low | Git (configs), rotation (logs) |

## Recovery Objectives

| Metric | Target | Notes |
|--------|--------|-------|
| **RPO** (Recovery Point Objective) — PostgreSQL | 24 hours | Maximum acceptable data loss |
| **RPO** — Redis | 1 hour | Cache can be rebuilt; locks are transient |
| **RPO** — NATS JetStream | 0 (durable) | Messages persisted to disk, replay available |
| **RTO** (Recovery Time Objective) | 4 hours | Time to restore full service |

## Backup Strategy

### PostgreSQL — Daily Full Backup

**Schedule**: Daily at 02:00 WIB (low traffic period)

**Method**: `pg_dump` with compression

```bash
#!/bin/bash
# /opt/parkir-pintar/scripts/backup-postgres.sh

BACKUP_DIR="/backups/postgres"
DB_NAME="parkir_pintar"
DB_USER="parkir"
RETENTION_DAYS=30
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

mkdir -p ${BACKUP_DIR}

# Full database dump with compression
docker exec parkir-postgres pg_dump \
  -U ${DB_USER} \
  -d ${DB_NAME} \
  --format=custom \
  --compress=9 \
  --file=/tmp/backup_${TIMESTAMP}.dump

# Copy from container
docker cp parkir-postgres:/tmp/backup_${TIMESTAMP}.dump \
  ${BACKUP_DIR}/parkir_pintar_${TIMESTAMP}.dump

# Cleanup old backups
find ${BACKUP_DIR} -name "*.dump" -mtime +${RETENTION_DAYS} -delete

# Verify backup integrity
pg_restore --list ${BACKUP_DIR}/parkir_pintar_${TIMESTAMP}.dump > /dev/null 2>&1
if [ $? -eq 0 ]; then
  echo "$(date): Backup successful - parkir_pintar_${TIMESTAMP}.dump" >> /var/log/backup.log
else
  echo "$(date): BACKUP VERIFICATION FAILED" >> /var/log/backup.log
  # Send alert
  curl -s -X POST "http://localhost:9093/api/v1/alerts" \
    -H "Content-Type: application/json" \
    -d '[{"labels":{"alertname":"BackupFailed","severity":"P1"}}]'
fi
```

**Cron entry:**
```cron
0 2 * * * /opt/parkir-pintar/scripts/backup-postgres.sh
```

### Redis — Hourly RDB Snapshots

**Schedule**: Every hour via Redis `save` configuration

**Redis config:**
```
save 3600 1
save 300 100
save 60 10000
dir /data
dbfilename dump.rdb
```

**Backup script** (copies RDB to backup location):
```bash
#!/bin/bash
# /opt/parkir-pintar/scripts/backup-redis.sh

BACKUP_DIR="/backups/redis"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
RETENTION_HOURS=72

mkdir -p ${BACKUP_DIR}

# Trigger RDB save
docker exec parkir-redis redis-cli BGSAVE
sleep 5

# Copy RDB file
docker cp parkir-redis:/data/dump.rdb ${BACKUP_DIR}/redis_${TIMESTAMP}.rdb

# Cleanup old snapshots (keep 72 hours)
find ${BACKUP_DIR} -name "*.rdb" -mmin +$((RETENTION_HOURS * 60)) -delete

echo "$(date): Redis backup complete - redis_${TIMESTAMP}.rdb" >> /var/log/backup.log
```

**Cron entry:**
```cron
5 * * * * /opt/parkir-pintar/scripts/backup-redis.sh
```

### NATS JetStream — Persistent Storage

- JetStream stores messages on disk at `/data/nats/jetstream/`
- Messages are retained based on stream configuration:
  - `PARKIR_EVENTS`: max age 7 days, max bytes 1GB
  - `PARKIR_BILLING`: max age 30 days (financial audit trail)
- Recovery via stream replay (consumers can rewind to any retained message)

## Backup Verification Process

### Automated Verification (Weekly)

```bash
#!/bin/bash
# /opt/parkir-pintar/scripts/verify-backup.sh

# 1. Restore latest backup to test database
LATEST_BACKUP=$(ls -t /backups/postgres/*.dump | head -1)

docker exec parkir-postgres createdb -U parkir parkir_test_restore 2>/dev/null
docker exec parkir-postgres pg_restore \
  -U parkir \
  -d parkir_test_restore \
  --clean --if-exists \
  /backups/postgres/$(basename ${LATEST_BACKUP})

# 2. Run integrity checks
RESERVATION_COUNT=$(docker exec parkir-postgres psql -U parkir -d parkir_test_restore \
  -t -c "SELECT count(*) FROM reservations;")

USER_COUNT=$(docker exec parkir-postgres psql -U parkir -d parkir_test_restore \
  -t -c "SELECT count(*) FROM users;")

# 3. Compare with production counts (should be close)
PROD_RESERVATIONS=$(docker exec parkir-postgres psql -U parkir -d parkir_pintar \
  -t -c "SELECT count(*) FROM reservations;")

# 4. Cleanup
docker exec parkir-postgres dropdb -U parkir parkir_test_restore

# 5. Report
echo "$(date): Backup verification complete" >> /var/log/backup.log
echo "  Backup reservations: ${RESERVATION_COUNT}" >> /var/log/backup.log
echo "  Production reservations: ${PROD_RESERVATIONS}" >> /var/log/backup.log
```

**Cron entry:**
```cron
0 4 * * 0 /opt/parkir-pintar/scripts/verify-backup.sh
```

### Manual Verification (Monthly)

1. Restore backup to a separate environment
2. Run application smoke tests against restored data
3. Verify all foreign key relationships intact
4. Check that recent transactions are present
5. Document results in operations log

## Disaster Recovery Scenarios

### Scenario 1: Single Service Failure

**Impact**: One microservice crashes or becomes unresponsive
**RPO**: 0 (no data loss — data in PostgreSQL/NATS)
**RTO**: 5 minutes

**Recovery:**
1. Docker auto-restarts the container (restart policy: `unless-stopped`)
2. If persistent crash: check logs, rollback to previous image
3. Service reconnects to DB/Redis/NATS automatically
4. NATS JetStream replays unacknowledged messages

```bash
# Verify auto-recovery
docker-compose ps <service-name>

# If not recovering:
docker-compose up -d --force-recreate <service-name>
```

---

### Scenario 2: Database Corruption

**Impact**: PostgreSQL data corrupted or inconsistent
**RPO**: Up to 24 hours (last backup)
**RTO**: 2 hours

**Recovery:**
1. Stop all application services immediately:
   ```bash
   docker-compose stop reservation-service billing-service notification-service api-gateway
   ```

2. Assess corruption scope:
   ```sql
   -- Check for corruption
   SELECT schemaname, relname, last_analyze, n_dead_tup
   FROM pg_stat_user_tables;
   
   -- Run integrity check
   SELECT * FROM pg_catalog.pg_class WHERE relkind = 'r';
   ```

3. If minor corruption (single table):
   ```bash
   # Try REINDEX
   docker exec parkir-postgres psql -U parkir -d parkir_pintar \
     -c "REINDEX TABLE <corrupted_table>;"
   ```

4. If major corruption (restore required):
   ```bash
   # Drop and recreate database
   docker exec parkir-postgres dropdb -U parkir parkir_pintar
   docker exec parkir-postgres createdb -U parkir parkir_pintar
   
   # Restore from latest backup
   LATEST=$(ls -t /backups/postgres/*.dump | head -1)
   docker exec parkir-postgres pg_restore \
     -U parkir -d parkir_pintar ${LATEST}
   
   # Run pending migrations
   migrate -path ./migrations -database "postgres://..." up
   ```

5. Replay NATS events for data created after backup:
   ```bash
   # Identify gap period
   # Replay events from JetStream starting from backup timestamp
   nats consumer add PARKIR_EVENTS recovery-consumer \
     --deliver=by-start-time \
     --start-time="2025-01-15T02:00:00Z"
   ```

6. Restart application services
7. Verify data integrity with spot checks

---

### Scenario 3: Full Server Loss

**Impact**: Complete server failure (hardware, VM deletion, etc.)
**RPO**: Up to 24 hours (last off-site backup)
**RTO**: 4 hours

**Recovery:**

1. **Provision new server** (same specs or better)

2. **Install prerequisites:**
   ```bash
   apt update && apt install -y docker.io docker-compose
   ```

3. **Restore configuration from Git:**
   ```bash
   git clone <repo-url> /opt/parkir-pintar
   cd /opt/parkir-pintar
   ```

4. **Restore secrets/env files** (from secure backup or secrets manager):
   ```bash
   # Copy .env files from secure storage
   scp backup-server:/secure/parkir-pintar/.env /opt/parkir-pintar/.env
   ```

5. **Restore PostgreSQL data:**
   ```bash
   # Start only PostgreSQL
   docker-compose up -d postgres
   sleep 10
   
   # Restore from off-site backup
   scp backup-server:/backups/postgres/latest.dump /tmp/
   docker cp /tmp/latest.dump parkir-postgres:/tmp/
   docker exec parkir-postgres pg_restore \
     -U parkir -d parkir_pintar /tmp/latest.dump
   ```

6. **Restore Redis (optional — cache rebuilds):**
   ```bash
   scp backup-server:/backups/redis/latest.rdb /tmp/
   docker cp /tmp/latest.rdb parkir-redis:/data/dump.rdb
   ```

7. **Start all services:**
   ```bash
   docker-compose up -d
   ```

8. **Verify:**
   - All containers running: `docker-compose ps`
   - Health checks passing: `curl localhost:<port>/healthz`
   - Data present: spot-check DB records
   - Monitoring reconnected: check Grafana

---

### Scenario 4: Redis Complete Loss

**Impact**: All cached data lost, distributed locks lost
**RPO**: 1 hour (RDB snapshot)
**RTO**: 30 minutes

**Recovery:**
1. Restart Redis (cache rebuilds on demand):
   ```bash
   docker-compose restart redis
   ```
2. If RDB restore needed:
   ```bash
   docker-compose stop redis
   LATEST=$(ls -t /backups/redis/*.rdb | head -1)
   docker cp ${LATEST} parkir-redis:/data/dump.rdb
   docker-compose start redis
   ```
3. Monitor cache hit rate recovery in Grafana
4. Watch for reservation lock conflicts (transient, self-resolving)

---

### Scenario 5: NATS JetStream Data Loss

**Impact**: Undelivered events lost, potential data inconsistency
**RPO**: 0 if JetStream storage intact; variable if storage lost
**RTO**: 1 hour

**Recovery:**
1. Restart NATS:
   ```bash
   docker-compose restart nats
   ```
2. Verify streams exist:
   ```bash
   nats stream ls
   ```
3. If streams lost, recreate:
   ```bash
   nats stream add PARKIR_EVENTS \
     --subjects="parkir.>" \
     --retention=limits \
     --max-age=168h \
     --storage=file
   ```
4. Recreate consumers for each service
5. Run data consistency check between services:
   ```bash
   # Compare reservation count with billing records
   # Identify orphaned records
   # Trigger reconciliation job
   ```

## Data Retention Policy

| Data Type | Retention Period | Reason |
|-----------|-----------------|--------|
| Active reservations | Indefinite | Business data |
| Completed reservations | 2 years | Audit/dispute resolution |
| Cancelled reservations | 6 months | Analytics |
| Billing records | 5 years | Tax/legal compliance |
| User accounts | Until deletion request + 30 days | GDPR-like compliance |
| Audit logs | 1 year | Security compliance |
| Application logs | 30 days | Debugging |
| Metrics (Prometheus) | 90 days | Trend analysis |
| Traces (Tempo) | 7 days | Debugging |
| PostgreSQL backups | 30 days | Recovery window |
| Redis snapshots | 72 hours | Cache recovery |
| NATS messages | 7 days (events), 30 days (billing) | Event replay |

## NATS JetStream Replay for Event Recovery

When data inconsistency is detected between services (e.g., reservation exists but billing record missing), use JetStream replay:

### Replay Procedure

```bash
# 1. Identify the gap (e.g., billing missed events from 14:00-14:30)

# 2. Create a temporary replay consumer starting from the gap
nats consumer add PARKIR_EVENTS billing-replay \
  --deliver=by-start-time \
  --start-time="2025-01-15T14:00:00+07:00" \
  --filter="parkir.reservation.confirmed" \
  --ack-explicit \
  --max-deliver=1

# 3. Run the billing service replay handler
# (Application has a dedicated replay mode that processes without side effects like notifications)
docker exec billing-service /app/billing replay \
  --consumer=billing-replay \
  --dry-run  # First verify what would be processed

docker exec billing-service /app/billing replay \
  --consumer=billing-replay

# 4. Verify reconciliation
docker exec parkir-postgres psql -U parkir -d parkir_pintar -c "
  SELECT r.id, r.status, b.id as billing_id
  FROM reservations r
  LEFT JOIN billing_records b ON b.reservation_id = r.id
  WHERE r.created_at > '2025-01-15 14:00:00'
  AND b.id IS NULL;
"

# 5. Cleanup replay consumer
nats consumer rm PARKIR_EVENTS billing-replay
```

### Automatic Reconciliation

A scheduled job runs daily to detect and fix inconsistencies:

```bash
# Cron: 0 3 * * * /opt/parkir-pintar/scripts/reconcile.sh
# Checks:
# - Every confirmed reservation has a billing record
# - Every completed reservation has a payment record
# - Spot availability matches reservation state
# Reports discrepancies to monitoring
```

## Off-Site Backup (Future Enhancement)

Current backups are stored on the same server. Planned improvements:

1. **S3-compatible storage** (MinIO or cloud S3) for off-site copies
2. **Encrypted backups** using `age` or GPG before transfer
3. **Cross-region replication** for disaster recovery
4. **Automated restore testing** in CI pipeline

## Revision History

| Date | Change | Author |
|------|--------|--------|
| 2025-01-15 | Initial data recovery plan | ParkirPintar Team |
