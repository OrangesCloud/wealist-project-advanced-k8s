.PHONY: help dev-up dev-down dev-logs kind-setup kind-load-images kind-apply kind-delete status clean
.PHONY: local-kind-apply local-tls-secret
.PHONY: sonar-up sonar-down sonar-logs sonar-status sonar-restart sonar-clean
.PHONY: auth-service-build auth-service-load auth-service-redeploy auth-service-all
.PHONY: board-service-build board-service-load board-service-redeploy board-service-all
.PHONY: chat-service-build chat-service-load chat-service-redeploy chat-service-all
.PHONY: frontend-build frontend-load frontend-redeploy frontend-all
.PHONY: noti-service-build noti-service-load noti-service-redeploy noti-service-all
.PHONY: storage-service-build storage-service-load storage-service-redeploy storage-service-all
.PHONY: user-service-build user-service-load user-service-redeploy user-service-all
.PHONY: video-service-build video-service-load video-service-redeploy video-service-all

# Kind cluster name
KIND_CLUSTER ?= wealist
LOCAL_REGISTRY ?= localhost:5001
K8S_NAMESPACE ?= wealist-dev
IMAGE_TAG ?= latest

help:
	@echo "Wealist Project"
	@echo ""
	@echo "  Development (Docker Compose):"
	@echo "    make dev-up       - Start all services"
	@echo "    make dev-down     - Stop all services"
	@echo "    make dev-logs     - View logs"
	@echo ""
	@echo "  SonarQube (Code Quality - Standalone):"
	@echo "    make sonar-up     - Start SonarQube only (lightweight)"
	@echo "    make sonar-down   - Stop SonarQube environment"
	@echo "    make sonar-logs   - View SonarQube logs"
	@echo "    make sonar-status - Check SonarQube status"
	@echo "    make sonar-restart - Restart SonarQube environment"
	@echo "    make sonar-clean  - Clean SonarQube data (destructive)"
	@echo ""
	@echo "  Kubernetes (Local - localhost) - 3 Step Setup:"
	@echo "    make kind-setup       - 1. Create cluster + registry"
	@echo "    make kind-load-images - 2. Build/pull all images (infra + services)"
	@echo "    make kind-apply       - 3. Deploy all to k8s (localhost)"
	@echo "    make kind-delete      - Delete cluster"
	@echo ""
	@echo "  Kubernetes (Local - local.wealist.co.kr):"
	@echo "    make local-kind-apply - Deploy with local.wealist.co.kr domain"
	@echo "    (Uses same cluster/images as kind-*, only ingress host differs)"
	@echo ""
	@echo "  Per-Service Commands:"
	@echo "    make <service>-build    - Build image only"
	@echo "    make <service>-load     - Build + push to registry"
	@echo "    make <service>-redeploy - Rollout restart in k8s"
	@echo "    make <service>-all      - Build + load + redeploy"
	@echo ""
	@echo "  Available services:"
	@echo "    auth-service, board-service, chat-service, frontend,"
	@echo "    noti-service, storage-service, user-service, video-service"
	@echo ""
	@echo "  Helm Charts (Recommended):"
	@echo "    make helm-lint           - Lint all Helm charts"
	@echo "    make helm-install-infra  - Install infrastructure chart"
	@echo "    make helm-install-services - Install all service charts"
	@echo "    make helm-install-all    - Install infrastructure + services"
	@echo "    make helm-upgrade-all    - Upgrade all charts"
	@echo "    make helm-uninstall-all  - Uninstall all charts"
	@echo "    make helm-validate       - Run comprehensive validation"
	@echo ""
	@echo "  Utility:"
	@echo "    make status       - Show pods status"
	@echo "    make clean        - Clean up"

# =============================================================================
# Development (Docker Compose)
# =============================================================================

dev-up:
	./docker/scripts/dev.sh up

dev-down:
	./docker/scripts/dev.sh down

dev-logs:
	./docker/scripts/dev.sh logs

# =============================================================================
# SonarQube (Code Quality - Standalone Environment)
# =============================================================================

sonar-up:
	@echo "🚀 Starting SonarQube standalone environment..."
	./docker/scripts/sonar.sh up

