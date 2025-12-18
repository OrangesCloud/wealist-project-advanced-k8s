# =============================================================================
# Helm Chart Commands
# =============================================================================

##@ Helm Charts (Recommended)

.PHONY: helm-deps-build helm-lint helm-validate
.PHONY: helm-install-cert-manager helm-install-infra helm-install-services
.PHONY: helm-install-all helm-install-all-init helm-upgrade-all helm-uninstall-all
.PHONY: helm-setup-route53-secret
.PHONY: helm-local-kind helm-local-ubuntu helm-dev helm-staging helm-prod

helm-deps-build: ## Build all Helm dependencies
	@echo "Building all Helm dependencies..."
	@helm dependency update $(HELM_CHARTS_DIR)/wealist-common 2>/dev/null || true
	@for chart in $(SERVICES); do \
		echo "Updating $$chart dependencies..."; \
		helm dependency update $(HELM_CHARTS_DIR)/$$chart; \
	done
	@helm dependency update $(HELM_CHARTS_DIR)/wealist-infrastructure
	@helm dependency update $(HELM_CHARTS_DIR)/cert-manager-config 2>/dev/null || true
	@echo "All dependencies built!"

helm-lint: ## Lint all Helm charts
	@echo "Linting all Helm charts..."
	@helm lint $(HELM_CHARTS_DIR)/wealist-common
	@helm lint $(HELM_CHARTS_DIR)/wealist-infrastructure
	@helm lint $(HELM_CHARTS_DIR)/cert-manager-config 2>/dev/null || echo "cert-manager-config: run 'helm dependency update' first"
	@for service in $(SERVICES); do \
		echo "Linting $$service..."; \
		helm lint $(HELM_CHARTS_DIR)/$$service; \
	done
	@echo "All charts linted successfully!"

helm-validate: ## Run comprehensive Helm validation
	@echo "Running comprehensive Helm validation..."
	@./main/helm/scripts/validate-all-charts.sh
	@echo ""
	@echo "Running ArgoCD Applications validation..."
	@./main/argocd/scripts/validate-applications.sh

##@ Helm Installation

helm-setup-route53-secret: ## Setup Route53 credentials secret for cert-manager
	@echo "Setting up Route53 credentials secret..."
	@kubectl create namespace cert-manager 2>/dev/null || true
	@if [ -z "$$AWS_SECRET_ACCESS_KEY" ]; then \
		echo "Error: AWS_SECRET_ACCESS_KEY environment variable not set"; \
		echo "Usage: AWS_SECRET_ACCESS_KEY=xxx make helm-setup-route53-secret"; \
		exit 1; \
	fi
	@kubectl create secret generic route53-credentials \
		--namespace cert-manager \
		--from-literal=secret-access-key=$$AWS_SECRET_ACCESS_KEY \
		--dry-run=client -o yaml | kubectl apply -f -
	@echo "Route53 credentials secret created/updated!"

helm-install-cert-manager: ## Install cert-manager (if enabled in env)
	@echo "Checking cert-manager configuration (ENV=$(ENV))..."
	@if grep -q "certManager:" "$(HELM_ENV_VALUES)" 2>/dev/null && \
	   grep -A1 "certManager:" "$(HELM_ENV_VALUES)" | grep -q "enabled: true"; then \
		echo "Installing cert-manager-config..."; \
		cd $(HELM_CHARTS_DIR)/cert-manager-config && helm dependency update && cd -; \
		helm upgrade --install cert-manager-config $(HELM_CHARTS_DIR)/cert-manager-config \
			-f $(HELM_BASE_VALUES) \
			-f $(HELM_ENV_VALUES) $(HELM_SECRETS_FLAG) \
			-n cert-manager --create-namespace --wait --timeout 5m; \
		echo "cert-manager installed!"; \
		echo "Waiting for cert-manager webhook to be ready..."; \
		sleep 10; \
	else \
		echo "Skipping cert-manager (disabled for $(ENV))"; \
	fi

