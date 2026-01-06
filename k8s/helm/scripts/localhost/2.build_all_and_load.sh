#!/bin/bash
# =============================================================================
# 모든 서비스 이미지 빌드 및 로드 (localhost 환경용)
# - Backend 서비스 + Frontend 포함
# - 레지스트리에 이미지가 있으면 스킵 (재사용)
# =============================================================================
#
# 사용법:
#   ./2.build_all_and_load.sh           # 레지스트리에 없는 이미지만 빌드
#   ./2.build_all_and_load.sh --force   # 모든 이미지 강제 재빌드
#   FORCE_BUILD=1 ./2.build_all_and_load.sh  # 환경변수로 강제 빌드

# set -e 제거 - 개별 빌드 실패해도 계속 진행

LOCAL_REG="localhost:5001"
TAG="${IMAGE_TAG:-latest}"
FORCE_BUILD="${FORCE_BUILD:-0}"

# 빌드 결과 추적
FAILED_BUILDS=""
SUCCESS_COUNT=0
FAIL_COUNT=0

# --force 플래그 처리
if [[ "$1" == "--force" ]] || [[ "$1" == "-f" ]]; then
    FORCE_BUILD=1
fi

echo "=== 서비스 이미지 빌드 및 로드 (localhost 환경) ==="
echo ""
echo "레지스트리: ${LOCAL_REG}"
echo "태그: ${TAG}"
if [[ "$FORCE_BUILD" == "1" ]]; then
    echo "모드: 강제 재빌드 (--force)"
else
    echo "모드: 캐시 사용 (레지스트리에 있으면 스킵)"
fi
echo ""

# 필수 도구 확인
if ! command -v jq &> /dev/null; then
    echo "ERROR: jq 설치 필요 (brew install jq 또는 apt install jq)"
    exit 1
fi

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

# 로컬 레지스트리에 이미지 있는지 확인 (매니페스트 + 레이어 검증)
image_exists() {
    local name=$1 tag=$2
    local manifest_url="http://${LOCAL_REG}/v2/${name}/manifests/${tag}"

    # 1. 매니페스트 조회 (v2, OCI, manifest list 모두 지원)
    local response
    response=$(curl -s -w "\n%{http_code}" \
        -H "Accept: application/vnd.docker.distribution.manifest.v2+json" \
        -H "Accept: application/vnd.oci.image.manifest.v1+json" \
        -H "Accept: application/vnd.docker.distribution.manifest.list.v2+json" \
        -H "Accept: application/vnd.oci.image.index.v1+json" \
        "$manifest_url" 2>/dev/null)

    local http_code=$(echo "$response" | tail -n1)
    local body=$(echo "$response" | sed '$d')

    # 2. HTTP 200 아니면 이미지 없음
    if [[ "$http_code" != "200" ]]; then
        return 1
    fi

    # 3. JSON 유효성 확인
    if ! echo "$body" | jq -e '.' > /dev/null 2>&1; then
        echo "  ⚠️  ${name}:${tag} - JSON 형식 오류"
        return 1
    fi

    # 4. manifest list인지 단일 manifest인지 확인
    local media_type
    media_type=$(echo "$body" | jq -r '.mediaType // .schemaVersion' 2>/dev/null)

    # manifest list (multi-platform) 또는 OCI index
    if echo "$body" | jq -e '.manifests' > /dev/null 2>&1; then
        # manifest list - manifests 배열이 비어있지 않으면 OK
        local manifest_count
        manifest_count=$(echo "$body" | jq '.manifests | length' 2>/dev/null)
        if [[ "$manifest_count" -gt 0 ]]; then
            return 0
        fi
        echo "  ⚠️  ${name}:${tag} - manifest list가 비어있음"
        return 1
    fi

    # 단일 manifest - config 확인
    if echo "$body" | jq -e '.config' > /dev/null 2>&1; then
        local config_digest
        config_digest=$(echo "$body" | jq -r '.config.digest' 2>/dev/null)

        if [[ -z "$config_digest" ]] || [[ "$config_digest" == "null" ]]; then
            echo "  ⚠️  ${name}:${tag} - config digest 없음"
            return 1
        fi

        # config blob 존재 확인
        local blob_check
        blob_check=$(curl -s -o /dev/null -w "%{http_code}" \
            "http://${LOCAL_REG}/v2/${name}/blobs/${config_digest}" 2>/dev/null)

        if [[ "$blob_check" != "200" ]]; then
            echo "  ⚠️  ${name}:${tag} - config blob 없음"
            return 1
        fi
        return 0
    fi

    # 어떤 형식도 아님
    echo "  ⚠️  ${name}:${tag} - 알 수 없는 매니페스트 형식"
    return 1
}

# 푸시 후 이미지 검증
verify_push() {
    local name=$1 tag=$2

    echo "  검증 중..."

    # 푸시 직후 검증 (최대 3회 재시도, 레지스트리 동기화 대기)
    local retry=0
    while [[ $retry -lt 3 ]]; do
        if image_exists "$name" "$tag"; then
            return 0
        fi
        ((retry++))
        sleep 1
    done

    echo "  ❌ 푸시 후 검증 실패: ${name}:${tag}"
    return 1
}

