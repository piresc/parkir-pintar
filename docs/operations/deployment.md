# Deployment Architecture

## Overview

ParkirPintar runs on a self-hosted Coolify instance with three isolated Docker Compose stacks connected via shared networks. CI/CD is fully automated: a push to `main` triggers GitHub Actions, which builds per-service images, pushes them to GHCR, and triggers a rolling deploy via Coolify webhook.

## Coolify Stack Architecture

```
┌─────────────────────────────────────────────────────────┐
│  Infra Stack (infra-backend network)                    │
│  PostgreSQL 14 · Redis 7 · NATS 2.10 JetStream         │
└─────────────────────────────────────────────────────────┘
         ▲                          ▲
         │ infra-backend            │ infra-backend
         │                          │
┌────────┴──────────────┐  ┌───────┴──────────────────────┐
│  App Stack            │  │  Observability Stack          │
│  gateway              │  │  Prometheus · Grafana         │
│  reservation          │  │  Tempo · Loki · Alloy        │
│  billing              │  │  Alertmanager · Exporters     │
│  payment              │  │  (postgres, redis, nats)      │
│  search               │  │                               │
│  presence             │  │  (observability network)      │
│  analytics            │  │                               │
│  frontend             │  │  UIs on Tailscale only        │
│                       │  │                               │
│  (parkir-app network) │  │                               │
└───────────────────────┘  └───────────────────────────────┘
```

### Stack Separation Rationale

| Stack | Purpose | Lifecycle |
|-------|---------|-----------|
| **Infra** | Stateful data stores (Postgres, Redis, NATS) | Rarely redeployed; data persistence is critical |
| **App** | Stateless microservices | Redeployed on every push to main |
| **Observability** | Monitoring, tracing, logging | Independent upgrade cycle; never disrupted by app deploys |

### Deployment Order

1. **Infra Stack** — creates the `infra-backend` network
2. **Observability Stack** — creates `observability` network, joins `infra-backend`
3. **App Stack** — joins both `infra-backend` and `observability` networks

## Network Topology

| Network | Purpose | Created By |
|---------|---------|------------|
| `infra-backend` | App/Obs services → Postgres, Redis, NATS | Infra stack |
| `observability` | App services → Alloy (OTLP ingestion) | Observability stack |
| `parkir-app` | Internal service mesh (gateway ↔ gRPC services) | App stack |

Traefik (managed by Coolify) handles TLS termination and reverse proxying for public-facing endpoints. Observability UIs (Grafana, Prometheus) are accessible only via Tailscale (private network).

## CI/CD Pipeline

```
Push to main
  → GitHub Actions CI
    ├── Secret Scan (gitleaks)
    ├── Lint (golangci-lint)
    ├── Test (race detector + coverage)
    ├── Security Scan (gosec)
    ├── Vulnerability Check (govulncheck)
    ├── Proto Check (buf lint + breaking)
    └── SonarCloud analysis
  → Build & Push (parallel per-service)
    └── docker/build-push-action → GHCR
  → Deploy to Staging
    └── curl Coolify webhook → rolling recreate
```

### Pipeline Stages

**Gate 1 — Secret Scan:** Runs gitleaks with full history scan. All subsequent jobs depend on this passing.

**Gate 2 — Quality (parallel):** Lint, test, security, vulncheck, and proto-check run concurrently after secret scan passes.

**Gate 3 — Build & Push:** Matrix build for all 7 backend services + frontend. Each service gets two tags:
- `ghcr.io/piresc/parkir-pintar/<service>:latest`
- `ghcr.io/piresc/parkir-pintar/<service>:main-<sha>`

**Gate 4 — Deploy:** Triggers only on push to main (not PRs). Calls the Coolify webhook endpoint.

### Pull Request Behavior

PRs run gates 1–2 only (no build/push/deploy). This provides fast feedback without producing artifacts.

## Docker Image Build Strategy

Each service has its own Dockerfile at `cmd/<service>/Dockerfile`. All images share the monorepo as build context, enabling shared `pkg/` dependencies without multi-repo complexity.

Build arguments injected at CI time:
- `VERSION` — git SHA
- `GIT_COMMIT` — git SHA (for runtime identification)
- `BUILD_TIME` — commit timestamp

Images are multi-stage builds: compile in a Go builder stage, copy the binary into a minimal Alpine runtime image.

## Rolling Deploy Process

The deploy script (`deploy/deploy.sh`) performs a health-checked rolling restart:

```bash
# Order: leaf services first, gateway last
for svc in billing search presence payment reservation analytics frontend gateway; do
  docker compose up -d --no-deps --force-recreate "$svc"
  # Wait up to 60s for healthcheck to pass
done
```

