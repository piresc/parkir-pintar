#!/bin/bash
# deploy-webhook.sh — lightweight HTTP webhook that pulls latest images and recreates containers
# Runs as a simple socat/netcat listener on the server, triggered by CI after image push.
# Uses Coolify's stored compose (has injected env vars and labels).

set -euo pipefail

COMPOSE_FILE="/data/coolify/services/y149c53qjx3ktas665msck77/docker-compose.yml"
PROJECT="y149c53qjx3ktas665msck77"
LOG="/var/log/parkir-deploy.log"

echo "$(date -Iseconds) Deploy triggered" >> "$LOG"

# Remove any orphan "Created" containers
docker ps -a --filter "status=created" --filter "name=$PROJECT" -q | xargs -r docker rm -f >> "$LOG" 2>&1

# Pull latest images
docker compose -f "$COMPOSE_FILE" -p "$PROJECT" pull >> "$LOG" 2>&1

# Recreate with latest images
docker compose -f "$COMPOSE_FILE" -p "$PROJECT" up -d >> "$LOG" 2>&1

echo "$(date -Iseconds) Deploy complete" >> "$LOG"
