# ParkirPintar Resource Allocation Plan

## Overview

This document outlines the resource allocation for ParkirPintar, covering team structure, infrastructure, tooling, time management, and budget planning.

## Team Structure

### Current Team

| Role                    | Count | Responsibilities                                      |
|-------------------------|-------|-------------------------------------------------------|
| Full-Stack Developer    | 1     | Architecture, backend, frontend, DevOps, testing      |
| AI Pair Programming     | 2+    | Code generation, review, documentation, testing       |

### AI Assistants Allocation

| Assistant        | Primary Focus                                          |
|------------------|--------------------------------------------------------|
| Coding Agent     | Implementation, refactoring, test writing              |
| Review Agent     | Code review, architecture decisions, documentation     |

### Effective Capacity

| Resource              | Weekly Hours | Effective Output (with AI) |
|-----------------------|--------------|----------------------------|
| Developer             | 40h          | ~2.5x multiplier with AI   |
| AI Coding Assistant   | On-demand    | Code generation, testing   |
| AI Review Assistant   | On-demand    | Review, docs, planning     |
| **Effective capacity**| **~100h equivalent** | Per week              |

### Future Scaling (When Needed)

| Phase        | Additional Roles                    | Trigger                        |
|--------------|-------------------------------------|--------------------------------|
| Phase 2      | +1 Backend Developer                | > 3 services in active dev     |
| Phase 3      | +1 Frontend Developer               | Mobile app development         |
| Phase 4      | +1 DevOps/SRE                       | Multi-region deployment        |

## Infrastructure Resources

### Current: VPS (Debian 12)

| Resource     | Specification          | Allocation                              |
|--------------|------------------------|-----------------------------------------|
| CPU          | 4 vCPU                 | Services: 60%, DB: 25%, Monitoring: 15% |
| RAM          | 16 GB                  | Services: 8GB, DB: 4GB, Cache: 2GB, OS+Mon: 2GB |
| Storage      | 200 GB SSD             | DB: 80GB, Logs: 40GB, Images: 40GB, OS: 40GB |
| Network      | 1 Gbps                 | Shared across all services              |
| OS           | Debian 12 (Bookworm)   | Minimal server installation             |

### Service Resource Allocation (Docker)

| Service              | CPU Limit | Memory Limit | Replicas | Notes              |
|----------------------|-----------|--------------|----------|--------------------|
| Traefik (LB)        | 0.5 CPU   | 256 MB       | 1        | Reverse proxy      |
| Gateway Service      | 0.5 CPU   | 512 MB       | 1        | HTTP→gRPC gateway  |
| Parking Service      | 0.5 CPU   | 512 MB       | 1        | Core business logic|
| Payment Service      | 0.25 CPU  | 256 MB       | 1        | Payment processing |
| User Service         | 0.25 CPU  | 256 MB       | 1        | User management    |
| PostgreSQL           | 1.0 CPU   | 2 GB         | 1        | Primary database   |
| Redis                | 0.25 CPU  | 512 MB       | 1        | Cache + locks      |
| NATS                 | 0.25 CPU  | 256 MB       | 1        | Event messaging    |
| Grafana              | 0.25 CPU  | 256 MB       | 1        | Dashboards         |
| Prometheus           | 0.25 CPU  | 512 MB       | 1        | Metrics collection |
| Loki                 | 0.25 CPU  | 256 MB       | 1        | Log aggregation    |
| Tempo                | 0.25 CPU  | 256 MB       | 1        | Distributed tracing|
| Coolify              | 0.5 CPU   | 512 MB       | 1        | Deployment platform|

### Planned: GCP (Production)

| Resource              | Specification          | Purpose                         |
|-----------------------|------------------------|---------------------------------|
| GKE Cluster           | 3 nodes, e2-medium     | Service orchestration           |
| Cloud SQL (PostgreSQL) | db-f1-micro → db-g1-small | Managed database          |
| Memorystore (Redis)   | Basic, 1GB             | Managed cache                   |
| Cloud NAT             | Standard               | Outbound connectivity           |
| Cloud Load Balancer   | Regional, external     | Traffic ingress                 |
| Cloud Storage         | Standard               | Backups, static assets          |
| Artifact Registry     | Standard               | Container images                |

## Tool Allocation

### Development Tools

| Tool              | Purpose                    | Tier/Plan    | Cost        |
|-------------------|----------------------------|--------------|-------------|
| GitHub            | Repository + CI/CD         | Free (public)| $0/mo       |
| GitHub Actions    | CI/CD pipelines            | Free tier    | $0/mo       |
| SonarCloud        | Code quality analysis      | Free (OSS)   | $0/mo       |
| Coolify           | Self-hosted PaaS           | Self-hosted  | $0/mo       |
| VS Code           | IDE                        | Free         | $0          |

### Monitoring & Observability

