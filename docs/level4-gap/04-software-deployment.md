# Gap Analysis: Kompetensi 4 — Software Deployment (Level 4)

**Project:** Parkir Pintar  
**Assessment Level:** 4  
**Date:** 2026-05-12  
**Status:** 🔴 Significant Gaps

---

## Sub-Kompetensi 1: Deploy Application

### Level 4 Requirements

- Memahami konsep dan implementasi Infrastructure as Code (IaC)
- Mampu merancang sistem deployment yang otomatis
- Mampu membuat template konfigurasi deployment yang modular dan reusable
- Mampu merumuskan best practices dalam deployment aplikasi

### Assessment Criteria

> Mengimplementasikan Infrastructure as Code. Mengimplementasikan automated deployment sesuai best practices.

### What Currently EXISTS ✅

| Asset | Path | Description |
|-------|------|-------------|
| Dockerfile | `Dockerfile` | Multi-stage build, non-root user, build metadata injection via `--build-arg` |
| Docker Compose | `docker-compose.yml` | Health checks, dependency ordering, env var references, named volumes |
| GitHub CD | `.github/workflows/cd.yml` | Triggered on semver tags, Docker build + push to GHCR |
| GitLab CI | `.gitlab-ci.yml` | 5-stage pipeline (lint → test → security → build → deploy), staging auto-deploy, production manual gate |
| Architecture Docs | `README.md` | "Cloud Deployment Reference Architecture" section |

### What's MISSING ❌

| Gap | Impact | Priority |
|-----|--------|----------|
| No Terraform/Pulumi/CDK modules | Cannot demonstrate IaC — **critical for Level 4** | 🔴 Critical |
| No Kubernetes manifests or Helm charts | No declarative infrastructure for container orchestration | 🔴 Critical |
| No per-service Dockerfiles | Single Dockerfile limits modular deployment | 🟡 Medium |
| GitHub CD deploy step is placeholder | Automated deployment is incomplete (commented out) | 🔴 Critical |
| No rollback strategy documented | No mechanism to revert failed deployments | 🔴 Critical |
| No blue-green/canary deployment | No progressive delivery strategy | 🟡 Medium |
| No environment promotion workflow | No formal staging → production promotion | 🟡 Medium |
| No deployment best practices document | Cannot demonstrate "merumuskan best practices" | 🔴 Critical |

### ACTION ITEMS

1. **Implement Terraform IaC modules**
   - `infra/terraform/modules/networking/` — VPC, subnets, security groups
   - `infra/terraform/modules/database/` — RDS PostgreSQL, Redis ElastiCache
   - `infra/terraform/modules/messaging/` — NATS JetStream cluster
   - `infra/terraform/modules/compute/` — ECS/EKS cluster definition
   - `infra/terraform/environments/staging/main.tf`
   - `infra/terraform/environments/production/main.tf`
   - Use remote state (S3 + DynamoDB locking)

2. **Create Kubernetes manifests with Helm**
   - `deploy/helm/parkir-pintar/Chart.yaml`
   - `deploy/helm/parkir-pintar/values.yaml` — base values
   - `deploy/helm/parkir-pintar/values-staging.yaml`
   - `deploy/helm/parkir-pintar/values-production.yaml`
   - `deploy/helm/parkir-pintar/templates/deployment.yaml` — per-service template
   - `deploy/helm/parkir-pintar/templates/service.yaml`
   - `deploy/helm/parkir-pintar/templates/ingress.yaml`
   - `deploy/helm/parkir-pintar/templates/hpa.yaml` — autoscaling
   - `deploy/helm/parkir-pintar/templates/configmap.yaml`
   - `deploy/helm/parkir-pintar/templates/secrets.yaml` (sealed-secrets or external-secrets)

