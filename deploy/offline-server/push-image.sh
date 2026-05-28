#!/usr/bin/env bash

# Build and push the offline-server image from the repository source tree.
# Usage: ./push-image.sh [DOCKERHUB_USERNAME] [IMAGE_NAME] [TAG]

set -euo pipefail

cd "$(dirname "$0")"

REPO_ROOT="$(cd ../.. && pwd)"
exec "${REPO_ROOT}/offline-server/docker-build-push.sh" "$@"
