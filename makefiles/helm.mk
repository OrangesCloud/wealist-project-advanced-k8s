# =============================================================================
# Helm Chart Commands
# =============================================================================

##@ Helm Charts (Recommended)

.PHONY: helm-deps-build helm-lint helm-validate
.PHONY: helm-install-cert-manager helm-install-infra helm-install-services
.PHONY: helm-install-all helm-install-all-init helm-upgrade-all helm-uninstall-all
.PHONY: helm-setup-route53-secret
.PHONY: helm-local-kind helm-local-ubuntu helm-dev helm-staging helm-prod

# Determine which services to deploy based on environment
# Frontend is only deployed in local environments (docker-compose, localhost)
# Cloud environments (dev, staging, prod) use CDN/S3/Route53 for frontend
ifeq ($(ENV),local-kind)
  HELM_SERVICES = $(SERVICES)
# DEPRECATED-SOON: local-ubuntu will be replaced by staging
else ifeq ($(ENV),local-ubuntu)
  HELM_SERVICES = $(BACKEND_SERVICES)
else
  HELM_SERVICES = $(BACKEND_SERVICES)
endif

helm-deps-build: ## Build all Helm dependencies
	@echo "Building all Helm dependencies..."
	@helm dependency update ./helm/charts/wealist-common 2>/dev/null || true
	@for chart in $(HELM_SERVICES); do \
		echo "Updating $$chart dependencies..."; \
		helm dependency update ./helm/charts/$$chart; \
	done
	@helm dependency update ./helm/charts/wealist-infrastructure
	@helm dependency update ./helm/charts/cert-manager-config 2>/dev/null || true
	@echo "All dependencies built!"

helm-lint: ## Lint all Helm charts
	@echo "Linting all Helm charts..."
	@helm lint ./helm/charts/wealist-common
	@helm lint ./helm/charts/wealist-infrastructure
	@helm lint ./helm/charts/istio-config
	@helm lint ./helm/charts/cert-manager-config 2>/dev/null || echo "cert-manager-config: run 'helm dependency update' first"
	@for service in $(HELM_SERVICES); do \
		echo "Linting $$service..."; \
		helm lint ./helm/charts/$$service; \
	done
	@echo "All charts linted successfully!"

helm-validate: ## Run comprehensive Helm validation
	@echo "Running comprehensive Helm validation..."
	@./helm/scripts/validate-all-charts.sh
	@echo ""
	@echo "Running ArgoCD Applications validation..."
	@./argocd/scripts/validate-applications.sh

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
		cd ./helm/charts/cert-manager-config && helm dependency update && cd -; \
		helm upgrade --install cert-manager-config ./helm/charts/cert-manager-config \
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
	helm install wealist-infrastructure ./helm/charts/wealist-infrastructure \
		-f $(HELM_BASE_VALUES) \
		-f $(HELM_ENV_VALUES) $(HELM_SECRETS_FLAG) \
		-n $(K8S_NAMESPACE) --create-namespace
	@echo "Infrastructure installed!"

helm-install-services: ## Install all service charts
	@echo "Installing services (ENV=$(ENV), NS=$(K8S_NAMESPACE))..."
	@echo "Services to install: $(HELM_SERVICES)"
	@for service in $(HELM_SERVICES); do \
		echo "Installing $$service..."; \
		helm install $$service ./helm/charts/$$service \
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
	@echo "Services to install: $(HELM_SERVICES)"
	@for service in $(HELM_SERVICES); do \
		echo "Installing $$service with DB migration..."; \
		helm install $$service ./helm/charts/$$service \
			-f $(HELM_BASE_VALUES) \
			-f $(HELM_ENV_VALUES) $(HELM_SECRETS_FLAG) \
			--set shared.config.DB_AUTO_MIGRATE=true \
			-n $(K8S_NAMESPACE); \
	done
	@echo "Initial setup complete! Future deploys will skip DB migration."

##@ Helm Upgrade/Uninstall

