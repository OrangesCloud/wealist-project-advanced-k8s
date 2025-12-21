# =============================================================================
# Common Variables
# =============================================================================

# Kind cluster configuration
KIND_CLUSTER ?= wealist
LOCAL_REGISTRY ?= localhost:5001
IMAGE_TAG ?= latest

# External Database Configuration
# false (default): Deploy PostgreSQL/Redis as pods inside cluster
# true: Use host machine's PostgreSQL/Redis (requires local installation)
EXTERNAL_DB ?= false

# Environment configuration (used across all commands)
# Options: localhost, dev, staging, prod
# DEPRECATED-SOON: local-ubuntu (will be replaced by staging)
ENV ?= localhost

# Namespace, Domain, and Protocol mapping based on environment
ifeq ($(ENV),localhost)
  K8S_NAMESPACE = wealist-localhost
  DOMAIN = localhost:8080
  PROTOCOL = http
else ifeq ($(ENV),localhost)
  K8S_NAMESPACE = wealist-localhost
  DOMAIN = localhost:8080
  PROTOCOL = http
# DEPRECATED-SOON: local-ubuntu will be replaced by staging
else ifeq ($(ENV),local-ubuntu)
  K8S_NAMESPACE = wealist-dev
  DOMAIN = local.wealist.co.kr
  PROTOCOL = https
else ifeq ($(ENV),dev)
  K8S_NAMESPACE = wealist-dev
  DOMAIN = dev.wealist.co.kr
  PROTOCOL = https
else ifeq ($(ENV),staging)
  K8S_NAMESPACE = wealist-staging
  DOMAIN = staging.wealist.co.kr
  PROTOCOL = https
else ifeq ($(ENV),prod)
  K8S_NAMESPACE = wealist-prod
  DOMAIN = wealist.co.kr
  PROTOCOL = https
else
  K8S_NAMESPACE = wealist-localhost
  DOMAIN = localhost:8080
  PROTOCOL = http
endif

# Helm values file paths
HELM_BASE_VALUES = ./k8s/helm/environments/base.yaml
HELM_ENV_VALUES = ./k8s/helm/environments/$(ENV).yaml
HELM_SECRETS_VALUES = ./k8s/helm/environments/secrets.yaml

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
