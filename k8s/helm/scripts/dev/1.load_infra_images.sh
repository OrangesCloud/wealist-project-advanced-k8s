#!/bin/bash
# =============================================================================
# 인프라 이미지 로드 (dev 환경)
# =============================================================================
# dev 환경:
# - PostgreSQL/Redis: 호스트 PC 외부 DB 사용 (이미지 불필요)
# - MinIO, LiveKit: 클러스터 내 Pod로 실행
# - 모니터링: Prometheus, Grafana, Loki, Promtail, Exporters
# - Backend: AWS ECR에서 pull (CI/CD로 자동 빌드)
#
# 환경변수:
#   SKIP_INFRA=true      - 인프라 이미지(MinIO, LiveKit) 건너뛰기
#   SKIP_MONITORING=true - 모니터링 이미지 건너뛰기
#   ONLY_INFRA=true      - 인프라 이미지만 로드
#   ONLY_MONITORING=true - 모니터링 이미지만 로드

# set -e 제거 - 개별 이미지 실패해도 계속 진행

CLUSTER_NAME="wealist"

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
echo "📦 Registry: Docker Hub (인프라 이미지)"
echo "🖥️  Architecture: ${ARCH} → Platform: ${PLATFORM}"
echo ""

# =============================================================================
# Docker Storage Driver 확인 (WSL native Docker 호환성)
# =============================================================================
# containerd storage driver 사용 시 kind load image-archive 실패할 수 있음
# overlay2로 변경하여 해결

check_docker_storage_driver() {
    echo "🔍 Docker Storage Driver 확인 중..."

    STORAGE_DRIVER=$(docker info 2>/dev/null | grep "Storage Driver" | awk '{print $3}')

    if [ -z "$STORAGE_DRIVER" ]; then
        echo "⚠️  Docker 정보를 가져올 수 없습니다."
        return 0
    fi

    echo "   현재 Storage Driver: $STORAGE_DRIVER"

    # stargz 또는 containerd 기반 드라이버 감지
    if echo "$STORAGE_DRIVER" | grep -qi "stargz\|containerd"; then
        echo ""
        echo "⚠️  $STORAGE_DRIVER 드라이버가 감지되었습니다."
        echo "   이 드라이버는 'kind load image-archive'와 호환되지 않을 수 있습니다."
        echo ""
        echo "overlay2 드라이버로 변경하시겠습니까? [Y/n]"
        read -r answer
        if [ "$answer" != "n" ] && [ "$answer" != "N" ]; then
            echo ""
            echo "🔧 Docker Storage Driver를 overlay2로 변경 중..."

            # 기존 daemon.json 백업 및 수정
            DAEMON_JSON="/etc/docker/daemon.json"
            if [ -f "$DAEMON_JSON" ]; then
                sudo cp "$DAEMON_JSON" "${DAEMON_JSON}.backup"
                echo "   📄 기존 daemon.json 백업됨: ${DAEMON_JSON}.backup"
            fi

            # overlay2 설정 적용
            if [ -f "$DAEMON_JSON" ] && grep -q "storage-driver" "$DAEMON_JSON"; then
                # 기존 storage-driver 설정 변경
                sudo sed -i 's/"storage-driver"[[:space:]]*:[[:space:]]*"[^"]*"/"storage-driver": "overlay2"/' "$DAEMON_JSON"
            else
                # daemon.json 생성 또는 추가
                if [ -f "$DAEMON_JSON" ]; then
                    # 기존 파일에 storage-driver 추가 (마지막 } 앞에)
                    sudo sed -i 's/}$/,\n  "storage-driver": "overlay2"\n}/' "$DAEMON_JSON"
                else
                    # 새 파일 생성
                    echo '{
  "storage-driver": "overlay2"
}' | sudo tee "$DAEMON_JSON" > /dev/null
                fi
            fi

            echo "   ✅ daemon.json 수정 완료"
            echo ""
            echo "🔄 Docker 재시작 중..."
            sudo systemctl restart docker
            sleep 5

            # 재시작 후 확인
            NEW_DRIVER=$(docker info 2>/dev/null | grep "Storage Driver" | awk '{print $3}')
            echo "   새 Storage Driver: $NEW_DRIVER"

            if [ "$NEW_DRIVER" = "overlay2" ]; then
                echo "   ✅ overlay2로 변경 완료!"
            else
                echo "   ⚠️  변경이 적용되지 않았습니다."
                echo "      수동으로 /etc/docker/daemon.json을 확인하세요."
            fi
        else
            echo ""
            echo "⚠️  드라이버 변경을 건너뜁니다."
            echo "   이미지 로드 시 오류가 발생할 수 있습니다."
        fi
    else
        echo "   ✅ $STORAGE_DRIVER - Kind와 호환됨"
    fi
    echo ""
}