helm-upgrade-all: helm-deps-build ## Upgrade all charts
	@echo "Upgrading all charts (ENV=$(ENV), NS=$(K8S_NAMESPACE))..."
	@echo "Services to upgrade: $(HELM_SERVICES)"
	@helm upgrade wealist-infrastructure ./helm/charts/wealist-infrastructure \
		-f $(HELM_BASE_VALUES) \
		-f $(HELM_ENV_VALUES) $(HELM_SECRETS_FLAG) \
		-n $(K8S_NAMESPACE)
	@for service in $(HELM_SERVICES); do \
		echo "Upgrading $$service..."; \
		helm upgrade $$service ./helm/charts/$$service \
			-f $(HELM_BASE_VALUES) \
			-f $(HELM_ENV_VALUES) $(HELM_SECRETS_FLAG) \
			-n $(K8S_NAMESPACE); \
	done
	@echo "All charts upgraded!"

helm-uninstall-all: ## Uninstall all charts
	@echo "Uninstalling all charts (ENV=$(ENV), NS=$(K8S_NAMESPACE))..."
	@# Uninstall all services (including frontend if it was installed)
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

# DEPRECATED-SOON: local-ubuntu will be replaced by staging
helm-local-ubuntu: ## Deploy to local Ubuntu (DEPRECATED-SOON: use staging instead)
	@$(MAKE) helm-install-all ENV=local-ubuntu

helm-dev: ## Deploy to dev server
	@$(MAKE) helm-install-all ENV=dev

helm-staging: ## Deploy to staging
	@$(MAKE) helm-install-all ENV=staging

helm-prod: ## Deploy to production
	@$(MAKE) helm-install-all ENV=prod

##@ Istio Service Mesh

.PHONY: istio-install istio-install-ambient istio-install-gateway istio-install-addons istio-install-config
.PHONY: istio-label-ns istio-label-ns-ambient istio-restart-pods istio-uninstall istio-status

ISTIO_VERSION ?= 1.24.0

GATEWAY_API_VERSION ?= v1.2.0

istio-install-ambient: ## Install Istio Ambient mode (recommended)
	@echo "Installing Istio Ambient $(ISTIO_VERSION)..."
	@echo ""
	@echo "Ambient mode uses:"
	@echo "  - ztunnel (DaemonSet): L4 mTLS, basic auth on each node"
	@echo "  - Waypoint Proxy: L7 features (routing, retries, JWT) per namespace"
	@echo "  - NO sidecar injection needed"
	@echo ""
	@echo "Step 1: Installing Kubernetes Gateway API CRDs $(GATEWAY_API_VERSION)..."
	@kubectl get crd gateways.gateway.networking.k8s.io >/dev/null 2>&1 || \
		kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/$(GATEWAY_API_VERSION)/standard-install.yaml
	@echo ""
	@echo "Step 2: Adding Istio Helm repository..."
	@helm repo add istio https://istio-release.storage.googleapis.com/charts 2>/dev/null || true
	@helm repo update
	@echo ""
	@echo "Step 3: Installing istio-base (CRDs)..."
	@helm upgrade --install istio-base istio/base \
		-n istio-system --create-namespace \
		--version $(ISTIO_VERSION) --wait
	@echo ""
	@echo "Step 4: Installing istio-cni (Ambient required)..."
	@helm upgrade --install istio-cni istio/cni \
		-n istio-system \
		--version $(ISTIO_VERSION) \
		--set profile=ambient --wait
	@echo ""
	@echo "Step 5: Installing istiod (Ambient profile)..."
	@helm upgrade --install istiod istio/istiod \
		-n istio-system \
		--version $(ISTIO_VERSION) \
		--set profile=ambient --wait
	@echo ""
	@echo "Step 6: Installing ztunnel (L4 secure overlay)..."
	@helm upgrade --install ztunnel istio/ztunnel \
		-n istio-system \
		--version $(ISTIO_VERSION) --wait
	@echo ""
	@echo "Istio Ambient core installation complete!"
	@echo ""
	@echo "Next steps:"
	@echo "  1. make istio-label-ns-ambient  # Enable Ambient for namespace"
	@echo "  2. make istio-install-config    # Install Gateway, VirtualService, Waypoint"
	@echo ""
	@echo "Note: Ambient mode uses Kubernetes Gateway API (Waypoint) instead of"
	@echo "      legacy istio-ingressgateway. If you need legacy gateway, run:"
	@echo "      make istio-install-gateway"

