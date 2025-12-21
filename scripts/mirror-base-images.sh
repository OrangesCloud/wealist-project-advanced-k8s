#!/bin/bash
# =============================================================================
# Base 이미지 GHCR 미러링 스크립트
# =============================================================================
# Docker Hub rate limit을 피하기 위해 자주 사용하는 base 이미지를
# GHCR (ghcr.io/orangescloud/base/)에 미러링합니다.
#
# 사용법: ./scripts/mirror-base-images.sh
# =============================================================================

set -e

GHCR_REGISTRY="ghcr.io/orangescloud/base"

# 미러링할 base 이미지 목록
# 빌드용 base 이미지
BASE_IMAGES=(
    "golang:1.24-bookworm"
    "alpine:latest"
    "gradle:8-jdk21"
    "eclipse-temurin:21-jre-jammy"
    "node:20-alpine"
    "nginx:stable-alpine"
)

# 인프라 이미지 (DB, 캐시, 스토리지, 통신)
INFRA_IMAGES=(
    "postgres:16-alpine"
    "redis:7-alpine"
    "minio/minio:latest"
    "livekit/livekit-server:latest"
)

# 모니터링 이미지
MONITORING_IMAGES=(
    "prom/prometheus:v2.48.0"
    "grafana/grafana:10.2.2"
    "grafana/loki:2.9.2"
    "grafana/promtail:2.9.2"
    "prometheuscommunity/postgres-exporter:v0.15.0"
    "oliver006/redis_exporter:v1.55.0"
)

# Istio Addon 이미지
ISTIO_ADDON_IMAGES=(
    "jaegertracing/all-in-one:1.52"
    "quay.io/kiali/kiali:v1.77"
)

# 전체 이미지 목록 (BASE + INFRA + MONITORING + ISTIO)
ALL_IMAGES=("${BASE_IMAGES[@]}" "${INFRA_IMAGES[@]}" "${MONITORING_IMAGES[@]}" "${ISTIO_ADDON_IMAGES[@]}")

echo "=============================================="
echo "  전체 이미지 GHCR 미러링"
echo "=============================================="
echo ""
echo "대상 레지스트리: ${GHCR_REGISTRY}"
echo "미러링할 이미지: ${#ALL_IMAGES[@]}개"
echo "  - 빌드용 base: ${#BASE_IMAGES[@]}개"
echo "  - 인프라: ${#INFRA_IMAGES[@]}개"
echo "  - 모니터링: ${#MONITORING_IMAGES[@]}개"
echo "  - Istio Addon: ${#ISTIO_ADDON_IMAGES[@]}개"
echo ""

# GHCR 로그인 확인 (docker config 파일에서 ghcr.io 확인)
echo "🔐 GHCR 로그인 확인 중..."
if ! grep -q "ghcr.io" ~/.docker/config.json 2>/dev/null; then
    echo "❌ GHCR 로그인이 필요합니다."
    echo ""
    echo "로그인 방법:"
    echo "  docker login ghcr.io -u YOUR_USERNAME"
    echo "  Password에 GitHub PAT (ghp_xxx) 입력"
    exit 1
fi
echo "✅ GHCR 로그인됨"
echo ""

# Docker Hub 로그인 확인 (rate limit 완화)
echo "🔐 Docker Hub 로그인 확인 중..."
if docker login --get-login 2>/dev/null | grep -q .; then
    echo "✅ Docker Hub 로그인됨 (rate limit 완화)"
else
    echo "⚠️  Docker Hub 미로그인 - rate limit 주의"
    echo "   권장: docker login -u YOUR_DOCKERHUB_USERNAME"
fi
echo ""

# buildx 빌더 확인 (멀티 아키텍처용)
echo "🔧 buildx 빌더 확인 중..."
if ! docker buildx inspect multiarch-builder >/dev/null 2>&1; then
    echo "   buildx 빌더 생성 중..."
    docker buildx create --name multiarch-builder --use --bootstrap
else
    docker buildx use multiarch-builder
fi
echo "✅ buildx 빌더 준비됨"
echo ""

echo "----------------------------------------------"
echo "  이미지 미러링 시작 (amd64 + arm64)"
echo "----------------------------------------------"
echo ""

for image in "${ALL_IMAGES[@]}"; do
    echo "📦 ${image}"

    # 이미지 이름 변환
    # 예: golang:1.24-bookworm → golang-1.24-bookworm
    # 예: minio/minio:latest → minio-latest
    # 예: livekit/livekit-server:latest → livekit-server-latest
    target_name=$(echo "$image" | sed 's|.*/||' | tr ':' '-')
    target_image="${GHCR_REGISTRY}/${target_name}"

    echo "   → ${target_image}"

    # 멀티 아키텍처로 빌드 및 푸시
    # --provenance=false: attestation 비활성화 (단순 이미지만)
    docker buildx build --platform linux/amd64,linux/arm64 \
        --build-arg BASE_IMAGE="$image" \
        -t "$target_image" \
        --provenance=false \
        --push \
        -f - . << 'DOCKERFILE'
ARG BASE_IMAGE
FROM ${BASE_IMAGE}
DOCKERFILE

    echo "   ✅ 미러링 완료"
    echo ""
done

echo "=============================================="
echo "  🎉 미러링 완료!"
echo "=============================================="
echo ""
echo "📦 빌드용 Base 이미지:"
for image in "${BASE_IMAGES[@]}"; do
    target_name=$(echo "$image" | sed 's|.*/||' | tr ':' '-')
    echo "  - ${GHCR_REGISTRY}/${target_name}"
done
echo ""
echo "🔧 인프라 이미지:"
for image in "${INFRA_IMAGES[@]}"; do
    target_name=$(echo "$image" | sed 's|.*/||' | tr ':' '-')
    echo "  - ${GHCR_REGISTRY}/${target_name}"
done
echo ""
echo "📊 모니터링 이미지:"
for image in "${MONITORING_IMAGES[@]}"; do
    target_name=$(echo "$image" | sed 's|.*/||' | tr ':' '-')
    echo "  - ${GHCR_REGISTRY}/${target_name}"
done
echo ""
echo "🔷 Istio Addon 이미지:"
for image in "${ISTIO_ADDON_IMAGES[@]}"; do
    target_name=$(echo "$image" | sed 's|.*/||' | tr ':' '-')
    echo "  - ${GHCR_REGISTRY}/${target_name}"
done
echo ""
echo "📝 사용 예시:"
echo ""
echo "  # Dockerfile"
echo "  FROM ghcr.io/orangescloud/base/golang-1.24-bookworm AS builder"
echo "  FROM ghcr.io/orangescloud/base/alpine-latest"
echo ""
echo "  # Helm values.yaml (인프라)"
echo "  postgres:"
echo "    image: ghcr.io/orangescloud/base/postgres-16-alpine"
echo "  redis:"
echo "    image: ghcr.io/orangescloud/base/redis-7-alpine"
echo "  minio:"
echo "    image: ghcr.io/orangescloud/base/minio-latest"
echo "  livekit:"
echo "    image: ghcr.io/orangescloud/base/livekit-server-latest"
echo ""
