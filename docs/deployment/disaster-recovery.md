# Disaster Recovery Plan — ParkirPintar

> **Status:** Approved  
> **Author:** piresc  
> **Date:** 2026-05-13  
> **Stack:** Go, Docker, Cloud Run, PostgreSQL (Cloud SQL), Redis (Memorystore), NATS, Traefik  
> **Last DR Test:** TBD  
> **Next DR Test:** 2026-08-13 (quarterly)

---

## 1. Recovery Objectives

| Objective | Target | Justification |
|-----------|--------|---------------|
| **RPO** (Recovery Point Objective) | 1 hour | Maximum acceptable data loss. WAL archiving every 5 min + hourly snapshots ensure < 1h loss. |
| **RTO** (Recovery Time Objective) | 15 minutes | Maximum acceptable downtime. Cloud Run instant failover + automated recovery scripts. |

### Tiered Recovery Targets

| Tier | Services | RPO | RTO | Priority |
|------|----------|-----|-----|----------|
| Tier 1 (Critical) | Gateway, Reservation, Payment | 15 min | 5 min | Highest |
| Tier 2 (Important) | Search, Billing, Presence | 1 hour | 15 min | High |
| Tier 3 (Non-critical) | Notification | 4 hours | 30 min | Medium |

---

## 2. Backup Strategy

### 2.1 PostgreSQL Backups

#### WAL Archiving (Continuous)

```
PostgreSQL → WAL segments → Cloud Storage (every 5 minutes)
```

**Cloud SQL Configuration:**
- Automated backups: enabled
- Backup window: 02:00-03:00 UTC (low traffic)
- Transaction log retention: 7 days
- Point-in-time recovery (PITR): enabled

**For self-managed PostgreSQL (staging):**

```bash
# postgresql.conf
archive_mode = on
archive_command = 'gsutil cp %p gs://parkir-pintar-wal-archive/%f'
wal_level = replica

# Continuous archiving via pg_basebackup
pg_basebackup -D /backup/base -Ft -z -P --wal-method=stream
```

#### Daily Snapshots

| Backup Type | Frequency | Retention | Storage |
|-------------|-----------|-----------|---------|
| WAL archive | Continuous (5 min) | 7 days | Cloud Storage (Nearline) |
| Automated snapshot | Daily (02:00 UTC) | 30 days | Cloud SQL automated |
| Manual snapshot | Before major releases | 90 days | Cloud SQL on-demand |
| Full export (pg_dump) | Weekly (Sunday 03:00) | 90 days | Cloud Storage (Coldline) |

#### Backup Verification

```bash
# Weekly automated restore test (Cloud Run Job)
#!/bin/bash
# 1. Restore latest backup to test instance
gcloud sql instances clone parkir-pintar-prod parkir-pintar-restore-test \
  --point-in-time="$(date -u -d '1 hour ago' +%Y-%m-%dT%H:%M:%S.000Z)"

# 2. Run integrity checks
psql "$RESTORE_TEST_URL" -c "SELECT count(*) FROM reservations;"
psql "$RESTORE_TEST_URL" -c "SELECT count(*) FROM payments;"

# 3. Run application health check against restored DB
# 4. Delete test instance
gcloud sql instances delete parkir-pintar-restore-test --quiet
```

### 2.2 Redis Backups

| Strategy | Frequency | Retention | Notes |
|----------|-----------|-----------|-------|
| RDB snapshot | Every 15 min | 24 hours | Memorystore automatic |
| Export to GCS | Daily | 7 days | Manual export job |

**Recovery approach:** Redis data is ephemeral (cache, locks, rate limits). On failure, services reconnect and rebuild cache organically. No restore needed — cold cache is acceptable.

**Exception:** If using Redis for session storage, sessions will be lost. Users re-authenticate (acceptable given JWT with 24h expiry).

### 2.3 NATS JetStream Backups

| Strategy | Frequency | Retention | Notes |
|----------|-----------|-----------|-------|
| Stream snapshots | Every 6 hours | 7 days | `nats stream backup` |
| Persistent volume backup | Daily | 30 days | GKE volume snapshot |

**Recovery approach:** NATS streams are replayed from the last snapshot. Consumers track their position via durable subscriptions. Missed events during outage are replayed on recovery.