istio-install-gateway: ## Install Istio Ingress Gateway (optional, for legacy support)
	@echo "Installing Istio Ingress Gateway..."
	@echo ""
	@echo "Note: For Ambient mode, consider using Kubernetes Gateway API instead."
	@echo "      This is for legacy sidecar mode or hybrid setups."
	@echo ""
	@helm upgrade --install istio-ingressgateway istio/gateway \
		-n istio-system \
		--version $(ISTIO_VERSION) \
		--set service.type=LoadBalancer --wait
	@echo ""
	@echo "Istio Ingress Gateway installed!"

istio-install: ## Install Istio Sidecar mode (legacy)
	@echo "Installing Istio $(ISTIO_VERSION)..."
	@echo ""
	@echo "Step 1: Adding Istio Helm repository..."
	@helm repo add istio https://istio-release.storage.googleapis.com/charts 2>/dev/null || true
	@helm repo update
	@echo ""
	@echo "Step 2: Installing istio-base (CRDs)..."
	@helm upgrade --install istio-base istio/base \
		-n istio-system --create-namespace \
		--version $(ISTIO_VERSION) --wait
	@echo ""
	@echo "Step 3: Installing istiod (control plane)..."
	@helm upgrade --install istiod istio/istiod \
		-n istio-system \
		--version $(ISTIO_VERSION) --wait
	@echo ""
	@echo "Step 4: Installing istio-ingressgateway..."
	@helm upgrade --install istio-ingressgateway istio/gateway \
		-n istio-system \
		--version $(ISTIO_VERSION) --wait
	@echo ""
	@echo "Istio core installation complete!"
	@echo ""
	@echo "Next steps:"
	@echo "  1. make istio-label-ns       # Enable sidecar injection for namespace"
	@echo "  2. make istio-install-config # Install Istio routing configuration"
	@echo "  3. make istio-restart-pods   # Restart pods to inject sidecars"
	@echo "  4. make istio-install-addons # Install Kiali, Jaeger (optional)"

istio-label-ns: ## Label namespace for Istio sidecar injection (legacy)
	@echo "Labeling namespace $(K8S_NAMESPACE) for Istio sidecar injection..."
	@kubectl label namespace $(K8S_NAMESPACE) istio-injection=enabled --overwrite
	@echo ""
	@echo "Namespace labeled! Pods will get Istio sidecar on restart."
	@echo "Run: make istio-restart-pods"

istio-label-ns-ambient: ## Label namespace for Istio Ambient mode
	@echo "Labeling namespace $(K8S_NAMESPACE) for Istio Ambient mode..."
	@kubectl label namespace $(K8S_NAMESPACE) istio.io/dataplane-mode=ambient --overwrite
	@kubectl label namespace $(K8S_NAMESPACE) istio-injection- 2>/dev/null || true
	@echo ""
	@echo "Namespace labeled for Ambient mode!"
	@echo "Pods are automatically enrolled - no restart needed."

istio-restart-pods: ## Restart all pods to inject Istio sidecars
	@echo "Restarting all deployments in $(K8S_NAMESPACE) to inject sidecars..."
	@kubectl rollout restart deployment -n $(K8S_NAMESPACE)
	@echo ""
	@echo "Pods are restarting. Check status with: make status"

istio-install-config: ## Install Istio configuration (Gateway, VirtualService, etc.)
	@echo "Installing Istio configuration (ENV=$(ENV), NS=$(K8S_NAMESPACE))..."
	@helm upgrade --install istio-config ./helm/charts/istio-config \
		-f $(HELM_BASE_VALUES) \
		-f $(HELM_ENV_VALUES) \
		-n $(K8S_NAMESPACE) --wait
	@echo ""
	@echo "Istio configuration installed!"
	@echo "Gateway, VirtualService, PeerAuthentication, DestinationRules, AuthorizationPolicy deployed."