sonar-down:
	@echo "⏹️  Stopping SonarQube standalone environment..."
	./docker/scripts/sonar.sh down

sonar-logs:
	./docker/scripts/sonar.sh logs

sonar-status:
	./docker/scripts/sonar.sh status

sonar-restart:
	@echo "🔄 Restarting SonarQube standalone environment..."
	./docker/scripts/sonar.sh restart

sonar-clean:
	@echo "🗑️  Cleaning SonarQube standalone environment..."
	./docker/scripts/sonar.sh clean

# =============================================================================
# Kubernetes (Local - Kind)
# =============================================================================

# Step 1: Create cluster + registry only
kind-setup:
	@echo "=== Step 1: Creating Kind cluster with local registry ==="
	./docker/scripts/dev/0.setup-cluster.sh
	@echo ""
	@echo "✅ Cluster ready! Next: make kind-load-images"

# Step 2: Build/pull all images (infra + services)
kind-load-images:
	@echo "=== Step 2: Loading all images ==="
	@echo ""
	@echo "--- Loading infrastructure images ---"
	./docker/scripts/dev/1.load_infra_images.sh
	@echo ""
	@echo "--- Building service images ---"
	./docker/scripts/dev/2.build_services_and_load.sh
	@echo ""
	@echo "✅ All images loaded!"
	@echo ""
	@echo "Next step (choose one):"
	@echo "  make kind-apply       - Deploy (localhost)"
	@echo "  make local-kind-apply - Deploy (local.wealist.co.kr)"

# Step 3: Deploy all to k8s
kind-apply:
	@echo "=== Step 3: Deploying to Kubernetes ==="
	@echo ""
	@echo "--- Deploying infrastructure ---"
	kubectl apply -k infrastructure/overlays/develop
	@echo ""
	@echo "Waiting for infra pods..."
	kubectl wait --namespace $(K8S_NAMESPACE) --for=condition=ready pod --selector=app=postgres --timeout=120s || true
	kubectl wait --namespace $(K8S_NAMESPACE) --for=condition=ready pod --selector=app=redis --timeout=120s || true
	@echo ""
	@echo "--- Deploying services ---"
	kubectl apply -k k8s/overlays/develop-registry/all-services
	@echo ""
	@echo "✅ Done! Check: make status"

kind-delete:
	kind delete cluster --name $(KIND_CLUSTER)
	@docker rm -f kind-registry 2>/dev/null || true

# =============================================================================
# Kubernetes (Local - local.wealist.co.kr)
# =============================================================================
# Uses same cluster and images as kind-* commands
# Only difference: ingress uses host: local.wealist.co.kr with TLS

local-tls-secret:
	@echo "=== Creating TLS secret for local.wealist.co.kr ==="
	@if kubectl get secret local-wealist-tls -n $(K8S_NAMESPACE) >/dev/null 2>&1; then \
		echo "TLS secret already exists, skipping..."; \
	else \
		echo "Generating self-signed certificate..."; \
		openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
			-keyout /tmp/local-wealist-tls.key \
			-out /tmp/local-wealist-tls.crt \
			-subj "/CN=local.wealist.co.kr/O=wealist" \
			-addext "subjectAltName=DNS:local.wealist.co.kr"; \
		kubectl create secret tls local-wealist-tls \
			--cert=/tmp/local-wealist-tls.crt \
			--key=/tmp/local-wealist-tls.key \
			-n $(K8S_NAMESPACE); \
		rm -f /tmp/local-wealist-tls.key /tmp/local-wealist-tls.crt; \
		echo "✅ TLS secret created"; \
	fi

local-kind-apply: local-tls-secret
	@echo "=== Deploying to Kubernetes (local.wealist.co.kr) ==="
	@echo ""
	@echo "--- Deploying infrastructure ---"
	kubectl apply -k infrastructure/overlays/develop
	@echo ""
	@echo "Waiting for infra pods..."
	kubectl wait --namespace $(K8S_NAMESPACE) --for=condition=ready pod --selector=app=postgres --timeout=120s || true
	kubectl wait --namespace $(K8S_NAMESPACE) --for=condition=ready pod --selector=app=redis --timeout=120s || true
	@echo ""
	@echo "--- Deploying services (local.wealist.co.kr) ---"
	kubectl apply -k k8s/overlays/develop-registry-local/all-services
	@echo ""
	@echo "✅ Done! Access: https://local.wealist.co.kr"
	@echo "(Self-signed cert - browser will show warning, click 'Advanced' → 'Proceed')"
	@echo "Check: make status"