```bash
# Backup NATS streams
nats stream backup RESERVATIONS /backup/nats/reservations-$(date +%Y%m%d)
nats stream backup BILLING /backup/nats/billing-$(date +%Y%m%d)
nats stream backup NOTIFICATIONS /backup/nats/notifications-$(date +%Y%m%d)

# Restore
nats stream restore RESERVATIONS /backup/nats/reservations-20260513
```

---

## 3. Multi-Region Failover Plan

### Current Architecture (Single Region)

```
Primary: asia-southeast1 (Jakarta)
├── Cloud Run services (all 7)
├── Cloud SQL (PostgreSQL)
├── Memorystore (Redis)
└── NATS (GKE pod)
```

### Failover Architecture (Target)

```
Primary: asia-southeast1 (Jakarta)          Failover: asia-southeast2 (Singapore)
├── Cloud Run services (active)              ├── Cloud Run services (standby)
├── Cloud SQL (primary)                      ├── Cloud SQL (read replica, promotable)
├── Memorystore (primary)                    ├── Memorystore (new instance on failover)
└── NATS (primary cluster)                   └── NATS (mirror cluster)
         │                                            ▲
         └────── Cross-region replication ────────────┘
```

### Failover Triggers

| Trigger | Detection | Action |
|---------|-----------|--------|
| Region outage | Cloud Monitoring uptime check fails 3x | Automatic DNS failover |
| Database failure | Cloud SQL health check fails | Promote read replica |
| Network partition | Cross-region ping fails > 5 min | Manual failover decision |
| Sustained high error rate | > 10% 5xx for > 5 min | Manual failover decision |

### Failover Procedure

```
┌─────────────────────────────────────────────────────────────┐
│                   FAILOVER RUNBOOK                            │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. DETECT: Automated alert fires (PagerDuty/Telegram)       │
│  2. ASSESS: Confirm outage is regional (not app bug)         │
│  3. DECIDE: On-call engineer approves failover               │
│  4. EXECUTE:                                                 │
│     a. Promote Cloud SQL read replica to primary             │
│        gcloud sql instances promote-replica \                │
│          parkir-pintar-replica --quiet                        │
│     b. Deploy Cloud Run services in failover region          │
│        gcloud run deploy parkir-pintar-gateway \             │
│          --region=asia-southeast2 \                          │
│          --image=ghcr.io/piresc/parkir-pintar:latest         │
│     c. Update DNS to point to failover region                │
│        gcloud dns record-sets update parkir-pintar \         │
│          --zone=piresc-dev --type=A \                        │
│          --rrdatas=<failover-ip>                             │
│     d. Spin up new Redis instance (cold cache acceptable)    │
│     e. Start NATS from latest backup                         │
│  5. VERIFY: Health checks pass in failover region            │
│  6. COMMUNICATE: Status page updated, team notified          │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### DNS Failover (Global Load Balancer)

```bash
# Using Cloud DNS health-checked routing
gcloud dns record-sets create parkir-pintar.piresc.dev \
  --zone=piresc-dev \
  --type=A \
  --routing-policy-type=FAILOVER \
  --routing-policy-primary-data=<primary-ip> \
  --routing-policy-backup-data=<failover-ip> \
  --health-check=parkir-pintar-health
```

---

## 4. Data Recovery Procedures

### 4.1 Point-in-Time Recovery (PITR)

**Scenario:** Accidental data deletion or corruption.

```bash
# Step 1: Identify the timestamp before the incident
# Check Cloud Logging for the destructive operation
gcloud logging read 'resource.type="cloudsql_database" AND textPayload:"DELETE"' \
  --limit=50 --format=json

# Step 2: Restore to point-in-time
gcloud sql instances clone parkir-pintar-prod parkir-pintar-recovered \
  --point-in-time="2026-05-13T14:30:00.000Z"

# Step 3: Verify recovered data
psql "$RECOVERED_DB_URL" -c "SELECT count(*) FROM reservations WHERE deleted_at IS NULL;"

# Step 4: Promote recovered instance (or export/import specific tables)
# Option A: Replace primary with recovered instance
gcloud sql instances patch parkir-pintar-prod --activation-policy=NEVER
gcloud sql instances patch parkir-pintar-recovered --activation-policy=ALWAYS

# Option B: Export specific tables and import to primary
pg_dump -h recovered-host -t reservations --data-only | psql -h primary-host
```

### 4.2 Full Database Restore

**Scenario:** Complete database loss.

```bash
# From automated backup
gcloud sql backups restore BACKUP_ID \
  --restore-instance=parkir-pintar-prod \
  --backup-instance=parkir-pintar-prod

