# Project Roadmap — ParkirPintar

## Document Information

| Field | Value |
|-------|-------|
| Project | ParkirPintar — Smart Parking Reservation System |
| Last Updated | 2026-05-13 |
| Timeline | March 2026 — August 2026 |
| Methodology | Agile (Scrum) with phased delivery |

---

## Roadmap Overview

```
MAR 2026        APR 2026        MAY 2026        JUN 2026        JUL 2026        AUG 2026
│───────────────│───────────────│───────────────│───────────────│───────────────│
│  Phase 1 ✅   │    Phase 2 ✅  │  Phase 3 🔄   │   Phase 4 📋  │       Phase 5 📋       │
│  MVP          │  Prod Ready   │  Quality      │  Prod Deploy  │  Advanced Features     │
│               │               │               │               │                        │
├─M1────────M2──┼──M3───────M4──┼──M5───────M6──┼──M7───────────┼──M8─────────────M9─────┤
```

---

## Phase 1: MVP — Core Reservation Flow ✅

| Field | Value |
|-------|-------|
| Status | Complete |
| Duration | 2026-03-03 → 2026-03-30 (4 weeks) |
| Sprints | Sprint 1, Sprint 2 |

### Objectives
- Establish domain models and data layer
- Implement core reservation lifecycle
- Enable inter-service communication
- Validate architecture with end-to-end flow

### Deliverables

| Deliverable | Description | Status |
|-------------|-------------|:------:|
| Domain Models | Parking lot, reservation, billing entities with state machines | ✅ |
| PostgreSQL Repositories | CRUD + spatial queries with PostGIS | ✅ |
| gRPC Services | Reservation, Search, Billing service implementations | ✅ |
| NATS Event Bus | JetStream-based async communication between services | ✅ |
| Distributed Locking | Redis-based lock for concurrent reservation prevention | ✅ |
| Database Migrations | Versioned schema migrations with golang-migrate | ✅ |

### Milestones

| ID | Milestone | Date | Status |
|----|-----------|------|:------:|
| M1 | Domain models and repositories complete | 2026-03-16 | ✅ |
| M2 | End-to-end reservation flow working via gRPC | 2026-03-30 | ✅ |

### Dependencies
- PostgreSQL 16 with PostGIS extension
- Redis 7.x for distributed locking
- NATS 2.10 with JetStream enabled

---

## Phase 2: Production Readiness — Observability, Security, CI/CD ✅

| Field | Value |
|-------|-------|
| Status | Complete |
| Duration | 2026-03-31 → 2026-04-27 (4 weeks) |
| Sprints | Sprint 3, Sprint 4 |

### Objectives
- Full observability (traces, metrics, logs)
- Secure all endpoints with authentication and authorization
- Automate build, test, and deployment pipeline
- Containerize and deploy to staging environment

### Deliverables

| Deliverable | Description | Status |
|-------------|-------------|:------:|
| OpenTelemetry Pipeline | Traces (Tempo), Metrics (Prometheus), Logs (Loki) | ✅ |
| Grafana Dashboards | RED metrics, resource usage, service health | ✅ |
| JWT Authentication | RS256-signed tokens with refresh rotation | ✅ |
| RBAC Authorization | Role-based access (driver, operator, admin) | ✅ |
| Rate Limiting | Token bucket per IP and per user | ✅ |
| GitHub Actions CI | Lint, test, build, scan on every PR | ✅ |
| GitHub Actions CD | Build → GHCR → Coolify webhook deploy | ✅ |
| Docker Containers | Multi-stage builds with distroless runtime | ✅ |
| Coolify Deployment | VPS staging with Caddy auto-TLS | ✅ |
| Terraform Modules | GCP Cloud Run IaC (prepared for Phase 4) | ✅ |

### Milestones

| ID | Milestone | Date | Status |
|----|-----------|------|:------:|
| M3 | Full observability pipeline operational | 2026-04-13 | ✅ |
| M4 | CI/CD pipeline deploying to staging automatically | 2026-04-27 | ✅ |

### Dependencies
- Phase 1 complete (services must exist to observe/deploy)
- GitHub repository with branch protection
- VPS with Docker and Coolify installed
- GHCR for container registry

---

## Phase 3: Quality Assurance — Load Testing, DAST, SLOs 🔄

| Field | Value |
|-------|-------|
| Status | In Progress |
| Duration | 2026-04-28 → 2026-05-18 (3 weeks, extended) |
| Sprints | Sprint 5 |

### Objectives
- Validate system performance under load
- Identify security vulnerabilities via DAST
- Define and monitor SLOs
- Complete engineering documentation

### Deliverables

