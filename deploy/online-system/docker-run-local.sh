#!/usr/bin/env bash

# Start the all-in-one online system locally with Docker Compose.
# Usage:
#   ./docker-run-local.sh [IMAGE]

set -euo pipefail

cd "$(dirname "$0")"

DEFAULT_IMAGE="crypto-custody-online-system:local"
export ONLINE_SYSTEM_IMAGE="${1:-${ONLINE_SYSTEM_IMAGE:-$DEFAULT_IMAGE}}"
export DOCKER_PLATFORM="${DOCKER_PLATFORM:-linux/amd64}"

if ! docker info >/dev/null 2>&1; then
    echo "Error: Docker is not running. Start Docker and retry."
    exit 1
fi

if docker compose version >/dev/null 2>&1; then
    COMPOSE=(docker compose)
elif command -v docker-compose >/dev/null 2>&1; then
    COMPOSE=(docker-compose)
else
    echo "Error: Docker Compose is not installed."
    exit 1
fi

mkdir -p database logs backups

if [[ ! -f ".env" ]]; then
    cp .env.example .env
fi

if [[ "${ONLINE_SYSTEM_IMAGE}" == *":local" ]] && ! docker image inspect "${ONLINE_SYSTEM_IMAGE}" >/dev/null 2>&1; then
    echo "Local image is missing: ${ONLINE_SYSTEM_IMAGE}"
    echo "Build it first:"
    echo "  ./docker-build-image.sh ${ONLINE_SYSTEM_IMAGE}"
    exit 1
fi

echo "======================================"
echo "Start online-system locally"
echo "======================================"
echo "Image: ${ONLINE_SYSTEM_IMAGE}"
echo "URL: http://127.0.0.1:${WEB_HOST_PORT:-8088}"
echo "======================================"

"${COMPOSE[@]}" up -d
"${COMPOSE[@]}" ps
