#!/bin/bash
# deploy.sh — Rolling deploy for parkir-pintar app stack
# Called by CI after images are pushed to GHCR.
# Uses docker compose to pull new images and recreate containers one by one.
set -euo pipefail

COMPOSE_DIR="/data/coolify/services/y149c53qjx3ktas665msck77"
PROJECT="y149c53qjx3ktas665msck77"

echo "=== Pulling latest images ==="
cd "$COMPOSE_DIR"
docker compose -p "$PROJECT" pull --ignore-pull-failures

echo "=== Recreating services (rolling) ==="
# Order matters: infra-dependent first, then dependents, then gateway last
for svc in billing search presence payment reservation frontend gateway; do
  echo "--- Restarting $svc ---"
  docker compose -p "$PROJECT" up -d --no-deps --force-recreate "$svc"
  # Wait for healthy
  timeout=60
  elapsed=0
  while [ $elapsed -lt $timeout ]; do
    status=$(docker inspect --format='{{.State.Health.Status}}' "${svc}-${PROJECT}" 2>/dev/null || echo "unknown")
    if [ "$status" = "healthy" ]; then
      echo "  ✓ $svc healthy"
      break
    fi
    sleep 3
    elapsed=$((elapsed + 3))
  done
  if [ $elapsed -ge $timeout ]; then
    echo "  ⚠ $svc did not become healthy within ${timeout}s"
  fi
done

# Clean up any orphaned containers
docker compose -p "$PROJECT" up -d --remove-orphans 2>/dev/null || true

echo "=== Deploy complete ==="
docker compose -p "$PROJECT" ps --format "table {{.Name}}\t{{.Status}}"
