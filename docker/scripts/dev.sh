#!/bin/bash
# =============================================================================
# weAlist - Development Environment Startup Script
# =============================================================================
# 개발 환경을 시작하는 스크립트입니다.
#
# 사용법:
#   ./docker/scripts/dev.sh [command]
#
# Commands:
#   up         - 개발 환경 시작 (기본값)
#   down       - 개발 환경 중지
#   restart    - 개발 환경 재시작
#   logs       - 로그 확인
#   build      - 이미지 다시 빌드
#   clean      - 볼륨 포함 모두 삭제
# =============================================================================

set -e

# 색상 정의
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 프로젝트 루트 디렉토리로 이동
cd "$(dirname "$0")/../.."

# 환경변수 파일 확인
ENV_FILE="docker/env/.env.dev"
if [ ! -f "$ENV_FILE" ]; then
    echo -e "${YELLOW}⚠️  환경변수 파일이 없습니다. 템플릿에서 생성합니다...${NC}"
    cp docker/env/.env.dev.example "$ENV_FILE"
    echo -e "${GREEN}✅ $ENV_FILE 파일이 생성되었습니다.${NC}"
    echo -e "${YELLOW}   필요한 값들을 수정한 후 다시 실행하세요.${NC}"
    exit 1
fi

# Docker Compose 파일 경로
COMPOSE_FILES="-f docker/compose/docker-compose.yml"

# BuildKit 활성화 (cache mount 사용을 위해 필수)
export DOCKER_BUILDKIT=1
export COMPOSE_DOCKER_CLI_BUILD=1

# 환경변수 파일을 명시적으로 지정 (compose 파일 내 변수 치환용)
ENV_FILE_OPTION="--env-file $ENV_FILE"

# =============================================================================
# [⭐️ 핵심 변경 사항]: 로컬 환경 API Base URL 강제 오버라이드
# 
# 프론트엔드 컨테이너의 환경 변수 VITE_API_BASE_URL을 
# .env 파일 내용과 관계없이 localhost로 강제 설정합니다.
# 이 쉘 변수는 docker compose 실행 시 .env 내용을 덮어씁니다.
# =============================================================================
export VITE_API_BASE_URL="http://localhost"
echo -e "${BLUE}⚙️  로컬 개발 환경 설정: VITE_API_BASE_URL=${VITE_API_BASE_URL}${NC}"

# 커맨드 처리
COMMAND=${1:-up}

