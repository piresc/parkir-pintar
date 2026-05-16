# Coolify Deployment — ParkirPintar

Replaces the old `deploy/staging/docker-compose.yml` + Watchtower setup.

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
│  search               │  │  Alertmanager · Exporters    │
│  billing              │  │                              │
│  payment              │  │  (observability network)     │
│                       │  │                              │
│  (parkir-app network) │  │  UIs on Tailscale only       │
└───────────────────────┘  └──────────────────────────────┘
```

## Shared Networks

| Network | Purpose | Created by |
|---------|---------|-----------|
| `infra-backend` | App/Obs → Postgres, Redis, NATS | Infra stack |
| `observability` | App → Alloy (traces/metrics) | Observability stack |
| `parkir-app` | Internal app service mesh | App stack |

Any future project can join `infra-backend` and `observability` to reuse the same infra.

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
- Set environment variables (DB_*, JWT_SECRET, REDIS_PASSWORD)
- Note the webhook URL from Coolify for this stack

### 4. Configure GitHub Secrets
Add these secrets to the GitHub repo (Settings → Secrets → Actions):
- `COOLIFY_WEBHOOK_URL` — the webhook URL from Coolify's app stack
- `COOLIFY_TOKEN` — API token from Coolify (Settings → API Tokens → Generate)

## Deployment Flow

```
Push to main
    → GitHub Actions CI (lint, test, scan, build)
    → Push image to GHCR (ghcr.io/piresc/parkir-pintar:latest)
    → Trigger Coolify webhook
    → Coolify pulls new image and redeploys app stack
```

## Webhook Security

The webhook is secured by:
1. Coolify's unique webhook UUID (unguessable URL)
2. Bearer token authentication (COOLIFY_TOKEN)

Both are stored as GitHub secrets and never exposed in logs.

## Migration from Old Setup

The old `deploy/staging/docker-compose.yml` with Watchtower is deprecated.

To migrate:
1. Deploy the 3 Coolify stacks (infra → obs → app)
2. Verify all services are healthy
3. Stop the old stack: `cd deploy/staging && docker compose down`
4. Remove Watchtower: `docker rm -f staging-watchtower`
5. Migrate volumes if needed (named volumes persist across stacks)

## Volume Migration Note

Old volumes: `staging-postgres_data`, `staging-redis_data`, `staging-nats_data`
New volumes: `postgres_data`, `redis_data`, `nats_data` (in Coolify's project scope)

To preserve data, either:
- Rename old volumes before deploying: `docker volume create --name <new> && docker run --rm -v <old>:/from -v <new>:/to alpine cp -a /from/. /to/`
- Or point new compose at old volume names
