# Work Breakdown Structure — ParkirPintar

## Document Information

| Field | Value |
|-------|-------|
| Project | ParkirPintar — Smart Parking Reservation System |
| Last Updated | 2026-05-13 |
| WBS Levels | 3 (Major Deliverable → Work Package → Task) |

---

## 1.0 Project Management

### 1.1 Planning & Governance
- 1.1.1 Define project scope and objectives
- 1.1.2 Create stakeholder analysis matrix
- 1.1.3 Develop risk register and mitigation plans
- 1.1.4 Establish sprint cadence and ceremonies
- 1.1.5 Define Definition of Done (DoD)

### 1.2 Tracking & Reporting
- 1.2.1 Maintain sprint backlog and burndown
- 1.2.2 Conduct sprint reviews and retrospectives
- 1.2.3 Track velocity and forecast delivery
- 1.2.4 Update project roadmap per phase

### 1.3 Documentation
- 1.3.1 Write Architecture Decision Records (ADRs)
- 1.3.2 Create project management artifacts (WBS, roadmap, risk register)
- 1.3.3 Maintain changelog and release notes
- 1.3.4 Prepare assessment submission package

---

## 2.0 Architecture & Design

### 2.1 System Architecture
- 2.1.1 Design microservices decomposition (reservation, search, billing)
- 2.1.2 Define inter-service communication patterns (gRPC sync, NATS async)
- 2.1.3 Design data architecture (per-service databases, event sourcing)
- 2.1.4 Create C4 architecture diagrams (context, container, component)

### 2.2 API Design
- 2.2.1 Define protobuf service contracts (.proto files)
- 2.2.2 Design REST gateway mapping (gRPC-Gateway)
- 2.2.3 Define event schemas for NATS subjects
- 2.2.4 Document API versioning strategy

### 2.3 Data Design
- 2.3.1 Design PostgreSQL schema with PostGIS extensions
- 2.3.2 Define database migration strategy
- 2.3.3 Design Redis caching and locking schema
- 2.3.4 Plan data retention and archival policy

---

## 3.0 Core Services Development

### 3.1 Reservation Service
- 3.1.1 Implement reservation domain model with state machine
- 3.1.2 Implement reservation repository (PostgreSQL)
- 3.1.3 Implement gRPC server (CreateReservation, CancelReservation, GetReservation)
- 3.1.4 Implement distributed lock acquisition for spot reservation
- 3.1.5 Publish reservation events to NATS (created, confirmed, cancelled, completed)
- 3.1.6 Add optimistic concurrency control
- 3.1.7 Write unit and integration tests

### 3.2 Search Service
- 3.2.1 Implement parking lot domain model with capacity tracking
- 3.2.2 Implement spatial search repository (PostGIS ST_DWithin)
- 3.2.3 Implement gRPC server (SearchParkingLots, GetAvailability)
- 3.2.4 Add filtering (price range, amenities, distance)
- 3.2.5 Implement Redis caching for hot queries
- 3.2.6 Subscribe to reservation events for availability updates
- 3.2.7 Write unit and integration tests

### 3.3 Billing Service
- 3.3.1 Implement billing domain model (invoice, line items, payment status)
- 3.3.2 Implement billing repository with transaction support
- 3.3.3 Implement gRPC server (CreateInvoice, ProcessPayment, GetInvoice)
- 3.3.4 Subscribe to reservation.confirmed events for invoice generation
- 3.3.5 Integrate payment gateway client (Midtrans)
- 3.3.6 Implement payment retry with exponential backoff
- 3.3.7 Write unit and integration tests

### 3.4 Shared Libraries
- 3.4.1 Implement gRPC interceptors (logging, auth, tracing, recovery)
- 3.4.2 Implement NATS publisher/subscriber abstractions
- 3.4.3 Implement Redis client wrapper with circuit breaker
- 3.4.4 Implement configuration management (env + file)
- 3.4.5 Implement graceful shutdown coordinator

---

## 4.0 Security

### 4.1 Authentication
- 4.1.1 Implement JWT token generation (RS256)
- 4.1.2 Implement token validation middleware
- 4.1.3 Implement refresh token rotation
- 4.1.4 Configure token expiry policies (access: 15min, refresh: 7d)

### 4.2 Authorization
- 4.2.1 Define RBAC roles (driver, operator, admin)
- 4.2.2 Implement role-based gRPC interceptor
- 4.2.3 Implement resource-level access control (own reservations only)

### 4.3 API Protection
- 4.3.1 Implement rate limiting (token bucket algorithm)
- 4.3.2 Add input validation with sanitization
- 4.3.3 Configure request size limits
- 4.3.4 Add security headers (CORS, CSP, HSTS)

### 4.4 Security Testing
- 4.4.1 Integrate OWASP ZAP DAST in CI pipeline
- 4.4.2 Run Trivy container vulnerability scanning
- 4.4.3 Conduct dependency audit (govulncheck)
- 4.4.4 Review and remediate findings

---

## 5.0 Observability

### 5.1 Distributed Tracing
- 5.1.1 Integrate OpenTelemetry SDK in all services
- 5.1.2 Configure trace propagation across gRPC and NATS
- 5.1.3 Deploy OTel Collector with OTLP receiver
- 5.1.4 Deploy Tempo for trace storage
- 5.1.5 Create trace-based Grafana dashboards

