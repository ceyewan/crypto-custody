#!/usr/bin/env bash

# Compatibility wrapper for the old combined build-and-push command.
# Prefer the split commands:
#   ./docker-build-image.sh [LOCAL_IMAGE]
#   ./docker-push-image.sh [LOCAL_IMAGE] [REMOTE_IMAGE]
#
# Usage:
#   ./docker-build-push.sh [DOCKERHUB_USERNAME] [IMAGE_NAME] [TAG]

set -euo pipefail

DEFAULT_USERNAME="ceyewan"
DEFAULT_IMAGE_NAME="crypto-custody-offline-server"
DEFAULT_TAG="latest"

DOCKERHUB_USERNAME="${1:-$DEFAULT_USERNAME}"
IMAGE_NAME="${2:-$DEFAULT_IMAGE_NAME}"
TAG="${3:-$DEFAULT_TAG}"

cd "$(dirname "$0")"

REMOTE_IMAGE="${DOCKERHUB_USERNAME}/${IMAGE_NAME}:${TAG}"
LOCAL_IMAGE="${OFFLINE_SERVER_LOCAL_IMAGE:-${REMOTE_IMAGE}}"

"$(pwd)/docker-build-image.sh" "${LOCAL_IMAGE}"
"$(pwd)/docker-push-image.sh" "${LOCAL_IMAGE}" "${REMOTE_IMAGE}"
