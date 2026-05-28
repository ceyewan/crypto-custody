#!/bin/bash

# crypto-custody online-server 的 Docker 构建和推送脚本。
# 用法: ./docker-build-push.sh [DOCKERHUB_USERNAME] [IMAGE_NAME] [TAG]
# 可选环境变量: PLATFORMS="linux/amd64,linux/arm64"

set -e

DEFAULT_USERNAME="ceyewan"
DEFAULT_IMAGE_NAME="crypto-custody-online-server"
DEFAULT_TAG="latest"

DOCKERHUB_USERNAME=${1:-$DEFAULT_USERNAME}
IMAGE_NAME=${2:-$DEFAULT_IMAGE_NAME}
TAG=${3:-$DEFAULT_TAG}
PLATFORMS=${PLATFORMS:-linux/amd64}

FULL_IMAGE_NAME="${DOCKERHUB_USERNAME}/${IMAGE_NAME}:${TAG}"

echo "======================================"
echo "Docker Buildx Build and Push"
echo "======================================"
echo "镜像: ${FULL_IMAGE_NAME}"
echo "平台: ${PLATFORMS}"
echo "======================================"

if ! docker info > /dev/null 2>&1; then
    echo "❌ 错误: Docker 未运行。请启动 Docker 后重试。"
    exit 1
fi

BUILDER_NAME="crypto-custody-builder"
if ! docker buildx ls | grep -q "${BUILDER_NAME}"; then
    echo "🔧 正在创建 buildx 构建器: ${BUILDER_NAME}..."
    docker buildx create --name $BUILDER_NAME --use
else
    echo "🔧 正在使用已有的 buildx 构建器: ${BUILDER_NAME}..."
    docker buildx use $BUILDER_NAME
fi

echo "🔨 正在构建并推送 Docker 镜像..."
docker buildx build --platform "${PLATFORMS}" -t "${FULL_IMAGE_NAME}" --push .

echo "======================================"
echo "✅ 构建并推送完成: ${FULL_IMAGE_NAME}"
echo "======================================"