# From weekly pg_dump (Cloud Storage)
gsutil cp gs://parkir-pintar-backups/weekly/parkir_pintar_20260512.sql.gz .
gunzip parkir_pintar_20260512.sql.gz
psql "$DATABASE_URL" < parkir_pintar_20260512.sql

# Then replay WAL to catch up
# (Handled automatically by Cloud SQL PITR)
```

### 4.3 Service Recovery (Cloud Run)

**Scenario:** Service crashes or becomes unresponsive.

```bash
# Cloud Run auto-heals by default (restarts unhealthy instances)
# Manual intervention only if all instances fail:

# Redeploy latest known-good version
gcloud run deploy parkir-pintar-gateway \
  --image=ghcr.io/piresc/parkir-pintar:v1.1.0 \
  --region=asia-southeast1

# Or route to previous revision
gcloud run services update-traffic parkir-pintar-gateway \
  --to-revisions=parkir-pintar-gateway-00042=100 \
  --region=asia-southeast1
```

### 4.4 NATS Recovery

**Scenario:** NATS cluster failure, message loss.

```bash
# Step 1: Restart NATS with persistent storage
kubectl rollout restart statefulset/nats -n parkir-pintar

# Step 2: If storage lost, restore from backup
nats stream restore RESERVATIONS /backup/nats/reservations-latest

# Step 3: Reset consumer positions if needed
nats consumer rm RESERVATIONS notification-consumer
nats consumer add RESERVATIONS notification-consumer \
  --deliver=last --ack=explicit --max-deliver=5

# Step 4: Verify streams are healthy
nats stream info RESERVATIONS
nats stream info BILLING
```

### 4.5 Redis Recovery

**Scenario:** Redis instance failure.

```bash
# Memorystore auto-recovers for Standard tier (HA)
# For Basic tier: create new instance, app reconnects automatically

# If manual recovery needed:
gcloud redis instances create parkir-pintar-redis-new \
  --size=1 --region=asia-southeast1 --tier=basic

# Update Cloud Run env vars to point to new instance
gcloud run services update parkir-pintar-gateway \
  --set-env-vars="REDIS_ADDR=<new-redis-ip>:6379" \
  --region=asia-southeast1

# Cache rebuilds organically from database queries
# Distributed locks re-acquired on next operation
```

---

## 5. DR Testing Schedule

### Quarterly DR Drill (Every 3 Months)

| Quarter | Test Type | Scope | Duration |
|---------|-----------|-------|----------|
| Q1 | Backup restore | Restore DB from backup, verify data integrity | 2 hours |
| Q2 | Service failover | Kill primary services, verify auto-recovery | 1 hour |
| Q3 | Full region failover | Simulate region outage, execute failover runbook | 4 hours |
| Q4 | Chaos engineering | Random failure injection (services, DB, Redis, NATS) | 2 hours |

### DR Test Checklist

```markdown
## DR Test Report — [DATE]

### Test Parameters
- **Type:** [Backup Restore / Service Failover / Region Failover / Chaos]
- **Date:** YYYY-MM-DD
- **Duration:** X hours
- **Participants:** [names]

### Execution
- [ ] Pre-test: Notify stakeholders
- [ ] Pre-test: Confirm monitoring is active
- [ ] Execute failure scenario
- [ ] Measure detection time (time to alert)
- [ ] Measure recovery time (time to healthy)
- [ ] Verify data integrity post-recovery
- [ ] Verify no data loss beyond RPO
- [ ] Document any issues encountered

### Results
- **Detection time:** X minutes (target: < 2 min)
- **Recovery time:** X minutes (target: < 15 min)
- **Data loss:** X minutes (target: < 60 min)
- **Issues found:** [list]

### Action Items
- [ ] [Issue] → [Fix] → [Owner] → [Due date]
```

### Automated DR Validation (Monthly)

```yaml
# Cloud Scheduler job — runs monthly
name: dr-validation
schedule: "0 3 1 * *"  # 1st of every month, 03:00 UTC
steps:
  - restore_backup_to_test_instance
  - run_data_integrity_checks
  - run_application_smoke_tests
  - cleanup_test_instance
  - report_results_to_slack
