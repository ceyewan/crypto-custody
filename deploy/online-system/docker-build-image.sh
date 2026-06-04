#!/usr/bin/env bash

# Build the all-in-one online system Docker image into the local Docker image store.
# Usage:
#   ./docker-build-image.sh [IMAGE]
#
# Optional env:
#   DOCKER_PLATFORM=linux/amd64

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

DEFAULT_IMAGE="crypto-custody-online-system:local"
IMAGE="${1:-${ONLINE_SYSTEM_IMAGE:-$DEFAULT_IMAGE}}"
PLATFORM="${DOCKER_PLATFORM:-linux/amd64}"

echo "======================================"
echo "Build online-system Docker image"
echo "======================================"
echo "Image: ${IMAGE}"
echo "Platform: ${PLATFORM}"
echo "Context: ${REPO_ROOT}"
echo "======================================"

if ! docker info >/dev/null 2>&1; then
    echo "Error: Docker is not running. Start Docker and retry."
    exit 1
fi

if ! docker buildx version >/dev/null 2>&1; then
    echo "Error: docker buildx is not available."
    exit 1
fi

docker buildx build \
    --platform "${PLATFORM}" \
    -f "${SCRIPT_DIR}/Dockerfile" \
    -t "${IMAGE}" \
    --load \
    "${REPO_ROOT}"

echo "======================================"
echo "Built: ${IMAGE}"
echo "======================================"
