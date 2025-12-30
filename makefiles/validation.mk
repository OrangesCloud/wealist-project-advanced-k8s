# =============================================================================
# Validation Commands
# =============================================================================
# 배포 전 코드 검증을 위한 명령어들
# 실행 순서대로 검증: Terraform → Helm → Go Services
# =============================================================================

##@ Validation

.PHONY: validate-all validate-terraform validate-helm validate-go validate-helm-services validate-helm-infra

validate-all: ## 전체 검증 실행 (Terraform + Helm + Go)
	@echo "=============================================="
	@echo "         전체 검증 시작"
	@echo "=============================================="
	@$(MAKE) validate-terraform
	@$(MAKE) validate-helm
	@$(MAKE) validate-go
	@echo ""
	@echo "=============================================="
	@echo "         검증 완료!"
	@echo "=============================================="

# -----------------------------------------------------------------------------
# Terraform Validation
# -----------------------------------------------------------------------------
validate-terraform: ## Terraform 검증 (foundation, compute, argocd-apps)
	@echo ""
	@echo "=== Terraform Validation ==="
	@echo "--- foundation ---"
	@cd terraform/prod/foundation && terraform validate
	@echo "--- compute ---"
	@cd terraform/prod/compute && terraform validate
	@echo "--- argocd-apps ---"
	@cd terraform/prod/argocd-apps && terraform validate
	@echo "✅ Terraform: All layers valid"

# -----------------------------------------------------------------------------
# Helm Chart Validation
# -----------------------------------------------------------------------------
validate-helm: validate-helm-services validate-helm-infra ## Helm 차트 검증 (전체)
	@echo "✅ Helm: Validation complete"

validate-helm-services: ## Helm 서비스 차트 검증
	@echo ""
	@echo "=== Helm Services Validation ==="
	@helm lint k8s/helm/charts/auth-service
	@helm lint k8s/helm/charts/board-service
	@helm lint k8s/helm/charts/chat-service
	@helm lint k8s/helm/charts/user-service
	@helm lint k8s/helm/charts/noti-service
	@helm lint k8s/helm/charts/storage-service
	@helm lint k8s/helm/charts/video-service

validate-helm-infra: ## Helm 인프라 차트 검증
	@echo ""
	@echo "=== Helm Infrastructure Validation ==="
	@helm lint k8s/helm/charts/wealist-common || true
	@helm lint k8s/helm/charts/istio-config
	@helm lint k8s/helm/charts/db-init

# -----------------------------------------------------------------------------
# Go Service Validation
# -----------------------------------------------------------------------------
validate-go: ## Go 서비스 빌드 검증
	@echo ""
	@echo "=== Go Services Build Validation ==="
	@echo "--- user-service ---"
	@cd services/user-service && go build -o /dev/null ./cmd/api
	@echo "--- board-service ---"
	@cd services/board-service && go build -o /dev/null ./cmd/api
	@echo "--- chat-service ---"
	@cd services/chat-service && go build -o /dev/null ./cmd/api
	@echo "--- noti-service ---"
	@cd services/noti-service && go build -o /dev/null ./cmd/api
	@echo "--- storage-service ---"
	@cd services/storage-service && go build -o /dev/null ./cmd/api
	@echo "--- video-service ---"
	@cd services/video-service && go build -o /dev/null ./cmd/api
	@echo "✅ Go Services: All builds passed"

# -----------------------------------------------------------------------------
# Quick Validation (Go only, fastest)
# -----------------------------------------------------------------------------
validate-quick: ## 빠른 검증 (Go 빌드만)
	@echo "=== Quick Validation (Go only) ==="
	@cd services/user-service && go build -o /dev/null ./cmd/api
	@cd services/board-service && go build -o /dev/null ./cmd/api
	@cd services/chat-service && go build -o /dev/null ./cmd/api
	@cd services/noti-service && go build -o /dev/null ./cmd/api
	@cd services/storage-service && go build -o /dev/null ./cmd/api
	@cd services/video-service && go build -o /dev/null ./cmd/api
	@echo "✅ Quick validation passed"
