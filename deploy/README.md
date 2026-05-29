# Deployment

ParkirPintar has three deployment environments:

## Local Development
- `deploy/local/docker-compose.yml` — Local dev stack (PostgreSQL, Redis, NATS, all services)
- Run with: `make docker-up`

## Staging (Coolify)
- `deploy/staging/` — Docker Compose stacks deployed via Coolify
  - `infra/` — PostgreSQL, Redis, NATS
  - `app/` — 7 Go services + frontend
  - `observability/` — Prometheus, Grafana, Tempo, Loki, Alloy
  - `deploy-webhook.sh` — Webhook endpoint triggered by CI/CD

## Production (AWS EKS)
- `deploy/production/` — AWS infrastructure and Kubernetes manifests
  - `terraform/` — Terraform IaC (VPC, EKS, RDS, ElastiCache, IAM)
  - `kubernetes/` — K8s manifests (services, autoscaling, observability, NATS)
  - `setup.sh` — One-time bootstrap script

## CI/CD
- `.github/workflows/pr-checks.yml` — PR quality gates
- `.github/workflows/deploy-staging.yml` — Push to main → build → deploy to staging
- `.github/workflows/build-prod.yml` — Tag v* → build images → GHCR
- `.github/workflows/deploy-prod.yml` — Auto-deploy to EKS on build success, or manual trigger
