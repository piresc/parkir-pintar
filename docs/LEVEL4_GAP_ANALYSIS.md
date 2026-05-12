# Level 4 Competency Gap Analysis — parkir-pintar

> **Assessment Date:** 2026-05-12
> **Codebase:** parkir-pintar (Go microservices parking application)
> **Target:** Level 4 (Senior/Lead Developer — Telkomsel)
> **Branch:** main (post-merge)

---

## Executive Summary

| Kompetensi | Score | Status |
|------------|-------|--------|
| 0 — Software Requirement | 45% | ⚠️ Gaps in formal process docs |
| 1 — Software Design | 60% | ⚠️ Strong implementation, weak ADRs |
| 2 — Software Construction | 85% | ✅ Near-complete |
| 3 — Software Quality | 80% | ✅ Strong, minor CI gaps |
| 4 — Software Deployment | 40% | 🔴 Major gaps (IaC, monitoring) |
| 5 — Software Security | 65% | ⚠️ Strong preventive, weak response |

**Overall Level 4 Readiness: ~63%**

---

## Kompetensi 0 — Software Requirement

### Sub: Menggali kebutuhan user (Level 4: 45%)

| Requirement | Status | Evidence |
|-------------|--------|----------|
| Memprediksi skenario alternatif | ⚠️ Partial | `docs/superpowers/specs/2026-05-06-payment-flow-design.md` shows before/after scenarios |
| Mengklarifikasi skenario alternatif | ⚠️ Partial | Design specs mention edge cases but no formal "questions list" |
| Memperkirakan kebutuhan integrasi | ✅ | Proto files define service contracts; docker-compose shows integration map |

**MISSING:**
- No formal requirements elicitation document (interview questions, stakeholder sessions)
- No documented alternative scenario predictions beyond payment flow
- No integration requirements matrix

### Sub: Menganalisa kebutuhan user (Level 4: 55%)

| Requirement | Status | Evidence |
|-------------|--------|----------|
| Menganalisa keterkaitan dengan aplikasi/service lain | ✅ | `docs/architecture.md` shows service dependencies |
| Menganalisa dampak terhadap resource (RAM, Storage, Network) | ❌ | No capacity planning or resource analysis docs |
| Merekomendasikan solusi alternatif | ⚠️ Partial | `docs/superpowers/plans/2026-05-07-two-phase-payment-flow.md` has Option a/b/c |
| Menjelaskan kelebihan/kekurangan alternatif | ⚠️ Partial | Some tradeoff discussion in plans, not formalized |

**MISSING:**
- Resource impact analysis (RAM/Storage/Network projections)
- Formal alternatives comparison matrix with pros/cons
- Documented recommendations to stakeholders

### Sub: Membuat spesifikasi kebutuhan (Level 4: 60%)

| Requirement | Status | Evidence |
|-------------|--------|----------|
| Membuat spesifikasi API | ✅ | 6 proto files with full RPC definitions; `docs/api.md` |
| Membuat aturan access control | ✅ | JWT auth + API key auth + rate limiting implemented |
| Spesifikasi non-functional (security) | ✅ | Rate limiting, circuit breaker, JWT algorithm pinning |
| Spesifikasi non-functional (scalability) | ⚠️ Partial | Schema-per-service, but no explicit scalability spec |
| Spesifikasi non-functional (performance) | ⚠️ Partial | PRD §18.1 mentions targets, load tests validate them |
| Spesifikasi non-functional (configurability) | ✅ | All config via env vars, `pkg/config/` with validation |

**MISSING:**
- Formal OpenAPI/Swagger spec (proto files exist but no REST spec)
- Explicit scalability specification document
- Performance SLA document with targets and measurement plan

### Sub: Mengelola perubahan kebutuhan (Level 4: 40%)

| Requirement | Status | Evidence |
|-------------|--------|----------|
| Mengkalkulasi dampak perubahan terhadap service lain | ⚠️ Partial | Migration 000004 shows cross-service impact awareness |
| Mengkalkulasi dampak terhadap resource | ❌ | No resource impact calculations |
| Menganalisa solusi alternatif | ⚠️ Partial | Some alternatives in plan docs |
| Menjelaskan detail alternatif ke user | ❌ | No stakeholder-facing change analysis docs |

**MISSING:**
- Change impact analysis template
- Resource impact calculator for requirement changes
- Stakeholder communication documents for change proposals

