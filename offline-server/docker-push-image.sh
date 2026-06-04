#!/usr/bin/env bash

# Tag and push an offline-server Docker image to DockerHub.
# Usage:
#   ./docker-push-image.sh [LOCAL_IMAGE] [REMOTE_IMAGE]
#
# Examples:
#   ./docker-push-image.sh
#   ./docker-push-image.sh crypto-custody-offline-server:local ceyewan/crypto-custody-offline-server:latest
#   ./docker-push-image.sh ceyewan/crypto-custody-offline-server:v1.0.0 ceyewan/crypto-custody-offline-server:v1.0.0

set -euo pipefail

cd "$(dirname "$0")"

DEFAULT_LOCAL_IMAGE="crypto-custody-offline-server:local"
DEFAULT_REMOTE_IMAGE="ceyewan/crypto-custody-offline-server:latest"

LOCAL_IMAGE="${1:-${OFFLINE_SERVER_LOCAL_IMAGE:-$DEFAULT_LOCAL_IMAGE}}"
REMOTE_IMAGE="${2:-${OFFLINE_SERVER_REMOTE_IMAGE:-$DEFAULT_REMOTE_IMAGE}}"

echo "======================================"
echo "Push offline-server Docker image"
echo "======================================"
echo "Local image: ${LOCAL_IMAGE}"
echo "Remote image: ${REMOTE_IMAGE}"
echo "======================================"

if ! docker info >/dev/null 2>&1; then
    echo "Error: Docker is not running. Start Docker and retry."
    exit 1
fi

if ! docker image inspect "${LOCAL_IMAGE}" >/dev/null 2>&1; then
    echo "Error: local image does not exist: ${LOCAL_IMAGE}"
    echo "Build it first, for example:"
    echo "  ./docker-build-image.sh ${LOCAL_IMAGE}"
    exit 1
fi

if [[ "${LOCAL_IMAGE}" != "${REMOTE_IMAGE}" ]]; then
    docker tag "${LOCAL_IMAGE}" "${REMOTE_IMAGE}"
fi

docker push "${REMOTE_IMAGE}"

echo "======================================"
echo "Pushed: ${REMOTE_IMAGE}"
echo "======================================"