3. **Create per-service Dockerfiles**
   - `services/gateway/Dockerfile`
   - `services/user/Dockerfile`
   - `services/parking/Dockerfile`
   - `services/payment/Dockerfile`
   - `services/notification/Dockerfile`
   - `services/analytics/Dockerfile`
   - `services/admin/Dockerfile`
   - Use shared base image pattern: `deploy/docker/base.Dockerfile`

4. **Complete CD pipeline with actual deployment**
   - Update `.github/workflows/cd.yml` to include:
     - Helm upgrade for staging (auto)
     - Smoke tests post-deploy
     - Manual approval gate for production
     - Helm upgrade for production
     - Rollback on failure
   - Add ArgoCD or Flux for GitOps (alternative approach)

5. **Implement deployment strategies**
   - `docs/deployment/rollback-strategy.md`
   - `docs/deployment/blue-green-deployment.md`
   - Configure Kubernetes rolling update with `maxSurge`/`maxUnavailable`
   - Add canary deployment via Istio VirtualService or Argo Rollouts

6. **Document deployment best practices**
   - `docs/deployment/best-practices.md` covering:
     - Immutable infrastructure principles
     - 12-factor app compliance
     - Secret management (Vault/Sealed Secrets)
     - Health check requirements before traffic routing
     - Graceful shutdown during rolling updates
     - Database migration ordering relative to app deploy
     - Feature flags for zero-downtime releases

---

## Sub-Kompetensi 2: Performing Data Migration

### Level 4 Requirements

- Mampu merumuskan aturan konversi data existing
- Mampu menentukan format data yang cocok untuk integrasi/migrasi antar aplikasi

### Assessment Criteria

> Membuat aturan konversi data. Menentukan format data untuk migrasi.

### What Currently EXISTS ✅

| Asset | Path | Description |
|-------|------|-------------|
| Migration files | `db/migrations/000001_*.up.sql` | Initial schema |
| Migration files | `db/migrations/000002_*.up.sql` | Schema evolution |
| Migration files | `db/migrations/000003_*.up.sql` | Schema evolution |
| Migration files | `db/migrations/000004_*.up.sql` + `.down.sql` | Latest migration with rollback |
| Migration tool | golang-migrate convention | Versioned, sequential migrations |

### What's MISSING ❌

| Gap | Impact | Priority |
|-----|--------|----------|
| Migrations 1-3 have NO `.down.sql` | Cannot rollback early migrations — no reversibility | 🔴 Critical |
| No migration validation in CI | Broken migrations could reach production | 🔴 Critical |
| No data conversion rules documentation | Cannot demonstrate "merumuskan aturan konversi data" | 🔴 Critical |
| No data format specification for inter-service migration | Cannot demonstrate format determination | 🔴 Critical |
| No seed data or test data migration scripts | No validation of migration correctness | 🟡 Medium |
| No migration dry-run mechanism | Cannot preview migration impact | 🟡 Medium |

### ACTION ITEMS

1. **Add rollback migrations for all existing migrations**
   - `db/migrations/000001_*.down.sql`
   - `db/migrations/000002_*.down.sql`
   - `db/migrations/000003_*.down.sql`

2. **Add migration validation to CI**
   - Add step in `.github/workflows/ci.yml`:
     ```yaml
     - name: Validate migrations
       run: |
         migrate -path db/migrations -database "$DB_URL" up
         migrate -path db/migrations -database "$DB_URL" down
         migrate -path db/migrations -database "$DB_URL" up
     ```
   - Validate idempotency and reversibility

3. **Document data conversion rules**
   - `docs/data-migration/conversion-rules.md` covering:
     - Data type mappings between services
     - Null handling and default value policies
     - Enum/status field mapping tables
     - Timestamp format standardization (UTC, RFC3339)
     - Currency/amount precision rules
     - ID format conventions (UUID v4)
   - `docs/data-migration/validation-rules.md`:
     - Pre-migration data integrity checks
     - Post-migration validation queries
     - Row count reconciliation procedures

