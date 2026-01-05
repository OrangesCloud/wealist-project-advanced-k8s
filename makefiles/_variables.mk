# =============================================================================
# Common Variables
# =============================================================================

# Kind cluster configuration
KIND_CLUSTER ?= wealist
LOCAL_REGISTRY ?= localhost:5001
IMAGE_TAG ?= latest

# External Database Configuration
# false: Deploy PostgreSQL/Redis as pods inside cluster (localhost)
# true: Use external PostgreSQL/Redis (dev, staging, prod)
# 환경별로 자동 설정됨 (아래 참조)

# Environment configuration (used across all commands)
# Options: localhost, dev, staging, prod
# DEPRECATED-SOON: local-ubuntu (will be replaced by staging)
ENV ?= localhost

# Namespace, Domain, and Protocol mapping based on environment
ifeq ($(ENV),localhost)
  K8S_NAMESPACE = wealist-localhost
  DOMAIN = localhost:8080
  PROTOCOL = http
  # localhost: 내부 DB Pod 사용
  EXTERNAL_DB ?= false
else ifeq ($(ENV),localhost)
  K8S_NAMESPACE = wealist-localhost
  DOMAIN = localhost:8080
  PROTOCOL = http
  EXTERNAL_DB ?= false
# DEPRECATED-SOON: local-ubuntu will be replaced by staging
else ifeq ($(ENV),local-ubuntu)
  K8S_NAMESPACE = wealist-dev
  DOMAIN = local.wealist.co.kr
  PROTOCOL = https
  EXTERNAL_DB ?= true
else ifeq ($(ENV),dev)
  K8S_NAMESPACE = wealist-dev
  DOMAIN = dev.wealist.co.kr
  PROTOCOL = https
  # dev: 호스트 PC의 외부 DB 사용
  EXTERNAL_DB ?= true
else ifeq ($(ENV),staging)
  K8S_NAMESPACE = wealist-staging
  DOMAIN = dev.wealist.co.kr
  PROTOCOL = https
  # staging/prod: 외부 DB (RDS 등) 사용
  EXTERNAL_DB ?= true
else ifeq ($(ENV),prod)
  K8S_NAMESPACE = wealist-prod
  DOMAIN = wealist.co.kr
  PROTOCOL = https
  EXTERNAL_DB ?= true
else
  K8S_NAMESPACE = wealist-localhost
  DOMAIN = localhost:8080
  PROTOCOL = http
  EXTERNAL_DB ?= false
endif

# Helm values file paths
HELM_BASE_VALUES = ./k8s/helm/environments/base.yaml
HELM_ENV_VALUES = ./k8s/helm/environments/$(ENV).yaml
HELM_SECRETS_VALUES = ./k8s/helm/environments/secrets.yaml

# Conditionally add secrets file if it exists
HELM_SECRETS_FLAG = $(shell test -f $(HELM_SECRETS_VALUES) && echo "-f $(HELM_SECRETS_VALUES)")

# Services list (all microservices)
# Backend services only (frontend is deployed via CDN/S3 in cloud environments)
BACKEND_SERVICES = auth-service user-service board-service chat-service noti-service storage-service

# Frontend (only deployed in local/docker-compose environments)
FRONTEND_SERVICE = frontend

# All services (for local development with frontend)
SERVICES = $(BACKEND_SERVICES) $(FRONTEND_SERVICE)

# Services for K8s cloud deployment (dev, staging, prod - no frontend)
K8S_SERVICES = $(BACKEND_SERVICES)

# Services with project root build context (use shared package)
ROOT_CONTEXT_SERVICES = chat-service noti-service storage-service user-service

# Services with local build context
LOCAL_CONTEXT_SERVICES = auth-service board-service frontend
