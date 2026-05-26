# CI/CD Production Pipeline — Design Spec

**Date**: 2026-05-26  
**Status**: Approved  
**Project**: ParkirPintar

## Overview

Extend the existing CI/CD pipeline (GitHub Actions → GHCR → Coolify/staging) with a production path to AWS EKS. Staging stays on Coolify (simple, fast feedback). Production goes to AWS EKS (managed K8s, Terraform-as-code).

## Architecture

```
Developer → PR → ci-pr.yml (checks, block merge)
                  │
                  └→ Merge to main
                       ├── build-push-staging.yml → GHCR (main-$SHA, latest)
                       │      └── Coolify auto-deploy (staging, unchanged)
                       │
                       └── Create release tag v1.0.0
                            ├── build-push-prod.yml → GHCR (v1.0.0, latest-prod)
                            └── (manual) deploy-prod.yml
                                  ├── Terraform plan → approve → apply
                                  ├── K8s migration Job (all schemas)
                                  ├── kubectl set image (rolling update)
                                  └── k6 smoke test → pass or rollback
```

## Component: AWS Production Infrastructure

All infra defined in `infra/aws/` (Terraform).

### Resources

| Resource | Spec | Purpose |
|----------|------|---------|
| EKS Cluster | Fargate profiles + 1 EC2 node group (t3.medium × 2) | Fargate for 7 Go services + frontend; EC2 for NATS JetStream |
| RDS PostgreSQL | db.t3.small, 20GB gp3 | Database (schema-per-service) |
| ElastiCache Redis | cache.t3.small | Cache, distributed locks, Asynq queue |
| ALB (via AWS LB Controller on EKS) | Internet-facing HTTPS | Ingress for frontend and gateway |
| NAT Gateway | 1 gateway | Pod outbound (image pulls, external APIs) |
| ACM | TLS cert for domain | parkir-pintar.piresc.dev |
| Route 53 | A record → ALB | DNS |
| S3 | Terraform state + backups | State locking via DynamoDB |
| Secrets Manager | DB creds, JWT secret, etc. | Secrets → K8s Secrets via CSI driver |

### K8s Manifests

```
infra/aws/k8s/
├── base/
│   ├── namespace.yaml
│   ├── nats-statefulset.yaml            # NATS JetStream (EC2 node group)
│   ├── nats-service.yaml
│   ├── ghcr-pull-secret.yaml             # Image pull secret for GHCR
│   └── configmaps/
│       └── per-service.yaml
├── services/
│   ├── gateway/
│   │   ├── deployment.yaml
│   │   └── service.yaml
│   ├── search/
│   │   ├── deployment.yaml
│   │   └── service.yaml
│   ├── reservation/
│   │   ├── deployment.yaml
│   │   └── service.yaml
│   ├── billing/
│   │   ├── deployment.yaml
│   │   └── service.yaml
│   ├── payment/
│   │   ├── deployment.yaml
│   │   └── service.yaml
│   ├── analytics/
│   │   ├── deployment.yaml
│   │   └── service.yaml
│   ├── presence/
│   │   ├── deployment.yaml
│   │   └── service.yaml
│   └── frontend/
│       ├── deployment.yaml
│       ├── service.yaml
│       └── ingress.yaml
└── migrations/
    └── migration-job.yaml               # Single Job, all schemas sequentially
```

### EKS Hybrid: Fargate + EC2

- **Fargate profiles**: gateway, search, reservation, billing, payment, analytics, presence, frontend
- **EC2 node group**: t3.medium × 2, taint `nats-only:true`, only NATS StatefulSet pods schedule here
- **NATS StatefulSet**: 3 replicas, gp3 SSD persistent volumes, internal ClusterIP service

## Component: 4 GitHub Actions Workflows

### ci-pr.yml

Trigger: `pull_request → main`  
Jobs: secret-scan (gitleaks) → lint (golangci-lint), test (race + coverage), security (gosec), vulncheck (govulncheck), proto-check (buf lint + breaking), frontend-check (ESLint + test) → sonar (needs test + security)

