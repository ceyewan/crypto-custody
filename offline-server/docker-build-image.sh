#!/usr/bin/env bash

# Build the offline-server Docker image into the local Docker image store.
# Usage:
#   ./docker-build-image.sh [IMAGE]
#
# Examples:
#   ./docker-build-image.sh
#   ./docker-build-image.sh crypto-custody-offline-server:local
#   ./docker-build-image.sh ceyewan/crypto-custody-offline-server:latest
#
# Optional env:
#   DOCKER_PLATFORM=linux/amd64

set -euo pipefail

cd "$(dirname "$0")"

DEFAULT_IMAGE="crypto-custody-offline-server:local"
IMAGE="${1:-${OFFLINE_SERVER_IMAGE:-$DEFAULT_IMAGE}}"
PLATFORM="${DOCKER_PLATFORM:-linux/amd64}"

echo "======================================"
echo "Build offline-server Docker image"
echo "======================================"
echo "Image: ${IMAGE}"
echo "Platform: ${PLATFORM}"
echo "Context: $(pwd)"
echo "======================================"

if [[ "${PLATFORM}" != "linux/amd64" ]]; then
    echo "Error: offline-server image currently supports linux/amd64 only."
    echo "Reason: the bundled gg20 manager is bin/gg20_sm_manager_linux_amd64."
    exit 1
fi

if ! docker info >/dev/null 2>&1; then
    echo "Error: Docker is not running. Start Docker and retry."
    exit 1
fi

if ! docker buildx version >/dev/null 2>&1; then
    echo "Error: docker buildx is not available."
    exit 1
fi

if [[ ! -x "bin/gg20_sm_manager_linux_amd64" ]]; then
    echo "Error: bin/gg20_sm_manager_linux_amd64 is missing or not executable."
    exit 1
fi

docker buildx build \
    --platform "${PLATFORM}" \
    -t "${IMAGE}" \
    --load \
    .

echo "======================================"
echo "Built: ${IMAGE}"
echo "======================================"
