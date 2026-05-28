#!/bin/bash

# Docker build and push script for crypto-custody frontend.
# Usage: ./docker-build-push.sh [DOCKERHUB_USERNAME] [IMAGE_NAME] [TAG] [API_BASE_URL]
# Optional env: PLATFORMS="linux/amd64,linux/arm64"

set -e

DEFAULT_USERNAME="ceyewan"
DEFAULT_IMAGE_NAME="crypto-custody-frontend"
DEFAULT_TAG="latest"
DEFAULT_API_BASE_URL="http://192.168.192.1:22221"

DOCKERHUB_USERNAME=${1:-$DEFAULT_USERNAME}
IMAGE_NAME=${2:-$DEFAULT_IMAGE_NAME}
TAG=${3:-$DEFAULT_TAG}
API_BASE_URL=${4:-$DEFAULT_API_BASE_URL}
PLATFORMS=${PLATFORMS:-linux/amd64}

FULL_IMAGE_NAME="${DOCKERHUB_USERNAME}/${IMAGE_NAME}:${TAG}"

echo "======================================"
echo "Docker Buildx Build and Push"
echo "======================================"
echo "Image: ${FULL_IMAGE_NAME}"
echo "Platforms: ${PLATFORMS}"
echo "API base URL: ${API_BASE_URL}"
echo "======================================"

if ! docker info > /dev/null 2>&1; then
    echo "❌ Error: Docker is not running. Please start Docker and try again."
    exit 1
fi

BUILDER_NAME="crypto-custody-builder"
if ! docker buildx ls | grep -q "${BUILDER_NAME}"; then
    echo "🔧 Creating buildx builder: ${BUILDER_NAME}..."
    docker buildx create --name "${BUILDER_NAME}" --use
else
    echo "🔧 Using existing buildx builder: ${BUILDER_NAME}..."
    docker buildx use "${BUILDER_NAME}"
fi

echo "🔨 Building and pushing Docker image..."
docker buildx build \
    --platform "${PLATFORMS}" \
    --build-arg "VUE_APP_API_BASE_URL=${API_BASE_URL}" \
    -t "${FULL_IMAGE_NAME}" \
    --push .

echo "======================================"
echo "✅ Build and push completed successfully: ${FULL_IMAGE_NAME}"
echo "======================================"
