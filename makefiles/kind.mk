# =============================================================================
# Kubernetes (Kind) Commands
# =============================================================================

##@ Kubernetes (Kind)

.PHONY: kind-setup kind-setup-db kind-load-images kind-load-images-mono kind-delete kind-recover
.PHONY: _setup-db-macos _setup-db-debian

kind-setup: kind-setup-db ## Create cluster + registry (with local DB setup)
	@echo "=== Step 2: Creating Kind cluster with local registry ==="
	./docker/scripts/dev/0.setup-cluster.sh
	@echo ""
	@echo "Cluster ready! Next: make kind-load-images"

kind-setup-db: ## Setup local PostgreSQL/Redis for Kind
	@echo "=== Step 1: Setting up local PostgreSQL and Redis ==="
	@echo ""
	@# Detect OS
	@if [ "$$(uname)" = "Darwin" ]; then \
		echo "Detected: macOS"; \
		$(MAKE) _setup-db-macos; \
	elif [ -f /etc/debian_version ]; then \
		echo "Detected: Debian/Ubuntu"; \
		$(MAKE) _setup-db-debian; \
	else \
		echo "Unsupported OS. Please install PostgreSQL and Redis manually."; \
		echo "  PostgreSQL: listening on 0.0.0.0:5432"; \
		echo "  Redis: listening on 0.0.0.0:6379"; \
	fi
	@echo ""
	@echo "Local DB setup complete!"
	@echo ""

_setup-db-macos:
	@# PostgreSQL
	@if ! command -v psql >/dev/null 2>&1; then \
		echo "Installing PostgreSQL..."; \
		brew install postgresql@14; \
		brew services start postgresql@14; \
	else \
		echo "PostgreSQL already installed"; \
		brew services start postgresql@14 2>/dev/null || brew services start postgresql 2>/dev/null || true; \
	fi
	@# Redis
	@if ! command -v redis-cli >/dev/null 2>&1; then \
		echo "Installing Redis..."; \
		brew install redis; \
		brew services start redis; \
	else \
		echo "Redis already installed"; \
		brew services start redis 2>/dev/null || true; \
	fi
	@# Create wealist databases
	@echo "Creating wealist databases..."
	@psql -U postgres -c "SELECT 1" 2>/dev/null || createuser -s postgres 2>/dev/null || true
	@for db in wealist wealist_auth wealist_user wealist_board wealist_chat wealist_noti wealist_storage wealist_video; do \
		psql -U postgres -tc "SELECT 1 FROM pg_database WHERE datname = '$$db'" | grep -q 1 || \
		psql -U postgres -c "CREATE DATABASE $$db" 2>/dev/null || true; \
	done
	@echo "PostgreSQL databases ready"

_setup-db-debian:
	@# PostgreSQL
	@if ! command -v psql >/dev/null 2>&1; then \
		echo "Installing PostgreSQL..."; \
		sudo apt-get update && sudo apt-get install -y postgresql postgresql-contrib; \
	else \
		echo "PostgreSQL already installed"; \
	fi
	@sudo systemctl start postgresql || true
	@# Configure PostgreSQL for external access
	@echo "Configuring PostgreSQL for Kind cluster access..."
	@PG_HBA=$$(sudo -u postgres psql -t -P format=unaligned -c "SHOW hba_file"); \
	if ! sudo grep -q "172.18.0.0/16" "$$PG_HBA" 2>/dev/null; then \
		echo "host    all    all    172.17.0.0/16    trust" | sudo tee -a "$$PG_HBA" >/dev/null; \
		echo "host    all    all    172.18.0.0/16    trust" | sudo tee -a "$$PG_HBA" >/dev/null; \
	fi
	@PG_CONF=$$(sudo -u postgres psql -t -P format=unaligned -c "SHOW config_file"); \
	sudo sed -i "s/#listen_addresses = 'localhost'/listen_addresses = '*'/" "$$PG_CONF" 2>/dev/null || true; \
	sudo sed -i "s/listen_addresses = 'localhost'/listen_addresses = '*'/" "$$PG_CONF" 2>/dev/null || true
	@sudo systemctl restart postgresql
	@# Create wealist databases
	@echo "Creating wealist databases..."
	@for db in wealist wealist_auth wealist_user wealist_board wealist_chat wealist_noti wealist_storage wealist_video; do \
		sudo -u postgres psql -tc "SELECT 1 FROM pg_database WHERE datname = '$$db'" | grep -q 1 || \
		sudo -u postgres psql -c "CREATE DATABASE $$db" 2>/dev/null || true; \
	done
	@echo "PostgreSQL databases ready"
	@# Redis
	@if ! command -v redis-cli >/dev/null 2>&1; then \
		echo "Installing Redis..."; \
		sudo apt-get install -y redis-server; \
	else \
		echo "Redis already installed"; \
	fi
	@# Configure Redis for external access
	@echo "Configuring Redis for Kind cluster access..."
	@sudo sed -i 's/^bind 127.0.0.1/bind 0.0.0.0/' /etc/redis/redis.conf 2>/dev/null || true
	@sudo sed -i 's/^protected-mode yes/protected-mode no/' /etc/redis/redis.conf 2>/dev/null || true
	@sudo systemctl restart redis-server || sudo systemctl restart redis
	@echo "Redis ready"

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
