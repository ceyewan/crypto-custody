#!/usr/bin/env bash

# Start the offline-server locally with Docker Compose.
# Usage:
#   ./docker-run-local.sh [IMAGE]
#
# Examples:
#   ./docker-build-image.sh
#   ./docker-run-local.sh
#   ./docker-run-local.sh ceyewan/crypto-custody-offline-server:latest
#
# Optional env:
#   OFFLINE_WEB_HOST_PORT=8080
#   OFFLINE_WS_HOST_PORT=8081
#   OFFLINE_MANAGER_PUBLIC_HOST=127.0.0.1
#   OFFLINE_MANAGER_PORT_START=18001
#   OFFLINE_MANAGER_PORT_END=18100
#   OFFLINE_SERVER_PULL_POLICY=missing

set -euo pipefail

cd "$(dirname "$0")"

DEFAULT_IMAGE="crypto-custody-offline-server:local"
export OFFLINE_SERVER_IMAGE="${1:-${OFFLINE_SERVER_IMAGE:-$DEFAULT_IMAGE}}"
export DOCKER_PLATFORM="${DOCKER_PLATFORM:-linux/amd64}"
export OFFLINE_MANAGER_PUBLIC_HOST="${OFFLINE_MANAGER_PUBLIC_HOST:-127.0.0.1}"
export OFFLINE_MANAGER_PORT_START="${OFFLINE_MANAGER_PORT_START:-18001}"
export OFFLINE_MANAGER_PORT_END="${OFFLINE_MANAGER_PORT_END:-18100}"

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

mkdir -p data logs private_keys
if [[ ! -f "private_keys/ec_private_key.pem" ]]; then
    echo "Error: private_keys/ec_private_key.pem is required."
    echo "Copy in the offline-server ECDSA private key that matches the SE applet public key."
    exit 1
fi
chmod 600 private_keys/ec_private_key.pem || true

if [[ "${OFFLINE_SERVER_IMAGE}" == *":local" ]] && ! docker image inspect "${OFFLINE_SERVER_IMAGE}" >/dev/null 2>&1; then
    echo "Local image is missing: ${OFFLINE_SERVER_IMAGE}"
    echo "Build it first:"
    echo "  ./docker-build-image.sh ${OFFLINE_SERVER_IMAGE}"
    exit 1
fi

echo "======================================"
echo "Start offline-server locally"
echo "======================================"
echo "Image: ${OFFLINE_SERVER_IMAGE}"
echo "Web: 127.0.0.1:${OFFLINE_WEB_HOST_PORT:-8080}"
echo "WebSocket: 127.0.0.1:${OFFLINE_WS_HOST_PORT:-8081}"
echo "Manager public host: ${OFFLINE_MANAGER_PUBLIC_HOST}"
echo "Manager ports: ${OFFLINE_MANAGER_PORT_START}-${OFFLINE_MANAGER_PORT_END}"
echo "======================================"

"${COMPOSE[@]}" up -d
"${COMPOSE[@]}" ps

echo "======================================"
echo "Started."
echo "Useful commands:"
echo "  ${COMPOSE[*]} logs -f offline-server"
echo "  ${COMPOSE[*]} down"
echo "======================================"
