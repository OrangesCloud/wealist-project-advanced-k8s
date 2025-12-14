#!/bin/bash
set -e

CLUSTER_NAME="wealist"
PROJECT_DIR="/home/resshome/tech-up/advanced-project/wealist-project-advanced-k8s"
KIND_CONFIG="$PROJECT_DIR/docker/scripts/dev/kind-config.yaml"
LOG_FILE="/tmp/kind-recovery.log"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a $LOG_FILE
}

# Docker 대기
log "Waiting for Docker..."
while ! docker info >/dev/null 2>&1; do
    sleep 2
done

# Kind 클러스터 확인
if kind get clusters 2>/dev/null | grep -q "^${CLUSTER_NAME}$"; then
    log "Cluster exists, restarting containers..."
    docker restart ${CLUSTER_NAME}-control-plane ${CLUSTER_NAME}-worker ${CLUSTER_NAME}-worker2 kind-registry 2>/dev/null || true
    sleep 30
else
    log "Cluster not found, creating..."
    kind create cluster --name $CLUSTER_NAME --config $KIND_CONFIG
fi

# kubeconfig 갱신
log "Exporting kubeconfig..."
kind export kubeconfig --name $CLUSTER_NAME

# API server 대기
log "Waiting for API server..."
until kubectl get nodes >/dev/null 2>&1; do
    sleep 5
done

log "Cluster ready!"
kubectl get nodes
