#!/bin/bash
# =============================================================================
# Initialize Local PostgreSQL for Wealist Services
# =============================================================================
# This script creates databases and users for all Wealist microservices
# on the local PostgreSQL installation (macOS/Linux/WSL).
#
# Prerequisites:
#   macOS: brew install postgresql@14 && brew services start postgresql@14
#   Linux: sudo apt install postgresql postgresql-contrib
#   WSL:   sudo apt install postgresql postgresql-contrib
#
# Usage:
#   macOS: ./scripts/init-local-postgres.sh
#   Linux: sudo ./scripts/init-local-postgres.sh
#
# =============================================================================

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if running on macOS
is_macos() {
    [[ "$(uname)" == "Darwin" ]]
}

# Check if running in WSL
is_wsl() {
    if grep -qi microsoft /proc/version 2>/dev/null; then
        return 0
    fi
    return 1
}

# Database configurations
# Format: db_name:user:password
# Note: Using 'postgres' as password to match Helm values.yaml configuration

# Mode: dev (default) or prod
# - dev: 단일 wealist 유저 (Kind 개발환경, ESO와 동일)
# - prod: 개별 서비스 유저 (EKS 운영환경)
MODE="${1:-dev}"

# PROD mode: 개별 서비스별 DB/유저
DATABASES_PROD=(
    "wealist_user_service_db:user_service:postgres"
    "wealist_board_service_db:board_service:postgres"
    "wealist_chat_service_db:chat_service:postgres"
    "wealist_noti_service_db:noti_service:postgres"
    "wealist_storage_service_db:storage_service:postgres"
    "wealist_video_service_db:video_service:postgres"
)

# DEV mode: 단일 wealist 유저 (ESO에서 사용하는 credentials와 동일)
# 비밀번호는 AWS Secrets Manager의 wealist/dev/database/endpoint와 일치해야 함
#
# 방법 1: kubectl로 ESO 시크릿에서 자동 가져오기 (클러스터가 실행 중일 때)
#   DEV_DB_PASSWORD=$(kubectl get secret wealist-shared-secret -n wealist-dev \
#       -o jsonpath='{.data.DB_PASSWORD}' | base64 -d)
#
# 방법 2: 환경변수로 직접 지정
#   export DEV_DB_PASSWORD="your-aws-secret-password"
#
DEV_USER="wealist"
DEV_PASSWORD="${DEV_DB_PASSWORD:-wealist-dev-password}"
DEV_DATABASE="wealist"

# 모드에 따라 DATABASES 배열 설정 (main 함수에서 호출)
set_databases_for_mode() {
    # Try to auto-fetch from ESO if kubectl is available and default password is set
    if [ "$MODE" = "dev" ] && [ "$DEV_PASSWORD" = "wealist-dev-password" ]; then
        if command -v kubectl &> /dev/null; then
            ESO_PASS=$(kubectl get secret wealist-shared-secret -n wealist-dev \
                -o jsonpath='{.data.DB_PASSWORD}' 2>/dev/null | base64 -d 2>/dev/null || echo "")
            if [ -n "$ESO_PASS" ]; then
                log_info "Auto-fetched password from ESO secret"
                DEV_PASSWORD="$ESO_PASS"
            else
                log_warn "Could not fetch ESO password - using default"
            fi
        fi
    fi

    if [ "$MODE" = "dev" ]; then
        DATABASES=("${DEV_DATABASE}:${DEV_USER}:${DEV_PASSWORD}")
        log_info "Dev 모드: 단일 wealist 유저 생성"
    else
        DATABASES=("${DATABASES_PROD[@]}")
        log_info "Prod 모드: 개별 서비스 유저 생성"
    fi
}

# Run psql command (handles macOS vs Linux differences)
run_psql() {
    if is_macos; then
        # macOS: Run as current user
        psql -U postgres "$@" 2>/dev/null || psql "$@" 2>/dev/null
    else
        # Linux: Run as postgres user
        sudo -u postgres psql "$@"
    fi
}

# Check if running as root or can use sudo (Linux only)
check_permissions() {
    if is_macos; then
        log_info "macOS 감지 - sudo 불필요"
        return 0
    fi

    if [ "$EUID" -ne 0 ]; then
        log_error "Linux에서는 sudo로 실행하세요: sudo $0"
        exit 1
    fi
}

