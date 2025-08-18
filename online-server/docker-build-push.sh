#!/bin/bash

# crypto-custody online-server 的 Docker 构建和推送脚本
# 该脚本构建一个多平台 Docker 镜像并将其推送到镜像仓库。
# 用法: ./docker-build-push.sh [DOCKERHUB_USERNAME] [IMAGE_NAME] [TAG]

set -e

# --- 配置 ---
# Docker 镜像详情的默认值。
# 这些值可以通过命令行参数覆盖。
DEFAULT_USERNAME="ceyewan"
DEFAULT_IMAGE_NAME="crypto-custody-online-server"
DEFAULT_TAG="amd64"
PLATFORM="linux/amd64"

FULL_IMAGE_NAME="${DEFAULT_USERNAME}/${DEFAULT_IMAGE_NAME}:${DEFAULT_TAG}"

# --- 脚本标题 ---
echo "======================================"
echo "Docker 构建与推送脚本"
echo "======================================"
echo "镜像: ${FULL_IMAGE_NAME}"
echo "平台: ${PLATFORM}"
echo "======================================"

# --- Docker 环境检查 ---
# 在继续之前，确保 Docker 正在运行。
if ! docker info > /dev/null 2>&1; then
    echo "❌ 错误: Docker 未运行。请启动 Docker 后重试。"
    exit 1
fi

# --- Docker Buildx 设置 ---
# 为多平台构建设置并使用一个构建器。
BUILDER_NAME="mybuilder"
if ! docker buildx ls | grep -q $BUILDER_NAME; then
    echo "🔧 正在创建新的 buildx 构建器: $BUILDER_NAME..."
    docker buildx create --name $BUILDER_NAME --use
else
    echo "🔧 正在使用已有的 buildx 构建器: $BUILDER_NAME..."
    docker buildx use $BUILDER_NAME
fi

# --- Docker 构建 ---
# 为指定平台构建 Docker 镜像。
# --platform 标志对于交叉编译至关重要。
# --load 标志将构建好的镜像加载到本地 Docker 守护进程中。
echo "🔨 正在为 ${PLATFORM} 构建 Docker 镜像..."
docker buildx build --platform "${PLATFORM}" -t "${FULL_IMAGE_NAME}" --load .

if [ $? -eq 0 ]; then
    echo "✅ Docker 镜像构建成功: ${FULL_IMAGE_NAME}"
else
    echo "❌ Docker 镜像构建失败"
    exit 1
fi

# --- Docker 登录 ---
# 登录到 Docker Hub 以推送镜像。
# 为了安全，建议使用个人访问令牌 (PAT)。
echo "🔐 正在登录到 DockerHub..."
echo "请输入您的 DockerHub 用户名 (或按回车使用 '${DOCKERHUB_USERNAME}'):"
read -r input_username
DOCKERHUB_USERNAME=${input_username:-$DOCKERHUB_USERNAME}

echo "请输入您的 DockerHub 密码或访问令牌:"
read -s DOCKERHUB_PASSWORD
echo

if ! echo "$DOCKERHUB_PASSWORD" | docker login -u "$DOCKERHUB_USERNAME" --password-stdin; then
    echo "❌ 登录 DockerHub 失败"
    exit 1
fi

# --- Docker 推送 ---
# 将构建好的镜像推送到 Docker Hub 仓库。
echo "📤 正在将镜像推送到 DockerHub..."
docker push "${FULL_IMAGE_NAME}"

if [ $? -eq 0 ]; then
    echo "✅ 成功将 ${FULL_IMAGE_NAME} 推送到 DockerHub！"
    echo ""
    echo "🚀 您现在可以使用以下命令运行您的容器:"
    echo "   docker run -p 8080:8080 --env-file ./.env ${FULL_IMAGE_NAME}"
    echo ""
    echo "🌐 或者从任何地方拉取镜像:"
    echo "   docker pull ${FULL_IMAGE_NAME}"
else
    echo "❌ 推送镜像到 DockerHub 失败"
    exit 1
fi

# --- 脚本页脚 ---
echo "======================================"
echo "✅ 构建和推送已成功完成！"
echo "======================================"