| Tool              | Purpose                    | Deployment   | Cost        |
|-------------------|----------------------------|--------------|-------------|
| Grafana           | Dashboards & visualization | Self-hosted  | $0/mo       |
| Prometheus        | Metrics collection         | Self-hosted  | $0/mo       |
| Loki              | Log aggregation            | Self-hosted  | $0/mo       |
| Tempo             | Distributed tracing        | Self-hosted  | $0/mo       |
| Alertmanager      | Alert routing              | Self-hosted  | $0/mo       |

### Security Tools

| Tool              | Purpose                    | Integration  | Cost        |
|-------------------|----------------------------|--------------|-------------|
| gosec             | SAST for Go                | CI pipeline  | $0          |
| govulncheck       | Vulnerability scanning     | CI pipeline  | $0          |
| gitleaks          | Secret detection           | CI + pre-commit | $0       |
| Trivy             | Container scanning         | CI pipeline  | $0          |
| OWASP ZAP         | DAST                       | Scheduled CI | $0          |

## Time Allocation Per Sprint (2-Week Sprint)

### Standard Sprint Distribution

| Activity          | Allocation | Hours/Sprint | Description                          |
|-------------------|------------|--------------|--------------------------------------|
| Implementation    | 40%        | 32h          | Feature development, bug fixes       |
| Testing           | 20%        | 16h          | Writing tests, test infrastructure   |
| Operations        | 20%        | 16h          | CI/CD, monitoring, deployment, infra |
| Documentation     | 20%        | 16h          | Technical docs, ADRs, API docs       |
| **Total**         | **100%**   | **80h**      | Per 2-week sprint                    |

### Activity Breakdown

#### Implementation (40% - 32h/sprint)

| Sub-Activity           | Hours | Notes                              |
|------------------------|-------|------------------------------------|
| Feature development    | 20h   | New functionality                  |
| Bug fixes              | 6h    | Defect resolution                  |
| Refactoring            | 4h    | Technical debt reduction           |
| Code review            | 2h    | Reviewing AI-generated code        |

#### Testing (20% - 16h/sprint)

| Sub-Activity           | Hours | Notes                              |
|------------------------|-------|------------------------------------|
| Unit test writing      | 8h    | New tests for features             |
| Integration tests      | 4h    | Database/service boundary tests    |
| E2E test maintenance   | 2h    | Updating E2E for new features      |
| Test infrastructure    | 2h    | Fixtures, helpers, CI config       |

#### Operations (20% - 16h/sprint)

| Sub-Activity           | Hours | Notes                              |
|------------------------|-------|------------------------------------|
| CI/CD maintenance      | 4h    | Pipeline updates, optimization     |
| Monitoring & alerting  | 4h    | Dashboard updates, alert tuning    |
| Infrastructure         | 4h    | Server maintenance, updates        |
| Incident response      | 4h    | Buffer for unexpected issues       |

#### Documentation (20% - 16h/sprint)

| Sub-Activity           | Hours | Notes                              |
|------------------------|-------|------------------------------------|
| Technical documentation| 6h    | Architecture, design docs          |
| API documentation      | 4h    | Proto comments, OpenAPI specs      |
| ADRs & decisions       | 3h    | Recording architectural decisions  |
| Process documentation  | 3h    | Runbooks, guides                   |

### Sprint Variants

| Sprint Type        | Impl | Test | Ops  | Docs | When                        |
|--------------------|------|------|------|------|-----------------------------|
| Standard           | 40%  | 20%  | 20%  | 20%  | Normal development          |
| Feature-heavy      | 55%  | 25%  | 10%  | 10%  | Major feature delivery      |
| Stabilization      | 20%  | 35%  | 30%  | 15%  | Pre-release hardening       |
| Infrastructure     | 15%  | 15%  | 55%  | 15%  | Migration, major infra work |
| Documentation      | 20%  | 10%  | 10%  | 60%  | Compliance, audit prep      |

## Budget

### Current Monthly Costs

| Item                    | Provider      | Cost/Month | Notes                    |
|-------------------------|---------------|------------|--------------------------|
| VPS (16GB RAM)          | Hetzner/DO    | ~$25       | Primary server           |
| Domain (parkir-pintar)  | Registrar     | ~$1        | Annual amortized         |
| DNS                     | Cloudflare    | $0         | Free tier                |
| GitHub                  | GitHub        | $0         | Free for public repos    |
| SonarCloud              | SonarSource   | $0         | Free for OSS             |
| SSL Certificates        | Let's Encrypt | $0         | Auto-renewed via Traefik |
| **Total Current**       |               | **~$26/mo**|                          |

### Projected GCP Costs (Production)

