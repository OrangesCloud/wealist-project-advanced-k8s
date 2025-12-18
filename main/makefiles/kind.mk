# =============================================================================
# Kubernetes (Kind) Commands
# =============================================================================

##@ Kubernetes (Kind)

.PHONY: kind-setup kind-load-images kind-load-images-mono kind-delete kind-recover

kind-setup: ## Create cluster + registry
	@echo "=== Step 1: Creating Kind cluster with local registry ==="
	./docker/scripts/dev/0.setup-cluster.sh
	@echo ""
	@echo "Cluster ready! Next: make kind-load-images"

kind-load-images: ## Build/pull all images (infra + services)
	@echo "=== Step 2: Loading all images ==="
	@echo ""
	@echo "--- Loading infrastructure images ---"
	./docker/scripts/dev/1.load_infra_images.sh
	@echo ""
	@echo "--- Building service images ---"
	./docker/scripts/dev/2.build_services_and_load.sh
	@echo ""
	@echo "All images loaded!"
	@echo ""
	@echo "Next: make helm-install-all ENV=local-kind"

kind-load-images-mono: ## Build Go services with monorepo pattern (faster rebuilds)
	@echo "=== Loading images using Monorepo Build (BuildKit cache) ==="
	@echo ""
	@echo "--- Loading infrastructure images ---"
	./docker/scripts/dev/1.load_infra_images.sh
	@echo ""
	@echo "--- Building Go services (monorepo pattern) ---"
	./docker/scripts/dev-mono.sh build
	@echo ""
	@echo "--- Tagging and pushing to local registry ---"
	@for svc in user-service board-service chat-service noti-service storage-service video-service; do \
		echo "Pushing $$svc..."; \
		docker tag wealist/$$svc:latest $(LOCAL_REGISTRY)/$$svc:$(IMAGE_TAG); \
		docker push $(LOCAL_REGISTRY)/$$svc:$(IMAGE_TAG); \
	done
	@echo ""
	@echo "--- Building auth-service and frontend ---"
	@$(MAKE) auth-service-load frontend-load
	@echo ""
	@echo "All images loaded! (Monorepo pattern)"
	@echo ""
	@echo "Next: make helm-install-all ENV=local-kind"

kind-delete: ## Delete cluster
	kind delete cluster --name $(KIND_CLUSTER)
	@docker rm -f kind-registry 2>/dev/null || true

kind-recover: ## Recover cluster after reboot
	@echo "Recovering Kind cluster..."
	@docker restart $(KIND_CLUSTER)-control-plane $(KIND_CLUSTER)-worker $(KIND_CLUSTER)-worker2 kind-registry 2>/dev/null || true
	@sleep 30
	@kind export kubeconfig --name $(KIND_CLUSTER)
	@echo "Waiting for API server..."
	@until kubectl get nodes >/dev/null 2>&1; do sleep 5; done
	@echo "Cluster recovered!"
	@kubectl get nodes

##@ Local Domain (local.wealist.co.kr)

.PHONY: local-tls-secret

local-tls-secret: ## Create TLS secret for local.wealist.co.kr
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
		echo "TLS secret created"; \
	fi

##@ Local Database

.PHONY: init-local-db

init-local-db: ## Init local PostgreSQL/Redis (Ubuntu, ENV=local-ubuntu)
	@echo "Initializing local PostgreSQL and Redis for Wealist..."
	@echo ""
	@echo "This will configure your local PostgreSQL and Redis to accept"
	@echo "connections from the Kind cluster (Docker network)."
	@echo ""
	@echo "Prerequisites:"
	@echo "  - PostgreSQL installed: sudo apt install postgresql postgresql-contrib"
	@echo "  - Redis installed: sudo apt install redis-server"
	@echo ""
	@echo "Running scripts with sudo..."
	@sudo ./scripts/init-local-postgres.sh
	@sudo ./scripts/init-local-redis.sh
	@echo ""
	@echo "Local database initialization complete!"
	@echo ""
	@echo "Next: make helm-install-all ENV=local-ubuntu"
