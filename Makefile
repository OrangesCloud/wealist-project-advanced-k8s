# =============================================================================
# weAlist Project Makefile
# =============================================================================
# Self-documenting Makefile using ## comments
# Run 'make help' to see all available commands
#
# Structure:
#   makefiles/_variables.mk  - Common variables (ENV, K8S_NAMESPACE, etc.)
#   makefiles/docker.mk      - Docker Compose commands (dev-*, sonar-*)
#   makefiles/kind.mk        - Kind cluster commands (kind-*, local-*)
#   makefiles/services.mk    - Per-service commands (*-build, *-load, *-redeploy)
#   makefiles/helm.mk        - Helm chart commands (helm-*)
# =============================================================================

.DEFAULT_GOAL := help

# Include all sub-makefiles
include makefiles/_variables.mk
include makefiles/docker.mk
include makefiles/kind.mk
include makefiles/services.mk
include makefiles/helm.mk
include makefiles/branch-based.mk

##@ General

.PHONY: help

help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\n\033[1mUsage:\033[0m\n  make \033[36m<target>\033[0m [ENV=<env>]\n\n\033[1mEnvironments:\033[0m\n  local-kind (default), local-ubuntu, dev, staging, prod\n"} \
		/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } \
		/^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
	@echo ""
	@echo "\033[1mPer-Service Commands:\033[0m"
	@echo "  \033[36m<service>-build\033[0m           Build Docker image"
	@echo "  \033[36m<service>-load\033[0m            Build and push to registry"
	@echo "  \033[36m<service>-redeploy\033[0m        Rollout restart in k8s"
	@echo "  \033[36m<service>-all\033[0m             Build, push, and redeploy"
	@echo ""
	@echo "  Services: auth-service, board-service, chat-service, frontend,"
	@echo "            noti-service, storage-service, user-service, video-service"
	@echo ""