---

## Kompetensi 1 — Software Design

### Sub: Design application architecture (Level 4: 60%)

| Requirement | Status | Evidence |
|-------------|--------|----------|
| Menganalisa berbagai alternatif desain arsitektur | ⚠️ Partial | Plans show some alternatives but no formal comparison |
| Review dan feedback terhadap rancangan Medium Developer | ⚠️ Partial | `REVIEW.md` exists, code review skill configured |
| Menganalisa dampak desain terhadap deployment/operasional | ⚠️ Partial | README has "Cloud Deployment Reference Architecture" |
| Menyediakan beberapa pilihan arsitektur | ❌ | No formal architecture options document |
| Impact analysis per pilihan | ❌ | No impact analysis matrix |
| Architecture design review | ⚠️ Partial | REVIEW.md but no formal review process |

**MISSING:**
- Formal ADR directory (`docs/adr/`) with numbered records
- Architecture alternatives comparison (e.g., "Why microservices over monolith?")
- Impact analysis per architecture option
- Architecture review board/process documentation

### Sub: Design component and module (Level 4: 75%)

| Requirement | Status | Evidence |
|-------------|--------|----------|
| Review struktur komponen/modul | ✅ | REVIEW.md with severity-ranked findings |
| Desain alternatif terhadap requirement | ⚠️ Partial | Some alternatives in plan docs |
| Rancang arsitektur messaging | ✅ | NATS JetStream with 4 streams, 12 subjects |
| Rancang aplikasi terdistribusi | ✅ | 7 microservices, gRPC, distributed locks |
| Rancang microservices | ✅ | Full microservices with schema-per-service |
| Rancang event sourcing | ⚠️ Partial | Event-driven (pub/sub) but not full event sourcing |

**MISSING:**
- Full event sourcing with event store and replay capability
- Formal alternative design documents per component
- CODEOWNERS file for review assignment

### Sub: Design data structure (Level 4: 55%)

| Requirement | Status | Evidence |
|-------------|--------|----------|
| Menguasai berbagai model struktur data | ✅ | PostgreSQL + Redis + NATS JetStream |
| Menganalisa dan membandingkan pilihan | ❌ | No comparison document |
| Menjelaskan kelebihan/kekurangan pilihan | ❌ | No documented justification |

**MISSING:**
- Data storage technology comparison document
- Justification for PostgreSQL over alternatives
- Justification for Redis over Memcached/alternatives
- Capacity planning for data growth

### Sub: Select and use framework and library (Level 4: 50%)

| Requirement | Status | Evidence |
|-------------|--------|----------|
| Menguasai beberapa alternatif technology stack | ⚠️ Partial | Go + React, but not polyglot microservices |
| Menjelaskan kelebihan/kekurangan alternatif | ❌ | No tech stack comparison doc |
| Mengkombinasikan stack dari berbagai platform | ⚠️ Partial | Go backend + React frontend + protobuf |

**MISSING:**
- Technology stack comparison document
- Sample applications in alternative stacks
- Presentation/documentation of stack tradeoffs

### Sub: Document Design Writing (Level 4: 70%)

| Requirement | Status | Evidence |
|-------------|--------|----------|
| Menulis dokumen desain | ✅ | README (656 lines), architecture.md, PRD.md (849 lines) |
| Mereview dokumen desain | ⚠️ Partial | REVIEW.md exists but no formal doc review process |

**MISSING:**
- Formal design document review process
- Document versioning/changelog
- Review approval stamps

---

## Kompetensi 2 — Software Construction

### Sub: Collaboration tools (Level 4: 80%)

| Requirement | Status | Evidence |
|-------------|--------|----------|
| Mendefinisikan workflow version control | ✅ | Feature branch workflow evident from branch naming |
| Merumuskan kebijakan CI/CD | ✅ | Dual pipeline (GitHub Actions + GitLab CI) with quality gates |
| Trunk-based / Feature branch | ✅ | Feature branch → main with PR |

**MISSING:**
- Documented branching strategy (CONTRIBUTING.md)
- Branch protection rules visible in repo
- Formal CI/CD policy document

### Sub: Writing source code (Level 4: 90%)