4. **Define inter-service data format specification**
   - `docs/data-migration/data-format-spec.md` covering:
     - Protobuf as canonical format for service-to-service
     - JSON schema for external integrations
     - Event schema for NATS JetStream messages
     - Avro/Protobuf schema registry consideration
   - `docs/data-migration/schema-evolution-policy.md`:
     - Backward/forward compatibility rules
     - Field deprecation process
     - Version negotiation strategy

5. **Create migration tooling**
   - `scripts/migrate-validate.sh` — validates up/down cycle
   - `scripts/migrate-dry-run.sh` — shows SQL without executing
   - `db/seeds/` — test data for migration validation
   - `db/migrations/README.md` — migration authoring guidelines

---

## Sub-Kompetensi 3: Document Application Feature

### Level 4 Requirements

- Mampu membuat sistem dokumentasi aplikasi yang terintegrasi dengan proses development
- Mampu merumuskan prosedur pembaruan dokumen agar selalu sinkron dengan perubahan aplikasi

### Assessment Criteria

> Mengimplementasikan integrasi dokumentasi dalam proses development. Membuat checklist dan melakukan code review untuk memeriksa pembaruan dokumen.

### What Currently EXISTS ✅

| Asset | Path | Description |
|-------|------|-------------|
| README | `README.md` | Project overview, architecture reference |
| Proto definitions | `proto/**/*.proto` | Service contracts (self-documenting to degree) |
| Code comments | Various | Inline documentation in Go code |

### What's MISSING ❌

| Gap | Impact | Priority |
|-----|--------|----------|
| No auto-generated API docs | No protoc-gen-doc, no Swagger/OpenAPI | 🔴 Critical |
| No documentation freshness checks in CI | Docs can drift from code silently | 🔴 Critical |
| No CHANGELOG.md automation | No automated release notes | 🟡 Medium |
| No PR template with "docs updated?" checkbox | No enforcement of doc updates | 🔴 Critical |
| No documentation site (e.g., MkDocs, Docusaurus) | No integrated documentation system | 🟡 Medium |
| No ADR (Architecture Decision Records) | No decision history | 🟡 Medium |
| No code review checklist for documentation | Cannot demonstrate review process | 🔴 Critical |

### ACTION ITEMS

1. **Implement auto-generated API documentation**
   - Add `protoc-gen-doc` to generate HTML/Markdown from `.proto` files
   - `docs/api/` — generated API reference
   - `Makefile` target: `make docs-generate`
   - Add generation step to CI pipeline
   - Consider `buf` for proto linting + breaking change detection

2. **Create documentation freshness CI checks**
   - `.github/workflows/docs-check.yml`:
     - Detect if `.proto` files changed without regenerating docs
     - Detect if migration files changed without updating data docs
     - Detect if service code changed without updating relevant docs
   - Use file modification timestamps or git diff analysis

3. **Add PR template with documentation checklist**
   - `.github/PULL_REQUEST_TEMPLATE.md`:
     ```markdown
     ## Documentation Checklist
     - [ ] API documentation updated (if endpoints changed)
     - [ ] README updated (if setup/architecture changed)
     - [ ] Migration docs updated (if schema changed)
     - [ ] CHANGELOG entry added
     - [ ] ADR created (if architectural decision made)
     ```

4. **Implement CHANGELOG automation**
   - `CHANGELOG.md` — conventional commits based
   - Add `release-please` or `semantic-release` GitHub Action
   - `.github/workflows/changelog.yml`
   - Enforce conventional commit format via `commitlint`

5. **Set up documentation site**
   - `docs/mkdocs.yml` or `docs/docusaurus.config.js`
   - Integrate with CI to auto-deploy on merge to main
   - Include: API reference, architecture, deployment guides, runbooks

6. **Create code review documentation procedure**
   - `docs/processes/documentation-review-procedure.md`:
     - When docs must be updated (triggers)
     - Who is responsible for doc updates
     - Review checklist for documentation PRs
     - Staleness detection and remediation process
   - `docs/processes/doc-sync-policy.md`:
     - Maximum allowed drift period
     - Automated reminders/alerts for stale docs