# 이미지 빌드 및 푸시 (캐시 체크 포함)
build_and_push() {
    local name=$1
    local context=$2
    local dockerfile=$3

    # 캐시 체크 (--force가 아니면)
    if [[ "$FORCE_BUILD" != "1" ]] && image_exists "$name" "$TAG"; then
        echo "✓ ${name}:${TAG} - 이미 있음 (스킵)"
        ((SUCCESS_COUNT++)) || true
        return 0
    fi

    echo "🔨 ${name}:${TAG} 빌드 중..."

    local build_cmd
    if [[ -n "$dockerfile" ]]; then
        build_cmd="docker build -t ${LOCAL_REG}/${name}:${TAG} -f $dockerfile $context"
    else
        build_cmd="docker build -t ${LOCAL_REG}/${name}:${TAG} $context"
    fi

    echo "  명령어: $build_cmd"

    if eval "$build_cmd"; then
        echo "  빌드 성공, 푸시 중..."
        if docker push "${LOCAL_REG}/${name}:${TAG}"; then
            # 푸시 후 레지스트리에서 이미지 검증
            if verify_push "$name" "$TAG"; then
                echo "✅ ${name} 푸시 및 검증 완료"
                ((SUCCESS_COUNT++)) || true
                return 0
            else
                echo "❌ ${name} 푸시는 성공했으나 검증 실패"
                FAILED_BUILDS="${FAILED_BUILDS} ${name}"
                ((FAIL_COUNT++)) || true
                return 1
            fi
        else
            echo "❌ ${name} 푸시 실패"
            FAILED_BUILDS="${FAILED_BUILDS} ${name}"
            ((FAIL_COUNT++)) || true
            return 1
        fi
    else
        echo "❌ ${name} 빌드 실패"
        FAILED_BUILDS="${FAILED_BUILDS} ${name}"
        ((FAIL_COUNT++)) || true
        return 1
    fi
}

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
)

for service in "${BACKEND_SERVICES[@]}"; do
    echo ""
    SERVICE_PATH="services/${service}"

    if [ ! -d "$SERVICE_PATH" ]; then
        echo "⚠️  ${SERVICE_PATH} 없음 - 스킵"
        continue
    fi

    # Dockerfile 확인 (루트 또는 docker/ 하위)
    if [ -f "${SERVICE_PATH}/Dockerfile" ]; then
        # 서비스 루트에 Dockerfile이 있으면 서비스 폴더를 컨텍스트로
        build_and_push "$service" "${SERVICE_PATH}" ""
    elif [ -f "${SERVICE_PATH}/docker/Dockerfile" ]; then
        # docker/ 하위에 Dockerfile이 있으면 프로젝트 루트를 컨텍스트로 (Go 모노레포)
        build_and_push "$service" "." "${SERVICE_PATH}/docker/Dockerfile"
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
    build_and_push "frontend" "${FRONTEND_PATH}" ""
else
    echo "⚠️  ${FRONTEND_PATH}/Dockerfile 없음 - 스킵"
fi

echo ""
echo "=========================================="
echo "  빌드 결과 요약"
echo "=========================================="
echo ""
echo "✅ 성공: ${SUCCESS_COUNT}개"
echo "❌ 실패: ${FAIL_COUNT}개"

if [[ -n "$FAILED_BUILDS" ]]; then
    echo ""
    echo "실패한 서비스:${FAILED_BUILDS}"
    echo ""
    echo "개별 재빌드:"
    for svc in $FAILED_BUILDS; do
        echo "  docker build -t localhost:5001/${svc}:latest ./services/${svc}"
        echo "  docker push localhost:5001/${svc}:latest"
    done
fi

echo ""
echo "=========================================="
echo "  레지스트리 이미지 최종 검증"
echo "=========================================="
echo ""

# 레지스트리의 모든 이미지 목록 조회
ALL_IMAGES=$(curl -s "http://${LOCAL_REG}/v2/_catalog" 2>/dev/null | jq -r '.repositories[]?' 2>/dev/null)

if [[ -z "$ALL_IMAGES" ]]; then
    echo "⚠️  레지스트리에 이미지가 없습니다!"
else
    echo "레지스트리 이미지 상태 (${LOCAL_REG}):"
    echo ""

    VERIFIED_COUNT=0
    INVALID_COUNT=0

    for img in $ALL_IMAGES; do
        # 태그 목록 조회
        TAGS=$(curl -s "http://${LOCAL_REG}/v2/${img}/tags/list" 2>/dev/null | jq -r '.tags[]?' 2>/dev/null)

        if [[ -z "$TAGS" ]]; then
            echo "  ⚠️  ${img}: 태그 없음"
            ((INVALID_COUNT++)) || true
            continue
        fi

        for tag in $TAGS; do
            if image_exists "$img" "$tag"; then
                echo "  ✅ ${img}:${tag}"
                ((VERIFIED_COUNT++)) || true
            else
                echo "  ❌ ${img}:${tag} (불완전)"
                ((INVALID_COUNT++)) || true
            fi
        done
    done

    echo ""
    echo "검증 결과: ✅ ${VERIFIED_COUNT}개 정상, ❌ ${INVALID_COUNT}개 문제"
fi

echo ""
echo "=========================================="
if [[ $FAIL_COUNT -eq 0 ]] && [[ ${INVALID_COUNT:-0} -eq 0 ]]; then
    echo "  🎉 모든 서비스 이미지 빌드 및 검증 완료!"
else
    echo "  ⚠️  일부 이미지에 문제가 있습니다"
fi
echo "=========================================="
echo ""
echo "다음 단계:"
echo "  make helm-install-all ENV=localhost"
echo ""
echo "💡 팁: 이미지 강제 재빌드하려면:"
echo "  ./2.build_all_and_load.sh --force"

# 실패가 있으면 exit code 1
if [[ $FAIL_COUNT -gt 0 ]]; then
    exit 1
fi
