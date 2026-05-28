#!/usr/bin/env bash

# Deploy offline-server with Docker Compose.
# Usage:
#   OFFLINE_MANAGER_PUBLIC_HOST=192.168.1.10 ./deploy.sh
#
# Required local file:
#   private_keys/ec_private_key.pem

set -euo pipefail

cd "$(dirname "$0")"

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

if [[ ! -f ".env" ]]; then
    cp .env.example .env
    echo "Created .env from .env.example"
fi

mkdir -p data logs private_keys
if [[ ! -f "private_keys/ec_private_key.pem" ]]; then
    echo "Error: private_keys/ec_private_key.pem is required."
    echo "Copy in the offline-server ECDSA private key that matches the SE applet public key."
    exit 1
fi
chmod 600 private_keys/ec_private_key.pem || true

PUBLIC_HOST="${OFFLINE_MANAGER_PUBLIC_HOST:-}"
if [[ -z "${PUBLIC_HOST}" ]]; then
    PUBLIC_HOST="$(grep -E '^OFFLINE_MANAGER_PUBLIC_HOST=' .env | tail -1 | cut -d= -f2- || true)"
fi
if [[ -z "${PUBLIC_HOST}" || "${PUBLIC_HOST}" == "127.0.0.1" || "${PUBLIC_HOST}" == "localhost" ]]; then
    echo "Warning: OFFLINE_MANAGER_PUBLIC_HOST is '${PUBLIC_HOST:-unset}'."
    echo "Remote desktop clients need this to be the offline server IP or DNS name."
fi

echo "======================================"
echo "Deploying crypto-custody offline-server"
echo "Image: ${OFFLINE_SERVER_IMAGE:-$(grep -E '^OFFLINE_SERVER_IMAGE=' .env | tail -1 | cut -d= -f2- || echo ceyewan/crypto-custody-offline-server:latest)}"
echo "Manager public host: ${PUBLIC_HOST:-unset}"
echo "Work dir: $(pwd)"
echo "======================================"

"${COMPOSE[@]}" pull || true
"${COMPOSE[@]}" up -d
"${COMPOSE[@]}" ps

echo "======================================"
echo "Deployment complete."
echo "Useful commands:"
echo "  ${COMPOSE[*]} logs -f offline-server"
echo "  ${COMPOSE[*]} down"
echo "======================================"