7. **Add Architecture Decision Records**
   - `docs/adr/0001-microservices-architecture.md`
   - `docs/adr/0002-grpc-communication.md`
   - `docs/adr/0003-nats-jetstream-messaging.md`
   - `docs/adr/template.md` — ADR template

---

## Sub-Kompetensi 4: Menambahkan Fitur Monitoring

### Level 4 Requirements

- Mampu mengimplementasikan sistem analisa terhadap data monitoring (peak time, idle time)
- Mampu membuat model prediksi penggunaan resource

### Assessment Criteria

> (Not explicitly specified — inferred: implement monitoring analytics and resource prediction)

### What Currently EXISTS ✅

| Asset | Path | Description |
|-------|------|-------------|
| OpenTelemetry tracing | `pkg/tracing/` | OTLP, stdout, New Relic exporters |
| Structured logging | `pkg/logger/` | JSON output, OTEL trace correlation |
| Health checks | `pkg/health/` | Liveness, readiness, per-dependency checks |

### What's MISSING ❌

| Gap | Impact | Priority |
|-----|--------|----------|
| No Prometheus metrics | **Zero metrics collection** — cannot analyze anything | 🔴 Critical |
| No Grafana dashboards | No visualization of system behavior | 🔴 Critical |
| No alerting rules | No Alertmanager/PagerDuty integration | 🔴 Critical |
| No SLO/SLI definitions | No service level objectives | 🟡 Medium |
| No peak/idle time analytics | Cannot demonstrate "sistem analisa" | 🔴 Critical |
| No resource prediction models | Cannot demonstrate "model prediksi" | 🔴 Critical |
| No capacity planning documentation | No resource forecasting | 🟡 Medium |

### ACTION ITEMS

1. **Implement Prometheus metrics**
   - `pkg/metrics/metrics.go` — metrics registry and common metrics
   - `pkg/metrics/middleware.go` — gRPC/HTTP interceptors for automatic metrics
   - Metrics to expose per service:
     - `http_requests_total` (counter, labels: method, path, status)
     - `http_request_duration_seconds` (histogram)
     - `grpc_server_handled_total` (counter)
     - `grpc_server_handling_seconds` (histogram)
     - `db_query_duration_seconds` (histogram)
     - `nats_messages_published_total` (counter)
     - `nats_messages_consumed_total` (counter)
     - `parking_slots_occupied` (gauge)
     - `parking_sessions_active` (gauge)
     - `payment_transactions_total` (counter, labels: status)
   - Add `/metrics` endpoint to each service

2. **Deploy monitoring stack**
   - `deploy/monitoring/prometheus/prometheus.yml` — scrape configs
   - `deploy/monitoring/prometheus/alerts.yml` — alerting rules
   - `deploy/monitoring/alertmanager/alertmanager.yml`
   - `deploy/monitoring/docker-compose.monitoring.yml` — Prometheus + Grafana + Alertmanager
   - Add to Helm chart: ServiceMonitor CRDs for Prometheus Operator

3. **Create Grafana dashboards**
   - `deploy/monitoring/grafana/dashboards/overview.json` — system overview
   - `deploy/monitoring/grafana/dashboards/per-service.json` — per-service detail
   - `deploy/monitoring/grafana/dashboards/parking-business.json` — business metrics
   - `deploy/monitoring/grafana/dashboards/infrastructure.json` — resource usage
   - `deploy/monitoring/grafana/provisioning/` — auto-provisioning config

4. **Implement peak/idle time analytics**
   - `services/analytics/internal/peaktime/analyzer.go`:
     - Aggregate request rates by hour/day
     - Identify peak windows (e.g., top 20% traffic periods)
     - Identify idle windows (e.g., bottom 20% traffic periods)
     - Generate daily/weekly reports
   - `deploy/monitoring/grafana/dashboards/peak-analysis.json`:
     - Heatmap of requests by hour/day-of-week
     - Peak vs idle comparison panels
   - Prometheus recording rules for pre-aggregation:
     - `deploy/monitoring/prometheus/recording-rules.yml`

