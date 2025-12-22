#!/bin/bash
# =============================================================================
# 서비스 이미지 빌드 및 GHCR 푸시 (dev 환경)
# =============================================================================
# Backend 서비스만 빌드:
#   - auth-service, board-service, chat-service, noti-service
#   - storage-service, user-service, video-service
#
# 제외 항목 (dev 환경에서는 외부 서비스 사용):
#   - postgres: AWS RDS 사용
#   - redis: AWS ElastiCache 사용
#   - frontend: S3/CloudFront CDN 배포
#
# GHCR (GitHub Container Registry)에 푸시

# set -e 제거 - 개별 빌드 실패해도 계속 진행

GHCR_REGISTRY="${GHCR_REGISTRY:-ghcr.io/orangescloud}"
TAG="${IMAGE_TAG:-latest}"
MAX_PARALLEL="${MAX_PARALLEL:-4}"

# 색상 출력
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo "=== 서비스 이미지 빌드 & GHCR 푸시 (dev 환경) ==="
echo ""
echo "📦 GHCR Registry: ${GHCR_REGISTRY}"
echo "🏷️  Image Tag: ${TAG}"
echo "⚡ 동시 빌드 수: ${MAX_PARALLEL}"
echo ""

# 필수 도구 확인
if ! command -v jq &> /dev/null; then
    echo -e "${RED}ERROR: jq 설치 필요 (brew install jq 또는 apt install jq)${NC}"
    exit 1
fi

# GHCR 로그인 확인
echo -e "${BLUE}🔐 GHCR 인증 확인 중...${NC}"
if ! docker pull ${GHCR_REGISTRY}/auth-service:latest 2>/dev/null; then
    echo -e "${YELLOW}⚠️  GHCR 로그인이 필요할 수 있습니다.${NC}"
    echo ""
    echo "   GHCR 로그인:"
    echo "   echo \$GHCR_TOKEN | docker login ghcr.io -u \$GHCR_USERNAME --password-stdin"
    echo ""
fi

# 프로젝트 루트로 이동
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"
cd "$PROJECT_ROOT"
echo "Working directory: $PROJECT_ROOT"
echo ""

# 임시 디렉토리
TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

# Go 서비스는 common packages 사용
uses_common_packages() {
    local name="$1"
    case "$name" in
        auth-service)
            return 1  # Spring Boot
            ;;
        *)
            return 0  # Go services
            ;;
    esac
}

# =============================================================================
# Backend 서비스만 빌드 (dev 환경)
# =============================================================================
# ❌ 제외:
#   - postgres: AWS RDS 사용
#   - redis: AWS ElastiCache 사용
#   - frontend: S3/CloudFront CDN 배포
# ✅ 포함:
#   - auth-service, board-service, chat-service, noti-service
#   - storage-service, user-service, video-service
# =============================================================================
declare -a SERVICES=(
    "auth-service|services/auth-service|Dockerfile"
    "board-service|services/board-service|docker/Dockerfile"
    "chat-service|services/chat-service|docker/Dockerfile"
    "noti-service|services/noti-service|docker/Dockerfile"
    "storage-service|services/storage-service|docker/Dockerfile"
    "user-service|services/user-service|docker/Dockerfile"
    "video-service|services/video-service|docker/Dockerfile"
)

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
    echo "  - $name → ${GHCR_REGISTRY}/${name}:${TAG}"
done
echo ""
echo -e "${YELLOW}참고: postgres, redis, frontend는 dev 환경에서 제외됨${NC}"
echo "  - postgres: AWS RDS 사용"
echo "  - redis: AWS ElastiCache 사용"
echo "  - frontend: S3/CloudFront CDN 배포"
echo ""

# GHCR 이미지 검증 함수
verify_ghcr_image() {
    local name=$1
    local tag=$2
    local full_image="${GHCR_REGISTRY}/${name}:${tag}"

    # docker manifest inspect로 이미지 존재 확인
    if docker manifest inspect "$full_image" > /dev/null 2>&1; then
        return 0
    fi

    # manifest가 안되면 pull 시도
    if docker pull "$full_image" > /dev/null 2>&1; then
        return 0
    fi

    return 1
}

# 단일 서비스 빌드 함수
build_service() {
    local service_info="$1"
    local name="${service_info%%|*}"
    local rest="${service_info#*|}"
    local path="${rest%%|*}"
    local dockerfile="${rest#*|}"
    local image_name="${GHCR_REGISTRY}/${name}:${TAG}"
    local log_file="${TEMP_DIR}/${name}.log"

    echo -e "${YELLOW}[START] $name${NC}"

    {
        echo "=== Building $name ==="
        echo "Path: $path"
        echo "Dockerfile: $dockerfile"
        echo "Image: $image_name"
        echo ""

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
            echo "Pushing to GHCR..."
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

    if [ -f "${TEMP_DIR}/${name}.success" ]; then
        echo -e "${GREEN}[SUCCESS] $name${NC}"
    else
        echo -e "${RED}[FAILED] $name${NC}"
    fi
}

# 병렬 빌드 실행
echo -e "${BLUE}🔨 병렬 빌드 시작...${NC}"
echo ""

running=0
pids=()

for svc in "${BUILD_SERVICES[@]}"; do
    while [ $running -ge $MAX_PARALLEL ]; do
        for i in "${!pids[@]}"; do
            if ! kill -0 "${pids[$i]}" 2>/dev/null; then
                unset 'pids[$i]'
                ((running--)) || true
            fi
        done
        pids=("${pids[@]}")
        sleep 0.5
    done

    build_service "$svc" &
    pids+=($!)
    ((running++)) || true
done

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
fi

echo ""
echo "=========================================="
echo "  GHCR 이미지 최종 검증"
echo "=========================================="
echo ""

verified_count=0
invalid_count=0

for svc in "${BUILD_SERVICES[@]}"; do
    name="${svc%%|*}"
    if verify_ghcr_image "$name" "$TAG"; then
        echo -e "  ${GREEN}✅ ${GHCR_REGISTRY}/${name}:${TAG}${NC}"
        ((verified_count++)) || true
    else
        echo -e "  ${RED}❌ ${GHCR_REGISTRY}/${name}:${TAG} (접근 불가)${NC}"
        ((invalid_count++)) || true
    fi
done

echo ""
echo -e "검증 결과: ${GREEN}✅ ${verified_count}개 정상${NC}, ${RED}❌ ${invalid_count}개 문제${NC}"

echo ""
echo "=========================================="
if [ $failed_count -eq 0 ] && [ $invalid_count -eq 0 ]; then
    echo -e "  ${GREEN}🎉 모든 서비스 이미지 빌드 및 검증 완료!${NC}"
else
    echo -e "  ${YELLOW}⚠️  일부 이미지에 문제가 있습니다${NC}"
fi
echo "=========================================="
echo ""
echo "📝 다음 단계:"
echo "   1. ArgoCD 배포:"
echo "      make bootstrap && make deploy"
echo ""
echo "   2. 또는 Helm 직접 배포:"
echo "      make helm-install-all ENV=dev"
echo ""

# 실패가 있으면 exit code 1
if [ $failed_count -gt 0 ] || [ $invalid_count -gt 0 ]; then
    exit 1
fi