| Requirement | Status | Evidence |
|-------------|--------|----------|
| Menguasai berbagai design pattern | ✅ | 10+ patterns: Repository, Strategy, Circuit Breaker, Adapter, Middleware, Observer, Factory, Token Bucket, Distributed Lock, Idempotency |
| Review dan feedback kode program | ✅ | REVIEW.md, code review skill, CI on PRs |
| Mendefinisikan coding convention | ⚠️ Partial | golangci-lint enforced but no explicit style guide |
| Membuat library/framework internal | ✅ | 14+ reusable `pkg/` libraries, all tested |
| Riset best-practices | ✅ | SOLID, Clean Architecture, DRY evident throughout |
| Analisa performance dan integritas data | ✅ | Load tests, race tests, FOR UPDATE, distributed locks |
| Distributed programming | ✅ | gRPC + NATS + Redis locks + circuit breaker |

**MISSING:**
- Explicit coding guideline document
- `.golangci.yml` with custom rules
- Go benchmarks (`func BenchmarkXxx`)
- pprof profiling integration

### Sub: Manage data in application (Level 4: 80%)

| Requirement | Status | Evidence |
|-------------|--------|----------|
| Akses data pada database terdistribusi | ✅ | Schema-per-service, CQRS read model |
| Eventual consistency | ✅ | NATS events sync search read model |
| CAP theorem awareness | ✅ | Cross-schema FK removal comment: "eventual consistency via NATS" |
| Event sourcing | ⚠️ Partial | Event-driven but not full event sourcing |
| Messaging-based data flow | ✅ | NATS JetStream with durable consumers |

**MISSING:**
- Full event sourcing with event store
- Outbox pattern for guaranteed event delivery
- Formal saga orchestrator

### Sub: Integrate and implement API (Level 4: 85%)

| Requirement | Status | Evidence |
|-------------|--------|----------|
| Review desain API | ✅ | Proto files reviewed, REVIEW.md |
| Berbagai alternatif desain API | ✅ | REST + gRPC dual protocol |
| Roadmap evolusi API | ❌ | No API evolution roadmap |
| Berbagai metode integrasi (messaging, file transfer) | ✅ | gRPC (sync) + NATS (async) |
| Rate limiting | ✅ | HTTP + gRPC token bucket |
| Circuit breaker | ✅ | Custom 3-state implementation |
| Retry/backoff | ✅ | Exponential backoff, context-aware |
| API versioning | ✅ | v1 in proto packages + REST paths |

**MISSING:**
- API evolution roadmap document
- API deprecation strategy
- OpenAPI/Swagger spec for REST endpoints

### Sub: Handling error/bug (Level 4: 60%)

| Requirement | Status | Evidence |
|-------------|--------|----------|
| Aggregator log dari banyak aplikasi | ⚠️ Partial | OpenTelemetry tracing spans across services |
| Rekapitulasi dan klasifikasi error | ❌ | No error classification system |
| Root cause analysis | ❌ | No RCA documentation |
| Instrumentasi/pengukuran otomatis | ✅ | OTEL tracing, structured logging |
| Solusi generik untuk mengurangi bug | ✅ | Circuit breaker, idempotency, rate limiting |
| Perubahan metode development | ⚠️ Partial | Code review process exists |
| Langkah preventif | ✅ | Property-based tests, race detection |
| Prosedur response time penanganan error | ❌ | No incident response procedure |
| Laporan post mortem | ❌ | No post-mortem template |

**MISSING:**
- Centralized log aggregation (ELK/Loki)
- Error classification and trending
- Root cause analysis documentation
- Post-mortem template and process
- Incident response procedure with SLAs

---

## Kompetensi 3 — Software Quality

### Sub: Prepare automated testing environment (Level 4: 85%)

| Requirement | Status | Evidence |
|-------------|--------|----------|
| Performance test environment | ✅ | `tests/e2e/load_test.go` — 6 load scenarios |
| Load test environment | ✅ | 100 concurrent users, sustained waves |
| Security test environment | ✅ | gosec + gitleaks + SonarCloud |

**MISSING:**
- Dedicated performance test environment (separate from E2E)
- k6/Vegeta/Gatling for HTTP-level load testing
- OWASP ZAP for DAST

### Sub: Write automated test (Level 4: 80%)

| Requirement | Status | Evidence |
|-------------|--------|----------|
| Automated non-functional test | ✅ | Load tests, race tests, security scans |
| Integrasi non-functional test dalam CI/CD | ⚠️ Partial | Security scans in CI ✅, but load/E2E tests NOT in pipeline |

