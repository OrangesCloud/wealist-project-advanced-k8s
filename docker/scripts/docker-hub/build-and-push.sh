#!/bin/bash
# =============================================================================
# Docker Hub Build and Push Script
# =============================================================================
# 사용법:
#   DOCKER_HUB_ID=myid ./docker/scripts/docker-hub/build-and-push.sh
#   DOCKER_HUB_ID=myid IMAGE_TAG=v2 ./docker/scripts/docker-hub/build-and-push.sh
#
# 환경변수:
#   DOCKER_HUB_ID  - Docker Hub 사용자/조직 ID (필수)
#   IMAGE_TAG      - 이미지 태그 (기본값: latest)
#   SERVICES       - 빌드할 서비스 목록 (기본값: 모든 서비스)
# =============================================================================

set -e

# Docker Hub ID 확인
if [ -z "$DOCKER_HUB_ID" ]; then
    echo "Error: DOCKER_HUB_ID environment variable is required"
    echo ""
    echo "Usage:"
    echo "  DOCKER_HUB_ID=your-docker-id ./docker/scripts/docker-hub/build-and-push.sh"
    echo ""
    echo "Example:"
    echo "  DOCKER_HUB_ID=your-docker-id ./docker/scripts/docker-hub/build-and-push.sh"
    exit 1
fi

# 설정
TAG="${IMAGE_TAG:-latest}"
REPO_ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"

# 서비스 목록 (순서대로 빌드)
ALL_SERVICES="auth-service user-service board-service chat-service noti-service storage-service video-service frontend"
SERVICES="${SERVICES:-$ALL_SERVICES}"

echo "=============================================="
echo "Docker Hub Build & Push"
echo "=============================================="
echo "Docker Hub ID: $DOCKER_HUB_ID"
echo "Image Tag:     $TAG"
echo "Services:      $SERVICES"
echo "=============================================="
echo ""

# Docker Hub 로그인 확인
if ! docker info 2>/dev/null | grep -q "Username"; then
    echo "Warning: Not logged in to Docker Hub"
    echo "Run 'docker login' first if push fails"
    echo ""
fi

cd "$REPO_ROOT"

# 서비스별 Dockerfile 경로 매핑
get_dockerfile() {
    local service=$1
    case $service in
        auth-service)
            echo "services/auth-service/Dockerfile"
            ;;
        frontend)
            echo "services/frontend/Dockerfile"
            ;;
        *)
            echo "services/$service/docker/Dockerfile"
            ;;
    esac
}

# 빌드 및 푸시
count=1
total=$(echo $SERVICES | wc -w)

for service in $SERVICES; do
    dockerfile=$(get_dockerfile $service)
    image_name="${DOCKER_HUB_ID}/wealist-${service}:${TAG}"

    echo "[$count/$total] Building $service..."
    echo "  Dockerfile: $dockerfile"
    echo "  Image: $image_name"

    if [ ! -f "$dockerfile" ]; then
        echo "  Error: Dockerfile not found at $dockerfile"
        exit 1
    fi

    docker build -t "$image_name" -f "$dockerfile" "services/$service"

    echo "  Pushing $image_name..."
    docker push "$image_name"

    echo "  Done!"
    echo ""

    count=$((count + 1))
done

echo "=============================================="
echo "All images pushed successfully!"
echo ""
echo "Images:"
for service in $SERVICES; do
    echo "  - ${DOCKER_HUB_ID}/wealist-${service}:${TAG}"
done
echo ""
echo "Next: Deploy to Kubernetes"
echo "  DOCKER_HUB_ID=$DOCKER_HUB_ID make k8s-apply-dockerhub"
echo "=============================================="
