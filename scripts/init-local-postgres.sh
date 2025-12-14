#!/bin/bash
# =============================================================================
# Initialize Local PostgreSQL for Wealist Services
# =============================================================================
# This script creates databases and users for all Wealist microservices
# on the local Ubuntu PostgreSQL installation.
#
# Prerequisites:
#   1. PostgreSQL installed: sudo apt install postgresql postgresql-contrib
#   2. PostgreSQL service running: sudo systemctl start postgresql
#   3. Run this script as root or with sudo
#
# Usage:
#   sudo ./scripts/init-local-postgres.sh
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

# Database configurations
# Format: db_name:user:password
DATABASES=(
    "wealist_user_service_db:user_service:user_service_password"
    "wealist_board_service_db:board_service:board_service_password"
    "wealist_chat_service_db:chat_service:chat_service_password"
    "wealist_noti_service_db:noti_service:noti_service_password"
    "wealist_storage_service_db:storage_service:storage_service_password"
    "wealist_video_service_db:video_service:video_service_password"
)

# Check if running as root or can use sudo
check_permissions() {
    if [ "$EUID" -ne 0 ]; then
        log_error "Please run this script with sudo or as root"
        exit 1
    fi
}

# Configure PostgreSQL to accept connections from Docker network
configure_pg_hba() {
    log_info "Configuring pg_hba.conf for Docker network access..."

    PG_HBA=$(sudo -u postgres psql -t -P format=unaligned -c "SHOW hba_file")

    # Check if Docker/Kind networks are already configured
    if grep -q "172.18.0.0/16" "$PG_HBA"; then
        log_info "Kind bridge network already configured in pg_hba.conf"
    else
        log_info "Adding Docker/Kind networks to pg_hba.conf"
        echo "# Allow connections from Docker/Kind networks" >> "$PG_HBA"
        echo "host    all    all    172.17.0.0/16    md5" >> "$PG_HBA"
        echo "host    all    all    172.18.0.0/16    md5" >> "$PG_HBA"
    fi
}

# Configure PostgreSQL to listen on all interfaces
configure_postgresql_conf() {
    log_info "Configuring postgresql.conf for network listening..."

    PG_CONF=$(sudo -u postgres psql -t -P format=unaligned -c "SHOW config_file")

    # Check current listen_addresses
    CURRENT_LISTEN=$(grep "^listen_addresses" "$PG_CONF" 2>/dev/null || echo "")

    if [[ "$CURRENT_LISTEN" == *"'*'"* ]]; then
        log_info "listen_addresses already set to '*'"
    else
        log_info "Setting listen_addresses to '*'"
        # Comment out existing and add new
        sed -i "s/^listen_addresses/#listen_addresses/" "$PG_CONF"
        echo "listen_addresses = '*'" >> "$PG_CONF"
    fi
}

# Create databases and users
create_databases() {
    log_info "Creating databases and users..."

    for db_config in "${DATABASES[@]}"; do
        IFS=':' read -r db_name db_user db_password <<< "$db_config"

        log_info "Processing: $db_name -> $db_user"

        # Create user if not exists
        sudo -u postgres psql -tc "SELECT 1 FROM pg_roles WHERE rolname='$db_user'" | grep -q 1 || {
            sudo -u postgres psql -c "CREATE USER $db_user WITH PASSWORD '$db_password';"
            log_info "Created user: $db_user"
        }

        # Create database if not exists
        sudo -u postgres psql -tc "SELECT 1 FROM pg_database WHERE datname='$db_name'" | grep -q 1 || {
            sudo -u postgres psql -c "CREATE DATABASE $db_name OWNER $db_user;"
            log_info "Created database: $db_name"
        }

        # Grant privileges
        sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE $db_name TO $db_user;"

        # Grant schema privileges
        sudo -u postgres psql -d "$db_name" -c "GRANT ALL ON SCHEMA public TO $db_user;"
        sudo -u postgres psql -d "$db_name" -c "ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO $db_user;"
        sudo -u postgres psql -d "$db_name" -c "ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO $db_user;"
    done
}

# Restart PostgreSQL to apply changes
restart_postgresql() {
    log_info "Restarting PostgreSQL..."
    systemctl restart postgresql

    # Wait for PostgreSQL to be ready
    sleep 3

    if systemctl is-active --quiet postgresql; then
        log_info "PostgreSQL restarted successfully"
    else
        log_error "Failed to restart PostgreSQL"
        exit 1
    fi
}

# Verify connection from Docker network
verify_connection() {
    log_info "Verifying PostgreSQL is accessible..."

    # Check if PostgreSQL is listening on all interfaces
    if netstat -tlnp 2>/dev/null | grep -q ":5432.*LISTEN" || ss -tlnp | grep -q ":5432.*LISTEN"; then
        log_info "PostgreSQL is listening on port 5432"
    else
        log_warn "PostgreSQL may not be listening on all interfaces"
    fi
}

# Print summary
print_summary() {
    echo ""
    echo "============================================="
    echo "PostgreSQL Initialization Complete!"
    echo "============================================="
    echo ""
    echo "Connection Info (for Kind cluster):"
    echo "  Host: 172.17.0.1 (Docker bridge gateway)"
    echo "  Port: 5432"
    echo "  Superuser: postgres / postgres"
    echo ""
    echo "Created Databases:"
    for db_config in "${DATABASES[@]}"; do
        IFS=':' read -r db_name db_user db_password <<< "$db_config"
        echo "  - $db_name ($db_user)"
    done
    echo ""
    echo "Next Steps:"
    echo "  1. Deploy with: make helm-install-all ENV=local-ubuntu"
    echo "  2. For initial tables: make helm-install-all-init ENV=local-ubuntu"
    echo ""
}

# Main execution
main() {
    log_info "Starting PostgreSQL initialization for Wealist..."
    echo ""

    check_permissions
    configure_pg_hba
    configure_postgresql_conf
    create_databases
    restart_postgresql
    verify_connection
    print_summary
}

main "$@"
