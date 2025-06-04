#!/bin/bash

# Docker build and push script for crypto-custody online-server
# Usage: ./docker-build-push.sh [DOCKERHUB_USERNAME] [IMAGE_NAME] [TAG]

set -e

# Default values
DEFAULT_USERNAME="ceyewan"
DEFAULT_IMAGE_NAME="crypto-custody-online-server"
DEFAULT_TAG="latest"

# Parse command line arguments
DOCKERHUB_USERNAME=${1:-$DEFAULT_USERNAME}
IMAGE_NAME=${2:-$DEFAULT_IMAGE_NAME}
TAG=${3:-$DEFAULT_TAG}

# Full image name
FULL_IMAGE_NAME="${DOCKERHUB_USERNAME}/${IMAGE_NAME}:${TAG}"

echo "======================================"
echo "Docker Build and Push Script"
echo "======================================"
echo "Image: ${FULL_IMAGE_NAME}"
echo "======================================"

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "❌ Error: Docker is not running. Please start Docker and try again."
    exit 1
fi

# Build the Docker image
echo "🔨 Building Docker image..."
docker build -t ${FULL_IMAGE_NAME} .

if [ $? -eq 0 ]; then
    echo "✅ Docker image built successfully: ${FULL_IMAGE_NAME}"
else
    echo "❌ Failed to build Docker image"
    exit 1
fi

# Test the image locally (optional)
echo "🧪 Testing the image locally..."
CONTAINER_ID=$(docker run -d -p 8080:8080 ${FULL_IMAGE_NAME})
sleep 5

# Check if container is running
if docker ps | grep -q ${CONTAINER_ID}; then
    echo "✅ Container is running successfully"
    docker stop ${CONTAINER_ID}
    docker rm ${CONTAINER_ID}
else
    echo "⚠️  Warning: Container test failed, but continuing with push..."
fi

# Login to DockerHub
echo "🔐 Logging in to DockerHub..."
echo "Please enter your DockerHub credentials:"
docker login

if [ $? -ne 0 ]; then
    echo "❌ Failed to login to DockerHub"
    exit 1
fi

# Push the image
echo "📤 Pushing image to DockerHub..."
docker push ${FULL_IMAGE_NAME}

if [ $? -eq 0 ]; then
    echo "✅ Successfully pushed ${FULL_IMAGE_NAME} to DockerHub!"
    echo ""
    echo "🚀 You can now run your container with:"
    echo "   docker run -p 8080:8080 ${FULL_IMAGE_NAME}"
    echo ""
    echo "🌐 Or pull it from anywhere with:"
    echo "   docker pull ${FULL_IMAGE_NAME}"
else
    echo "❌ Failed to push image to DockerHub"
    exit 1
fi

echo "======================================"
echo "✅ Build and push completed successfully!"
echo "======================================"