### 5.2 Metrics
- 5.2.1 Instrument services with Prometheus metrics (RED pattern)
- 5.2.2 Add custom business metrics (reservations/min, revenue)
- 5.2.3 Deploy Prometheus with service discovery
- 5.2.4 Create Grafana dashboards for service health

### 5.3 Logging
- 5.3.1 Implement structured logging (slog) with trace correlation
- 5.3.2 Deploy Loki for log aggregation
- 5.3.3 Configure log levels and PII redaction
- 5.3.4 Create Grafana log exploration views

### 5.4 Alerting & SLOs
- 5.4.1 Define SLOs (availability, latency, error rate)
- 5.4.2 Implement multi-window burn rate alerts
- 5.4.3 Create SLO dashboard in Grafana
- 5.4.4 Define error budget policy and escalation

---

## 6.0 CI/CD Pipeline

### 6.1 Continuous Integration
- 6.1.1 Configure GitHub Actions workflow (lint, test, build)
- 6.1.2 Add golangci-lint with custom configuration
- 6.1.3 Add test coverage reporting (threshold: 80%)
- 6.1.4 Add Trivy scan step (fail on HIGH/CRITICAL)
- 6.1.5 Add OWASP ZAP DAST scan step
- 6.1.6 Configure branch protection and PR requirements

### 6.2 Continuous Deployment
- 6.2.1 Build multi-stage Docker images
- 6.2.2 Push images to GitHub Container Registry (GHCR)
- 6.2.3 Trigger Coolify deployment via webhook
- 6.2.4 Run post-deploy smoke tests
- 6.2.5 Configure Watchtower for automated pulls

### 6.3 Infrastructure as Code
- 6.3.1 Write Terraform modules for GCP Cloud Run
- 6.3.2 Write Terraform modules for Cloud SQL
- 6.3.3 Write Terraform modules for Memorystore Redis
- 6.3.4 Configure Terraform state backend (GCS)
- 6.3.5 Write docker-compose for local development

---

## 7.0 Testing & Quality

### 7.1 Unit Testing
- 7.1.1 Write unit tests for domain models
- 7.1.2 Write unit tests for repository layer (with testcontainers)
- 7.1.3 Write unit tests for service layer (mocked dependencies)
- 7.1.4 Maintain coverage above 80%

### 7.2 Integration Testing
- 7.2.1 Write gRPC integration tests (end-to-end flow)
- 7.2.2 Write NATS event integration tests
- 7.2.3 Write database migration tests
- 7.2.4 Implement contract tests for service boundaries

### 7.3 Performance Testing
- 7.3.1 Write k6 load test scripts (search, reserve, billing)
- 7.3.2 Establish performance baseline (500 concurrent users)
- 7.3.3 Identify and resolve bottlenecks
- 7.3.4 Document performance characteristics and limits

### 7.4 Security Testing
- 7.4.1 Run DAST scans and remediate findings
- 7.4.2 Validate authentication bypass scenarios
- 7.4.3 Test rate limiting under attack simulation
- 7.4.4 Verify input validation coverage

---

## 8.0 Infrastructure & Deployment

### 8.1 Staging Environment (Coolify/VPS)
- 8.1.1 Provision VPS with Docker runtime
- 8.1.2 Deploy Coolify for container orchestration
- 8.1.3 Configure Caddy reverse proxy with auto-TLS
- 8.1.4 Deploy PostgreSQL, Redis, NATS containers
- 8.1.5 Deploy monitoring stack (Prometheus, Grafana, Loki, Tempo)

### 8.2 Production Environment (GCP Cloud Run)
- 8.2.1 Apply Terraform for Cloud Run services
- 8.2.2 Configure Cloud SQL with automated backups
- 8.2.3 Configure Memorystore Redis
- 8.2.4 Set up Cloud Armor WAF rules
- 8.2.5 Configure blue-green deployment with traffic splitting
- 8.2.6 Set up Secret Manager for credentials

### 8.3 Operational Readiness
- 8.3.1 Write operational runbooks
- 8.3.2 Define on-call rotation and escalation
- 8.3.3 Configure PagerDuty alerting integration
- 8.3.4 Document disaster recovery procedures
- 8.3.5 Implement automated backup verification

---

## WBS Dictionary (Summary)

| WBS ID | Work Package | Estimated Effort | Sprint |
|--------|-------------|:----------------:|:------:|
| 1.0 | Project Management | Ongoing | All |
| 2.0 | Architecture & Design | 2 weeks | Pre-Sprint 1 |
| 3.0 | Core Services Development | 4 weeks | Sprint 1–2 |
| 4.0 | Security | 2 weeks | Sprint 3 |
| 5.0 | Observability | 2 weeks | Sprint 3 |
| 6.0 | CI/CD Pipeline | 2 weeks | Sprint 4 |
| 7.0 | Testing & Quality | 3 weeks | Sprint 5 |
| 8.0 | Infrastructure & Deployment | 4 weeks | Sprint 4–6 |

**Total Work Packages:** 8 major deliverables, 30 work packages, 100+ tasks