Key properties:
- **Ordered restart:** Infrastructure-dependent services restart first; the gateway (entry point) restarts last.
- **Health-gated:** Each service must pass its Docker healthcheck before the next service restarts.
- **Timeout protection:** If a service doesn't become healthy within 60 seconds, the deploy continues with a warning (no rollback — the previous container is already replaced).
- **Orphan cleanup:** Removes any dangling containers after the deploy completes.

## Environment Management

Secrets are injected via Coolify's environment variable UI per stack:

| Secret | Used By |
|--------|---------|
| `DB_USERNAME` / `DB_PASSWORD` | All backend services |
| `REDIS_PASSWORD` | Gateway, Search, Reservation, Presence |
| `JWT_SECRET` | All backend services |
| `COOLIFY_TOKEN` | GitHub Actions (deploy trigger) |
| `COOLIFY_WEBHOOK_URL` | GitHub Actions (deploy endpoint) |
| `SONAR_TOKEN` | GitHub Actions (SonarCloud) |

Non-secret configuration (ports, timeouts, pool sizes) lives in YAML files baked into the image at build time. See [Configuration](./configuration.md) for details.

## Health Checks

Every service defines a Docker healthcheck:

| Service | Protocol | Check |
|---------|----------|-------|
| Gateway | HTTP | `curl -f http://localhost:8080/health` |
| gRPC services | TCP | `nc -z localhost 9090` |
| PostgreSQL | CLI | `pg_isready -U $user -d $db` |
| Redis | CLI | `redis-cli -a $pass ping` |
| NATS | HTTP | `wget --spider http://localhost:8222/healthz` |

All healthchecks use: interval 10s, timeout 5s, retries 3–5, start period 10–30s.

---

## Production — AWS EKS

### Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│  AWS EKS Cluster (piresc-parkir)  —  ap-southeast-3 (Jakarta)       │
│                                                                      │
│  ┌─── Fargate (serverless) ──────────────────────────────────────┐  │
│  │  gateway ×2       → REST API, port 8080 (NLB public port 80)  │  │
│  │  reservation ×2   → gRPC :9091                                │  │
│  │  billing ×2       → gRPC :9092                                │  │
│  │  payment ×2       → gRPC :9093                                │  │
│  │  search ×2        → gRPC :9094                                │  │
│  │  analytics ×2     → gRPC :9095                                │  │
│  │  presence ×2      → gRPC :9095                                │  │
│  └────────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  ┌─── EC2 Node Group (2× t3.medium) ────────────────────────────┐  │
│  │  nats-0  → NATS JetStream :4222 (10Gi gp2 persistent volume) │  │
│  └────────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  ┌─── Observability (Fargate) ───────────────────────────────────┐  │
│  │  alloy ×1 → OTLP collector :4317                              │  │
│  │  prometheus ×1 → metrics :9090                                 │  │
│  │  tempo ×1 → traces :3200                                      │  │
│  │  loki ×1 → logs :3100                                         │  │
│  │  grafana ×1 → dashboards :3000                                │  │
│  └────────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  ┌─── Managed Services ──────────────────────────────────────────┐  │
│  │  RDS PostgreSQL 14 (db.t3.small, 20GB gp3, encrypted)         │  │
│  │  ElastiCache Redis 7.0 (cache.t3.small)                       │  │
│  │  NLB (internet-facing, IP-mode targeting Fargate pods)         │  │
│  └────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
```

### Infrastructure (Terraform)

All AWS resources are defined in `infra/aws/` and managed by Terraform:

| File | Creates |
|------|---------|
| `network.tf` | VPC, 2 AZs, public/private subnets, NAT Gateway |
| `eks.tf` | EKS cluster, Fargate profile, EC2 node group, EBS CSI driver |
| `rds.tf` | RDS PostgreSQL 14 |
| `elasticache.tf` | ElastiCache Redis 7.0 |
| `iam-oidc.tf` | GitHub Actions OIDC provider + IAM role |
| `s3.tf` | Terraform state bucket + DynamoDB lock |

### Kubernetes Manifests

```
infra/aws/k8s/
├── base/              # NATS StatefulSet, ConfigMaps, Secrets
├── services/          # 7 Go service Deployments + gateway LoadBalancer
├── autoscaling/       # HPA (CPU-based, 2→10 replicas)
├── observability/     # Prometheus, Grafana, Tempo, Loki, Alloy
└── migrations/        # DB migration Job template
```

### CI/CD Pipeline (Production)

```
Tag v1.0.0 → build-push-prod.yml → GHCR (7 images tagged v1.0.0 + latest-prod)
                                      │
                                      │ (on success, auto-triggers)
                                      ▼
                               deploy-prod.yml
                                 ├── Configure AWS (OIDC)
                                 ├── kubectl apply services + HPA
                                 ├── kubectl set image (v1.0.0)
                                 ├── kubectl rollout status (wait)
                                 ├── Smoke test (curl /health)
                                 └── On failure → kubectl rollout undo
