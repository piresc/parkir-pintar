# Deployment Strategy — ParkirPintar

> **Status:** Approved  
> **Author:** piresc  
> **Date:** 2026-05-13  
> **Stack:** Go, Docker, Coolify (staging), GCP Cloud Run (production)

---

## 1. Overview

ParkirPintar uses a progressive deployment model with increasing rigor as changes move toward production:

```
Local Dev → CI (lint/test/build) → Staging (auto) → Production (manual tag)
```

| Environment | Strategy | Platform | Trigger |
|-------------|----------|----------|---------|
| Staging | Rolling update (Watchtower/Coolify) | Coolify on bare metal VM | Push to `main` |
| Production | Blue-green (Cloud Run revisions) | GCP Cloud Run | Git tag `v*.*.*` |

**Principle:** Build once, deploy many. The same Docker image (`ghcr.io/piresc/parkir-pintar:<tag>`) is promoted from staging to production without rebuilding.

---

## 2. Staging: Rolling Update via Coolify

### How It Works

1. Developer merges PR to `main`
2. GitHub Actions builds Docker image, pushes to GHCR with tag `main-<short-sha>`
3. Coolify webhook triggers pull of new image
4. Coolify performs rolling restart of containers
5. Health check (`GET /health/ready`) confirms readiness
6. Old container terminated after new one is healthy

### Configuration

```yaml
# Coolify application settings
image: ghcr.io/piresc/parkir-pintar:main-latest
health_check:
  path: /health/ready
  interval: 10s
  timeout: 5s
  retries: 3
  start_period: 30s
deploy:
  update_config:
    order: start-first    # New container starts before old stops
    failure_action: rollback
```

### Characteristics

- **Downtime:** Zero (start-first strategy)
- **Rollback speed:** ~30s (redeploy previous image via Coolify UI)
- **Risk:** Low (staging only, no real users)
- **Validation:** Automated health check + manual smoke testing

---

## 3. Production: Blue-Green via Cloud Run Revisions

### How It Works

Cloud Run natively supports blue-green deployments through its revision system:

1. Git tag (`v1.2.0`) triggers production CI pipeline
2. New Cloud Run revision deployed with `--no-traffic` flag
3. Health check validates new revision
4. Traffic shifted 100% to new revision (atomic cutover)
5. Previous revision kept warm for instant rollback

### Deployment Flow

```
┌─────────────────────────────────────────────────────────────┐
│                    Production Deploy                          │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. Build image (already done in CI)                         │
│  2. Tag image: ghcr.io/piresc/parkir-pintar:v1.2.0          │
│  3. Deploy new revision (no traffic)                         │
│  4. Run migrations (forward-only)                            │
│  5. Health check new revision                                │
│  6. Shift traffic: 0% → 100% (atomic)                       │
│  7. Monitor error rate for 5 minutes                         │
│  8. If errors > 1%: auto-rollback to previous revision       │
│  9. If healthy: mark deployment successful                   │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Cloud Run Commands

```bash
# Deploy new revision without traffic
gcloud run deploy parkir-pintar-gateway \
  --image=ghcr.io/piresc/parkir-pintar:v1.2.0 \
  --region=asia-southeast1 \
  --no-traffic \
  --tag=canary

# Verify health of new revision
curl -s https://canary---parkir-pintar-gateway-xxxxx.run.app/health/ready

# Shift all traffic to new revision
gcloud run services update-traffic parkir-pintar-gateway \
  --to-latest \
  --region=asia-southeast1

# Keep previous revision available for rollback
gcloud run services update-traffic parkir-pintar-gateway \
  --set-tags=previous=REVISION_NAME \
  --region=asia-southeast1