| Deliverable | Description | Status |
|-------------|-------------|:------:|
| k6 Load Tests | Scripts for search, reservation, billing flows | ✅ |
| Performance Baseline | 500 concurrent users, P99 < 1s validated | ✅ |
| OWASP ZAP DAST | Automated security scan in CI | ✅ |
| SLO Definitions | Availability, latency, error rate targets | ✅ |
| SLO Dashboard | Grafana burn rate visualization | ✅ |
| ADRs | Architecture Decision Records for key choices | 🔄 |
| Project Management Docs | Risk register, WBS, roadmap, sprint planning | 🔄 |
| Runbooks | Operational procedures for common scenarios | 🔄 |
| API Documentation | Protobuf-generated service docs | ✅ |

### Milestones

| ID | Milestone | Date | Status |
|----|-----------|------|:------:|
| M5 | Load test baseline established, no critical issues | 2026-05-04 | ✅ |
| M6 | All documentation complete, assessment-ready | 2026-05-18 | 🔄 |

### Dependencies
- Phase 2 complete (need deployed system to test)
- k6 installed in CI runner
- OWASP ZAP Docker image available

---

## Phase 4: Production Deployment — GCP Cloud Run 📋

| Field | Value |
|-------|-------|
| Status | Planned |
| Duration | 2026-05-19 → 2026-06-15 (4 weeks) |
| Sprints | Sprint 6, Sprint 7 |

### Objectives
- Deploy to GCP Cloud Run for production workloads
- Implement blue-green deployment strategy
- Set up production-grade database (Cloud SQL)
- Configure CDN and global load balancing

### Planned Deliverables

| Deliverable | Description | Status |
|-------------|-------------|:------:|
| GCP Cloud Run Deployment | Apply Terraform modules, deploy all services | 📋 |
| Cloud SQL PostgreSQL | Managed database with automated backups | 📋 |
| Memorystore Redis | Managed Redis for caching and locking | 📋 |
| Blue-Green Deploy | Zero-downtime deployment with traffic splitting | 📋 |
| Cloud Armor | WAF and DDoS protection | 📋 |
| Secret Manager | Centralized secrets management | 📋 |
| Production Monitoring | Uptime checks, PagerDuty integration, error budget alerts | 📋 |
| Disaster Recovery | Automated backups, cross-region replication plan | 📋 |

### Milestones

| ID | Milestone | Date | Status |
|----|-----------|------|:------:|
| M7 | Production environment live with blue-green deploy | 2026-06-15 | 📋 |

### Dependencies
- Phase 3 complete (quality validated before production)
- GCP project with billing enabled
- Domain name configured with Cloud DNS
- Terraform state backend (GCS bucket)

---

## Phase 5: Advanced Features — Analytics, ML, Multi-tenant 📋

| Field | Value |
|-------|-------|
| Status | Planned |
| Duration | 2026-06-16 → 2026-08-10 (8 weeks) |
| Sprints | Sprint 8 — Sprint 11 |

### Objectives
- Build analytics dashboard for operators
- Implement ML-based parking demand prediction
- Support multi-tenant architecture for white-label deployment
- Add real-time notifications and mobile push

### Planned Deliverables

| Deliverable | Description | Status |
|-------------|-------------|:------:|
| Analytics Dashboard | Occupancy trends, revenue reports, peak hour analysis | 📋 |
| ML Demand Prediction | Time-series forecasting for parking availability | 📋 |
| Multi-tenant Support | Tenant isolation, custom branding, separate billing | 📋 |
| Real-time Notifications | WebSocket/SSE for live spot updates | 📋 |
| Mobile Push | FCM integration for reservation reminders | 📋 |
| Dynamic Pricing | Surge pricing based on demand prediction | 📋 |
| Operator API | Self-service lot management and reporting API | 📋 |

### Milestones

| ID | Milestone | Date | Status |
|----|-----------|------|:------:|
| M8 | Analytics dashboard live for operators | 2026-07-13 | 📋 |
| M9 | ML prediction model deployed, multi-tenant ready | 2026-08-10 | 📋 |

### Dependencies
- Phase 4 complete (production infrastructure required)
- Historical data collection (minimum 4 weeks)
- ML model training pipeline (Vertex AI or self-hosted)
- FCM project configuration

---

## Key Dependencies Graph

```
Phase 1 (MVP)
    │
    ├──→ Phase 2 (Prod Readiness)
    │        │
    │        ├──→ Phase 3 (Quality)
    │        │        │
    │        │        └──→ Phase 4 (Prod Deploy)
    │        │                 │
    │        │                 └──→ Phase 5 (Advanced)
    │        │
    │        └──→ Phase 4 (Terraform modules from Phase 2)
    │
    └──→ Phase 3 (Load tests need working services)
```

## Risk to Timeline

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Phase 3 documentation scope creep | Delays Phase 4 start | Timebox documentation, prioritize assessment-critical items |
| GCP budget constraints | Blocks Phase 4 | Use free tier where possible, apply for academic credits |
| ML model accuracy insufficient | Phase 5 feature incomplete | Start with rule-based heuristics, iterate with ML |
| Team availability (exam period) | Velocity reduction | Front-load critical work, maintain buffer sprints |
