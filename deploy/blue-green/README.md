# ParkirPintar Blue-Green Deployment

## Overview

Blue-green deployment provides zero-downtime releases by maintaining two identical production environments. At any time, only one environment (blue or green) serves live traffic while the other stands by for the next release or as a rollback target.

## Architecture

```
                    ┌─────────────┐
    Internet ──────►│   Traefik   │
                    │  (Router)   │
                    └──────┬──────┘
                           │
              ┌────────────┼────────────┐
              │ priority=100            │ priority=50
              ▼                         ▼
    ┌─────────────────┐      ┌─────────────────┐
    │  BLUE (active)  │      │ GREEN (standby) │
    │                 │      │                 │
    │  gateway-blue   │      │  gateway-green  │
    │  search-blue    │      │  search-green   │
    │  reservation-   │      │  reservation-   │
    │  billing-blue   │      │  billing-green  │
    │  payment-blue   │      │  payment-green  │
    │  presence-blue  │      │  presence-green │
    │  notification-  │      │  notification-  │
    └─────────────────┘      └─────────────────┘
              │                         │
              └────────────┬────────────┘
                           │
              ┌────────────┼────────────┐
              ▼            ▼            ▼
         PostgreSQL      Redis        NATS
         (shared)       (shared)    (shared)
```

## How It Works

1. **Traefik Priority Routing**: Both deployments register with Traefik using the same domain. The active deployment has `priority=100`, the standby has `priority=50`. Traefik routes all traffic to the higher-priority router.

2. **Shared Infrastructure**: PostgreSQL, Redis, and NATS are shared between blue and green. Database migrations must be backward-compatible.

3. **Independent Application Services**: Each color has its own set of 7 application services (gateway, search, reservation, billing, payment, presence, notification).

## Prerequisites

- Docker and Docker Compose v2+
- Traefik reverse proxy running on the `parkir-net` network
- `.env` file with required credentials (copy from `.env.example`)
- External network `parkir-net` created: `docker network create parkir-net`

## Quick Start

### Initial Setup

```bash
# Create the shared network
docker network create parkir-net

# Start with blue as active
export IMAGE_TAG=v1.0.0
docker compose -f docker-compose.blue.yml up -d
```

### Deploy a New Version

```bash
# 1. Deploy new version to the standby environment (green)
export IMAGE_TAG=v1.1.0
docker compose -f docker-compose.green.yml up -d

# 2. Wait for health checks to pass, then switch traffic
./switch.sh green

# 3. Verify the new deployment is working
curl -s https://parkir-pintar.piresc.dev/health

# 4. (Optional) Remove old deployment after confidence period
docker compose -f docker-compose.blue.yml down
```

### Rollback

```bash
# Quick rollback to previous deployment
./switch.sh --rollback

# Or explicitly target a color
./switch.sh blue
```

## switch.sh Usage

```
./switch.sh <blue|green> [options]

Options:
    --rollback  Switch to whichever deployment is NOT currently active
    --force     Skip health checks (emergency use only)
    --dry-run   Show what would happen without making changes
    --help      Show help
```

## Deployment Workflow

### Standard Release

1. Identify current active color (e.g., blue)
2. Build and tag new image: `ghcr.io/piresc/parkir-pintar:v1.2.0`
3. Update standby (green) with new image tag
4. Run database migrations (must be backward-compatible)
5. Start green deployment
6. Verify green health checks pass
7. Switch traffic: `./switch.sh green`
8. Monitor for errors (5-15 minutes)
9. If issues: `./switch.sh --rollback`
10. If stable: tear down old blue deployment

### Database Migrations

Since both deployments share the same database, migrations must follow the **expand-contract** pattern:

1. **Expand**: Add new columns/tables without removing old ones
2. **Deploy**: Switch traffic to new version that uses new schema
3. **Contract**: Remove old columns/tables in a subsequent release

## Environment Variables

| Variable | Description |
|----------|-------------|
| `IMAGE_TAG` | Docker image tag (e.g., `v1.2.0`) |
| `DOMAIN` | Production domain for Traefik routing |
| `ALLOWED_ORIGINS` | CORS allowed origins |
| `DB_USERNAME` | PostgreSQL username |
| `DB_PASSWORD` | PostgreSQL password |
| `DB_DATABASE` | PostgreSQL database name |
| `REDIS_PASSWORD` | Redis password |
| `JWT_SECRET` | JWT signing secret |
| `TRACING_OTLP_ENDPOINT` | OpenTelemetry collector endpoint |

## Monitoring

Check deployment status:

```bash
# See which containers are running
docker ps --filter "label=parkir.deployment" --format "table {{.Names}}\t{{.Status}}\t{{.Label \"parkir.version\"}}"

# Check gateway health
curl -s http://localhost:8080/health | jq .

# View logs for a specific color
docker compose -f docker-compose.green.yml logs -f gateway-green
```

## Troubleshooting

### Health checks failing
```bash
# Check individual service health
docker inspect --format='{{.State.Health}}' parkir-gateway-green

# View service logs
docker logs parkir-gateway-green --tail 50
```

### Traffic not switching
```bash
# Verify Traefik can see the routers
curl -s http://localhost:8080/api/http/routers | jq '.[] | select(.name | contains("parkir"))'

# Check container labels
docker inspect parkir-gateway-blue --format='{{json .Config.Labels}}' | jq .
```

### Rollback not working
```bash
# Force switch without health checks
./switch.sh blue --force

# Nuclear option: restart Traefik to re-read labels
docker restart traefik
```
