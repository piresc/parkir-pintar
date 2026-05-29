# Coolify Deployment — ParkirPintar

## Architecture

Three separate stacks in Coolify, connected via shared Docker networks:

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
│  billing              │  │  Alertmanager · Exporters    │
│  payment              │  │  (postgres, redis, nats)     │
│  search               │  │                              │
│  presence             │  │  (observability network)     │
│  analytics            │  │                              │
│  frontend             │  │  UIs on Tailscale only       │
│                       │  │  (100.79.123.39)             │
│  (parkir-app network) │  │                              │
└───────────────────────┘  └──────────────────────────────┘
```

## Shared Networks

| Network | Purpose | Created by |
|---------|---------|------------|
| `infra-backend` | App/Obs → Postgres, Redis, NATS | Infra stack |
| `observability` | App → Alloy (traces/metrics) | Observability stack |
| `parkir-app` | Internal app service mesh | App stack |

## Deployment Flow

```
Push to main
    → GitHub Actions CI (lint, test, build per-service images)
    → Push images to GHCR (ghcr.io/piresc/parkir-pintar/<service>:latest)
    → CI calls Coolify API to restart app stack
    → deploy.sh does rolling recreate per service
```

Per-service images:
- `ghcr.io/piresc/parkir-pintar/gateway:latest`
- `ghcr.io/piresc/parkir-pintar/reservation:latest`
- `ghcr.io/piresc/parkir-pintar/billing:latest`
- `ghcr.io/piresc/parkir-pintar/payment:latest`
- `ghcr.io/piresc/parkir-pintar/search:latest`
- `ghcr.io/piresc/parkir-pintar/presence:latest`
- `ghcr.io/piresc/parkir-pintar/analytics:latest`
- `ghcr.io/piresc/parkir-pintar/frontend:latest`

## Setup Order

1. **Infra Stack** — deploy first (creates `infra-backend` network)
2. **Observability Stack** — deploy second (creates `observability` network, joins `infra-backend`)
3. **App Stack** — deploy last (joins both networks)

## Coolify Setup Steps

### 1. Create Infra Stack
- Coolify → Projects → New → Docker Compose
- Source: `deploy/coolify/infra/docker-compose.yml`
- Set environment variables (DB_DATABASE, DB_USERNAME, DB_PASSWORD, REDIS_PASSWORD)

### 2. Create Observability Stack
- Coolify → Projects → New → Docker Compose
- Source: `deploy/coolify/observability/docker-compose.yml`
- Set environment variables (DB_*, REDIS_PASSWORD, GRAFANA_PASSWORD)
- Copy config files (prometheus, grafana, tempo, loki, alloy, alertmanager) into the build path

### 3. Create App Stack
- Coolify → Projects → New → Docker Compose
- Source: `deploy/coolify/app/docker-compose.yml`
- Set environment variables (DB_*, JWT_SECRET, REDIS_PASSWORD, NATS_URL)

### 4. Configure GitHub Secrets
Add these secrets to the GitHub repo (Settings → Secrets → Actions):
- `COOLIFY_TOKEN` — API token from Coolify (Settings → API Tokens → Generate)

## Stack UUIDs (current)

| Stack | UUID |
|-------|------|
| App | `y149c53qjx3ktas665msck77` |
| Infra | `jm5muzbk2w3hjipndzrk0x26` |
| Observability | `u4487ozi9oim2vj88ut3sxdp` |
