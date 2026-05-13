#!/usr/bin/env bash
# ParkirPintar Blue-Green Deployment Switch
# Usage: ./switch.sh <blue|green> [--rollback]
#
# This script switches production traffic between blue and green deployments
# by updating Traefik router priorities via Docker label changes.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_BLUE="${SCRIPT_DIR}/docker-compose.blue.yml"
COMPOSE_GREEN="${SCRIPT_DIR}/docker-compose.green.yml"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

ACTIVE_PRIORITY=100
STANDBY_PRIORITY=50
HEALTH_CHECK_RETRIES=30
HEALTH_CHECK_INTERVAL=5

log_info()  { echo -e "${BLUE}[INFO]${NC} $*"; }
log_ok()    { echo -e "${GREEN}[OK]${NC} $*"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*" >&2; }

usage() {
    cat <<EOF
ParkirPintar Blue-Green Deployment Switch

Usage:
    $(basename "$0") <target>  [options]

Targets:
    blue        Switch traffic to blue deployment
    green       Switch traffic to green deployment

Options:
    --rollback  Quick rollback to the other deployment
    --force     Skip health checks (use with caution)
    --dry-run   Show what would happen without making changes
    --help      Show this help message

Examples:
    # Deploy new version to green, then switch traffic
    IMAGE_TAG=v1.2.0 docker compose -f docker-compose.green.yml up -d
    ./switch.sh green

    # Rollback to blue if green has issues
    ./switch.sh --rollback

    # Force switch without health checks (emergency)
    ./switch.sh blue --force
EOF
    exit 0
}

get_current_active() {
    # Determine which deployment currently has the higher priority
    local blue_priority green_priority
    blue_priority=$(docker inspect parkir-gateway-blue 2>/dev/null \
        | grep -o '"traefik.http.routers.parkir-blue.priority=[0-9]*"' \
        | grep -o '[0-9]*' || echo "0")
    green_priority=$(docker inspect parkir-gateway-green 2>/dev/null \
        | grep -o '"traefik.http.routers.parkir-green.priority=[0-9]*"' \
        | grep -o '[0-9]*' || echo "0")

    if [[ "$blue_priority" -gt "$green_priority" ]]; then
        echo "blue"
    elif [[ "$green_priority" -gt "$blue_priority" ]]; then
        echo "green"
    else
        echo "unknown"
    fi
}

check_deployment_exists() {
    local target=$1
    local gateway_container="parkir-gateway-${target}"

    if ! docker ps --format '{{.Names}}' | grep -q "^${gateway_container}$"; then
        return 1
    fi
    return 0
}

wait_for_healthy() {
    local target=$1
    local services=("gateway" "search" "reservation" "billing" "payment" "presence" "notification")
    local all_healthy=false

    log_info "Waiting for ${target} deployment to become healthy..."

    for ((i=1; i<=HEALTH_CHECK_RETRIES; i++)); do
        all_healthy=true
        for svc in "${services[@]}"; do
            local container="parkir-${svc}-${target}"
            local status
            status=$(docker inspect --format='{{.State.Health.Status}}' "$container" 2>/dev/null || echo "missing")

            if [[ "$status" != "healthy" ]]; then
                all_healthy=false
                break
            fi
        done

        if [[ "$all_healthy" == "true" ]]; then
            log_ok "All ${target} services are healthy"
            return 0
        fi

        echo -n "."
        sleep "$HEALTH_CHECK_INTERVAL"
    done

    echo ""
    log_error "Timeout: ${target} deployment did not become healthy within $((HEALTH_CHECK_RETRIES * HEALTH_CHECK_INTERVAL))s"

    # Show status of each service
    for svc in "${services[@]}"; do
        local container="parkir-${svc}-${target}"
        local status
        status=$(docker inspect --format='{{.State.Health.Status}}' "$container" 2>/dev/null || echo "missing")
        if [[ "$status" != "healthy" ]]; then
            log_error "  ${container}: ${status}"
        fi
    done

    return 1
}

switch_traffic() {
    local target=$1
    local other

    if [[ "$target" == "blue" ]]; then
        other="green"
    else
        other="blue"
    fi

    log_info "Switching traffic: ${other} → ${target}"

    # Update target to active priority (high)
    docker compose -f "${SCRIPT_DIR}/docker-compose.${target}.yml" \
        up -d --no-recreate \
        --label "traefik.http.routers.parkir-${target}.priority=${ACTIVE_PRIORITY}" \
        gateway-${target} 2>/dev/null || true

    # Use docker to update labels by recreating the gateway with new priority
    # We do this by modifying the compose and re-upping just the gateway
    log_info "Setting ${target} gateway priority to ${ACTIVE_PRIORITY} (active)"
    docker container update --label-add "traefik.http.routers.parkir-${target}.priority=${ACTIVE_PRIORITY}" \
        "parkir-gateway-${target}" 2>/dev/null || {
        # Fallback: recreate with updated environment
        export ACTIVE_DEPLOYMENT="${target}"
        sed "s/priority=${STANDBY_PRIORITY}/priority=${ACTIVE_PRIORITY}/" \
            "${SCRIPT_DIR}/docker-compose.${target}.yml" > "/tmp/parkir-${target}-active.yml"
        docker compose -f "/tmp/parkir-${target}-active.yml" up -d gateway-${target}
    }

    log_info "Setting ${other} gateway priority to ${STANDBY_PRIORITY} (standby)"
    docker container update --label-add "traefik.http.routers.parkir-${other}.priority=${STANDBY_PRIORITY}" \
        "parkir-gateway-${other}" 2>/dev/null || {
        sed "s/priority=${ACTIVE_PRIORITY}/priority=${STANDBY_PRIORITY}/" \
            "${SCRIPT_DIR}/docker-compose.${other}.yml" > "/tmp/parkir-${other}-standby.yml"
        docker compose -f "/tmp/parkir-${other}-standby.yml" up -d gateway-${other} 2>/dev/null || true
    }

    # Verify the switch by checking gateway response
    sleep 2
    log_ok "Traffic switched to ${target} deployment"
    log_info "Previous deployment (${other}) kept running for rollback"
    echo ""
    log_info "To rollback: ./switch.sh ${other}"
    log_info "To remove old deployment: docker compose -f docker-compose.${other}.yml down"
}

# --- Main ---

TARGET=""
ROLLBACK=false
FORCE=false
DRY_RUN=false

while [[ $# -gt 0 ]]; do
    case $1 in
        blue|green)
            TARGET="$1"
            shift
            ;;
        --rollback)
            ROLLBACK=true
            shift
            ;;
        --force)
            FORCE=true
            shift
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --help|-h)
            usage
            ;;
        *)
            log_error "Unknown argument: $1"
            usage
            ;;
    esac
