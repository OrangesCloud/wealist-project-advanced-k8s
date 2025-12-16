#!/bin/bash
# 서비스 이미지 빌드 후 로컬 레지스트리에 푸시하는 스크립트
# Docker Hub rate limit 및 kind load 문제 완전 우회
# macOS bash 3.x 호환
# 병렬 빌드 지원

set -e

REG_PORT="5001"
LOCAL_REG="localhost:${REG_PORT}"
TAG="${IMAGE_TAG:-latest}"  # 환경변수로 오버라이드 가능, 기본값 latest
MAX_PARALLEL="${MAX_PARALLEL:-4}"  # 동시 빌드 수 (기본 4)
SKIP_FRONTEND="${SKIP_FRONTEND:-false}"  # 프론트엔드 제외 (CDN/S3 배포 환경용)

# 색상 출력
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo "=== 서비스 이미지 빌드 & 로컬 레지스트리 푸시 (병렬) ==="
echo "로컬 레지스트리: ${LOCAL_REG}"
echo "동시 빌드 수: ${MAX_PARALLEL}"
echo "프론트엔드 제외: ${SKIP_FRONTEND}"
echo ""

# 레지스트리 실행 확인
if ! curl -s "http://${LOCAL_REG}/v2/" > /dev/null 2>&1; then
    echo -e "${RED}ERROR: 로컬 레지스트리가 실행 중이 아닙니다!${NC}"
    echo "먼저 ./0.setup-cluster.sh 를 실행하세요."
    exit 1
fi

# 프로젝트 루트로 이동 (스크립트는 docker/scripts/dev/ 에 위치)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
cd "$PROJECT_ROOT"
echo "Working directory: $PROJECT_ROOT"
echo ""

# 임시 디렉토리 (빌드 결과 저장)
TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

# Detect if service uses common packages (requires project root context)
# All Go services use common packages, only auth-service (Spring Boot) and frontend (React) don't
uses_common_packages() {
    local name="$1"
    case "$name" in
        auth-service|frontend)
            return 1  # false - Spring Boot and React don't use Go packages
            ;;
        *)
            return 0  # true - all Go services use common packages
            ;;
    esac
}

# 서비스 정보 배열 (backend services)
declare -a BACKEND_SERVICES=(
    "auth-service|services/auth-service|Dockerfile"
    "board-service|services/board-service|docker/Dockerfile"
    "chat-service|services/chat-service|docker/Dockerfile"
    "noti-service|services/noti-service|docker/Dockerfile"
    "storage-service|services/storage-service|docker/Dockerfile"
    "user-service|services/user-service|docker/Dockerfile"
    "video-service|services/video-service|docker/Dockerfile"
)

# Frontend service (only for local development, not for CDN/S3 deployments)
declare -a FRONTEND_SERVICE=(
    "frontend|services/frontend|Dockerfile"
)

# Build full service list based on SKIP_FRONTEND flag
declare -a SERVICES=("${BACKEND_SERVICES[@]}")
if [ "$SKIP_FRONTEND" != "true" ]; then
    SERVICES+=("${FRONTEND_SERVICE[@]}")
fi