```

---

## 6. Communication Plan During DR Events

### Severity Levels

| Level | Definition | Example | Response |
|-------|-----------|---------|----------|
| SEV-1 | Complete service outage | All APIs returning 5xx | Immediate failover |
| SEV-2 | Partial degradation | One service down, others functional | Investigate + fix |
| SEV-3 | Minor issue | Elevated latency, no errors | Monitor + fix in hours |

### Communication Timeline

```
T+0 min:  Alert fires (automated)
T+2 min:  On-call acknowledges
T+5 min:  Initial assessment posted to #incidents
T+10 min: Status page updated (if user-facing)
T+15 min: Stakeholder notification (if SEV-1)
T+30 min: First update (progress, ETA)
T+60 min: Hourly updates until resolved
T+resolve: Resolution notification
T+24h:    Post-mortem scheduled
T+72h:    Post-mortem published
```

### Communication Channels

| Audience | Channel | Trigger |
|----------|---------|---------|
| Engineering team | Telegram #incidents | All SEV levels |
| On-call engineer | PagerDuty alert | SEV-1, SEV-2 |
| Stakeholders | Email + Telegram #status | SEV-1 |
| End users | Status page (statuspage.io) | SEV-1, extended SEV-2 |

### Status Page Updates Template

```
[INVESTIGATING] We are investigating reports of [issue description].
[IDENTIFIED] The issue has been identified as [root cause]. We are working on a fix.
[MONITORING] A fix has been deployed. We are monitoring the situation.
[RESOLVED] The issue has been resolved. [Brief summary of impact and duration].
```

### Escalation Matrix

| Time Since Alert | Action | Who |
|-----------------|--------|-----|
| 0-5 min | Acknowledge + assess | On-call engineer |
| 5-15 min | Begin recovery | On-call engineer |
| 15-30 min | Escalate if not resolved | Engineering lead |
| 30-60 min | Executive notification | CTO/Product lead |
| > 60 min | All-hands if needed | Full engineering team |

---

## 7. Disaster Scenarios and Playbooks

### Scenario 1: Database Corruption

```
Detection: Application errors "relation does not exist" or data inconsistency
Response:
1. Immediately switch to read-only mode (feature flag: maintenance_mode=true)
2. Identify corruption scope (which tables, when it started)
3. PITR restore to timestamp before corruption
4. Verify data integrity
5. Switch back to read-write mode
6. Post-mortem: identify root cause (bad migration? disk failure?)
```

### Scenario 2: Credential Compromise

```
Detection: Unusual API activity, unauthorized access in logs
Response:
1. Rotate ALL secrets immediately (JWT secret, API keys, DB passwords)
2. Invalidate all active sessions (flush Redis session store)
3. Deploy with new secrets (Cloud Run env var update)
4. Audit access logs for scope of compromise
5. Notify affected users if data accessed
6. Post-mortem: identify how credentials leaked
```

### Scenario 3: DDoS Attack

```
Detection: Sudden traffic spike, rate limiter overwhelmed
Response:
1. Enable Cloud Armor WAF rules (if not already active)
2. Tighten rate limits via feature flag (RATE_LIMIT_RPS=10)
3. Enable geographic blocking if attack is region-specific
4. Scale up Gateway instances (increase max to 20)
5. If sustained: enable Cloudflare proxy (DNS change)
6. Post-mortem: review rate limiting strategy
```

### Scenario 4: Cloud Provider Outage

```
Detection: GCP status page shows incident in asia-southeast1
Response:
1. Confirm via independent monitoring (external uptime check)
2. If < 15 min expected: wait (GCP SLA covers this)
3. If > 15 min or unknown: execute region failover
4. Communicate to users via status page
5. Monitor GCP status for resolution
6. Failback when primary region recovers
```

---

## 8. Recovery Validation Checklist

After any DR event, verify full system health:

- [ ] All 7 services responding to health checks
- [ ] Database connections established (check pool metrics)
- [ ] Redis connectivity confirmed
- [ ] NATS streams operational, consumers active
- [ ] No message backlog in NATS
- [ ] API responses returning correct data
- [ ] Authentication working (JWT validation)
- [ ] Rate limiting functional
- [ ] Distributed locks acquirable
- [ ] OpenTelemetry traces flowing
- [ ] No data integrity issues (run consistency checks)
- [ ] Monitoring and alerting active
- [ ] DNS resolving correctly
- [ ] SSL certificates valid
