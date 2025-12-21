#!/bin/bash
# 인프라 이미지를 로컬 레지스트리에 푸시
# 이미 로컬 레지스트리에 있으면 스킵

set -e

LOCAL_REG="localhost:5001"

echo "=== 인프라 이미지 → 로컬 레지스트리 ==="

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

# AWS ECR Public (무료)
load "public.ecr.aws/docker/library/postgres:15-alpine" "postgres" "15-alpine"
load "public.ecr.aws/docker/library/redis:7-alpine" "redis" "7-alpine"

# Docker Hub
load "coturn/coturn:4.6" "coturn" "4.6"
load "livekit/livekit-server:v1.5" "livekit" "v1.5"

echo "완료!"
