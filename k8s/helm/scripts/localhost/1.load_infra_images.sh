#!/bin/bash
# =============================================================================
# 인프라 이미지를 로컬 레지스트리에 로드 (localhost 환경용)
# =============================================================================
# localhost 환경:
# - PostgreSQL, Redis: 클러스터 내부 Pod로 실행
# - MinIO, LiveKit: 클러스터 내 Pod로 실행
# - 모니터링: Prometheus, Grafana, Loki, Promtail, Exporters

# set -e 제거 - 개별 이미지 실패해도 계속 진행

LOCAL_REG="localhost:5001"

echo "=== 인프라 이미지 → 로컬 레지스트리 (localhost 환경) ==="
echo ""
echo "ℹ️  localhost 환경 구성:"
echo "   - 데이터베이스: PostgreSQL 16, Redis 7"
echo "   - 스토리지/통신: MinIO, LiveKit"
echo "   - 모니터링: Prometheus, Grafana, Loki, Promtail"
echo "   - Exporters: PostgreSQL, Redis"
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

# Docker Hub에서 이미지 로드
load_from_dockerhub() {
    local src=$1 name=$2 tag=$3

    if image_exists "$name" "$tag"; then
        echo "✓ ${name}:${tag} - 이미 있음 (스킵)"
        return
    fi

    echo "📦 ${name}:${tag}"
    echo "   Docker Hub: $src"

    if docker pull --platform linux/amd64 "$src" 2>/dev/null; then
        docker tag "$src" "${LOCAL_REG}/${name}:${tag}"
        docker push "${LOCAL_REG}/${name}:${tag}"
        echo "   ✅ 로드 완료"
    else
        echo "   ❌ 이미지 로드 실패: ${name}:${tag}"
        return 1
    fi
}

# 데이터베이스 이미지
echo "--- 데이터베이스 이미지 ---"
load_from_dockerhub "postgres:16-alpine" "postgres" "16-alpine"
load_from_dockerhub "redis:7-alpine" "redis" "7-alpine"

# 스토리지 이미지
echo ""
echo "--- 스토리지 이미지 ---"
load_from_dockerhub "minio/minio:latest" "minio" "latest"

# 실시간 통신 이미지
echo ""
echo "--- 실시간 통신 이미지 ---"
load_from_dockerhub "livekit/livekit-server:latest" "livekit" "latest"

# =============================================================================
# 모니터링 이미지
# =============================================================================
echo ""
echo "--- 모니터링 이미지 ---"

# Prometheus
load_from_dockerhub "prom/prometheus:v2.48.0" "prometheus" "v2.48.0"

# Grafana
load_from_dockerhub "grafana/grafana:10.2.2" "grafana" "10.2.2"

# Loki
load_from_dockerhub "grafana/loki:2.9.2" "loki" "2.9.2"

# Promtail
load_from_dockerhub "grafana/promtail:2.9.2" "promtail" "2.9.2"

# PostgreSQL Exporter
load_from_dockerhub "prometheuscommunity/postgres-exporter:v0.15.0" "postgres-exporter" "v0.15.0"

# Redis Exporter
load_from_dockerhub "oliver006/redis_exporter:v1.55.0" "redis-exporter" "v1.55.0"

echo ""
echo "✅ 인프라 이미지 로드 완료!"
