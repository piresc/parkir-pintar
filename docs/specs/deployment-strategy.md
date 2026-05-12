# Deployment Strategy Specification — parkir-pintar

> **Status:** Draft
> **Author:** piresc
> **Date:** 2026-05-12
> **Domain:** piresc.dev

---

## 1. Overview

Artifact-based deployment with two environments:

| Environment | Trigger | Platform | Domain |
|-------------|---------|----------|--------|
| **Staging** | Push to `main` | Coolify (bare metal VM) | `staging.parkir-pintar.piresc.dev` |
| **Production** | Git tag (`v*.*.*`) | GCP (Cloud Run / GKE) | `parkir-pintar.piresc.dev` |

**Principle:** Build once, deploy many. The same Docker image promoted from staging to production.

---

## 2. Domain & Routing

### DNS Layout

```
piresc.dev
├── parkir-pintar.piresc.dev          → Production (GCP)
├── staging.parkir-pintar.piresc.dev  → Staging (Coolify / VM)
└── (future) admin.parkir-pintar.piresc.dev → Admin panel
```

### URL Structure

```
https://parkir-pintar.piresc.dev/v1/reservations
https://parkir-pintar.piresc.dev/v1/availability
https://parkir-pintar.piresc.dev/v1/payments
https://parkir-pintar.piresc.dev/health

https://staging.parkir-pintar.piresc.dev/v1/reservations
https://staging.parkir-pintar.piresc.dev/health
```

**Why subdomains over path-based (`/staging/v1/...`):**
- API code is environment-agnostic (no path prefix logic)
- Independent SSL certificates (or wildcard `*.parkir-pintar.piresc.dev`)
- DNS can point to completely different infrastructure
- No confusion between environment prefix and API versioning
- Easier to restrict staging access (IP allowlist, basic auth)

### SSL/TLS

- **Staging:** Let's Encrypt via Coolify (auto-managed)
- **Production:** Google-managed SSL certificate (Cloud Run) or Let's Encrypt (GKE + cert-manager)
- **Wildcard option:** `*.parkir-pintar.piresc.dev` covers both

---

## 3. Artifact Strategy

### Docker Image

```
ghcr.io/piresc/parkir-pintar:<tag>
```

**Tagging convention:**

| Trigger | Image Tag | Example |
|---------|-----------|---------|
| Push to main | `main-<short-sha>` | `main-2c71753` |
| Git tag | `<semver>` + `latest` | `v1.2.0`, `latest` |

### Build Matrix

Single multi-service image (current architecture):

```dockerfile
# Built once in CI
FROM golang:1.23-alpine AS builder
# ... builds all 7 service binaries

FROM alpine:3.20 AS runtime
COPY --from=builder /build/bin/gateway /app/gateway
COPY --from=builder /build/bin/reservation /app/reservation
COPY --from=builder /build/bin/billing /app/billing
COPY --from=builder /build/bin/payment /app/payment
COPY --from=builder /build/bin/search /app/search
COPY --from=builder /build/bin/presence /app/presence
COPY --from=builder /build/bin/notification /app/notification
COPY --from=builder /build/db/migrations /app/migrations
```

**Future consideration:** Per-service images for independent scaling. Not needed now.

### Artifact Promotion Flow

```
┌─────────┐     ┌──────────┐     ┌────────────┐
│  Build  │────▶│ Staging  │────▶│ Production │
│  (CI)   │     │ (Coolify)│     │   (GCP)    │
└─────────┘     └──────────┘     └────────────┘
     │                │                  │
     │  main-abc123   │  main-abc123     │  v1.2.0
     │  (GHCR)        │  (auto-deploy)   │  (manual tag)
```

**Key rule:** Production ONLY deploys images that have been validated in staging.

---

## 4. CI/CD Pipeline

### Pipeline Stages

```yaml
# Triggered on: push to main, pull_request, tags
stages:
  - lint        # golangci-lint
  - test        # unit + race + coverage
  - security    # gosec + gitleaks
  - build       # Docker build + push to GHCR
  - deploy-staging    # auto on main
  - deploy-production # manual on tag
```

### Stage: Build (on push to main OR tag)

