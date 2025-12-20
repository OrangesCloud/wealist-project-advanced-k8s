#!/bin/bash
# =============================================================================
# 인프라 이미지를 로컬 레지스트리에 로드 (localhost 환경용)
# - PostgreSQL, Redis 포함 (클러스터 내부 Pod로 실행)
# =============================================================================

set -e

LOCAL_REG="localhost:5001"

echo "=== 인프라 이미지 → 로컬 레지스트리 (localhost 환경) ==="
echo ""
echo "※ 모든 인프라 이미지를 로드합니다 (DB 포함)"
echo ""

# 레지스트리 확인
if ! curl -s "http://${LOCAL_REG}/v2/" > /dev/null 2>&1; then
    echo "ERROR: 레지스트리 없음. make kind-setup 먼저 실행"
    exit 1
fi

# 로컬 레지스트리에 이미지 있는지 확인
image_exists() {
    local name=$1 tag=$2
    curl -sf "http://${LOCAL_REG}/v2/${name}/manifests/${tag}" > /dev/null 2>&1
}

load() {
    local src=$1 name=$2 tag=$3

    if image_exists "$name" "$tag"; then
        echo "✓ ${name}:${tag} - 이미 있음 (스킵)"
        return
    fi

    echo "$src → ${LOCAL_REG}/${name}:${tag}"
    docker pull --platform linux/amd64 "$src"
    docker tag "$src" "${LOCAL_REG}/${name}:${tag}"
    docker push "${LOCAL_REG}/${name}:${tag}"
}

# AWS ECR Public (무료) - DB 이미지
echo "--- 데이터베이스 이미지 ---"
load "public.ecr.aws/docker/library/postgres:15-alpine" "postgres" "15-alpine"
load "public.ecr.aws/docker/library/redis:7-alpine" "redis" "7-alpine"

# Docker Hub - LiveKit (실시간 통신)
echo ""
echo "--- 실시간 통신 이미지 ---"
load "livekit/livekit-server:v1.5" "livekit" "v1.5"

echo ""
echo "✅ 인프라 이미지 로드 완료!"
