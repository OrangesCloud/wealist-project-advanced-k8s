#!/bin/bash
# =============================================================================
# 인프라 이미지 로드 (dev 환경)
# =============================================================================
# dev 환경:
# - PostgreSQL/Redis: 호스트 PC 외부 DB 사용 (이미지 불필요)
# - MinIO: 클러스터 내 Pod로 실행 (이미지 필요)
# - Backend: GHCR에서 pull

set -e

CLUSTER_NAME="wealist"
GHCR_REGISTRY="ghcr.io/orangescloud"

# 아키텍처 감지
ARCH=$(uname -m)
case "$ARCH" in
    x86_64)  PLATFORM="linux/amd64" ;;
    aarch64) PLATFORM="linux/arm64" ;;
    arm64)   PLATFORM="linux/arm64" ;;
    *)       PLATFORM="linux/amd64" ;;
esac

echo "=== dev 환경 인프라 이미지 로드 ==="
echo ""
echo "📦 Registry: ${GHCR_REGISTRY}"
echo "🖥️  Architecture: ${ARCH} → Platform: ${PLATFORM}"
echo ""
echo "ℹ️  dev 환경 구성:"
echo "   - PostgreSQL: 호스트 PC (외부) - 이미지 불필요"
echo "   - Redis: 호스트 PC (외부) - 이미지 불필요"
echo "   - MinIO: 클러스터 내 Pod - 이미지 로드 필요"
echo "   - Backend: GHCR 이미지"
echo ""

# GHCR 인증 확인 (토큰 유효성만 체크, 이미지 존재 여부와 무관)
echo "🔐 GHCR 인증 확인 중..."
if docker login ghcr.io --get-login 2>/dev/null | grep -q .; then
    echo "✅ GHCR 로그인 상태: $(docker login ghcr.io --get-login 2>/dev/null)"
else
    echo "⚠️  GHCR 로그인 필요"
    echo ""
    echo "   GHCR 로그인:"
    echo "   echo \$GHCR_TOKEN | docker login ghcr.io -u \$GHCR_USERNAME --password-stdin"
fi

echo ""
echo "--- 인프라 이미지 로드 (Kind 클러스터) ---"

# Kind 클러스터에 이미지 로드하는 함수
# Docker Desktop containerd 호환성을 위해 tar 파일로 저장 후 로드
load_to_kind() {
    local image=$1
    local tar_file="/tmp/kind-image-$(echo "$image" | tr '/:' '-').tar"
    echo "  📦 ${image}"

    # 기존 이미지 삭제 (containerd 캐시 문제 방지)
    docker rmi "$image" 2>/dev/null || true

    # 플랫폼 명시하여 pull
    echo "     Pulling with platform: ${PLATFORM}"
    docker pull --platform "${PLATFORM}" "$image"

    # tar 파일로 저장 후 Kind에 로드 (containerd 우회)
    echo "     Saving to tar..."
    docker save "$image" -o "$tar_file"

    echo "     Loading to Kind cluster..."
    kind load image-archive "$tar_file" --name "$CLUSTER_NAME"

    # 임시 파일 삭제
    rm -f "$tar_file"
    echo "     ✅ 로드 완료"
}

# MinIO - S3 호환 스토리지
echo ""
echo "🗄️  MinIO 이미지 로드 중..."
load_to_kind "minio/minio:latest"

# LiveKit - 실시간 통신 (필요시)
echo ""
echo "📹 LiveKit 이미지 로드 중..."
load_to_kind "livekit/livekit-server:v1.5"

echo ""
echo "✅ 인프라 이미지 로드 완료!"
echo ""
echo "📝 다음 단계:"
echo "   1. 서비스 이미지 확인/푸시:"
echo "      make ghcr-push-all ENV=dev"
echo ""
echo "   2. Helm 배포:"
echo "      make helm-install-all ENV=dev"