5. **Implement resource prediction model**
   - `services/analytics/internal/prediction/model.go`:
     - Linear regression on historical resource usage
     - Time-series forecasting (e.g., Holt-Winters or Prophet-style)
     - Predict CPU/memory/storage needs for next 7/30/90 days
   - `services/analytics/internal/prediction/capacity.go`:
     - Capacity threshold alerts (e.g., "storage will be full in 14 days")
     - Scaling recommendations based on predicted load
   - `docs/monitoring/resource-prediction.md`:
     - Model description and assumptions
     - Data sources and collection frequency
     - Accuracy metrics and validation approach

6. **Define SLOs/SLIs**
   - `docs/monitoring/slo-sli.md`:
     - Availability SLO: 99.9% uptime
     - Latency SLI: p99 < 500ms for API calls
     - Error rate SLI: < 0.1% 5xx responses
     - Parking session SLO: entry-to-exit processing < 3s
   - Error budget tracking in Grafana dashboard

---

## Sub-Kompetensi 5: Memperbaiki Error yang Terjadi di Production

### Level 4 Requirements

- Mampu menentukan langkah preventif untuk mencegah error terjadi lagi
- Mampu merumuskan prosedur/mekanisme untuk memperpendek response time penanganan error
- Mampu membuat laporan post mortem terhadap error yang terjadi di production

### Assessment Criteria

> (Not explicitly specified — inferred: demonstrate preventive measures, incident response procedures, and post-mortem reports)

### What Currently EXISTS ✅

| Asset | Path | Description |
|-------|------|-------------|
| Circuit breaker | `pkg/circuitbreaker/` | Prevents cascade failures (preventive) |
| Rate limiting | `pkg/ratelimit/` | Protects services from overload (preventive) |
| Idempotency | `pkg/idempotency/` | Prevents duplicate processing (preventive) |
| Graceful shutdown | `pkg/server/` | Clean connection draining |
| Health checks | `pkg/health/` | Dependency failure detection |
| Structured logging | `pkg/logger/` | JSON logs with trace correlation for debugging |
| Distributed tracing | `pkg/tracing/` | Request flow visibility across services |

### What's MISSING ❌

| Gap | Impact | Priority |
|-----|--------|----------|
| No post-mortem template | Cannot demonstrate "laporan post mortem" | 🔴 Critical |
| No incident response runbook | Cannot demonstrate "memperpendek response time" | 🔴 Critical |
| No on-call documentation | No escalation procedures | 🟡 Medium |
| No error budget tracking | No quantified reliability targets | 🟡 Medium |
| No incident timeline tooling | Manual incident tracking only | 🟡 Medium |
| No preventive measures documentation | Existing measures not documented as strategy | 🔴 Critical |
| No chaos engineering / fault injection | Cannot validate resilience proactively | 🟡 Medium |

### ACTION ITEMS

1. **Create post-mortem template and example**
   - `docs/incidents/post-mortem-template.md`:
     ```markdown
     # Post-Mortem: [Incident Title]
     ## Summary
     ## Impact (duration, affected users, revenue impact)
     ## Timeline
     ## Root Cause
     ## Contributing Factors
     ## Resolution
     ## Preventive Actions (with owners and deadlines)
     ## Lessons Learned
     ## Action Items Tracking
     ```
   - `docs/incidents/examples/2026-01-example-db-connection-exhaustion.md` — sample post-mortem