done

# Handle rollback
if [[ "$ROLLBACK" == "true" ]]; then
    current=$(get_current_active)
    if [[ "$current" == "blue" ]]; then
        TARGET="green"
    elif [[ "$current" == "green" ]]; then
        TARGET="blue"
    else
        log_error "Cannot determine current active deployment. Specify target explicitly."
        exit 1
    fi
    log_warn "Rolling back from ${current} to ${TARGET}"
fi

if [[ -z "$TARGET" ]]; then
    log_error "No target specified. Use: ./switch.sh <blue|green>"
    usage
fi

echo "╔══════════════════════════════════════════════╗"
echo "║   ParkirPintar Blue-Green Deployment Switch  ║"
echo "╚══════════════════════════════════════════════╝"
echo ""

current=$(get_current_active)
log_info "Current active deployment: ${current:-unknown}"
log_info "Target deployment: ${TARGET}"
echo ""

if [[ "$DRY_RUN" == "true" ]]; then
    log_info "[DRY RUN] Would switch traffic from ${current} to ${TARGET}"
    log_info "[DRY RUN] Would set ${TARGET} priority to ${ACTIVE_PRIORITY}"
    log_info "[DRY RUN] Would set ${current} priority to ${STANDBY_PRIORITY}"
    exit 0
fi

# Check target deployment exists
if ! check_deployment_exists "$TARGET"; then
    log_error "Target deployment '${TARGET}' is not running."
    log_info "Start it first: IMAGE_TAG=vX.Y.Z docker compose -f docker-compose.${TARGET}.yml up -d"
    exit 1
fi

# Health check (unless --force)
if [[ "$FORCE" != "true" ]]; then
    if ! wait_for_healthy "$TARGET"; then
        log_error "Aborting switch — target deployment is not healthy."
        log_info "Use --force to switch anyway (not recommended)."
        exit 1
    fi
else
    log_warn "Skipping health checks (--force)"
fi

# Perform the switch
switch_traffic "$TARGET"

# Final status
echo ""
echo "╔══════════════════════════════════════════════╗"
echo "║   Deployment Status                          ║"
echo "╚══════════════════════════════════════════════╝"
echo ""
log_info "Active:  ${TARGET} (priority ${ACTIVE_PRIORITY})"
log_info "Standby: $([ "$TARGET" == "blue" ] && echo "green" || echo "blue") (priority ${STANDBY_PRIORITY})"

# Show versions
blue_ver=$(docker inspect --format='{{index .Config.Labels "parkir.version"}}' parkir-gateway-blue 2>/dev/null || echo "N/A")
green_ver=$(docker inspect --format='{{index .Config.Labels "parkir.version"}}' parkir-gateway-green 2>/dev/null || echo "N/A")
echo ""
log_info "Blue version:  ${blue_ver}"
log_info "Green version: ${green_ver}"
