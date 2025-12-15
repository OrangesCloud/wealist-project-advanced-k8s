#!/bin/bash
# =============================================================================
# weAlist - Monorepo Build Development Script
# =============================================================================
# Multi-Service Dockerfile을 사용하여 Go 서비스를 빌드합니다.
# shared package (wealist-advanced-go-pkg)를 한 번만 컴파일합니다.
#
# 사용법:
#   ./docker/scripts/dev-mono.sh [command]
#
# Commands:
#   up         - 개발 환경 시작 (멀티서비스 빌드)
#   down       - 개발 환경 중지
#   build      - Go 서비스만 멀티빌드
#   clean      - 볼륨 포함 모두 삭제
# =============================================================================

set -e

# BuildKit 활성화 (병렬 빌드 + 캐시)
export DOCKER_BUILDKIT=1
export COMPOSE_DOCKER_CLI_BUILD=1

# 색상 정의
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# 프로젝트 루트 디렉토리로 이동
cd "$(dirname "$0")/../.."

echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}🔧 Monorepo Build Mode - shared package 1회 컴파일${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

# =============================================================================
# 환경 파일 자동 생성 함수 (dev.sh에서 복사)
# =============================================================================
setup_env_files() {
    local created_files=()
    local needs_review=false

    echo -e "${BLUE}🔧 환경 파일 확인 중...${NC}"

    if [ ! -f "docker/env/.env.dev" ]; then
        if [ -f "docker/env/.env.dev.example" ]; then
            cp docker/env/.env.dev.example docker/env/.env.dev
            created_files+=("docker/env/.env.dev")
            needs_review=true
        else
            echo -e "${RED}❌ docker/env/.env.dev.example 파일이 없습니다.${NC}"
            exit 1
        fi
    fi

    if [ ${#created_files[@]} -gt 0 ]; then
        echo -e "${GREEN}✅ 환경 파일이 생성되었습니다.${NC}"
        if [ "$needs_review" = true ]; then
            echo -e "${YELLOW}💡 docker/env/.env.dev 파일을 확인하세요.${NC}"
        fi
    fi
}

setup_env_files

# 환경변수 설정
ENV_FILE="docker/env/.env.dev"
ENV_FILE_OPTION="--env-file $ENV_FILE"
export VITE_API_BASE_URL="http://localhost"

# Go 서비스 목록
GO_SERVICES=(
    "user-service"
    "board-service"
    "chat-service"
    "noti-service"
    "storage-service"
    "video-service"
)

# =============================================================================
# Multi-Service Dockerfile로 Go 서비스 빌드
# =============================================================================
build_go_services() {
    echo -e "${BLUE}🔨 Go 서비스 멀티빌드 시작...${NC}"
    echo -e "${YELLOW}   (shared package는 한 번만 컴파일됩니다)${NC}"
    echo ""

    local start_time=$(date +%s)

    for service in "${GO_SERVICES[@]}"; do
        echo -e "${CYAN}▶ Building $service...${NC}"
        docker build \
            -f docker/base/Dockerfile.go-services \
            --target "${service}" \
            -t "wealist/${service}:latest" \
            . 2>&1 | tail -5
        echo -e "${GREEN}✓ $service 완료${NC}"
    done

    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    echo ""
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}✅ Go 서비스 빌드 완료! (${duration}초)${NC}"
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

# =============================================================================
# 나머지 서비스 빌드 및 시작
# =============================================================================
start_all_services() {
    echo -e "${BLUE}🚀 전체 서비스 시작 중...${NC}"

    # auth-service, frontend는 기존 방식으로 빌드
    docker compose $ENV_FILE_OPTION -f docker/compose/docker-compose.yml up -d --build auth-service frontend-service

    # 인프라 + 나머지 서비스 시작 (빌드된 이미지 사용)
    docker compose $ENV_FILE_OPTION -f docker/compose/docker-compose.yml up -d

    echo -e "${GREEN}✅ 모든 서비스가 시작되었습니다.${NC}"
    echo ""
    echo -e "${BLUE}📊 서비스 접속 정보:${NC}"
    echo "   - Frontend:    http://localhost (nginx) 또는 http://localhost:3000"
    echo "   - API Gateway: http://localhost (nginx)"
    echo "   - Grafana:     http://localhost:3001"
    echo ""
    echo -e "${BLUE}💡 로그 확인: ./docker/scripts/dev-mono.sh logs${NC}"
}

# =============================================================================
# 커맨드 처리
# =============================================================================
COMMAND=${1:-up}

case $COMMAND in
    up)
        build_go_services
        start_all_services
        ;;

    build)
        build_go_services
        ;;

    build-parallel)
        echo -e "${BLUE}🔨 Go 서비스 병렬 빌드 시작...${NC}"
        local start_time=$(date +%s)

        # 병렬로 빌드 (백그라운드)
        for service in "${GO_SERVICES[@]}"; do
            (
                echo -e "${CYAN}▶ Building $service...${NC}"
                docker build \
                    -f docker/base/Dockerfile.go-services \
                    --target "${service}" \
                    -t "wealist/${service}:latest" \
                    . > /dev/null 2>&1
                echo -e "${GREEN}✓ $service 완료${NC}"
            ) &
        done

        # 모든 백그라운드 작업 대기
        wait

        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        echo -e "${GREEN}✅ 병렬 빌드 완료! (${duration}초)${NC}"
        ;;

    down)
        echo -e "${YELLOW}⏹️  개발 환경을 중지합니다...${NC}"
        docker compose $ENV_FILE_OPTION -f docker/compose/docker-compose.yml down
        echo -e "${GREEN}✅ 개발 환경이 중지되었습니다.${NC}"
        ;;

    logs)
        SERVICE=${2:-}
        if [ -z "$SERVICE" ]; then
            docker compose $ENV_FILE_OPTION -f docker/compose/docker-compose.yml logs -f
        else
            docker compose $ENV_FILE_OPTION -f docker/compose/docker-compose.yml logs -f "$SERVICE"
        fi
        ;;

    clean)
        echo -e "${RED}⚠️  모든 컨테이너, 볼륨을 삭제합니다.${NC}"
        read -p "계속하시겠습니까? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            docker compose $ENV_FILE_OPTION -f docker/compose/docker-compose.yml down -v --remove-orphans
            echo -e "${GREEN}✅ 정리가 완료되었습니다.${NC}"
        fi
        ;;

    ps)
        docker compose $ENV_FILE_OPTION -f docker/compose/docker-compose.yml ps
        ;;

    *)
        echo -e "${RED}❌ 알 수 없는 명령어: $COMMAND${NC}"
        echo ""
        echo "사용 가능한 명령어:"
        echo "  up              - Go 멀티빌드 후 전체 시작"
        echo "  build           - Go 서비스만 멀티빌드 (순차)"
        echo "  build-parallel  - Go 서비스 병렬 빌드"
        echo "  down            - 개발 환경 중지"
        echo "  logs            - 로그 확인"
        echo "  clean           - 모두 삭제"
        echo "  ps              - 실행 중인 서비스 확인"
        exit 1
        ;;
esac