# Configure PostgreSQL to accept connections from Docker network
configure_pg_hba() {
    log_info "Configuring pg_hba.conf for Docker network access..."

    if is_macos; then
        # macOS: Find pg_hba.conf location
        PG_HBA=""
        for path in \
            "/opt/homebrew/var/postgres/pg_hba.conf" \
            "/opt/homebrew/var/postgresql@14/pg_hba.conf" \
            "/opt/homebrew/var/postgresql@15/pg_hba.conf" \
            "/usr/local/var/postgres/pg_hba.conf" \
            "/usr/local/var/postgresql@14/pg_hba.conf" \
            "$HOME/.postgres/pg_hba.conf"; do
            if [ -f "$path" ]; then
                PG_HBA="$path"
                break
            fi
        done

        if [ -z "$PG_HBA" ]; then
            log_warn "pg_hba.conf를 찾을 수 없습니다. macOS에서는 기본적으로 로컬 연결만 허용됩니다."
            log_info "Docker Desktop의 host.docker.internal을 통해 연결합니다."
            return 0
        fi
    else
        # Linux: Get path from PostgreSQL
        PG_HBA=$(sudo -u postgres psql -t -P format=unaligned -c "SHOW hba_file")
    fi

    log_info "pg_hba.conf: $PG_HBA"

    # Check if Docker/Kind networks are already configured
    if grep -q "172.18.0.0/16" "$PG_HBA" 2>/dev/null; then
        log_info "Kind bridge network already configured in pg_hba.conf"
    else
        log_info "Adding Docker/Kind networks to pg_hba.conf"
        if is_macos; then
            echo "# Allow connections from Docker/Kind networks" >> "$PG_HBA"
            echo "host    all    all    172.17.0.0/16    trust" >> "$PG_HBA"
            echo "host    all    all    172.18.0.0/16    trust" >> "$PG_HBA"
        else
            echo "# Allow connections from Docker/Kind networks" >> "$PG_HBA"
            echo "host    all    all    172.17.0.0/16    md5" >> "$PG_HBA"
            echo "host    all    all    172.18.0.0/16    md5" >> "$PG_HBA"
        fi
    fi

    # WSL-specific: Add WSL network subnet
    if is_wsl; then
        WSL_IP=$(hostname -I | awk '{print $1}')
        # Extract /16 subnet (e.g., 172.29.0.0/16)
        WSL_SUBNET=$(echo "$WSL_IP" | sed 's/\.[0-9]*\.[0-9]*$/.0.0\/16/')
        log_info "WSL 환경 감지 - WSL 서브넷 추가: $WSL_SUBNET"

        if ! grep -q "$WSL_SUBNET" "$PG_HBA"; then
            echo "# Allow connections from WSL network (for Kind pods)" >> "$PG_HBA"
            echo "host    all    all    $WSL_SUBNET    md5" >> "$PG_HBA"
            log_info "Added WSL subnet to pg_hba.conf"
        else
            log_info "WSL subnet already configured in pg_hba.conf"
        fi
    fi
}

# Configure PostgreSQL to listen on all interfaces
configure_postgresql_conf() {
    log_info "Configuring postgresql.conf for network listening..."

    if is_macos; then
        # macOS: Find postgresql.conf location
        PG_CONF=""
        for path in \
            "/opt/homebrew/var/postgres/postgresql.conf" \
            "/opt/homebrew/var/postgresql@14/postgresql.conf" \
            "/opt/homebrew/var/postgresql@15/postgresql.conf" \
            "/usr/local/var/postgres/postgresql.conf" \
            "/usr/local/var/postgresql@14/postgresql.conf" \
            "$HOME/.postgres/postgresql.conf"; do
            if [ -f "$path" ]; then
                PG_CONF="$path"
                break
            fi
        done

        if [ -z "$PG_CONF" ]; then
            log_warn "postgresql.conf를 찾을 수 없습니다."
            return 0
        fi
    else
        # Linux: Get path from PostgreSQL
        PG_CONF=$(sudo -u postgres psql -t -P format=unaligned -c "SHOW config_file")
    fi

    log_info "postgresql.conf: $PG_CONF"

    # Check current listen_addresses
    CURRENT_LISTEN=$(grep "^listen_addresses" "$PG_CONF" 2>/dev/null || echo "")

    if [[ "$CURRENT_LISTEN" == *"'*'"* ]]; then
        log_info "listen_addresses already set to '*'"
    else
        log_info "Setting listen_addresses to '*'"
        # Comment out existing and add new
        if is_macos; then
            sed -i '' "s/^listen_addresses/#listen_addresses/" "$PG_CONF" 2>/dev/null || true
        else
            sed -i "s/^listen_addresses/#listen_addresses/" "$PG_CONF"
        fi
        echo "listen_addresses = '*'" >> "$PG_CONF"
    fi
}

