#!/usr/bin/env bash

# Compatibility wrapper: build the all-in-one image and push it.
# Usage:
#   ./docker-build-push.sh [IMAGE]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
DEFAULT_IMAGE="ceyewan/crypto-custody-online-system:latest"
IMAGE="${1:-${ONLINE_SYSTEM_IMAGE:-$DEFAULT_IMAGE}}"

"${SCRIPT_DIR}/docker-build-image.sh" "${IMAGE}"
"${SCRIPT_DIR}/docker-push-image.sh" "${IMAGE}"