```yaml
build:
  runs-on: ubuntu-latest
  if: github.ref == 'refs/heads/main' || startsWith(github.ref, 'refs/tags/v')
  steps:
    - uses: actions/checkout@v4
    - uses: docker/setup-buildx-action@v3
    - uses: docker/login-action@v3
      with:
        registry: ghcr.io
        username: ${{ github.actor }}
        password: ${{ secrets.GITHUB_TOKEN }}
    - uses: docker/build-push-action@v5
      with:
        push: true
        tags: |
          ghcr.io/piresc/parkir-pintar:${{ github.sha }}
          ghcr.io/piresc/parkir-pintar:main-${{ github.sha.substring(0,7) }}
        cache-from: type=gha
        cache-to: type=gha,mode=max
        build-args: |
          VERSION=${{ github.ref_name }}
          GIT_COMMIT=${{ github.sha }}
          BUILD_TIME=${{ github.event.head_commit.timestamp }}
```

### Stage: Deploy Staging (auto on main)

```yaml
deploy-staging:
  runs-on: ubuntu-latest
  needs: [build, test, security]
  if: github.ref == 'refs/heads/main'
  environment: staging
  steps:
    - name: Deploy to Coolify
      run: |
        curl -X POST "${{ secrets.COOLIFY_WEBHOOK_URL }}" \
          -H "Authorization: Bearer ${{ secrets.COOLIFY_API_TOKEN }}" \
          -H "Content-Type: application/json" \
          -d '{"image": "ghcr.io/piresc/parkir-pintar:main-${{ github.sha.substring(0,7) }}"}'
    - name: Verify deployment health
      run: |
        sleep 30
        for i in {1..10}; do
          STATUS=$(curl -s -o /dev/null -w "%{http_code}" https://staging.parkir-pintar.piresc.dev/health)
          if [ "$STATUS" = "200" ]; then
            echo "✅ Staging healthy"
            exit 0
          fi
          sleep 10
        done
        echo "❌ Staging health check failed"
        exit 1
```

### Stage: Deploy Production (on tag)

```yaml
deploy-production:
  runs-on: ubuntu-latest
  needs: [build, test, security]
  if: startsWith(github.ref, 'refs/tags/v')
  environment: production
  steps:
    - name: Tag image with semver
      run: |
        docker pull ghcr.io/piresc/parkir-pintar:${{ github.sha }}
        docker tag ghcr.io/piresc/parkir-pintar:${{ github.sha }} \
          ghcr.io/piresc/parkir-pintar:${{ github.ref_name }}
        docker tag ghcr.io/piresc/parkir-pintar:${{ github.sha }} \
          ghcr.io/piresc/parkir-pintar:latest
        docker push ghcr.io/piresc/parkir-pintar:${{ github.ref_name }}
        docker push ghcr.io/piresc/parkir-pintar:latest
    - name: Deploy to GCP Cloud Run
      uses: google-github-actions/deploy-cloudrun@v2
      with:
        service: parkir-pintar-gateway
        image: ghcr.io/piresc/parkir-pintar:${{ github.ref_name }}
        region: asia-southeast1
        env_vars_file: .env.production
    - name: Run migrations
      run: |
        # Run migrations against Cloud SQL
        gcloud run jobs execute parkir-pintar-migrate \
          --region asia-southeast1 \
          --args="--source=/app/migrations,--database=${{ secrets.PROD_DATABASE_URL }}"
    - name: Verify production health
      run: |
        sleep 30
        STATUS=$(curl -s -o /dev/null -w "%{http_code}" https://parkir-pintar.piresc.dev/health)
        if [ "$STATUS" != "200" ]; then
          echo "❌ Production health check failed — initiating rollback"
          # Rollback to previous revision
          gcloud run services update-traffic parkir-pintar-gateway \
            --to-revisions=LATEST=0 --region=asia-southeast1
          exit 1
        fi
        echo "✅ Production healthy"
```

---

## 5. Environment Configuration

### Staging (Coolify)

| Config | Value |
|--------|-------|
| Platform | Coolify on bare metal VM |
| Image source | `ghcr.io/piresc/parkir-pintar:main-<sha>` |
| Database | PostgreSQL (same VM, isolated DB) |
| Redis | Same VM, separate instance |
| NATS | Same VM, JetStream enabled |
| Domain | `staging.parkir-pintar.piresc.dev` |
| SSL | Let's Encrypt (Coolify auto) |
| Deploy trigger | Coolify webhook on image push |
| Resources | Shared VM (4 vCPU, 32GB RAM) |

**Coolify setup:**
- Create application from Docker image
- Set image: `ghcr.io/piresc/parkir-pintar`
- Enable auto-deploy on webhook
- Configure environment variables
- Set health check: `GET /health`
- Map port 8082 → 443 (via Coolify proxy)

### Production (GCP)