helm-install-infra: ## Install infrastructure chart
	@echo "Installing infrastructure (ENV=$(ENV), NS=$(K8S_NAMESPACE))..."
	helm install wealist-infrastructure $(HELM_CHARTS_DIR)/wealist-infrastructure \
		-f $(HELM_BASE_VALUES) \
		-f $(HELM_ENV_VALUES) $(HELM_SECRETS_FLAG) \
		-n $(K8S_NAMESPACE) --create-namespace
	@echo "Infrastructure installed!"

helm-install-services: ## Install all service charts
	@echo "Installing services (ENV=$(ENV), NS=$(K8S_NAMESPACE))..."
	@for service in $(SERVICES); do \
		echo "Installing $$service..."; \
		helm install $$service $(HELM_CHARTS_DIR)/$$service \
			-f $(HELM_BASE_VALUES) \
			-f $(HELM_ENV_VALUES) $(HELM_SECRETS_FLAG) \
			-n $(K8S_NAMESPACE); \
	done
	@echo "All services installed!"
	@echo ""
	@echo "Next: make status"

helm-install-all: helm-deps-build helm-install-cert-manager helm-install-infra ## Install all charts
	@sleep 5
	@$(MAKE) helm-install-services ENV=$(ENV)

helm-install-all-init: helm-deps-build helm-install-cert-manager helm-install-infra ## Install all with DB migration (first time)
	@sleep 5
	@echo "Installing services with DB migration enabled (initial setup)..."
	@for service in $(SERVICES); do \
		echo "Installing $$service with DB migration..."; \
		helm install $$service $(HELM_CHARTS_DIR)/$$service \
			-f $(HELM_BASE_VALUES) \
			-f $(HELM_ENV_VALUES) $(HELM_SECRETS_FLAG) \
			--set shared.config.DB_AUTO_MIGRATE=true \
			-n $(K8S_NAMESPACE); \
	done
	@echo "Initial setup complete! Future deploys will skip DB migration."

##@ Helm Upgrade/Uninstall

helm-upgrade-all: helm-deps-build ## Upgrade all charts
	@echo "Upgrading all charts (ENV=$(ENV), NS=$(K8S_NAMESPACE))..."
	@helm upgrade wealist-infrastructure $(HELM_CHARTS_DIR)/wealist-infrastructure \
		-f $(HELM_BASE_VALUES) \
		-f $(HELM_ENV_VALUES) $(HELM_SECRETS_FLAG) \
		-n $(K8S_NAMESPACE)
	@for service in $(SERVICES); do \
		echo "Upgrading $$service..."; \
		helm upgrade $$service $(HELM_CHARTS_DIR)/$$service \
			-f $(HELM_BASE_VALUES) \
			-f $(HELM_ENV_VALUES) $(HELM_SECRETS_FLAG) \
			-n $(K8S_NAMESPACE); \
	done
	@echo "All charts upgraded!"

helm-uninstall-all: ## Uninstall all charts
	@echo "Uninstalling all charts (ENV=$(ENV), NS=$(K8S_NAMESPACE))..."
	@for service in $(SERVICES); do \
		echo "Uninstalling $$service..."; \
		helm uninstall $$service -n $(K8S_NAMESPACE) 2>/dev/null || true; \
	done
	@helm uninstall wealist-infrastructure -n $(K8S_NAMESPACE) 2>/dev/null || true
	@echo "Checking if cert-manager should be uninstalled..."
	@if helm list -n cert-manager 2>/dev/null | grep -q cert-manager-config; then \
		echo "Uninstalling cert-manager-config..."; \
		helm uninstall cert-manager-config -n cert-manager 2>/dev/null || true; \
	fi
	@echo "All charts uninstalled!"

##@ Quick Environment Switches

helm-local-kind: ## Deploy to local Kind cluster
	@$(MAKE) helm-install-all ENV=local-kind

helm-local-ubuntu: ## Deploy to local Ubuntu
	@$(MAKE) helm-install-all ENV=local-ubuntu

helm-dev: ## Deploy to dev server
	@$(MAKE) helm-install-all ENV=dev

helm-staging: ## Deploy to staging
	@$(MAKE) helm-install-all ENV=staging

helm-prod: ## Deploy to production
	@$(MAKE) helm-install-all ENV=prod