# =============================================================================
# Per-Service Commands
# =============================================================================

# Service definitions: name|path|dockerfile|k8s-deployment-name
define build-service
	@echo "Building $(1)..."
	docker build -t $(LOCAL_REGISTRY)/$(1):$(IMAGE_TAG) -f $(2)/$(3) $(2)
	@echo "✅ Built $(LOCAL_REGISTRY)/$(1):$(IMAGE_TAG)"
endef

define load-service
	@echo "Building and pushing $(1) to registry..."
	docker build -t $(LOCAL_REGISTRY)/$(1):$(IMAGE_TAG) -f $(2)/$(3) $(2)
	docker push $(LOCAL_REGISTRY)/$(1):$(IMAGE_TAG)
	@echo "✅ Pushed $(LOCAL_REGISTRY)/$(1):$(IMAGE_TAG)"
endef

define redeploy-service
	@echo "Redeploying $(1)..."
	kubectl rollout restart deployment/$(1) -n $(K8S_NAMESPACE)
	@echo "✅ Rollout restart triggered for $(1)"
endef

# --- auth-service ---
auth-service-build:
	$(call build-service,auth-service,services/auth-service,Dockerfile)

auth-service-load:
	$(call load-service,auth-service,services/auth-service,Dockerfile)

auth-service-redeploy:
	$(call redeploy-service,auth-service)

auth-service-all: auth-service-load auth-service-redeploy

# --- board-service ---
board-service-build:
	$(call build-service,board-service,services/board-service,docker/Dockerfile)

board-service-load:
	$(call load-service,board-service,services/board-service,docker/Dockerfile)

board-service-redeploy:
	$(call redeploy-service,board-service)

board-service-all: board-service-load board-service-redeploy

# --- chat-service ---
chat-service-build:
	@echo "Building chat-service..."
	docker build -t $(LOCAL_REGISTRY)/chat-service:$(IMAGE_TAG) -f services/chat-service/docker/Dockerfile .
	@echo "✅ Built $(LOCAL_REGISTRY)/chat-service:$(IMAGE_TAG)"

chat-service-load:
	@echo "Building and pushing chat-service to registry..."
	docker build -t $(LOCAL_REGISTRY)/chat-service:$(IMAGE_TAG) -f services/chat-service/docker/Dockerfile .
	docker push $(LOCAL_REGISTRY)/chat-service:$(IMAGE_TAG)
	@echo "✅ Pushed $(LOCAL_REGISTRY)/chat-service:$(IMAGE_TAG)"

chat-service-redeploy:
	$(call redeploy-service,chat-service)

chat-service-all: chat-service-load chat-service-redeploy

# --- frontend ---
frontend-build:
	$(call build-service,frontend,services/frontend,Dockerfile)

frontend-load:
	$(call load-service,frontend,services/frontend,Dockerfile)

frontend-redeploy:
	$(call redeploy-service,frontend)

frontend-all: frontend-load frontend-redeploy

# --- noti-service ---
noti-service-build:
	$(call build-service,noti-service,services/noti-service,docker/Dockerfile)

noti-service-load:
	$(call load-service,noti-service,services/noti-service,docker/Dockerfile)

noti-service-redeploy:
	$(call redeploy-service,noti-service)

noti-service-all: noti-service-load noti-service-redeploy

# --- storage-service ---
storage-service-build:
	$(call build-service,storage-service,services/storage-service,docker/Dockerfile)

storage-service-load:
	$(call load-service,storage-service,services/storage-service,docker/Dockerfile)

storage-service-redeploy:
	$(call redeploy-service,storage-service)

storage-service-all: storage-service-load storage-service-redeploy

# --- user-service ---
user-service-build:
	@echo "Building user-service..."
	docker build -t $(LOCAL_REGISTRY)/user-service:$(IMAGE_TAG) -f services/user-service/docker/Dockerfile .
	@echo "✅ Built $(LOCAL_REGISTRY)/user-service:$(IMAGE_TAG)"