# Set postgres superuser password (for Helm charts that use postgres user)
set_postgres_password() {
    log_info "Setting postgres superuser password..."

    if is_macos; then
        # macOS: Check if postgres role exists, create if not
        if ! psql -U postgres -c "SELECT 1" >/dev/null 2>&1; then
            # Try with current user
            if psql -c "SELECT 1 FROM pg_roles WHERE rolname='postgres'" 2>/dev/null | grep -q 1; then
                psql -c "ALTER USER postgres PASSWORD 'postgres';" >/dev/null 2>&1
            else
                log_info "Creating postgres superuser..."
                createuser -s postgres 2>/dev/null || true
                psql -c "ALTER USER postgres PASSWORD 'postgres';" >/dev/null 2>&1 || \
                psql -U "$(whoami)" -c "ALTER USER postgres PASSWORD 'postgres';" >/dev/null 2>&1 || true
            fi
        else
            psql -U postgres -c "ALTER USER postgres PASSWORD 'postgres';" >/dev/null 2>&1
        fi
    else
        sudo -u postgres psql -c "ALTER USER postgres PASSWORD 'postgres';" >/dev/null 2>&1
    fi

    log_info "postgres user password set to 'postgres'"
}

# Create databases and users
create_databases() {
    log_info "Creating databases and users..."

    for db_config in "${DATABASES[@]}"; do
        IFS=':' read -r db_name db_user db_password <<< "$db_config"

        log_info "Processing: $db_name -> $db_user"

        # Create user if not exists
        if run_psql -tc "SELECT 1 FROM pg_roles WHERE rolname='$db_user'" | grep -q 1; then
            log_info "User $db_user exists, updating password..."
        else
            run_psql -c "CREATE USER $db_user WITH PASSWORD '$db_password';"
            log_info "Created user: $db_user"
        fi

        # Always update password (in case it changed)
        run_psql -c "ALTER USER $db_user WITH PASSWORD '$db_password';" >/dev/null 2>&1
        log_info "Password updated for: $db_user"

        # Create database if not exists
        run_psql -tc "SELECT 1 FROM pg_database WHERE datname='$db_name'" | grep -q 1 || {
            run_psql -c "CREATE DATABASE $db_name OWNER $db_user;"
            log_info "Created database: $db_name"
        }

        # Grant privileges
        run_psql -c "GRANT ALL PRIVILEGES ON DATABASE $db_name TO $db_user;"

        # Grant schema privileges (PostgreSQL 15+ requires explicit schema ownership)
        run_psql -d "$db_name" -c "ALTER SCHEMA public OWNER TO $db_user;" 2>/dev/null || true
        run_psql -d "$db_name" -c "GRANT ALL ON SCHEMA public TO $db_user;"
        run_psql -d "$db_name" -c "GRANT CREATE ON SCHEMA public TO $db_user;"
        run_psql -d "$db_name" -c "GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO $db_user;"
        run_psql -d "$db_name" -c "GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO $db_user;"
        run_psql -d "$db_name" -c "ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO $db_user;"
        run_psql -d "$db_name" -c "ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO $db_user;"
        log_info "Granted full privileges on $db_name to $db_user"
    done
}

# Restart PostgreSQL to apply changes
restart_postgresql() {
    log_info "Restarting PostgreSQL..."

    if is_macos; then
        log_info "macOS 감지 - brew services 사용"

        # Try different PostgreSQL versions
        if brew services restart postgresql@14 2>/dev/null; then
            log_info "PostgreSQL@14 restarted via brew services"
        elif brew services restart postgresql@15 2>/dev/null; then
            log_info "PostgreSQL@15 restarted via brew services"
        elif brew services restart postgresql 2>/dev/null; then
            log_info "PostgreSQL restarted via brew services"
        else
            log_warn "brew services restart 실패 - PostgreSQL이 실행 중인지 확인하세요"
            log_info "수동 재시작: brew services restart postgresql@14"
        fi

        sleep 2

    elif is_wsl; then
        log_info "WSL 환경 감지 - systemd 대신 pg_ctlcluster 사용"

        # Get PostgreSQL version
        PG_VERSION=$(ls /usr/lib/postgresql/ 2>/dev/null | sort -rn | head -1)
        log_info "PostgreSQL version: $PG_VERSION"

        # Try pg_ctlcluster first (Debian/Ubuntu standard)
        if pg_ctlcluster "$PG_VERSION" main restart 2>/dev/null; then
            log_info "PostgreSQL restarted via pg_ctlcluster"
        else
            # Fallback to direct pg_ctl
            PG_DATA="/var/lib/postgresql/$PG_VERSION/main"
            log_info "Fallback to pg_ctl with data dir: $PG_DATA"
            sudo -u postgres /usr/lib/postgresql/"$PG_VERSION"/bin/pg_ctl stop -D "$PG_DATA" -m fast 2>/dev/null || true
            sleep 2
            sudo -u postgres /usr/lib/postgresql/"$PG_VERSION"/bin/pg_ctl start -D "$PG_DATA" -l /var/log/postgresql/postgresql.log
        fi

        # Wait for PostgreSQL to be ready
        sleep 3

        # Verify using pg_isready
        if sudo -u postgres pg_isready -q; then
            log_info "PostgreSQL started successfully (WSL)"
        else
            log_error "Failed to start PostgreSQL"
            exit 1
        fi
    else
        # Standard Linux with systemd
        systemctl restart postgresql

        # Wait for PostgreSQL to be ready
        sleep 3

        if systemctl is-active --quiet postgresql; then
            log_info "PostgreSQL restarted successfully"
        else
            log_error "Failed to restart PostgreSQL"
            exit 1
        fi
    fi
}