```

---

## 4. Canary Releases: Traffic Splitting

For high-risk changes (new payment provider, auth changes, schema migrations), use canary deployment with gradual traffic shifting:

### Traffic Split Strategy

```
Phase 1: 10% canary / 90% stable    (5 minutes, monitor)
Phase 2: 50% canary / 50% stable    (10 minutes, monitor)
Phase 3: 100% canary                 (full rollout)
```

### Implementation

```bash
# Phase 1: 10% traffic to new revision
gcloud run services update-traffic parkir-pintar-gateway \
  --to-revisions=parkir-pintar-gateway-v120=10,parkir-pintar-gateway-v119=90 \
  --region=asia-southeast1

# Monitor error rate during canary
# Alert if canary error rate > 2x baseline

# Phase 2: 50% traffic
gcloud run services update-traffic parkir-pintar-gateway \
  --to-revisions=parkir-pintar-gateway-v120=50,parkir-pintar-gateway-v119=50 \
  --region=asia-southeast1

# Phase 3: Full rollout
gcloud run services update-traffic parkir-pintar-gateway \
  --to-latest \
  --region=asia-southeast1
```

### Canary Criteria

| Metric | Threshold to Proceed | Threshold to Rollback |
|--------|---------------------|----------------------|
| Error rate (5xx) | < 1% | > 3% |
| Latency p95 | < 300ms | > 1000ms |
| Latency p99 | < 1000ms | > 3000ms |
| Business metric (reservations/min) | Within 20% of baseline | Drop > 50% |

### When to Use Canary

- Payment flow changes
- Authentication/authorization changes
- Database schema migrations affecting queries
- New external service integrations
- Major refactors of critical paths (reservation, billing)

---

## 5. Rollback Procedure

### Staging Rollback

**Time to rollback:** ~30 seconds

```bash
# Option 1: Via Coolify UI
# Deployments → Select previous deployment → Redeploy

# Option 2: Via Coolify API
curl -X POST "${COOLIFY_URL}/api/v1/applications/${APP_ID}/restart" \
  -H "Authorization: Bearer ${COOLIFY_TOKEN}" \
  -d '{"image": "ghcr.io/piresc/parkir-pintar:main-<previous-sha>"}'

# Option 3: Manual Docker (SSH to VM)
ssh deploy@staging-vm
docker pull ghcr.io/piresc/parkir-pintar:main-<previous-sha>
docker compose up -d --force-recreate
```

**Verify rollback:**
```bash
curl -s https://staging.parkir-pintar.piresc.dev/health | jq .version
# Should show previous version
```

### Production Rollback

**Time to rollback:** ~15 seconds (traffic shift is instant)

```bash
# Step 1: Identify previous healthy revision
gcloud run revisions list --service=parkir-pintar-gateway \
  --region=asia-southeast1 --limit=5

# Step 2: Route 100% traffic to previous revision
gcloud run services update-traffic parkir-pintar-gateway \
  --to-revisions=parkir-pintar-gateway-<previous>=100 \
  --region=asia-southeast1

# Step 3: Verify
curl -s https://parkir-pintar.piresc.dev/health | jq .version

