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
