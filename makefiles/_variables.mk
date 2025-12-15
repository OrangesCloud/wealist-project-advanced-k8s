# =============================================================================
# Common Variables
# =============================================================================

# Kind cluster configuration
KIND_CLUSTER ?= wealist
LOCAL_REGISTRY ?= localhost:5001
IMAGE_TAG ?= latest

# Environment configuration (used across all commands)
# Options: local-kind, dev, staging, prod
# DEPRECATED-SOON: local-ubuntu (will be replaced by staging)
ENV ?= local-kind

# Namespace mapping based on environment
ifeq ($(ENV),local-kind)
  K8S_NAMESPACE = wealist-kind-local
# DEPRECATED-SOON: local-ubuntu will be replaced by staging
else ifeq ($(ENV),local-ubuntu)
  K8S_NAMESPACE = wealist-dev
else ifeq ($(ENV),dev)
  K8S_NAMESPACE = wealist-dev
else ifeq ($(ENV),staging)
  K8S_NAMESPACE = wealist-staging
else ifeq ($(ENV),prod)
  K8S_NAMESPACE = wealist-prod
else
  K8S_NAMESPACE = wealist-kind-local
endif

# Helm values file paths
HELM_BASE_VALUES = ./helm/environments/base.yaml
HELM_ENV_VALUES = ./helm/environments/$(ENV).yaml
HELM_SECRETS_VALUES = ./helm/environments/$(ENV)-secrets.yaml

# Conditionally add secrets file if it exists
HELM_SECRETS_FLAG = $(shell test -f $(HELM_SECRETS_VALUES) && echo "-f $(HELM_SECRETS_VALUES)")

# Services list (all microservices)
# Backend services only (frontend is deployed via CDN/S3 in cloud environments)
BACKEND_SERVICES = auth-service user-service board-service chat-service noti-service storage-service video-service

# Frontend (only deployed in local/docker-compose environments)
FRONTEND_SERVICE = frontend

# All services (for local development with frontend)
SERVICES = $(BACKEND_SERVICES) $(FRONTEND_SERVICE)

# Services for K8s cloud deployment (dev, staging, prod - no frontend)
K8S_SERVICES = $(BACKEND_SERVICES)

# Services with project root build context (use shared package)
ROOT_CONTEXT_SERVICES = chat-service noti-service storage-service user-service video-service

# Services with local build context
LOCAL_CONTEXT_SERVICES = auth-service board-service frontend