| Config | Value |
|--------|-------|
| Platform | Cloud Run (or GKE if needed later) |
| Image source | `ghcr.io/piresc/parkir-pintar:v*.*.*` |
| Database | Cloud SQL (PostgreSQL 14) |
| Redis | Memorystore for Redis |
| NATS | GKE-hosted or Cloud Pub/Sub adapter |
| Domain | `parkir-pintar.piresc.dev` |
| SSL | Google-managed certificate |
| Deploy trigger | GitHub Actions on semver tag |
| Region | `asia-southeast1` (Jakarta) |
| Scaling | Min 1, Max 10 instances |

**GCP services needed:**

| Service | Purpose | Estimated Cost |
|---------|---------|---------------|
| Cloud Run | Application hosting | ~$5-20/mo (low traffic) |
| Cloud SQL (PostgreSQL) | Primary database | ~$10-30/mo (db-f1-micro) |
| Memorystore (Redis) | Cache + locks | ~$15/mo (basic tier) |
| Cloud DNS | Domain management | ~$0.50/mo |
| Artifact Registry | Backup image store | ~$1/mo |
| Secret Manager | Secrets | ~$0.06/secret/mo |
| Cloud Monitoring | Metrics + alerts | Free tier |

**Estimated total: ~$35-70/month** for low-traffic staging/demo.

---

## 6. Environment Variables

### Shared (both environments)

```env
# Application
APP_ENV=<staging|production>
SERVER_PORT=8082
SERVER_SHUTDOWN_TIMEOUT=30s

# Auth
JWT_SECRET=<from-secret-manager>
JWT_ISSUER=parkir-pintar
JWT_EXPIRY=24h
AUTH_API_KEYS=<from-secret-manager>

# gRPC
GRPC_PORT=9090

# Tracing
TRACING_ENABLED=true
TRACING_EXPORTER=otlp
OTEL_EXPORTER_OTLP_ENDPOINT=<collector-endpoint>
```

### Staging-specific

```env
APP_ENV=staging
DB_HOST=localhost
DB_PORT=5432
DB_NAME=parkir_pintar_staging
DB_SSL_MODE=disable
REDIS_ADDR=localhost:6379
NATS_URL=nats://localhost:4222
SERVER_ALLOWED_ORIGINS=https://staging.parkir-pintar.piresc.dev
LOG_LEVEL=debug
RATE_LIMIT_RPS=1000  # relaxed for testing
```

### Production-specific

```env
APP_ENV=production
DB_HOST=/cloudsql/project:region:instance  # Unix socket
DB_PORT=5432
DB_NAME=parkir_pintar
DB_SSL_MODE=require
REDIS_ADDR=<memorystore-ip>:6379
NATS_URL=nats://<nats-host>:4222
SERVER_ALLOWED_ORIGINS=https://parkir-pintar.piresc.dev
LOG_LEVEL=info
RATE_LIMIT_RPS=100  # strict for production
```

---

## 7. Migration Strategy

### Staging

Migrations run automatically on deploy:
1. Coolify pre-deploy hook runs `migrate -path /app/migrations -database $DATABASE_URL up`
2. If migration fails → deploy is aborted, previous container stays running

### Production

Migrations run as a separate Cloud Run Job:
1. Tag triggers CD pipeline
2. Migration job runs FIRST (before new app version)
3. If migration fails → pipeline stops, no new version deployed
4. If migration succeeds → new app version deployed
5. Rollback: run down migration + revert to previous Cloud Run revision

### Migration Safety Rules

- All migrations MUST be backward-compatible (old code works with new schema)
- Destructive changes (drop column, rename) require 2-phase:
  1. Deploy code that doesn't use the column
  2. Next release: drop the column
- All migrations MUST have `.down.sql` files
- Migrations tested in CI before deploy

---

## 8. Rollback Strategy

### Staging (Coolify)

```bash
# Rollback to previous image
coolify deploy --image ghcr.io/piresc/parkir-pintar:main-<previous-sha>
# Or via Coolify UI: Deployments → Redeploy previous
```

### Production (Cloud Run)

```bash
# Instant rollback — route traffic to previous revision
gcloud run services update-traffic parkir-pintar-gateway \
  --to-revisions=<previous-revision>=100 \
  --region=asia-southeast1

# Or rollback to specific version
gcloud run deploy parkir-pintar-gateway \
  --image=ghcr.io/piresc/parkir-pintar:v1.1.0 \
  --region=asia-southeast1
```

**Automatic rollback:** If health check fails within 60s of deploy, traffic routes back to previous revision automatically (Cloud Run default behavior).

---

## 9. Secrets Management

