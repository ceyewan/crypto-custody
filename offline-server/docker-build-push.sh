#!/usr/bin/env bash

# crypto-custody offline-server Docker build and push script.
# Usage: ./docker-build-push.sh [DOCKERHUB_USERNAME] [IMAGE_NAME] [TAG]
# Optional env: PLATFORMS=linux/amd64

set -euo pipefail

DEFAULT_USERNAME="ceyewan"
DEFAULT_IMAGE_NAME="crypto-custody-offline-server"
DEFAULT_TAG="latest"

DOCKERHUB_USERNAME="${1:-$DEFAULT_USERNAME}"
IMAGE_NAME="${2:-$DEFAULT_IMAGE_NAME}"
TAG="${3:-$DEFAULT_TAG}"
PLATFORMS="${PLATFORMS:-linux/amd64}"

FULL_IMAGE_NAME="${DOCKERHUB_USERNAME}/${IMAGE_NAME}:${TAG}"

cd "$(dirname "$0")"

echo "======================================"
echo "Docker Buildx Build and Push"
echo "======================================"
echo "Image: ${FULL_IMAGE_NAME}"
echo "Platforms: ${PLATFORMS}"
echo "Context: $(pwd)"
echo "======================================"

if [[ "${PLATFORMS}" != "linux/amd64" ]]; then
    echo "Error: offline-server image currently supports linux/amd64 only."
    echo "Reason: the bundled gg20 manager is gg20_sm_manager_linux_amd64."
    exit 1
fi

if ! docker info >/dev/null 2>&1; then
    echo "Error: Docker is not running. Start Docker and retry."
    exit 1
fi

if [[ ! -x "bin/gg20_sm_manager_linux_amd64" ]]; then
    echo "Error: bin/gg20_sm_manager_linux_amd64 is missing or not executable."
    exit 1
fi

BUILDER_NAME="crypto-custody-builder"
if ! docker buildx ls | grep -q "${BUILDER_NAME}"; then
    echo "Creating buildx builder: ${BUILDER_NAME}"
    docker buildx create --name "${BUILDER_NAME}" --use
else
    echo "Using existing buildx builder: ${BUILDER_NAME}"
    docker buildx use "${BUILDER_NAME}"
fi

echo "Building and pushing Docker image..."
docker buildx build \
    --platform "${PLATFORMS}" \
    -t "${FULL_IMAGE_NAME}" \
    --push \
    .

echo "======================================"
echo "Done: ${FULL_IMAGE_NAME}"
echo "======================================"