# Step 4: If migration was applied, assess if rollback migration needed
# (All migrations must be backward-compatible, so usually not needed)
```

### Rollback Decision Tree

```
Is the issue affecting users?
├── YES → Immediate rollback (don't debug in production)
│   ├── Traffic shift to previous revision
│   ├── Notify team in #incidents channel
│   └── Debug using canary revision (tagged, no traffic)
└── NO → Investigate first
    ├── Check logs: gcloud logging read
    ├── Check traces: Cloud Trace
    └── If not resolvable in 5 min → rollback anyway
```

---

## 6. Deployment Checklist

### Pre-Deploy

- [ ] All CI checks pass (lint, test, security scan)
- [ ] Docker image built and pushed to GHCR
- [ ] Image tested in staging environment
- [ ] Database migrations tested (if any)
- [ ] Migration is backward-compatible (old code works with new schema)
- [ ] Feature flags configured for gradual rollout (if applicable)
- [ ] Changelog updated
- [ ] No active incidents on dependent services
- [ ] Team notified in deployment channel

### Deploy

- [ ] Create annotated git tag: `git tag -a v1.2.0 -m "Release 1.2.0: description"`
- [ ] Push tag: `git push origin v1.2.0`
- [ ] Monitor CI pipeline execution
- [ ] Verify migration job completes (if applicable)
- [ ] Confirm new revision deployed (no traffic yet)
- [ ] Health check passes on new revision
- [ ] Shift traffic (100% or canary split)
- [ ] Monitor error rate for 5 minutes

### Post-Deploy Verification

- [ ] Health endpoints return 200: `/health`, `/health/ready`, `/health/detailed`
- [ ] Version endpoint shows new version
- [ ] Key user flows work (create reservation, search, payment)
- [ ] Error rate stable (< 0.1%)
- [ ] Latency p95 within SLO (< 200ms)
- [ ] No new error patterns in logs
- [ ] Database connection pool healthy
- [ ] Redis connectivity confirmed
- [ ] NATS consumers processing events (no backlog)
- [ ] Notify team: deployment successful

### Post-Deploy (30 minutes)

- [ ] Review Cloud Monitoring dashboard
- [ ] Check for any delayed error patterns
- [ ] Confirm no memory leaks (stable RSS)
- [ ] Close deployment tracking issue/ticket

---

## 7. Feature Flags Strategy

### Purpose

Feature flags enable:
- Gradual rollout of new features without redeployment
- Quick disable of problematic features without rollback
- A/B testing of different implementations
- Decoupling deployment from release

### Implementation

Feature flags stored in Redis with fallback to environment variables:

```go
// pkg/featureflag/flag.go
type FeatureFlags struct {
    redis  *redis.Client
    prefix string
}

func (ff *FeatureFlags) IsEnabled(ctx context.Context, flag string, userID string) bool {
    // Check Redis first (dynamic flags)
    key := fmt.Sprintf("pp:flags:%s", flag)
    val, err := ff.redis.Get(ctx, key).Result()
    if err == nil {
        return ff.evaluateFlag(val, userID)
    }
    // Fallback to env var
    return os.Getenv(fmt.Sprintf("FF_%s", strings.ToUpper(flag))) == "true"
}
```

### Flag Types

| Type | Use Case | Storage | Example |
|------|----------|---------|---------|
| Boolean | On/off toggle | Redis / env var | `FF_NEW_PAYMENT_FLOW=true` |
| Percentage | Gradual rollout | Redis | `{"type":"percentage","value":25}` |
| User list | Beta testers | Redis | `{"type":"allowlist","users":["u1","u2"]}` |
| Environment | Env-specific | Env var | `FF_DEBUG_MODE` (staging only) |

### Current Feature Flags

| Flag | Type | Description | Default |
|------|------|-------------|---------|
| `new_payment_flow` | Percentage | Two-phase payment processing | 0% (disabled) |
| `push_notifications` | Boolean | Mobile push via FCM | false |
| `dynamic_pricing` | Percentage | Surge pricing algorithm | 0% |
| `v2_search` | User list | New search ranking | beta users only |

### Flag Lifecycle

```
Created → Testing (staging) → Gradual Rollout → Fully Enabled → Cleanup (remove flag)
```

Flags should not live longer than 30 days after full rollout. Stale flags are tech debt.

---

## 8. Database Migration Strategy

### Principles

1. **Forward-only in production** — never run `down` migrations in prod
2. **Backward-compatible** — old code must work with new schema
3. **Additive first** — add columns/tables before removing old ones
4. **Two-phase for destructive changes** — separate deploy from cleanup

### Migration Tool

Using `golang-migrate/migrate` with PostgreSQL driver:

```bash
# Run migrations
migrate -path db/migrations -database "$DATABASE_URL" up

# Check current version
migrate -path db/migrations -database "$DATABASE_URL" version
```

### Migration Execution

| Environment | When | How | Rollback |
|-------------|------|-----|----------|
| Local | On `docker compose up` | Auto via entrypoint | `migrate down N` |
| Staging | Pre-deploy hook | Coolify pre-deploy command | `migrate down N` |
| Production | Cloud Run Job (before app deploy) | Separate CI step | New forward migration |

### Two-Phase Migration Pattern

For breaking schema changes (rename column, change type, drop column):

**Phase 1 — Expand (Release N):**
```sql
-- Add new column, keep old one
ALTER TABLE reservations ADD COLUMN slot_id UUID;
-- Backfill new column from old
UPDATE reservations SET slot_id = parking_spot_id WHERE slot_id IS NULL;
-- Add trigger to keep both in sync
CREATE TRIGGER sync_slot_id ...
```

**Phase 2 — Contract (Release N+1, after all code uses new column):**
```sql
-- Remove old column (only after confirming no code references it)
ALTER TABLE reservations DROP COLUMN parking_spot_id;
DROP TRIGGER sync_slot_id ...
```

### Migration Safety Checks (CI)

```yaml
# .github/workflows/ci.yml
migration-check:
  steps:
    - name: Validate migrations
      run: |
        # Check all migrations have down files
        for up in db/migrations/*.up.sql; do
          down="${up/.up.sql/.down.sql}"
          if [ ! -f "$down" ]; then
            echo "Missing down migration: $down"
            exit 1
          fi
        done
    - name: Test migration roundtrip
      run: |
        migrate -path db/migrations -database "$TEST_DB_URL" up
        migrate -path db/migrations -database "$TEST_DB_URL" down
        migrate -path db/migrations -database "$TEST_DB_URL" up
    - name: Check backward compatibility
      run: |
        # Run old version's tests against new schema
        migrate -path db/migrations -database "$TEST_DB_URL" up
        go test ./... -tags=schema_compat
```

### Migration Naming Convention

```
{timestamp}_{description}.up.sql
{timestamp}_{description}.down.sql

Example:
20260513080000_add_slot_id_to_reservations.up.sql
20260513080000_add_slot_id_to_reservations.down.sql
```

---

## 9. Deployment Architecture Diagram

```
                    ┌─────────────────────────────────────────┐
                    │              GitHub Actions               │
                    │  lint → test → security → build → push   │
                    └──────────────┬──────────────┬────────────┘
                                   │              │
                          (main push)        (v*.*.* tag)
                                   │              │
                    ┌──────────────▼──┐    ┌──────▼──────────────┐
                    │    Staging       │    │    Production        │
                    │    (Coolify)     │    │    (Cloud Run)       │
                    │                  │    │                      │
                    │  ┌────────────┐  │    │  ┌───────────────┐  │
                    │  │  Traefik   │  │    │  │ Cloud LB/CDN  │  │
                    │  └─────┬──────┘  │    │  └───────┬───────┘  │
                    │        │         │    │          │           │
                    │  ┌─────▼──────┐  │    │  ┌───────▼───────┐  │
                    │  │  Gateway   │  │    │  │   Gateway     │  │
                    │  │  (Docker)  │  │    │  │ (Cloud Run)   │  │
                    │  └─────┬──────┘  │    │  └───────┬───────┘  │
                    │        │ gRPC    │    │          │ gRPC     │
                    │  ┌─────▼──────┐  │    │  ┌───────▼───────┐  │
                    │  │ Services   │  │    │  │  Services     │  │
                    │  │ (Docker)   │  │    │  │ (Cloud Run)   │  │
                    │  └────────────┘  │    │  └───────────────┘  │
                    │                  │    │                      │
                    │  PostgreSQL      │    │  Cloud SQL           │
                    │  Redis           │    │  Memorystore         │
                    │  NATS            │    │  NATS (GKE)          │
                    └──────────────────┘    └──────────────────────┘
```