**MISSING:**
- Load tests in CI/CD pipeline (nightly/scheduled)
- E2E tests in CI/CD pipeline (Docker-in-Docker)
- DAST (ZAP) in CI/CD pipeline
- Chaos/resilience testing (toxiproxy)
- API contract tests (`buf breaking`)

---

## Kompetensi 4 — Software Deployment

### Sub: Deploy application (Level 4: 35%)

| Requirement | Status | Evidence |
|-------------|--------|----------|
| Infrastructure as Code | ❌ | No Terraform/Pulumi/Helm |
| Sistem deployment otomatis | ⚠️ Partial | CD pipeline exists but staging deploy is placeholder |
| Template konfigurasi modular/reusable | ⚠️ Partial | Multi-stage Dockerfile, but single image for all services |
| Best practices deployment | ⚠️ Partial | Non-root container, health checks, graceful shutdown |

**MISSING:**
- Terraform/Pulumi for cloud infrastructure
- Helm charts or Kubernetes manifests
- Per-service Dockerfiles
- Blue-green or canary deployment
- Rollback strategy documentation
- Environment promotion workflow

### Sub: Performing data migration (Level 4: 50%)

| Requirement | Status | Evidence |
|-------------|--------|----------|
| Merumuskan aturan konversi data | ⚠️ Partial | Migration 000004 has data conversion (seed read model) |
| Format data untuk integrasi/migrasi | ⚠️ Partial | Proto files define data contracts |

**MISSING:**
- Down migrations for files 1-3
- Migration validation in CI
- Data conversion rules documentation
- Migration testing (dry-run)

### Sub: Document application feature (Level 4: 45%)

| Requirement | Status | Evidence |
|-------------|--------|----------|
| Dokumentasi terintegrasi dengan development | ⚠️ Partial | Docs in repo, updated with code |
| Prosedur pembaruan dokumen sinkron dengan perubahan | ❌ | No doc freshness checks |
| Code review untuk memeriksa pembaruan dokumen | ❌ | No doc review in PR checklist |

**MISSING:**
- Auto-generated API docs from proto files
- Documentation freshness checks in CI
- CHANGELOG.md automation
- PR template with "docs updated?" checkbox

### Sub: Monitoring (Level 4: 30%)

| Requirement | Status | Evidence |
|-------------|--------|----------|
| Database historis kinerja aplikasi | ❌ | No Prometheus/metrics |
| Sistem notifikasi/alert | ❌ | No alerting |
| Analisa data monitoring (peak/idle time) | ❌ | No analytics |
| Model prediksi penggunaan resource | ❌ | No predictive models |

**EXISTS (foundation):**
- OpenTelemetry tracing (`pkg/tracing/`)
- Structured logging with trace correlation (`pkg/logger/`)
- Health checks with dependency timing (`pkg/health/`)

**MISSING:**
- Prometheus metrics exposition
- Grafana dashboards (or dashboard-as-code)
- Alerting rules (Alertmanager/PagerDuty/OpsGenie)
- SLO/SLI definitions
- Peak/idle time analytics
- Resource prediction models

### Sub: Memperbaiki error di production (Level 4: 25%)

| Requirement | Status | Evidence |
|-------------|--------|----------|
| Langkah preventif | ✅ | Circuit breaker, rate limiting, idempotency |
| Prosedur memperpendek response time | ❌ | No incident response procedure |
| Laporan post mortem | ❌ | No post-mortem template |

**MISSING:**
- Incident response runbook
- Post-mortem template
- On-call rotation documentation
- Error budget tracking
- Automated alerting pipeline

---

## Kompetensi 5 — Software Security

### Sub: Develop secure application (Level 4: 65%)

| Requirement | Status | Evidence |
|-------------|--------|----------|
| Implementasi standar security (OWASP) | ✅ | 8/10 OWASP Top 10 mitigated |
| Security testing tools dalam development | ✅ | gosec + gitleaks + SonarCloud in CI |
| Langkah preventif kebocoran | ✅ | JWT pinning, rate limiting, parameterized SQL |
| Langkah recovery data bocor/hilang | ❌ | No recovery plan |
| Laporan post mortem security breach | ❌ | No incident response docs |
| PCI/DSS compliance | ❌ | No compliance documentation |
| Automated security testing in CI/CD | ✅ | gosec + gitleaks block pipeline |