```

Production deploys are **automatic** — every tagged release is built and deployed without manual intervention. The `deploy-prod.yml` workflow also supports manual triggering via `workflow_dispatch` for emergency releases.

### First-Time Setup

Prerequisites: AWS CLI, Terraform, kubectl installed.

```bash
# 1. Bootstrap state storage
aws s3api create-bucket --bucket pirescer-parkir-pintar-tfstate --region ap-southeast-3 --create-bucket-configuration LocationConstraint=ap-southeast-3
aws dynamodb create-table --table-name pirescer-parkir-pintar-tfstate-lock --attribute-definitions AttributeName=LockID,AttributeType=S --key-schema AttributeName=LockID,KeyType=HASH --billing-mode PAY_PER_REQUEST --region ap-southeast-3

# 2. Create terraform.tfvars (from example)
cd infra/aws && cp terraform.tfvars.example terraform.tfvars
# Edit with real passwords

# 3. Deploy infrastructure (~15 min)
terraform init && terraform apply

# 4. Connect kubectl
aws eks update-kubeconfig --region ap-southeast-3 --name piresc-parkir

# 5. Deploy K8s resources
bash setup.sh
```

### Subsequent Deploys

```bash
# Via GitHub Actions (recommended):
# 1. Tag a release
git tag v1.1.0 && git push origin v1.1.0
# 2. Go to Actions → Deploy Production → Run workflow (enter v1.1.0)

# Or manually:
kubectl set image deployment/gateway gateway=ghcr.io/piresc/parkir-pintar/gateway:v1.1.0 -n pirescer-parkir-pintar
# Repeat for all services
```

### Service Discovery

Services find each other via environment variables:

| Env Var | Value |
|---------|-------|
| `GRPC_RESERVATION_TARGET` | `reservation:9091` |
| `GRPC_SEARCH_TARGET` | `search:9094` |
| `GRPC_BILLING_TARGET` | `billing:9092` |
| `GRPC_PAYMENT_TARGET` | `payment:9093` |
| `GRPC_ANALYTICS_TARGET` | `analytics:9095` |
| `GRPC_PRESENCE_TARGET` | `presence:9095` |
| `DB_HOST` | RDS IP (from `terraform output db_endpoint`) |
| `REDIS_HOST` | ElastiCache IP |
| `NATS_URL` | `nats.pirescer-parkir-pintar.svc.cluster.local:4222` |
| `DB_SCHEMA` | Per-service: reservation, billing, payment, search, presence, analytics |

### Auto-Scaling (HPA)

| Service | Min | Max | Target CPU |
|---------|-----|-----|-----------|
| reservation | 2 | 10 | 60% |
| search | 2 | 10 | 60% |
| gateway | 2 | 8 | 60% |
| billing | 2 | 8 | 60% |
| payment | 2 | 6 | 60% |
| analytics | 2 | 6 | 60% |
| presence | 2 | 6 | 60% |

### Health Probes (EKS)

| Service | Probe Type | Port |
|---------|-----------|------|
| Gateway | HTTP GET `/health` | 8080 |
| All gRPC services | TCP Socket | Service port |
| NATS | HTTP GET `/healthz` | 8222 |

### Security

| Resource | Access |
|----------|--------|
| Gateway NLB | Public (port 80, JWT-protected) |
| EKS API | Public (IAM auth required) |
| RDS PostgreSQL | Private (VPC only, port 5432) |
| ElastiCache Redis | Private (VPC only, port 6379) |
| NATS | Private (ClusterIP only) |
| Grafana/Prometheus | Private (ClusterIP only) |

### Cost (~$320/mo)

| Resource | Monthly |
|----------|---------|
| EKS Fargate (15 pods) | ~$150 |
| EC2 node group (2× t3.medium) | ~$60 |
| RDS PostgreSQL | ~$25 |
| ElastiCache Redis | ~$29 |
| NAT Gateway | ~$40 |
| NLB | ~$18 |

### Staging vs Production

| | Staging | Production |
|---|---|---|
| Platform | Coolify (Docker Compose) | AWS EKS (Kubernetes) |
| URL | `https://parkir-pintar.piresc.dev` | `http://k8s-pirescer-gateway-....elb.ap-southeast-3.amazonaws.com` |
| TLS | ✅ (Traefik + Let's Encrypt) | ❌ (HTTP only) |
| Replicas | 1 per service | 2 per service (auto-scales) |
| Database | Docker PostgreSQL | AWS RDS (managed) |
| Redis | Docker Redis | AWS ElastiCache (managed) |
| Deploy trigger | Auto on push to main | Auto (on tag push) |
| Frontend | ✅ Deployed | ❌ Backend only |
| Rollback | Manual | Automatic on smoke test failure |