# Storage Driver 확인 (WSL 환경에서만)
if grep -qi microsoft /proc/version 2>/dev/null; then
    check_docker_storage_driver
fi

echo "ℹ️  dev 환경 구성:"
echo "   - PostgreSQL: 호스트 PC (외부) - 이미지 불필요"
echo "   - Redis: 호스트 PC (외부) - 이미지 불필요"
echo "   - MinIO, LiveKit: 클러스터 내 Pod (Docker Hub)"
echo "   - 모니터링: Prometheus, Grafana, Loki, Promtail (Docker Hub)"
echo "   - Exporters: PostgreSQL, Redis (Docker Hub)"
echo "   - Backend: AWS ECR 이미지 (CI/CD 자동 빌드)"
echo ""
echo "--- 인프라 이미지 로드 (Kind 클러스터) ---"

# Kind 클러스터에 이미지 로드하는 함수
# 방법 1: kind load docker-image (빠름, 일부 환경에서 동작 안함)
# 방법 2: kind load image-archive (tar 저장 후 로드)
# 방법 3: 노드에 직접 ctr import (fallback)
load_to_kind() {
    local image=$1
    local tar_file="/tmp/kind-image-$(echo "$image" | tr '/:' '-').tar"
    echo "  📦 ${image}"

    # 기존 이미지 삭제 (캐시 문제 방지)
    docker rmi "$image" 2>/dev/null || true

    # 플랫폼 명시하여 pull
    echo "     Pulling with platform: ${PLATFORM}"
    docker pull --platform "${PLATFORM}" "$image"

    # 방법 1: kind load docker-image 시도
    echo "     Loading to Kind cluster (docker-image)..."
    if kind load docker-image "$image" --name "$CLUSTER_NAME" 2>/dev/null; then
        echo "     ✅ 로드 완료 (docker-image)"
        return 0
    fi

    echo "     ⚠️  docker-image 방식 실패, image-archive 시도..."

    # 방법 2: tar 저장 후 image-archive 로드
    echo "     Saving to tar..."
    docker save "$image" -o "$tar_file"

    echo "     Loading to Kind cluster (image-archive)..."
    if kind load image-archive "$tar_file" --name "$CLUSTER_NAME" 2>/dev/null; then
        rm -f "$tar_file"
        echo "     ✅ 로드 완료 (image-archive)"
        return 0
    fi

    echo "     ⚠️  image-archive 방식 실패, 직접 import 시도..."

    # 방법 3: 노드에 직접 ctr import (최후의 수단)
    # Kind 노드의 containerd에 직접 이미지 로드
    # 중요: 모든 노드(control-plane + workers)에 로드해야 함
    local nodes=("${CLUSTER_NAME}-control-plane" "${CLUSTER_NAME}-worker" "${CLUSTER_NAME}-worker2")
    local loaded=false

    for node in "${nodes[@]}"; do
        # 노드 존재 여부 확인
        if ! docker inspect "$node" &>/dev/null; then
            continue
        fi

        echo "     Loading to node: $node"
        if docker exec -i "$node" ctr --namespace=k8s.io images import - < "$tar_file" 2>/dev/null; then
            echo "       ✅ $node 로드 완료"
            loaded=true
        else
            echo "       ⚠️  $node 로드 실패"
        fi
    done

    rm -f "$tar_file"

    if [ "$loaded" = true ]; then
        echo "     ✅ 로드 완료 (direct ctr import)"
        return 0
    fi

    # 모든 방법 실패
    echo "     ❌ 이미지 로드 실패: $image"
    echo ""
    echo "     수동 로드 방법:"
    echo "       docker pull $image"
    echo "       docker save $image -o /tmp/image.tar"
    echo "       # 모든 노드에 로드 필요:"
    echo "       for node in ${CLUSTER_NAME}-control-plane ${CLUSTER_NAME}-worker ${CLUSTER_NAME}-worker2; do"
    echo "         docker exec -i \$node ctr -n k8s.io images import - < /tmp/image.tar"
    echo "       done"
    echo ""
    return 1
}

