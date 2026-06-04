#!/usr/bin/env bash

# Push an online-system Docker image to DockerHub or another registry.
# Usage:
#   ./docker-push-image.sh [IMAGE]
#
# Examples:
#   ./docker-push-image.sh ceyewan/crypto-custody-online-system:latest

set -euo pipefail

DEFAULT_IMAGE="ceyewan/crypto-custody-online-system:latest"
IMAGE="${1:-${ONLINE_SYSTEM_IMAGE:-$DEFAULT_IMAGE}}"

echo "======================================"
echo "Push online-system Docker image"
echo "======================================"
echo "Image: ${IMAGE}"
echo "======================================"

if ! docker info >/dev/null 2>&1; then
    echo "Error: Docker is not running. Start Docker and retry."
    exit 1
fi

if ! docker image inspect "${IMAGE}" >/dev/null 2>&1; then
    echo "Error: local image not found: ${IMAGE}"
    echo "Build or tag it first."
    exit 1
fi

docker push "${IMAGE}"

echo "======================================"
echo "Pushed: ${IMAGE}"
echo "======================================"