Fixes from current:
- Pin all tool versions (no @latest)
- Add Go module caching (actions/cache)
- Add frontend lint + test step
- SonarCloud `continue-on-error` removed

### build-push-staging.yml

Trigger: `push → main`  
Jobs: Matrix build 7 Go services + frontend → GHCR tags: `main-$SHA`, `latest`

(Reuses current `ci.yml:169-268` logic, split into own file)

### build-push-prod.yml

Trigger: `tags: v*` (e.g. v1.0.0)  
Jobs: Matrix build 7 Go services + frontend → GHCR tags: `v1.0.0`, `latest-prod`

Same build logic as staging, different tag strategy.

### deploy-prod.yml

Trigger: `workflow_dispatch` with input `version` (e.g. v1.0.0)  
Jobs (sequential):
1. Configure AWS credentials (OIDC)
2. Terraform plan (post summary to job)
3. Manual approval (GitHub Environment "production")
4. Terraform apply
5. Run migration K8s Job (`kubectl apply -f migration-job.yaml`, wait for completion)
6. Rolling update (`kubectl set image` for all deployments, wait for rollout status)
7. k6 smoke test → fail triggers `kubectl rollout undo`
8. Create GitHub release notes

## Component: Reusable vs. New

| Reuse as-is | Modify slightly | Create new |
|-------------|----------------|------------|
| All 8 Dockerfiles | Go build/test steps (pin versions, add caching) | `infra/aws/` Terraform |
| .golangci.yml | Docker build-push steps (different tag strategy) | K8s manifests |
| .gitleaks.toml | Frontend build step (add lint + test) | build-push-prod.yml |
| sonar-project.properties | ci.yml → ci-pr.yml (split) | deploy-prod.yml |
| buf.yaml + buf.gen.yaml | | ci-pr.yml (from current ci.yml) |
| frontend build config | | build-push-staging.yml (from current ci.yml) |
| k6 load test scripts | | |
| db/migrations/ SQL files | | |
| Makefile | | |

## Configuration

### Environment Variables (AWS)

Injected via K8s ConfigMaps/Secrets (populated from GitHub Secrets → Terraform → AWS Secrets Manager → CSI driver → K8s):

- DB_USERNAME, DB_PASSWORD, DB_HOST, DB_PORT, DB_DATABASE
- REDIS_HOST, REDIS_PORT, REDIS_PASSWORD
- JWT_SECRET
- NATS_URL (internal: nats.parkir-pintar.svc.cluster.local:4222)
- OTEL_EXPORTER_OTLP_ENDPOINT (Tempo)

### GitHub Environments

| Environment | Protection |
|-------------|------------|
| `staging` | No approval, auto-deploy on push to main |
| `production` | 1 reviewer required, restricted to `workflow_dispatch` only |

## Rollback Strategy

If k6 smoke test fails after deploy:
1. `kubectl rollout undo deployment/<service> -n parkir-pintar`
2. Log failure to GitHub Actions output
3. Post alert (future: Slack webhook)

No DB rollback (migrations should be backward-compatible: add-only, no drops).

## Cost

| Resource | Monthly |
|----------|---------|
| EKS (Fargate + EC2 NATS) | ~$210 |
| RDS PostgreSQL | ~$25 |
| ElastiCache Redis | ~$29 |
| ALB + NAT Gateway | ~$55 |
| S3 + misc | ~$2 |
| **AWS Total** | **~$321** |
| Coolify staging VPS (existing) | ~$30 |
| **Combined Total** | **~$351** |

## Migration Strategy

Production uses the same GHCR images as staging. AWS infra is greenfield (no existing data to migrate). The project starts with a fresh AWS deployment alongside the existing Coolify staging.

Cutover:
1. Deploy to AWS EKS (fresh)
2. Run DB migrations (fresh DB, clean state)
3. Point DNS (parkir-pintar.piresc.dev) to AWS ALB
4. Coolify staging stays as `staging.parkir-pintar.piresc.dev`