2. **Create incident response runbook**
   - `docs/incidents/runbook.md`:
     - Severity classification (P1-P4)
     - Response time targets per severity
     - Escalation matrix
     - Communication templates (internal + external)
     - First responder checklist
     - Service-specific troubleshooting guides
   - `docs/incidents/runbook-per-service/`:
     - `gateway.md` — common gateway issues
     - `payment.md` — payment failure scenarios
     - `parking.md` — parking session issues
     - `database.md` — PostgreSQL troubleshooting
     - `messaging.md` — NATS JetStream issues

3. **Document preventive measures strategy**
   - `docs/incidents/preventive-measures.md`:
     - Circuit breaker configuration and rationale (`pkg/circuitbreaker/`)
     - Rate limiting thresholds and tuning (`pkg/ratelimit/`)
     - Idempotency implementation coverage (`pkg/idempotency/`)
     - Input validation strategy
     - Timeout budgets per service call
     - Retry policies with exponential backoff
     - Dead letter queue handling for failed messages
     - Database connection pool sizing
     - Memory/goroutine leak prevention patterns

4. **Implement incident response automation**
   - `scripts/incident-response/`:
     - `collect-diagnostics.sh` — gather logs, metrics, traces for time window
     - `rollback-service.sh` — quick rollback script
     - `scale-service.sh` — emergency scaling
   - Integrate with alerting (PagerDuty/OpsGenie webhook)
   - Slack/Teams notification on P1/P2 incidents

5. **Create on-call documentation**
   - `docs/incidents/on-call-guide.md`:
     - On-call rotation schedule
     - Handoff procedures
     - Tools access checklist
     - Common false-positive alerts and how to handle
     - When to wake up the next person

6. **Implement chaos engineering (stretch goal)**
   - `tests/chaos/` — fault injection tests
   - Use `toxiproxy` for network fault simulation
   - Scenarios: database unavailable, NATS partition, high latency injection
   - Document results and improvements made

---

## Summary: Overall Readiness

| Sub-Kompetensi | Readiness | Critical Gaps |
|----------------|-----------|---------------|
| 1. Deploy Application | 🟡 30% | No IaC, incomplete CD, no deployment strategies |
| 2. Data Migration | 🟡 40% | No rollback migrations, no conversion rules docs |
| 3. Document Features | 🔴 15% | No auto-docs, no CI checks, no review process |
| 4. Monitoring | 🟡 35% | No metrics, no dashboards, no analytics/prediction |
| 5. Production Errors | 🟡 45% | No post-mortem, no runbook (but good preventive code) |

### Priority Implementation Order

1. **Week 1-2:** Terraform modules + Helm charts (Sub 1) — highest impact for Level 4
2. **Week 2-3:** Prometheus metrics + Grafana dashboards (Sub 4) — enables analytics
3. **Week 3-4:** Post-mortem template + incident runbook (Sub 5) — documentation-heavy, quick wins
4. **Week 4-5:** Migration rollbacks + data format docs (Sub 2)
5. **Week 5-6:** API doc generation + PR template + CI checks (Sub 3)
6. **Week 6-8:** Peak/idle analytics + resource prediction (Sub 4 advanced)

### Key Dependencies

```
Terraform/Helm (Sub 1) ──→ Prometheus in K8s (Sub 4) ──→ Peak/Idle Analytics (Sub 4)
                                    │
                                    ▼
                        Alerting Rules (Sub 4) ──→ Incident Response (Sub 5)
                        
Proto docs generation (Sub 3) ──→ Documentation CI checks (Sub 3)

Migration rollbacks (Sub 2) ──→ Migration validation in CI (Sub 2)
```

---

## References

- [Terraform Best Practices](https://www.terraform-best-practices.com/)
- [Helm Chart Best Practices](https://helm.sh/docs/chart_best_practices/)
- [Google SRE Book — Post-Mortems](https://sre.google/sre-book/postmortem-culture/)
- [Prometheus Monitoring Best Practices](https://prometheus.io/docs/practices/)
- [ADR GitHub Organization](https://adr.github.io/)
- [Conventional Commits](https://www.conventionalcommits.org/)