| Environment | Method |
|-------------|--------|
| Local dev | `.env` file (gitignored) |
| Staging | Coolify environment variables (encrypted at rest) |
| Production | GCP Secret Manager → mounted as env vars in Cloud Run |

**Secrets to manage:**
- `JWT_SECRET`
- `AUTH_API_KEYS`
- `DB_PASSWORD`
- `REDIS_PASSWORD`
- `NATS_TOKEN` (if auth enabled)
- `NEW_RELIC_LICENSE_KEY`

---

## 10. Monitoring & Observability

### Staging

- **Logs:** Coolify log viewer + structured JSON to stdout
- **Health:** `/health`, `/health/live`, `/health/ready`
- **Tracing:** OpenTelemetry → stdout (or Jaeger if configured)

### Production

- **Logs:** Cloud Logging (automatic with Cloud Run)
- **Metrics:** Cloud Monitoring + custom Prometheus metrics
- **Tracing:** Cloud Trace (OTLP exporter → Google endpoint)
- **Alerts:** Cloud Monitoring alerting policies
  - Error rate > 1% → PagerDuty/Telegram
  - Latency p95 > 500ms → warning
  - Instance count at max → capacity alert
- **Dashboards:** Cloud Monitoring dashboard with:
  - Request rate, error rate, latency (RED metrics)
  - Instance count, CPU, memory
  - Database connections, query latency

---

## 11. Access Control

| Environment | Access |
|-------------|--------|
| Staging | Team only — IP allowlist or basic auth header |
| Production | Public API — rate limited, JWT required for protected routes |

**Staging protection options:**
1. Cloudflare Access (zero-trust, email-based)
2. Basic auth via Coolify proxy
3. IP allowlist (Tailscale network only)

---

## 12. Release Process

### Staging (Continuous)

```
Developer → PR → Review → Merge to main → CI passes → Auto-deploy staging
```

No manual intervention. Every merge to main is live on staging within ~5 minutes.

### Production (Controlled)

```
1. Verify staging is stable (manual or automated smoke tests)
2. Create annotated tag:
   git tag -a v1.2.0 -m "Release 1.2.0: payment flow improvements"
   git push origin v1.2.0
3. CI builds → tags image → deploys to production
4. Health check passes → done
5. Health check fails → auto-rollback
```

### Versioning Convention

```
v<major>.<minor>.<patch>

major: breaking API changes
minor: new features, backward-compatible
patch: bug fixes, no new features
```

---

## 13. Implementation Roadmap

### Phase 1: Staging on Coolify (Week 1)

- [ ] Configure DNS: `staging.parkir-pintar.piresc.dev` → VM IP
- [ ] Create Coolify application from Docker image
- [ ] Configure environment variables in Coolify
- [ ] Set up webhook for auto-deploy
- [ ] Update GitHub Actions CI to build + push image on main
- [ ] Verify health check endpoint works
- [ ] Test full deploy cycle (push to main → live on staging)

### Phase 2: Production on GCP (Week 2-3)

- [ ] Create GCP project
- [ ] Set up Cloud SQL (PostgreSQL)
- [ ] Set up Memorystore (Redis)
- [ ] Deploy NATS on GKE or find managed alternative
- [ ] Configure Secret Manager with all secrets
- [ ] Deploy first version to Cloud Run
- [ ] Configure custom domain mapping (`parkir-pintar.piresc.dev`)
- [ ] Set up Cloud Monitoring alerts
- [ ] Test full deploy cycle (tag → live on production)

### Phase 3: Polish (Week 3-4)

- [ ] Add rollback automation
- [ ] Add smoke tests post-deploy
- [ ] Set up Cloud Monitoring dashboard
- [ ] Configure alerting (Telegram/email)
- [ ] Document runbook for common operations
- [ ] Add staging access control (Cloudflare Access or IP allowlist)
- [ ] Performance baseline on production

---

## 14. Decision Log

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Staging platform | Coolify (existing VM) | Already running, free, sufficient for staging |
| Production platform | GCP Cloud Run | Serverless, auto-scaling, managed SSL, pay-per-use |
| Image registry | GHCR | Free for public repos, integrated with GitHub Actions |
| Trigger: staging | Push to main | Continuous delivery, fast feedback |
| Trigger: production | Git tag (semver) | Controlled releases, clear versioning |
| URL structure | Subdomains | Environment-agnostic code, clean routing |
| Region | asia-southeast1 | Closest to Indonesia users |
| Migration strategy | Forward-only + backward-compatible | Safe rollbacks without data loss |
| Secrets | Coolify env (staging) / GCP Secret Manager (prod) | Platform-native, encrypted at rest |