# =============================================================================
# 인프라 이미지 (Docker Hub에서 직접 pull)
# =============================================================================
# 인프라 이미지는 공식 Docker Hub 레지스트리에서 직접 가져옴
# 서비스 이미지는 AWS ECR에서 K8s가 직접 pull (이 스크립트와 무관)

# Docker Hub에서 이미지 로드
load_image_from_dockerhub() {
    local image=$1
    local name=$2

    echo ""
    echo "📦 ${name} 이미지 로드 중..."
    echo "   Docker Hub: ${image}"

    # Docker Hub에서 pull
    if ! docker pull --platform "${PLATFORM}" "${image}"; then
        echo "   ❌ Docker Hub pull 실패: ${image}"
        return 1
    fi

    # Kind에 로드
    load_to_kind "${image}"
}

# =============================================================================
# 인프라 이미지 로드 (SKIP_INFRA, ONLY_MONITORING으로 건너뛰기 가능)
# =============================================================================
if [ "${SKIP_INFRA}" != "true" ] && [ "${ONLY_MONITORING}" != "true" ]; then
    echo ""
    echo "--- 인프라 이미지 로드 ---"

    # MinIO - S3 호환 스토리지
    load_image_from_dockerhub "minio/minio:latest" "MinIO"

    # LiveKit - 실시간 통신
    load_image_from_dockerhub "livekit/livekit-server:latest" "LiveKit"
else
    echo ""
    echo "--- 인프라 이미지 건너뜀 (SKIP_INFRA=${SKIP_INFRA:-false}, ONLY_MONITORING=${ONLY_MONITORING:-false}) ---"
fi

# =============================================================================
# 모니터링 이미지 (SKIP_MONITORING, ONLY_INFRA로 건너뛰기 가능)
# =============================================================================
if [ "${SKIP_MONITORING}" != "true" ] && [ "${ONLY_INFRA}" != "true" ]; then
    echo ""
    echo "--- 모니터링 이미지 로드 ---"

    # Prometheus - 메트릭 수집
    load_image_from_dockerhub "prom/prometheus:v2.48.0" "Prometheus"

    # Grafana - 시각화
    load_image_from_dockerhub "grafana/grafana:10.2.2" "Grafana"

    # Loki - 로그 수집
    load_image_from_dockerhub "grafana/loki:2.9.2" "Loki"

    # Promtail - 로그 수집 에이전트
    load_image_from_dockerhub "grafana/promtail:2.9.2" "Promtail"

    # PostgreSQL Exporter - DB 메트릭
    load_image_from_dockerhub "prometheuscommunity/postgres-exporter:v0.15.0" "PostgreSQL Exporter"

    # Redis Exporter - 캐시 메트릭
    load_image_from_dockerhub "oliver006/redis_exporter:v1.55.0" "Redis Exporter"
else
    echo ""
    echo "--- 모니터링 이미지 건너뜀 (SKIP_MONITORING=${SKIP_MONITORING:-false}, ONLY_INFRA=${ONLY_INFRA:-false}) ---"
fi

echo ""
echo "✅ 인프라 이미지 로드 완료!"
echo ""
echo "📝 다음 단계:"
echo "   서비스 이미지는 CI/CD가 AWS ECR에 자동으로 빌드/푸시합니다."
echo "   (service-deploy-dev 브랜치에 push 시 자동 실행)"
echo ""
echo "   Helm 배포:"
echo "      make helm-install-all ENV=dev"