**MISSING:**
- `SECURITY.md` with responsible disclosure policy
- Incident response plan for security breaches
- Data recovery procedures
- Post-mortem template for security incidents
- PCI/DSS compliance documentation
- DAST (OWASP ZAP) in pipeline
- Dependency vulnerability scanning (`govulncheck`)
- mTLS between services
- Security headers (HSTS, CSP, X-Frame-Options)
- RBAC enforcement in handlers (role extracted but not checked)

---

## Priority Action Items (to reach Level 4)

### 🔴 Critical (Must-Have)

| # | Action | Competency | Effort |
|---|--------|-----------|--------|
| 1 | Create ADR directory with 5+ architecture decisions | Design | 2-3 days |
| 2 | Add Terraform/Helm for infrastructure | Deployment | 3-5 days |
| 3 | Implement Prometheus metrics + Grafana dashboards | Deployment | 2-3 days |
| 4 | Create incident response runbook + post-mortem template | Deployment/Security | 1-2 days |
| 5 | Add alerting (Alertmanager/PagerDuty) | Deployment | 1-2 days |
| 6 | Write `SECURITY.md` + incident response plan | Security | 1 day |
| 7 | Add load/E2E tests to CI/CD (nightly pipeline) | Quality | 1 day |

### 🟡 Important (Should-Have)

| # | Action | Competency | Effort |
|---|--------|-----------|--------|
| 8 | Create technology comparison docs (Why Go? Why PostgreSQL? Why NATS?) | Design | 1-2 days |
| 9 | Add OpenAPI/Swagger spec generation from proto | Construction | 1 day |
| 10 | Create coding style guide (`CONTRIBUTING.md`) | Construction | 1 day |
| 11 | Add `.golangci.yml` with custom rules | Construction | 0.5 day |
| 12 | Add Go benchmarks for hot paths | Construction | 1 day |
| 13 | Add OWASP ZAP DAST to pipeline | Security | 1 day |
| 14 | Add `govulncheck` to CI | Security | 0.5 day |
| 15 | Add security headers middleware | Security | 0.5 day |
| 16 | Create API evolution roadmap | Construction | 1 day |
| 17 | Add down migrations for 000001-000003 | Deployment | 1 day |
| 18 | Add resource capacity planning doc | Requirement | 1 day |

### 🟢 Nice-to-Have (Bonus)

| # | Action | Competency | Effort |
|---|--------|-----------|--------|
| 19 | Implement full event sourcing for one service | Design/Construction | 3-5 days |
| 20 | Add saga orchestrator pattern | Construction | 2-3 days |
| 21 | Add chaos testing (toxiproxy) | Quality | 1-2 days |
| 22 | Add mTLS between services | Security | 1-2 days |
| 23 | Create SLO/SLI definitions | Deployment | 1 day |
| 24 | Add `buf breaking` API contract tests | Quality | 0.5 day |
| 25 | Per-service Dockerfiles | Deployment | 1 day |

---

## What Already Exceeds Level 4

These areas are **beyond** typical Level 4 expectations:

1. **Property-based testing** — Using `pgregory.net/rapid` for fuzzing (uncommon even at senior level)
2. **Distributed locking with Lua** — Atomic Redis operations with proper TTL
3. **CQRS read model** — Event-driven search sync via NATS
4. **Dual CI/CD pipelines** — Both GitHub Actions and GitLab CI
5. **14 reusable pkg/ libraries** — All tested, well-documented
6. **Idempotency middleware** — SETNX sentinel + polling pattern
7. **Schema-per-service isolation** — Database boundary enforcement
8. **OpenTelemetry distributed tracing** — Cross-service correlation

---

## Estimated Timeline to Full Level 4

| Phase | Items | Duration |
|-------|-------|----------|
| Phase 1: Documentation | ADRs, SECURITY.md, runbooks, style guide, tech comparisons | 1 week |
| Phase 2: Infrastructure | Terraform/Helm, Prometheus, Grafana, alerting | 1 week |
| Phase 3: CI/CD Enhancement | Load tests in pipeline, ZAP, govulncheck, E2E in CI | 3 days |
| Phase 4: Polish | Benchmarks, OpenAPI, down migrations, API roadmap | 3 days |

**Total estimated effort: ~3 weeks** to close all critical and important gaps.