# 빌드할 서비스 선택
if [ $# -eq 0 ]; then
    BUILD_SERVICES=("${SERVICES[@]}")
else
    BUILD_SERVICES=()
    for arg in "$@"; do
        for svc in "${SERVICES[@]}"; do
            name="${svc%%|*}"
            if [ "$name" = "$arg" ]; then
                BUILD_SERVICES+=("$svc")
                break
            fi
        done
    done
fi

echo "빌드 대상 (${#BUILD_SERVICES[@]}개):"
for svc in "${BUILD_SERVICES[@]}"; do
    name="${svc%%|*}"
    echo "  - $name"
done
echo ""

# 단일 서비스 빌드 함수
build_service() {
    local service_info="$1"
    local name="${service_info%%|*}"
    local rest="${service_info#*|}"
    local path="${rest%%|*}"
    local dockerfile="${rest#*|}"
    local image_name="${LOCAL_REG}/${name}:${TAG}"
    local log_file="${TEMP_DIR}/${name}.log"

    echo -e "${YELLOW}[START] $name${NC}"

    # 빌드 및 푸시
    {
        echo "=== Building $name ==="
        echo "Path: $path"
        echo "Dockerfile: $dockerfile"
        echo "Image: $image_name"
        echo ""

        # Determine build context based on service type
        local build_context
        if uses_common_packages "$name"; then
            build_context="."
            echo "Using project root context (common packages)"
        else
            build_context="$path"
            echo "Using service directory context"
        fi

        if docker build -t "$image_name" -f "$path/$dockerfile" "$build_context" 2>&1; then
            echo ""
            echo "Pushing to local registry..."
            if docker push "$image_name" 2>&1; then
                echo "SUCCESS"
                echo "$name" > "${TEMP_DIR}/${name}.success"
            else
                echo "PUSH_FAILED"
                echo "$name" > "${TEMP_DIR}/${name}.failed"
            fi
        else
            echo "BUILD_FAILED"
            echo "$name" > "${TEMP_DIR}/${name}.failed"
        fi
    } > "$log_file" 2>&1

    # 결과 출력
    if [ -f "${TEMP_DIR}/${name}.success" ]; then
        echo -e "${GREEN}[SUCCESS] $name${NC}"
    else
        echo -e "${RED}[FAILED] $name${NC}"
    fi
}

# 병렬 빌드 실행
echo -e "${BLUE}🔨 병렬 빌드 시작...${NC}"
echo ""

# 현재 실행 중인 빌드 수 추적
running=0
pids=()

for svc in "${BUILD_SERVICES[@]}"; do
    # 최대 병렬 수에 도달하면 대기
    while [ $running -ge $MAX_PARALLEL ]; do
        # 완료된 프로세스 확인
        for i in "${!pids[@]}"; do
            if ! kill -0 "${pids[$i]}" 2>/dev/null; then
                unset 'pids[$i]'
                ((running--)) || true
            fi
        done
        # 재배열
        pids=("${pids[@]}")
        sleep 0.5
    done

    # 백그라운드로 빌드 시작
    build_service "$svc" &
    pids+=($!)
    ((running++)) || true
done

# 모든 빌드 완료 대기
echo ""
echo -e "${BLUE}⏳ 모든 빌드 완료 대기 중...${NC}"
wait

# 결과 요약
echo ""
echo "=== 빌드 결과 요약 ==="

success_count=0
failed_count=0
failed_services=""

for svc in "${BUILD_SERVICES[@]}"; do
    name="${svc%%|*}"
    if [ -f "${TEMP_DIR}/${name}.success" ]; then
        ((success_count++)) || true
        echo -e "  ${GREEN}✅ $name${NC}"
    else
        ((failed_count++)) || true
        failed_services="${failed_services} $name"
        echo -e "  ${RED}❌ $name${NC}"
        # 실패 로그 출력
        if [ -f "${TEMP_DIR}/${name}.log" ]; then
            echo -e "     ${RED}--- Log ---${NC}"
            tail -20 "${TEMP_DIR}/${name}.log" | sed 's/^/     /'
            echo -e "     ${RED}-----------${NC}"
        fi
    fi
done

echo ""
echo -e "성공: ${GREEN}${success_count}${NC}, 실패: ${RED}${failed_count}${NC}"

if [ $failed_count -gt 0 ]; then
    echo ""
    echo -e "${RED}실패한 서비스:${failed_services}${NC}"
    echo "개별 로그: ${TEMP_DIR}/<service>.log"
fi

echo ""
echo "=== 완료! ==="
echo ""
echo "로컬 레지스트리 이미지 확인:"
echo "  curl -s http://${LOCAL_REG}/v2/_catalog"
echo ""
echo "배포 명령어:"
echo "  make helm-install-all ENV=local-kind"
echo ""