case $COMMAND in
    up)
        echo -e "${BLUE}🚀 개발 환경을 백그라운드로 시작합니다...${NC}"

        # Swagger 문서 생성 스킵 (빌드 속도 개선)
        # echo -e "${BLUE}📝 Swagger 문서 확인 중...${NC}"
        # ./docker/scripts/generate-swagger.sh all 2>/dev/null || echo -e "${YELLOW}⚠️  Swagger 생성 스킵 (swag 미설치 - Docker에서 생성됨)${NC}"

        echo -e "${BLUE}🔨 이미지 빌드 및 컨테이너 시작 중 (병렬 처리)...${NC}"
        # --parallel: 서비스 빌드를 병렬로 수행
        # --build: 빌드 완료된 서비스부터 바로 시작
        docker compose $ENV_FILE_OPTION $COMPOSE_FILES build --parallel
        docker compose $ENV_FILE_OPTION $COMPOSE_FILES up -d
        echo -e "${GREEN}✅ 개발 환경이 시작되었습니다.${NC}"
        echo -e "${BLUE}📊 서비스 접속 정보:${NC}"
        echo "   - Frontend:    http://localhost:3000"
        echo "   - Auth API:    http://localhost:8080 (OAuth2, JWT)"
        echo "   - User API:    http://localhost:8081"
        echo "   - Board API:   http://localhost:8000"
        echo "   - Chat API:    http://localhost:8001"
        echo "   - Noti API:    http://localhost:8002"
        echo "   - Storage API: http://localhost:8003"
        echo "   - Video API:   http://localhost:8004"
        echo "   - LiveKit:     ws://localhost:7880 (WebRTC SFU)"
        echo "   - PostgreSQL:  localhost:5432"
        echo "   - Redis:       localhost:6379"
        echo "   - MinIO:       http://localhost:9000 (Console: http://localhost:9001)"
        echo -e ""
        echo -e "${BLUE}📈 모니터링:${NC}"
        echo "   - Grafana:     http://localhost:3001 (admin/admin)"
        echo "   - Prometheus:  http://localhost:9090"
        echo "   - Loki:        http://localhost:3100"
        echo -e ""
        echo -e "${BLUE}📚 Swagger 문서:${NC}"
        echo "   - Auth API:    http://localhost:8080/swagger-ui/index.html"
        echo "   - User API:    http://localhost:8081/swagger/index.html"
        echo "   - Board API:   http://localhost:8000/swagger/index.html"
        echo "   - Chat API:    http://localhost:8001/swagger/index.html"
        echo "   - Noti API:    http://localhost:8002/swagger/index.html"
        echo "   - Storage API: http://localhost:8003/swagger/index.html"
        echo "   - Video API:   http://localhost:8004/swagger/index.html"
        echo -e ""
        echo -e "${BLUE}💡 로그 확인: ./docker/scripts/dev.sh logs${NC}"
        ;;

    up-fg)
        echo -e "${BLUE}🚀 개발 환경을 포그라운드로 시작합니다...${NC}"
        docker compose $ENV_FILE_OPTION $COMPOSE_FILES up
        ;;

    down)
        echo -e "${YELLOW}⏹️  개발 환경을 중지합니다...${NC}"
        docker compose $ENV_FILE_OPTION $COMPOSE_FILES down
        echo -e "${GREEN}✅ 개발 환경이 중지되었습니다.${NC}"
        ;;

    restart)
        echo -e "${YELLOW}🔄 개발 환경을 재시작합니다...${NC}"
        docker compose $ENV_FILE_OPTION $COMPOSE_FILES restart
        echo -e "${GREEN}✅ 개발 환경이 재시작되었습니다.${NC}"
        ;;

    logs)
        SERVICE=${2:-}
        if [ -z "$SERVICE" ]; then
            docker compose $ENV_FILE_OPTION $COMPOSE_FILES logs -f
        else
            docker compose $ENV_FILE_OPTION $COMPOSE_FILES logs -f "$SERVICE"
        fi
        ;;

    build)
        echo -e "${BLUE}🔨 이미지를 다시 빌드합니다 (병렬)...${NC}"
        docker compose $ENV_FILE_OPTION $COMPOSE_FILES build --parallel --no-cache
        echo -e "${GREEN}✅ 빌드가 완료되었습니다.${NC}"
        ;;

    rebuild)
        echo -e "${BLUE}🔨 이미지를 다시 빌드하고 시작합니다 (병렬 처리)...${NC}"
        docker compose $ENV_FILE_OPTION $COMPOSE_FILES build --parallel
        docker compose $ENV_FILE_OPTION $COMPOSE_FILES up -d
        echo -e "${GREEN}✅ 빌드 및 시작이 완료되었습니다.${NC}"
        ;;

    clean)
        echo -e "${RED}⚠️  모든 컨테이너, 볼륨, 이미지를 삭제합니다.${NC}"
        read -p "계속하시겠습니까? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            docker compose $ENV_FILE_OPTION $COMPOSE_FILES down -v --remove-orphans
            echo -e "${GREEN}✅ 정리가 완료되었습니다.${NC}"
        else
            echo -e "${YELLOW}취소되었습니다.${NC}"
        fi
        ;;

    ps)
        docker compose $ENV_FILE_OPTION $COMPOSE_FILES ps
        ;;

    exec)
        SERVICE=${2:-user-service}
        SHELL=${3:-bash}
        docker compose $ENV_FILE_OPTION $COMPOSE_FILES exec "$SERVICE" "$SHELL"
        ;;

    swagger)
        SERVICE=${2:-all}
        FORCE_FLAG=${3:-}
        echo -e "${BLUE}📝 Swagger 문서를 생성합니다...${NC}"
        ./docker/scripts/generate-swagger.sh "$SERVICE" "$FORCE_FLAG"
        ;;

    *)
        echo -e "${RED}❌ 알 수 없는 명령어: $COMMAND${NC}"
        echo ""
        echo "사용 가능한 명령어:"
        echo "  up         - 개발 환경 시작 (백그라운드)"
        echo "  up-fg      - 개발 환경 시작 (포그라운드)"
        echo "  down       - 개발 환경 중지"
        echo "  restart    - 개발 환경 재시작"
        echo "  logs       - 로그 확인 (logs [service])"
        echo "  build      - 이미지 다시 빌드"
        echo "  rebuild    - 빌드 후 시작"
        echo "  clean      - 모두 삭제 (볼륨 포함)"
        echo "  ps         - 실행 중인 서비스 확인"
        echo "  exec       - 컨테이너 접속 (exec [service] [shell])"
        echo "  swagger    - Swagger 문서 생성 (swagger [service] [--force])"
        exit 1
        ;;
esac