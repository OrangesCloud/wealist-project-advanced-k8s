#!/bin/bash
# =============================================================================
# 모든 서비스 이미지 빌드 및 로드 (localhost 환경용)
# - Backend 서비스 + Frontend 포함
# =============================================================================

set -e

LOCAL_REG="localhost:5001"
TAG="${IMAGE_TAG:-latest}"

echo "=== 서비스 이미지 빌드 및 로드 (localhost 환경) ==="
echo ""
echo "레지스트리: ${LOCAL_REG}"
echo "태그: ${TAG}"
echo ""

# 레지스트리 확인
if ! curl -s "http://${LOCAL_REG}/v2/" > /dev/null 2>&1; then
    echo "ERROR: 레지스트리 없음. make kind-setup 먼저 실행"
    exit 1
fi

# 프로젝트 루트로 이동 (스크립트는 k8s/helm/scripts/localhost/ 에 위치)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"
cd "$PROJECT_ROOT"
echo "Working directory: $PROJECT_ROOT"
echo ""

# =============================================================================
# Backend 서비스 빌드
# =============================================================================
echo "=========================================="
echo "  Backend 서비스 빌드"
echo "=========================================="

BACKEND_SERVICES=(
    "auth-service"
    "user-service"
    "board-service"
    "chat-service"
    "noti-service"
    "storage-service"
    "video-service"
)

for service in "${BACKEND_SERVICES[@]}"; do
    echo ""
    echo "--- ${service} 빌드 중 ---"

    SERVICE_PATH="services/${service}"
    if [ ! -d "$SERVICE_PATH" ]; then
        echo "⚠️  ${SERVICE_PATH} 없음 - 스킵"
        continue
    fi

    # Dockerfile 확인 (루트 또는 docker/ 하위)
    if [ -f "${SERVICE_PATH}/Dockerfile" ]; then
        docker build -t "${LOCAL_REG}/${service}:${TAG}" "${SERVICE_PATH}"
        docker push "${LOCAL_REG}/${service}:${TAG}"
        echo "✅ ${service} 푸시 완료"
    elif [ -f "${SERVICE_PATH}/docker/Dockerfile" ]; then
        docker build -t "${LOCAL_REG}/${service}:${TAG}" -f "${SERVICE_PATH}/docker/Dockerfile" "${SERVICE_PATH}"
        docker push "${LOCAL_REG}/${service}:${TAG}"
        echo "✅ ${service} 푸시 완료"
    else
        echo "⚠️  ${SERVICE_PATH}/Dockerfile 없음 - 스킵"
    fi
done

# =============================================================================
# Frontend 빌드
# =============================================================================
echo ""
echo "=========================================="
echo "  Frontend 빌드"
echo "=========================================="

FRONTEND_PATH="services/frontend"
if [ -d "$FRONTEND_PATH" ] && [ -f "${FRONTEND_PATH}/Dockerfile" ]; then
    echo ""
    echo "--- frontend 빌드 중 ---"
    docker build -t "${LOCAL_REG}/frontend:${TAG}" "${FRONTEND_PATH}"
    docker push "${LOCAL_REG}/frontend:${TAG}"
    echo "✅ frontend 푸시 완료"
else
    echo "⚠️  ${FRONTEND_PATH}/Dockerfile 없음 - 스킵"
fi

echo ""
echo "=========================================="
echo "  🎉 모든 서비스 이미지 빌드 완료!"
echo "=========================================="
echo ""
echo "다음 단계:"
echo "  make helm-install-all ENV=localhost"
