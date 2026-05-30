# Coolify Staging — Setup Runbook

> Full architecture details: [docs/operations/deployment.md](../../docs/operations/deployment.md)

## Setup Order

1. **Infra Stack** — deploy first (creates `infra-backend` network)
2. **Observability Stack** — deploy second (creates `observability` network, joins `infra-backend`)
3. **App Stack** — deploy last (joins both networks)

## Coolify Setup Steps

### 1. Create Infra Stack
- Coolify → Projects → New → Docker Compose
- Source: `deploy/coolify/infra/docker-compose.yml`
- Set environment variables: `DB_DATABASE`, `DB_USERNAME`, `DB_PASSWORD`, `REDIS_PASSWORD`

### 2. Create Observability Stack
- Coolify → Projects → New → Docker Compose
- Source: `deploy/coolify/observability/docker-compose.yml`
- Set environment variables: `DB_*`, `REDIS_PASSWORD`, `GRAFANA_PASSWORD`
- Copy config files (prometheus, grafana, tempo, loki, alloy, alertmanager) into the build path

### 3. Create App Stack
- Coolify → Projects → New → Docker Compose
- Source: `deploy/coolify/app/docker-compose.yml`
- Set environment variables: `DB_*`, `JWT_SECRET`, `REDIS_PASSWORD`, `NATS_URL`

### 4. Configure GitHub Secrets
Add these secrets to the GitHub repo (Settings → Secrets → Actions):
- `COOLIFY_TOKEN` — API token from Coolify (Settings → API Tokens → Generate)

## Stack UUIDs (current)

| Stack | UUID |
|-------|------|
| App | `y149c53qjx3ktas665msck77` |
| Infra | `jm5muzbk2w3hjipndzrk0x26` |
| Observability | `u4487ozi9oim2vj88ut3sxdp` |
