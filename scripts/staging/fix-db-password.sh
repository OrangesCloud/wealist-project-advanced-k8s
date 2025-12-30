#!/bin/bash
# =============================================================================
# Fix Database Password Mismatch for Staging Environment
# =============================================================================
# This script syncs the local PostgreSQL wealist user password with the
# password stored in AWS Secrets Manager (via ESO).
#
# Problem: Services fail with "password authentication failed for user wealist"
# Cause: Local PostgreSQL password doesn't match ESO secret from AWS
#
# Usage (on the mini-PC/staging host):
#   sudo ./scripts/staging/fix-db-password.sh
#
# =============================================================================

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_step() { echo -e "${CYAN}[STEP]${NC} $1"; }

NAMESPACE="wealist-staging"
SECRET_NAME="wealist-shared-secret"

echo ""
echo "============================================="
echo "  Fix Staging DB Password Mismatch"
echo "============================================="
echo ""

# Step 1: Get password from ESO secret
log_step "1. Getting DB password from ESO secret..."

ESO_PASSWORD=$(kubectl get secret "$SECRET_NAME" -n "$NAMESPACE" -o jsonpath='{.data.DB_PASSWORD}' 2>/dev/null | base64 -d)

if [ -z "$ESO_PASSWORD" ]; then
    log_error "Failed to get password from secret $SECRET_NAME"
    echo ""
    echo "Make sure:"
    echo "  1. kubectl is configured correctly"
    echo "  2. ESO has synced the secret from AWS"
    echo ""
    echo "Check with: kubectl get secret $SECRET_NAME -n $NAMESPACE"
    exit 1
fi

log_info "Got password from ESO secret (length: ${#ESO_PASSWORD} chars)"

# Step 2: Update local PostgreSQL password
log_step "2. Updating local PostgreSQL wealist user password..."

sudo -u postgres psql -c "ALTER USER wealist WITH PASSWORD '$ESO_PASSWORD';" 2>/dev/null

if [ $? -eq 0 ]; then
    log_info "PostgreSQL wealist user password updated successfully"
else
    log_error "Failed to update PostgreSQL password"
    exit 1
fi

# Step 3: Verify connection
log_step "3. Verifying database connection..."

# Test with psql from pod
kubectl run -it --rm pg-test-verify --image=postgres:15 -n "$NAMESPACE" --restart=Never -- \
    psql "postgresql://wealist:${ESO_PASSWORD}@172.17.0.1:5432/wealist" -c "SELECT 1 as connection_test;" 2>/dev/null

if [ $? -eq 0 ]; then
    log_info "Database connection verified!"
else
    log_warn "Connection test failed - check pg_hba.conf settings"
fi

# Step 4: Restart services
log_step "4. Restarting services to pick up correct password..."

kubectl rollout restart deployment \
    board-service \
    chat-service \
    noti-service \
    video-service \
    user-service \
    storage-service \
    -n "$NAMESPACE" 2>/dev/null || log_warn "Some services may not exist yet"

log_info "Services restarted. Check status with:"
echo ""
echo "  kubectl get pods -n $NAMESPACE"
echo ""

# Step 5: Wait and check status
log_step "5. Waiting for pods to restart..."
sleep 10

echo ""
echo "============================================="
echo "  Current Pod Status"
echo "============================================="
kubectl get pods -n "$NAMESPACE" -l "app.kubernetes.io/component=service" 2>/dev/null || \
    kubectl get pods -n "$NAMESPACE" | grep -E "service|Service" || \
    kubectl get pods -n "$NAMESPACE"

echo ""
log_info "Done! If services are still crashing, check logs with:"
echo "  kubectl logs -n $NAMESPACE deployment/<service-name> --tail=50"
echo ""
