#!/bin/bash
# =============================================================================
# weAlist - SonarQube Standalone Environment Script
# =============================================================================
# SonarQube 독립 환경을 관리하는 스크립트입니다.
#
# 사용법:
#   ./docker/scripts/sonar.sh [command]
#
# Commands:
#   up         - SonarQube 환경 시작 (기본값)
#   down       - SonarQube 환경 중지
#   restart    - SonarQube 환경 재시작
#   logs       - 로그 확인
#   status     - 상태 확인
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
COMPOSE_FILES="-f docker/compose/docker-compose.sonarqube.yml"

# BuildKit 활성화
export DOCKER_BUILDKIT=1
export COMPOSE_DOCKER_CLI_BUILD=1

# 환경변수 파일을 명시적으로 지정
ENV_FILE_OPTION="--env-file $ENV_FILE"

# 환경변수 검증 함수
validate_env_vars() {
    echo -e "${BLUE}🔍 환경변수 검증 중...${NC}"
    
    # 필수 환경변수 목록
    REQUIRED_VARS=(
        "SONARQUBE_PORT"
        "SONARQUBE_DB_NAME"
        "SONARQUBE_DB_USER"
        "SONARQUBE_DB_PASSWORD"
        "POSTGRES_SUPERUSER"
        "POSTGRES_SUPERUSER_PASSWORD"
    )
    
    # .env 파일에서 환경변수 로드
    source "$ENV_FILE"
    
    local missing_vars=()
    
    for var in "${REQUIRED_VARS[@]}"; do
        if [ -z "${!var}" ]; then
            missing_vars+=("$var")
        else
            echo -e "${GREEN}   ✅ $var=${!var}${NC}"
        fi
    done
    
    if [ ${#missing_vars[@]} -ne 0 ]; then
        echo -e "${RED}❌ 다음 환경변수가 설정되지 않았습니다:${NC}"
        for var in "${missing_vars[@]}"; do
            echo -e "${RED}   - $var${NC}"
        done
        echo -e "${YELLOW}   $ENV_FILE 파일을 확인하고 필요한 값을 설정하세요.${NC}"
        exit 1
    fi
    
    echo -e "${GREEN}✅ 모든 필수 환경변수가 설정되었습니다.${NC}"
}

# 볼륨 존재 확인 및 생성 함수
ensure_volumes() {
    echo -e "${BLUE}📦 필요한 볼륨 확인 중...${NC}"
    
    # 필요한 볼륨 목록
    VOLUMES=(
        "wealist-postgres-data"
        "wealist-sonarqube-data"
        "wealist-sonarqube-extensions"
        "wealist-sonarqube-logs"
    )
    
    for volume in "${VOLUMES[@]}"; do
        if ! docker volume inspect "$volume" >/dev/null 2>&1; then
            echo -e "${YELLOW}   볼륨 생성: $volume${NC}"
            docker volume create "$volume"
        else
            echo -e "${GREEN}   볼륨 존재: $volume${NC}"
        fi
    done
}

# PostgreSQL 준비 대기 함수
wait_for_postgres() {
    echo -e "${BLUE}⏳ PostgreSQL 시작 대기 중...${NC}"
    
    local max_attempts=12
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        if docker compose $ENV_FILE_OPTION $COMPOSE_FILES exec -T postgres pg_isready -U postgres >/dev/null 2>&1; then
            echo -e "${GREEN}✅ PostgreSQL이 준비되었습니다!${NC}"
            return 0
        fi
        
        echo -e "${YELLOW}   시도 $attempt/$max_attempts - PostgreSQL 시작 중...${NC}"
        sleep 5
        ((attempt++))
    done
    
    echo -e "${RED}❌ PostgreSQL 시작 시간 초과${NC}"
    return 1
}

# 헬스체크 함수
wait_for_sonarqube() {
    echo -e "${BLUE}⏳ SonarQube 시작 대기 중...${NC}"
    
    local max_attempts=30
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        # SonarQube API 상태 확인
        local status_response=$(curl -s http://localhost:9000/api/system/status 2>/dev/null)
        if echo "$status_response" | grep -q '"status":"UP"'; then
            echo -e "${GREEN}✅ SonarQube가 준비되었습니다!${NC}"
            
            # 추가 정보 출력
            local version=$(echo "$status_response" | grep -o '"version":"[^"]*"' | cut -d'"' -f4)
            if [ -n "$version" ]; then
                echo -e "${BLUE}   📋 SonarQube 버전: $version${NC}"
            fi
            
            return 0
        fi
        
        echo -e "${YELLOW}   시도 $attempt/$max_attempts - SonarQube 시작 중...${NC}"
        sleep 10
        ((attempt++))
    done
    
    echo -e "${RED}❌ SonarQube 시작 시간 초과. 로그를 확인하세요: ./docker/scripts/sonar.sh logs${NC}"
    return 1
}

# 커맨드 처리
COMMAND=${1:-up}

case $COMMAND in
    up)
        echo -e "${BLUE}🚀 SonarQube 독립 환경을 시작합니다...${NC}"
        
        # 환경변수 검증
        validate_env_vars
        
        # 볼륨 확인 및 생성
        ensure_volumes
        
        echo -e "${BLUE}🔨 컨테이너 시작 중...${NC}"
        docker compose $ENV_FILE_OPTION $COMPOSE_FILES up -d
        
        # PostgreSQL 준비 대기
        if wait_for_postgres && wait_for_sonarqube; then
            echo -e "${GREEN}✅ SonarQube 독립 환경이 시작되었습니다.${NC}"
            echo -e ""
            echo -e "${BLUE}📊 SonarQube 접속 정보:${NC}"
            echo "   - SonarQube:   http://localhost:9000"
            echo "   - 기본 로그인: admin / admin (첫 로그인 시 비밀번호 변경 필요)"
            echo "   - PostgreSQL:  localhost:5433 (포트 충돌 방지)"
            echo -e ""
            echo -e "${BLUE}💡 다음 단계:${NC}"
            echo "   1. 브라우저에서 http://localhost:9000 접속"
            echo "   2. admin/admin으로 로그인 후 비밀번호 변경"
            echo "   3. 프로젝트 생성 및 토큰 발급"
            echo "   4. 코드 분석 시작"
            echo -e ""
            echo -e "${BLUE}📚 코드 분석 예시:${NC}"
            echo "   # Go 서비스 분석 (예: user-service)"
            echo "   cd services/user-service"
            echo "   go test -coverprofile=coverage.out ./..."
            echo "   sonar-scanner -Dsonar.projectKey=wealist-user-service \\"
            echo "                 -Dsonar.host.url=http://localhost:9000 \\"
            echo "                 -Dsonar.token=YOUR_TOKEN"
            echo -e ""
            echo -e "${BLUE}🔧 유용한 명령어:${NC}"
            echo "   - 로그 확인:   make sonar-logs 또는 ./docker/scripts/sonar.sh logs"
            echo "   - 상태 확인:   make sonar-status 또는 ./docker/scripts/sonar.sh status"
            echo "   - 환경 중지:   make sonar-down 또는 ./docker/scripts/sonar.sh down"
            echo "   - 환경 재시작: make sonar-restart"
            echo -e ""
            echo -e "${YELLOW}⚠️  주의사항:${NC}"
            echo "   - 이 환경은 기존 전체 환경(make dev-up)과 독립적으로 동작합니다"
            echo "   - PostgreSQL은 포트 5433을 사용하여 충돌을 방지합니다"
            echo "   - 데이터는 기존 환경과 공유되므로 분석 결과가 유지됩니다"
        else
            echo -e "${RED}❌ SonarQube 시작에 실패했습니다.${NC}"
            exit 1
        fi
        ;;

    down)
        echo -e "${YELLOW}⏹️  SonarQube 독립 환경을 중지합니다...${NC}"
        docker compose $ENV_FILE_OPTION $COMPOSE_FILES down
        echo -e "${GREEN}✅ SonarQube 독립 환경이 중지되었습니다.${NC}"
        ;;

    restart)
        echo -e "${YELLOW}🔄 SonarQube 독립 환경을 재시작합니다...${NC}"
        docker compose $ENV_FILE_OPTION $COMPOSE_FILES restart
        
        if wait_for_sonarqube; then
            echo -e "${GREEN}✅ SonarQube 독립 환경이 재시작되었습니다.${NC}"
        fi
        ;;

    logs)
        SERVICE=${2:-}
        if [ -z "$SERVICE" ]; then
            echo -e "${BLUE}📋 모든 서비스 로그:${NC}"
            docker compose $ENV_FILE_OPTION $COMPOSE_FILES logs -f
        else
            echo -e "${BLUE}📋 $SERVICE 서비스 로그:${NC}"
            docker compose $ENV_FILE_OPTION $COMPOSE_FILES logs -f "$SERVICE"
        fi
        ;;

    status)
        echo -e "${BLUE}📊 SonarQube 독립 환경 상태:${NC}"
        echo ""
        
        # 컨테이너 상태
        echo -e "${BLUE}🐳 컨테이너 상태:${NC}"
        docker compose $ENV_FILE_OPTION $COMPOSE_FILES ps
        echo ""
        
        # SonarQube 헬스체크
        echo -e "${BLUE}🏥 SonarQube 헬스체크:${NC}"
        if curl -s http://localhost:9000/api/system/status >/dev/null 2>&1; then
            echo -e "   ${GREEN}✅ SonarQube: 정상 동작 중${NC}"
            echo "   📍 접속 URL: http://localhost:9000"
        else
            echo -e "   ${RED}❌ SonarQube: 응답 없음${NC}"
        fi
        
        # PostgreSQL 헬스체크
        echo -e "${BLUE}🗄️  PostgreSQL 헬스체크:${NC}"
        if docker compose $ENV_FILE_OPTION $COMPOSE_FILES exec -T postgres pg_isready >/dev/null 2>&1; then
            echo -e "   ${GREEN}✅ PostgreSQL: 정상 동작 중${NC}"
        else
            echo -e "   ${RED}❌ PostgreSQL: 응답 없음${NC}"
        fi
        ;;

    clean)
        echo -e "${RED}⚠️  SonarQube 독립 환경의 모든 데이터를 삭제합니다.${NC}"
        echo -e "${YELLOW}   이 작업은 SonarQube 분석 결과와 설정을 모두 제거합니다.${NC}"
        read -p "계속하시겠습니까? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            docker compose $ENV_FILE_OPTION $COMPOSE_FILES down -v --remove-orphans
            echo -e "${GREEN}✅ 정리가 완료되었습니다.${NC}"
        else
            echo -e "${YELLOW}취소되었습니다.${NC}"
        fi
        ;;

    *)
        echo -e "${RED}❌ 알 수 없는 명령어: $COMMAND${NC}"
        echo ""
        echo "사용 가능한 명령어:"
        echo "  up         - SonarQube 환경 시작"
        echo "  down       - SonarQube 환경 중지"
        echo "  restart    - SonarQube 환경 재시작"
        echo "  logs       - 로그 확인 (logs [service])"
        echo "  status     - 상태 확인"
        echo "  clean      - 모든 데이터 삭제 (볼륨 포함)"
        echo ""
        echo "예시:"
        echo "  ./docker/scripts/sonar.sh up"
        echo "  ./docker/scripts/sonar.sh logs sonarqube"
        echo "  ./docker/scripts/sonar.sh status"
        exit 1
        ;;
esac