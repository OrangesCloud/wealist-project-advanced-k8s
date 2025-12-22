#!/bin/bash
# 로컬에 빌드된 서비스 이미지를 로컬 레지스트리에 푸시
# 이미 로컬 레지스트리에 있으면 스킵

set -e

LOCAL_REG="localhost:5001"
TAG="${IMAGE_TAG:-latest}"
IMAGE_PREFIX="${IMAGE_PREFIX:-localhost:5001}"  # 이미지 네임스페이스 다른 환경이면 바꿔야 할지도

# 색상 출력
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== 로컬 서비스 이미지 → 로컬 레지스트리 ===${NC}"
echo "로컬 레지스트리: ${LOCAL_REG}"
echo "이미지 네임스페이스: ${IMAGE_PREFIX}"
echo "이미지 태그: ${TAG}"
echo ""

# 레지스트리 확인
if ! curl -s "http://${LOCAL_REG}/v2/" > /dev/null 2>&1; then
    echo -e "${RED}ERROR: 레지스트리 없음. ./0.setup-cluster.sh 먼저 실행${NC}"
    exit 1
fi

# 로컬 레지스트리에 이미지 있는지 확인
image_exists() {
    local name=$1 tag=$2
    curl -sf "http://${LOCAL_REG}/v2/${name}/manifests/${tag}" > /dev/null 2>&1
}

# 로컬 이미지를 레지스트리에 푸시
push_local_image() {
    local service_name=$1
    local src_image="${IMAGE_PREFIX}/${service_name}:${TAG}"
    local dest_image="${LOCAL_REG}/${service_name}:${TAG}"

    # 로컬 레지스트리에 이미 있으면 스킵
    if image_exists "$service_name" "$TAG"; then
        echo -e "${GREEN}✓${NC} ${service_name}:${TAG} - 이미 있음 (스킵)"
        return 0
    fi

    # 로컬에 이미지가 있는지 확인
    if ! docker image inspect "$src_image" > /dev/null 2>&1; then
        echo -e "${YELLOW}⚠${NC} $src_image - 로컬에 없음 (스킵)"
        return 1
    fi

    echo -e "${BLUE}📤${NC} $src_image → $dest_image"
    
    # 태그 및 푸시
    if docker tag "$src_image" "$dest_image" && docker push "$dest_image"; then
        echo -e "${GREEN}✅${NC} ${service_name} 푸시 완료"
        return 0
    else
        echo -e "${RED}❌${NC} ${service_name} 푸시 실패"
        return 1
    fi
}

# 서비스 목록 (프로젝트 구조에 맞게)
SERVICES=(
    "auth-service"
    "board-service" 
    "chat-service"
    "noti-service"
    "storage-service"
    "user-service"
    "video-service"
)

# 빌드할 서비스 선택 (인자가 있으면 해당 서비스만, 없으면 전체)
if [ $# -eq 0 ]; then
    PUSH_SERVICES=("${SERVICES[@]}")
else
    PUSH_SERVICES=("$@")
fi

echo -e "${BLUE}푸시 대상 서비스 (${#PUSH_SERVICES[@]}개):${NC}"
for svc in "${PUSH_SERVICES[@]}"; do
    echo "  - ${IMAGE_PREFIX}/$svc:${TAG}"
done
echo ""

# 결과 추적
success_count=0
failed_count=0
failed_services=""
skipped_count=0

# 각 서비스 푸시
for service in "${PUSH_SERVICES[@]}"; do
    # 서비스가 유효한지 확인
    if [[ ! " ${SERVICES[@]} " =~ " ${service} " ]]; then
        echo -e "${RED}⚠${NC} 알 수 없는 서비스: $service (스킵)"
        continue
    fi

    echo ""
    echo -e "${YELLOW}[처리 중] $service${NC}"
    
    if push_local_image "$service"; then
        if image_exists "$service" "$TAG"; then
            ((success_count++)) || true
        else
            ((skipped_count++)) || true
        fi
    else
        ((failed_count++)) || true
        failed_services="${failed_services} $service"
    fi
done

# 결과 요약
echo ""
echo -e "${BLUE}=== 푸시 결과 요약 ===${NC}"
echo -e "성공: ${GREEN}${success_count}${NC}"
echo -e "스킵: ${YELLOW}${skipped_count}${NC}" 
echo -e "실패: ${RED}${failed_count}${NC}"

if [ $failed_count -gt 0 ]; then
    echo -e "${RED}실패한 서비스:${failed_services}${NC}"
fi

echo ""
echo -e "${BLUE}=== 완료! ===${NC}"
echo ""
echo "로컬 레지스트리 이미지 확인:"
echo "  curl -s http://${LOCAL_REG}/v2/_catalog | jq"
echo ""
echo "특정 서비스 태그 확인:"
echo "  curl -s http://${LOCAL_REG}/v2/<service-name>/tags/list | jq"
echo ""
echo "배포 명령어:"
echo "  make helm-deploy"
echo ""

# 성공한 서비스가 있으면 성공 종료
if [ $success_count -gt 0 ] || [ $skipped_count -gt 0 ]; then
    exit 0
else
    exit 1
fi