# Verify connection from Docker network
verify_connection() {
    log_info "Verifying PostgreSQL is accessible..."

    # Check if PostgreSQL is listening on all interfaces
    if is_macos; then
        if lsof -i :5432 2>/dev/null | grep -q LISTEN; then
            log_info "PostgreSQL is listening on port 5432"
        else
            log_warn "PostgreSQL may not be listening - check 'brew services list'"
        fi
    else
        if netstat -tlnp 2>/dev/null | grep -q ":5432.*LISTEN" || ss -tlnp | grep -q ":5432.*LISTEN"; then
            log_info "PostgreSQL is listening on port 5432"
        else
            log_warn "PostgreSQL may not be listening on all interfaces"
        fi
    fi
}

# Print summary
print_summary() {
    echo ""
    echo "============================================="
    echo "PostgreSQL Initialization Complete!"
    echo "============================================="
    echo ""

    # Get host IP for Kind cluster
    if is_macos; then
        echo "Connection Info (for Kind cluster - macOS):"
        echo "  Host: host.docker.internal (Docker Desktop 자동 제공)"
        echo "  Port: 5432"
    elif is_wsl; then
        HOST_IP=$(hostname -I | awk '{print $1}')
        echo "Connection Info (for Kind cluster - WSL):"
        echo "  Host: $HOST_IP (WSL IP)"
        echo "  Note: WSL IP may change after reboot"
        echo "  Port: 5432"
    else
        echo "Connection Info (for Kind cluster):"
        echo "  Host: 172.18.0.1 (Docker bridge gateway)"
        echo "  Port: 5432"
    fi
    echo "  Superuser: postgres / postgres"
    echo ""
    echo "Created Databases:"
    for db_config in "${DATABASES[@]}"; do
        IFS=':' read -r db_name db_user db_password <<< "$db_config"
        echo "  - $db_name ($db_user)"
    done
    echo ""
    if is_macos; then
        echo "macOS Note:"
        echo "  PostgreSQL 재시작: brew services restart postgresql@14"
        echo ""
    elif is_wsl; then
        echo "WSL Note:"
        echo "  PostgreSQL를 수동으로 시작하려면:"
        PG_VERSION=$(ls /usr/lib/postgresql/ 2>/dev/null | sort -rn | head -1)
        echo "    sudo pg_ctlcluster $PG_VERSION main start"
        echo ""
    fi
    echo "Next Steps:"
    if [ "$MODE" = "staging" ]; then
        echo "  1. Deploy with: make kind-staging-setup"
        echo "  2. ArgoCD가 자동으로 서비스 배포"
        echo ""
        echo "Note: AWS Secrets Manager의 비밀번호와 일치해야 합니다!"
        echo "  STAGING_DB_PASSWORD 환경변수로 비밀번호 지정 가능"
    else
        echo "  1. Deploy with: make helm-install-all ENV=dev"
        echo "  2. For initial tables: make helm-install-all-init ENV=dev"
    fi
    echo ""
}

# Main execution
main() {
    log_info "Starting PostgreSQL initialization for Wealist..."
    log_info "Mode: $MODE"
    echo ""

    check_permissions
    set_databases_for_mode
    configure_pg_hba
    configure_postgresql_conf
    set_postgres_password
    create_databases
    restart_postgresql
    verify_connection
    print_summary
}

main "$@"
