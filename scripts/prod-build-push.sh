#!/bin/bash
# =============================================================================
# Production Docker Build & Push Script
# =============================================================================
# Terraform apply 후 ECR에 이미지를 빌드하고 푸시하는 스크립트
#
# 사용법:
#   ./scripts/prod-build-push.sh              # 모든 서비스 빌드
#   ./scripts/prod-build-push.sh auth-service # 특정 서비스만 빌드
#
# 사전 요구사항:
#   1. AWS CLI 설정 (aws configure)
#   2. Docker 실행 중
#   3. Terraform apply 완료 (ECR 생성됨)

set -e

# 색상 정의
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 프로젝트 루트 디렉토리
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# 태그
TAG="${TAG:-prod-latest}"

# 서비스 목록
GO_SERVICES=("user-service" "board-service" "chat-service" "noti-service" "storage-service" "video-service")
SPRING_SERVICES=("auth-service")
FRONTEND_SERVICES=("frontend")

# ECR Registry URL 가져오기
get_ecr_registry() {
    cd "$PROJECT_ROOT/terraform/prod/foundation"

    if ! terraform output -raw ecr_registry_url 2>/dev/null; then
        echo -e "${RED}Error: Terraform output 'ecr_registry_url' not found${NC}" >&2
        echo -e "${YELLOW}Run 'terraform apply' in terraform/prod/foundation first${NC}" >&2
        exit 1
    fi
}

# ECR 로그인
ecr_login() {
    local registry=$1
    echo -e "${YELLOW}Logging in to ECR...${NC}"
    aws ecr get-login-password --region ap-northeast-2 | \
        docker login --username AWS --password-stdin "$registry"
    echo -e "${GREEN}ECR login successful${NC}"
}

# Go 서비스 빌드
build_go_service() {
    local service=$1
    local registry=$2
    local image="$registry/prod/$service:$TAG"

    echo -e "${YELLOW}Building $service...${NC}"

    cd "$PROJECT_ROOT"
    docker build \
        -t "$image" \
        -f "services/$service/docker/Dockerfile" \
        .

    echo -e "${GREEN}Built: $image${NC}"
}

# Spring Boot 서비스 빌드
build_spring_service() {
    local service=$1
    local registry=$2
    local image="$registry/prod/$service:$TAG"

    echo -e "${YELLOW}Building $service (Spring Boot)...${NC}"

    cd "$PROJECT_ROOT/services/$service"

    # Gradle 빌드
    ./gradlew bootJar --no-daemon

    # Docker 빌드
    docker build \
        -t "$image" \
        -f docker/Dockerfile \
        .

    echo -e "${GREEN}Built: $image${NC}"
}

# Frontend 빌드
build_frontend() {
    local registry=$1
    local image="$registry/prod/frontend:$TAG"

    echo -e "${YELLOW}Building frontend...${NC}"

    cd "$PROJECT_ROOT/frontend"
    docker build \
        -t "$image" \
        -f Dockerfile \
        .

    echo -e "${GREEN}Built: $image${NC}"
}

# 이미지 푸시
push_image() {
    local registry=$1
    local service=$2
    local image="$registry/prod/$service:$TAG"

    echo -e "${YELLOW}Pushing $image...${NC}"
    docker push "$image"
    echo -e "${GREEN}Pushed: $image${NC}"
}

# 모든 서비스 빌드 및 푸시
build_all() {
    local registry=$1

    echo -e "${YELLOW}========================================${NC}"
    echo -e "${YELLOW}Building all services${NC}"
    echo -e "${YELLOW}========================================${NC}"

    # Go 서비스
    for service in "${GO_SERVICES[@]}"; do
        build_go_service "$service" "$registry"
        push_image "$registry" "$service"
    done

    # Spring Boot 서비스
    for service in "${SPRING_SERVICES[@]}"; do
        build_spring_service "$service" "$registry"
        push_image "$registry" "$service"
    done

    # Frontend
    build_frontend "$registry"
    push_image "$registry" "frontend"

    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}All services built and pushed!${NC}"
    echo -e "${GREEN}========================================${NC}"
}

# 단일 서비스 빌드 및 푸시
build_single() {
    local service=$1
    local registry=$2

    echo -e "${YELLOW}Building single service: $service${NC}"

    if [[ " ${GO_SERVICES[*]} " =~ " $service " ]]; then
        build_go_service "$service" "$registry"
    elif [[ " ${SPRING_SERVICES[*]} " =~ " $service " ]]; then
        build_spring_service "$service" "$registry"
    elif [[ " ${FRONTEND_SERVICES[*]} " =~ " $service " ]]; then
        build_frontend "$registry"
        service="frontend"
    else
        echo -e "${RED}Unknown service: $service${NC}"
        echo "Available services: ${GO_SERVICES[*]} ${SPRING_SERVICES[*]} ${FRONTEND_SERVICES[*]}"
        exit 1
    fi

    push_image "$registry" "$service"
}

# 메인
main() {
    echo -e "${YELLOW}========================================${NC}"
    echo -e "${YELLOW}Production Build & Push Script${NC}"
    echo -e "${YELLOW}========================================${NC}"

    # ECR Registry 가져오기
    ECR_REGISTRY=$(get_ecr_registry)
    echo -e "ECR Registry: ${GREEN}$ECR_REGISTRY${NC}"
    echo -e "Tag: ${GREEN}$TAG${NC}"

    # ECR 로그인
    ecr_login "$ECR_REGISTRY"

    # 빌드
    if [ -z "$1" ]; then
        build_all "$ECR_REGISTRY"
    else
        build_single "$1" "$ECR_REGISTRY"
    fi
}

main "$@"
