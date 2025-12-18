# =============================================================================
# Common Variables
# =============================================================================

# Kind cluster configuration
KIND_CLUSTER ?= wealist
LOCAL_REGISTRY ?= localhost:5001
IMAGE_TAG ?= latest

# Environment configuration (used across all commands)
# Options: local-kind, local-ubuntu, dev, staging, prod
ENV ?= local-kind

# Namespace mapping based on environment
ifeq ($(ENV),local-kind)
  K8S_NAMESPACE = wealist-kind-local
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

# Helm paths
HELM_CHARTS_DIR = ./main/helm/charts
HELM_ENVS_DIR = ./main/helm/environments

# Helm values file paths
HELM_BASE_VALUES = $(HELM_ENVS_DIR)/base.yaml
HELM_ENV_VALUES = $(HELM_ENVS_DIR)/$(ENV).yaml
HELM_SECRETS_VALUES = $(HELM_ENVS_DIR)/$(ENV)-secrets.yaml

# Conditionally add secrets file if it exists
HELM_SECRETS_FLAG = $(shell test -f $(HELM_SECRETS_VALUES) && echo "-f $(HELM_SECRETS_VALUES)")

# Services list (all microservices)
SERVICES = auth-service user-service board-service chat-service noti-service storage-service video-service frontend

# Services with project root build context (use shared package)
ROOT_CONTEXT_SERVICES = chat-service noti-service storage-service user-service video-service

# Services with local build context
LOCAL_CONTEXT_SERVICES = auth-service board-service frontend