| Item                    | Specification       | Est. Cost/Month | Notes              |
|-------------------------|---------------------|-----------------|--------------------|
| GKE Cluster             | 3x e2-medium        | ~$75            | Autopilot mode     |
| Cloud SQL (PostgreSQL)  | db-f1-micro         | ~$10            | Shared core        |
| Memorystore (Redis)     | Basic, 1GB          | ~$35            | Managed cache      |
| Cloud Load Balancer     | Regional            | ~$20            | + data processing  |
| Cloud Storage           | 10GB Standard       | ~$1             | Backups            |
| Artifact Registry       | Standard            | ~$5             | Container images   |
| Cloud NAT               | Standard            | ~$5             | Outbound traffic   |
| Networking (egress)     | ~50GB/mo            | ~$5             | Data transfer      |
| **Total GCP Projected** |                     | **~$156/mo**    |                    |

### Cost Optimization Strategies

| Strategy                        | Savings Est. | Implementation                    |
|---------------------------------|--------------|-----------------------------------|
| GKE Autopilot (pay per pod)     | 20-30%       | Only pay for running workloads    |
| Preemptible/Spot VMs            | 60-80%       | For non-critical workloads        |
| Committed use discounts         | 20-57%       | 1-3 year commitments              |
| Right-sizing                    | 10-30%       | Monitor and adjust resources      |
| Off-hours scaling               | 30-50%       | Scale down dev/staging at night   |

## Skill Matrix & Training Needs

### Current Skills Assessment

| Skill Area              | Proficiency | Priority | Notes                        |
|-------------------------|-------------|----------|------------------------------|
| Go (backend)            | Advanced    | Core     | Primary language             |
| gRPC / Protobuf         | Intermediate| High     | Service communication        |
| PostgreSQL              | Advanced    | Core     | Primary data store           |
| Redis                   | Intermediate| High     | Caching, locking             |
| NATS / Event-driven     | Intermediate| High     | Async messaging              |
| Docker / Containers     | Advanced    | Core     | Deployment platform          |
| Kubernetes / GKE        | Beginner    | Medium   | Future production platform   |
| Terraform               | Beginner    | Medium   | Infrastructure as Code       |
| Observability (OTel)    | Intermediate| High     | Monitoring stack             |
| Security (AppSec)       | Intermediate| High     | Security architecture        |
| Frontend (React/Next)   | Intermediate| Medium   | Dashboard/admin UI           |
| CI/CD (GitHub Actions)  | Advanced    | Core     | Pipeline management          |

### Training Plan

| Skill Gap               | Learning Path                    | Timeline    | Priority |
|-------------------------|----------------------------------|-------------|----------|
| Kubernetes (GKE)        | GCP Associate Cloud Engineer     | Q3 2026     | Medium   |
| Terraform               | HashiCorp Terraform Associate    | Q3 2026     | Medium   |
| Advanced Security       | OWASP resources, CTF practice    | Ongoing     | High     |
| Performance Engineering | k6 University, Go profiling      | Q2 2026     | High     |
| Event-driven patterns   | NATS documentation, patterns     | Q2 2026     | High     |

### AI-Augmented Skill Coverage

Areas where AI assistants compensate for skill gaps:

| Area                    | AI Contribution                              |
|-------------------------|----------------------------------------------|
| Code review             | Automated pattern detection, best practices  |
| Documentation           | Generation, formatting, consistency          |
| Test generation         | Table-driven test scaffolding, edge cases    |
| Security review         | Vulnerability pattern recognition            |
| Architecture decisions  | Trade-off analysis, pattern suggestions      |
| Boilerplate code        | CRUD operations, middleware, handlers        |

## Resource Constraints & Mitigations

| Constraint                    | Impact                    | Mitigation                          |
|-------------------------------|---------------------------|-------------------------------------|
| Single developer              | Bus factor = 1            | Comprehensive docs, AI assistants   |
| Limited budget                | No managed services       | Self-hosted tools, free tiers       |
| Single VPS                    | No HA, limited scale      | Docker Compose, planned GCP migration|
| No dedicated QA               | Testing burden on dev     | Automated testing, AI test generation|
| No dedicated SRE              | Ops burden on dev         | Automation, self-healing, alerting  |

## Resource Utilization Monitoring

### Infrastructure Metrics to Track

| Metric                  | Alert Threshold | Action                              |
|-------------------------|-----------------|-------------------------------------|
| CPU utilization         | > 80% sustained | Scale up or optimize                |
| Memory utilization      | > 85%           | Investigate leaks, scale up         |
| Disk usage              | > 75%           | Clean logs, expand storage          |
| Network bandwidth       | > 70%           | Optimize payloads, CDN              |
| Container restarts      | > 3/hour        | Investigate OOM, crashes            |

### Capacity Planning Triggers

| Trigger                        | Action                                    |
|--------------------------------|-------------------------------------------|
| Sustained CPU > 70%            | Evaluate GCP migration timeline           |
| Database size > 50GB           | Plan Cloud SQL migration                  |
| > 100 concurrent users         | Add service replicas, load balancer       |
| Response time degradation      | Profile, optimize, or scale horizontally  |

---

*Last updated: 2026-05-13*
*Owner: ParkirPintar Engineering*
*Review cycle: Monthly (budget), Quarterly (strategy)*