user-service-load:
	@echo "Building and pushing user-service to registry..."
	docker build -t $(LOCAL_REGISTRY)/user-service:$(IMAGE_TAG) -f services/user-service/docker/Dockerfile .
	docker push $(LOCAL_REGISTRY)/user-service:$(IMAGE_TAG)
	@echo "✅ Pushed $(LOCAL_REGISTRY)/user-service:$(IMAGE_TAG)"

user-service-redeploy:
	$(call redeploy-service,user-service)

user-service-all: user-service-load user-service-redeploy

# --- video-service ---
video-service-build:
	$(call build-service,video-service,services/video-service,docker/Dockerfile)

video-service-load:
	$(call load-service,video-service,services/video-service,docker/Dockerfile)

video-service-redeploy:
	$(call redeploy-service,video-service)

video-service-all: video-service-load video-service-redeploy

# =============================================================================
# Utility
# =============================================================================

status:
	@echo "=== Kubernetes Pods ==="
	@kubectl get pods -n wealist-dev 2>/dev/null || echo "Namespace not found"

clean:
	./docker/scripts/clean.sh

# =============================================================================
# Helm Charts
# =============================================================================

.PHONY: helm-lint helm-install-infra helm-install-services helm-install-all
.PHONY: helm-upgrade-all helm-uninstall-all helm-validate

HELM_NAMESPACE ?= wealist-dev
HELM_VALUES_FILE ?= values-develop-registry-local.yaml
SERVICES = auth-service user-service board-service chat-service noti-service storage-service video-service frontend

helm-lint:
	@echo "🔍 Linting all Helm charts..."
	@helm lint ./helm/charts/wealist-common
	@helm lint ./helm/charts/wealist-infrastructure
	@for service in $(SERVICES); do \
		echo "Linting $$service..."; \
		helm lint ./helm/charts/$$service; \
	done
	@echo "✅ All charts linted successfully!"

helm-install-infra:
	@echo "📦 Installing infrastructure chart..."
	helm install wealist-infrastructure ./helm/charts/wealist-infrastructure \
		-f ./helm/charts/wealist-infrastructure/$(HELM_VALUES_FILE) \
		-n $(HELM_NAMESPACE) --create-namespace
	@echo "✅ Infrastructure installed!"

helm-install-services:
	@echo "📦 Installing all service charts..."
	@for service in $(SERVICES); do \
		echo "Installing $$service..."; \
		helm install $$service ./helm/charts/$$service \
			-f ./helm/charts/$$service/$(HELM_VALUES_FILE) \
			-n $(HELM_NAMESPACE); \
	done
	@echo "✅ All services installed!"

helm-install-all: helm-install-infra
	@sleep 5
	@$(MAKE) helm-install-services

helm-upgrade-all:
	@echo "🔄 Upgrading all charts..."
	@helm upgrade wealist-infrastructure ./helm/charts/wealist-infrastructure \
		-f ./helm/charts/wealist-infrastructure/$(HELM_VALUES_FILE) \
		-n $(HELM_NAMESPACE)
	@for service in $(SERVICES); do \
		echo "Upgrading $$service..."; \
		helm upgrade $$service ./helm/charts/$$service \
			-f ./helm/charts/$$service/$(HELM_VALUES_FILE) \
			-n $(HELM_NAMESPACE); \
	done
	@echo "✅ All charts upgraded!"

helm-uninstall-all:
	@echo "🗑️  Uninstalling all charts..."
	@for service in $(SERVICES); do \
		echo "Uninstalling $$service..."; \
		helm uninstall $$service -n $(HELM_NAMESPACE) 2>/dev/null || true; \
	done
	@helm uninstall wealist-infrastructure -n $(HELM_NAMESPACE) 2>/dev/null || true
	@echo "✅ All charts uninstalled!"

helm-validate:
	@echo "🔍 Running comprehensive Helm validation..."
	@./helm/scripts/validate-all-charts.sh
	@echo ""
	@echo "🔍 Running ArgoCD Applications validation..."
	@./argocd/scripts/validate-applications.sh
