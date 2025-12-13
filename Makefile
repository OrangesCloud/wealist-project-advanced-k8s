.PHONY: help dev-up dev-down dev-logs kind-setup kind-load-images kind-apply kind-delete status clean
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
LOCAL_DOMAIN ?= local.wealist.co.kr

help:
	@echo "Wealist Project"
	@echo ""
	@echo "  Development (Docker Compose):"
	@echo "    make dev-up       - Start all services"
	@echo "    make dev-down     - Stop all services"
	@echo "    make dev-logs     - View logs"
	@echo ""
	@echo "  Kubernetes (Kind) - 3 Step Setup:"
	@echo "    make kind-setup       - 1. Create cluster + registry"
	@echo "    make kind-load-images - 2. Build/pull all images (infra + services)"
	@echo "    make kind-apply       - 3. Deploy to k8s (default: local.wealist.co.kr)"
	@echo "    make kind-apply LOCAL_DOMAIN=<domain> - Deploy with custom domain"
	@echo "    make kind-delete      - Delete cluster"
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
	@echo "Next: make kind-apply"
	@echo "  (Or: make kind-apply LOCAL_DOMAIN=wonny.wealist.co.kr)"

# Step 3: Deploy all to k8s
kind-apply:
	@echo "=== Step 3: Deploying to Kubernetes ($(LOCAL_DOMAIN)) ==="
	@echo ""
	@echo "--- Deploying infrastructure ---"
	kubectl apply -k infrastructure/overlays/develop
	@echo ""
	@echo "Waiting for infra pods..."
	kubectl wait --namespace $(K8S_NAMESPACE) --for=condition=ready pod --selector=app=postgres --timeout=120s || true
	kubectl wait --namespace $(K8S_NAMESPACE) --for=condition=ready pod --selector=app=redis --timeout=120s || true
	@echo ""
	@echo "--- Deploying services ($(LOCAL_DOMAIN)) ---"
	@# Replace placeholder with actual domain in template files
	@sed -i.bak 's/__LOCAL_DOMAIN__/$(LOCAL_DOMAIN)/g' \
		k8s/overlays/develop-registry/all-services/ingress.yaml \
		k8s/overlays/develop-registry/all-services/kustomization.yaml \
		services/auth-service/k8s/base/configmap.yaml \
		k8s/base/namespace-dev/configmap.yaml \
		services/video-service/k8s/base/deployment.yaml \
		infrastructure/base/livekit/configmap.yaml
	@kubectl apply -k k8s/overlays/develop-registry/all-services || \
		(mv k8s/overlays/develop-registry/all-services/ingress.yaml.bak \
			k8s/overlays/develop-registry/all-services/ingress.yaml && \
		 mv k8s/overlays/develop-registry/all-services/kustomization.yaml.bak \
			k8s/overlays/develop-registry/all-services/kustomization.yaml && \
		 mv services/auth-service/k8s/base/configmap.yaml.bak \
			services/auth-service/k8s/base/configmap.yaml && \
		 mv k8s/base/namespace-dev/configmap.yaml.bak \
			k8s/base/namespace-dev/configmap.yaml && \
		 mv services/video-service/k8s/base/deployment.yaml.bak \
			services/video-service/k8s/base/deployment.yaml && \
		 mv infrastructure/base/livekit/configmap.yaml.bak \
			infrastructure/base/livekit/configmap.yaml && exit 1)
	@# Restore template files
	@mv k8s/overlays/develop-registry/all-services/ingress.yaml.bak \
		k8s/overlays/develop-registry/all-services/ingress.yaml
	@mv k8s/overlays/develop-registry/all-services/kustomization.yaml.bak \
		k8s/overlays/develop-registry/all-services/kustomization.yaml
	@mv services/auth-service/k8s/base/configmap.yaml.bak \
		services/auth-service/k8s/base/configmap.yaml
	@mv k8s/base/namespace-dev/configmap.yaml.bak \
		k8s/base/namespace-dev/configmap.yaml
	@mv services/video-service/k8s/base/deployment.yaml.bak \
		services/video-service/k8s/base/deployment.yaml
	@mv infrastructure/base/livekit/configmap.yaml.bak \
		infrastructure/base/livekit/configmap.yaml
	@echo ""
	@echo "✅ Done! Access: http://$(LOCAL_DOMAIN)"
	@echo "Check: make status"

kind-delete:
	kind delete cluster --name $(KIND_CLUSTER)
	@docker rm -f kind-registry 2>/dev/null || true

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