istio-install-addons: ## Install Istio addons (Kiali, Jaeger)
	@echo "Installing Istio observability addons..."
	@echo ""
	@echo "Installing Kiali (Service Graph)..."
	@kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.20/samples/addons/kiali.yaml
	@echo ""
	@echo "Installing Jaeger (Distributed Tracing)..."
	@kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.20/samples/addons/jaeger.yaml
	@echo ""
	@echo "Installing Prometheus (if not exists)..."
	@kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.20/samples/addons/prometheus.yaml 2>/dev/null || true
	@echo ""
	@echo "Addons installed!"
	@echo ""
	@echo "Access Kiali dashboard: kubectl port-forward svc/kiali -n istio-system 20001:20001"
	@echo "Access Jaeger dashboard: kubectl port-forward svc/tracing -n istio-system 16686:80"

istio-status: ## Show Istio installation status
	@echo "=== Istio System Components ==="
	@kubectl get pods -n istio-system
	@echo ""
	@echo "=== Istio Mode Status ($(K8S_NAMESPACE)) ==="
	@echo -n "Ambient mode: "; kubectl get namespace $(K8S_NAMESPACE) -o jsonpath='{.metadata.labels.istio\.io/dataplane-mode}' 2>/dev/null && echo "" || echo "not enabled"
	@echo -n "Sidecar injection: "; kubectl get namespace $(K8S_NAMESPACE) -o jsonpath='{.metadata.labels.istio-injection}' 2>/dev/null && echo "" || echo "not enabled"
	@echo ""
	@echo "=== ztunnel Status (Ambient) ==="
	@kubectl get pods -n istio-system -l app=ztunnel 2>/dev/null || echo "ztunnel not installed"
	@echo ""
	@echo "=== Waypoint Proxies ($(K8S_NAMESPACE)) ==="
	@kubectl get gateway -n $(K8S_NAMESPACE) -l istio.io/waypoint-for 2>/dev/null || echo "No waypoint proxies"
	@echo ""
	@echo "=== Pods ($(K8S_NAMESPACE)) ==="
	@kubectl get pods -n $(K8S_NAMESPACE) -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{range .spec.containers[*]}{.name}{" "}{end}{"\n"}{end}' 2>/dev/null | grep -v "^$$" || echo "No pods found"

istio-uninstall: ## Uninstall Istio completely
	@echo "Uninstalling Istio..."
	@echo ""
	@echo "Step 1: Removing Istio configuration..."
	@helm uninstall istio-config -n $(K8S_NAMESPACE) 2>/dev/null || true
	@echo ""
	@echo "Step 2: Removing namespace labels..."
	@kubectl label namespace $(K8S_NAMESPACE) istio-injection- 2>/dev/null || true
	@kubectl label namespace $(K8S_NAMESPACE) istio.io/dataplane-mode- 2>/dev/null || true
	@echo ""
	@echo "Step 3: Removing Istio addons..."
	@kubectl delete -f https://raw.githubusercontent.com/istio/istio/release-$(ISTIO_VERSION)/samples/addons/kiali.yaml 2>/dev/null || true
	@kubectl delete -f https://raw.githubusercontent.com/istio/istio/release-$(ISTIO_VERSION)/samples/addons/jaeger.yaml 2>/dev/null || true
	@echo ""
	@echo "Step 4: Removing Istio core (including Ambient components)..."
	@helm uninstall istio-ingressgateway -n istio-system 2>/dev/null || true
	@helm uninstall ztunnel -n istio-system 2>/dev/null || true
	@helm uninstall istiod -n istio-system 2>/dev/null || true
	@helm uninstall istio-cni -n istio-system 2>/dev/null || true
	@helm uninstall istio-base -n istio-system 2>/dev/null || true
	@echo ""
	@echo "Istio uninstalled!"
	@echo "Note: For sidecar mode, restart pods to remove sidecars: make istio-restart-pods"
