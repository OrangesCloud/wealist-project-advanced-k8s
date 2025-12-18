# # =============================================================================
# # Per-Service Commands
# # =============================================================================
# # Usage: make <service>-build, make <service>-load, make <service>-redeploy, make <service>-all
# # Services: auth-service, board-service, chat-service, frontend,
# #           noti-service, storage-service, user-service, video-service

# ##@ Per-Service Commands

# .PHONY: $(addsuffix -build,$(SERVICES))
# .PHONY: $(addsuffix -load,$(SERVICES))
# .PHONY: $(addsuffix -redeploy,$(SERVICES))
# .PHONY: $(addsuffix -all,$(SERVICES))
# .PHONY: redeploy-all status clean

# # -----------------------------------------------------------------------------
# # Build targets for ROOT context services (use shared package from project root)
# # -----------------------------------------------------------------------------

# chat-service-build: ## Build chat-service image
# 	@echo "Building chat-service..."
# 	docker build -t $(LOCAL_REGISTRY)/chat-service:$(IMAGE_TAG) \
# 		-f services/chat-service/docker/Dockerfile .
# 	@echo "Built $(LOCAL_REGISTRY)/chat-service:$(IMAGE_TAG)"

# noti-service-build: ## Build noti-service image
# 	@echo "Building noti-service..."
# 	docker build -t $(LOCAL_REGISTRY)/noti-service:$(IMAGE_TAG) \
# 		-f services/noti-service/docker/Dockerfile .
# 	@echo "Built $(LOCAL_REGISTRY)/noti-service:$(IMAGE_TAG)"

# storage-service-build: ## Build storage-service image
# 	@echo "Building storage-service..."
# 	docker build -t $(LOCAL_REGISTRY)/storage-service:$(IMAGE_TAG) \
# 		-f services/storage-service/docker/Dockerfile .
# 	@echo "Built $(LOCAL_REGISTRY)/storage-service:$(IMAGE_TAG)"

# user-service-build: ## Build user-service image
# 	@echo "Building user-service..."
# 	docker build -t $(LOCAL_REGISTRY)/user-service:$(IMAGE_TAG) \
# 		-f services/user-service/docker/Dockerfile .
# 	@echo "Built $(LOCAL_REGISTRY)/user-service:$(IMAGE_TAG)"

# video-service-build: ## Build video-service image
# 	@echo "Building video-service..."
# 	docker build -t $(LOCAL_REGISTRY)/video-service:$(IMAGE_TAG) \
# 		-f services/video-service/docker/Dockerfile .
# 	@echo "Built $(LOCAL_REGISTRY)/video-service:$(IMAGE_TAG)"

# # -----------------------------------------------------------------------------
# # Build targets for LOCAL context services (self-contained)
# # -----------------------------------------------------------------------------

# auth-service-build: ## Build auth-service image
# 	@echo "Building auth-service..."
# 	docker build -t $(LOCAL_REGISTRY)/auth-service:$(IMAGE_TAG) \
# 		-f services/auth-service/Dockerfile services/auth-service
# 	@echo "Built $(LOCAL_REGISTRY)/auth-service:$(IMAGE_TAG)"

# board-service-build: ## Build board-service image
# 	@echo "Building board-service..."
# 	docker build -t $(LOCAL_REGISTRY)/board-service:$(IMAGE_TAG) \
# 		-f services/board-service/docker/Dockerfile .
# 	@echo "Built $(LOCAL_REGISTRY)/board-service:$(IMAGE_TAG)"

# frontend-build: ## Build frontend image
# 	@echo "Building frontend..."
# 	docker build -t $(LOCAL_REGISTRY)/frontend:$(IMAGE_TAG) \
# 		-f services/frontend/Dockerfile services/frontend
# 	@echo "Built $(LOCAL_REGISTRY)/frontend:$(IMAGE_TAG)"

# # -----------------------------------------------------------------------------
# # Load targets (build + push to registry)
# # -----------------------------------------------------------------------------

# auth-service-load: auth-service-build ## Build and push auth-service
# 	docker push $(LOCAL_REGISTRY)/auth-service:$(IMAGE_TAG)
# 	@echo "Pushed $(LOCAL_REGISTRY)/auth-service:$(IMAGE_TAG)"

# board-service-load: board-service-build ## Build and push board-service
# 	docker push $(LOCAL_REGISTRY)/board-service:$(IMAGE_TAG)
# 	@echo "Pushed $(LOCAL_REGISTRY)/board-service:$(IMAGE_TAG)"

# chat-service-load: chat-service-build ## Build and push chat-service
# 	docker push $(LOCAL_REGISTRY)/chat-service:$(IMAGE_TAG)
# 	@echo "Pushed $(LOCAL_REGISTRY)/chat-service:$(IMAGE_TAG)"

# frontend-load: frontend-build ## Build and push frontend
# 	docker push $(LOCAL_REGISTRY)/frontend:$(IMAGE_TAG)
# 	@echo "Pushed $(LOCAL_REGISTRY)/frontend:$(IMAGE_TAG)"

# noti-service-load: noti-service-build ## Build and push noti-service
# 	docker push $(LOCAL_REGISTRY)/noti-service:$(IMAGE_TAG)
# 	@echo "Pushed $(LOCAL_REGISTRY)/noti-service:$(IMAGE_TAG)"

# storage-service-load: storage-service-build ## Build and push storage-service
# 	docker push $(LOCAL_REGISTRY)/storage-service:$(IMAGE_TAG)
# 	@echo "Pushed $(LOCAL_REGISTRY)/storage-service:$(IMAGE_TAG)"

# user-service-load: user-service-build ## Build and push user-service
# 	docker push $(LOCAL_REGISTRY)/user-service:$(IMAGE_TAG)
# 	@echo "Pushed $(LOCAL_REGISTRY)/user-service:$(IMAGE_TAG)"

# video-service-load: video-service-build ## Build and push video-service
# 	docker push $(LOCAL_REGISTRY)/video-service:$(IMAGE_TAG)
# 	@echo "Pushed $(LOCAL_REGISTRY)/video-service:$(IMAGE_TAG)"

# # -----------------------------------------------------------------------------
# # Redeploy targets (rollout restart)
# # -----------------------------------------------------------------------------

# auth-service-redeploy: ## Rollout restart auth-service
# 	kubectl rollout restart deployment/auth-service -n $(K8S_NAMESPACE)
# 	@echo "Rollout restart triggered for auth-service"

# board-service-redeploy: ## Rollout restart board-service
# 	kubectl rollout restart deployment/board-service -n $(K8S_NAMESPACE)
# 	@echo "Rollout restart triggered for board-service"

# chat-service-redeploy: ## Rollout restart chat-service
# 	kubectl rollout restart deployment/chat-service -n $(K8S_NAMESPACE)
# 	@echo "Rollout restart triggered for chat-service"

# frontend-redeploy: ## Rollout restart frontend
# 	kubectl rollout restart deployment/frontend -n $(K8S_NAMESPACE)
# 	@echo "Rollout restart triggered for frontend"

# noti-service-redeploy: ## Rollout restart noti-service
# 	kubectl rollout restart deployment/noti-service -n $(K8S_NAMESPACE)
# 	@echo "Rollout restart triggered for noti-service"

# storage-service-redeploy: ## Rollout restart storage-service
# 	kubectl rollout restart deployment/storage-service -n $(K8S_NAMESPACE)
# 	@echo "Rollout restart triggered for storage-service"

# user-service-redeploy: ## Rollout restart user-service
# 	kubectl rollout restart deployment/user-service -n $(K8S_NAMESPACE)
# 	@echo "Rollout restart triggered for user-service"

# video-service-redeploy: ## Rollout restart video-service
# 	kubectl rollout restart deployment/video-service -n $(K8S_NAMESPACE)
# 	@echo "Rollout restart triggered for video-service"

# # -----------------------------------------------------------------------------
# # All-in-one targets (build + load + redeploy)
# # -----------------------------------------------------------------------------

# auth-service-all: auth-service-load auth-service-redeploy ## Build, push, and redeploy auth-service

# board-service-all: board-service-load board-service-redeploy ## Build, push, and redeploy board-service

# chat-service-all: chat-service-load chat-service-redeploy ## Build, push, and redeploy chat-service

# frontend-all: frontend-load frontend-redeploy ## Build, push, and redeploy frontend

# noti-service-all: noti-service-load noti-service-redeploy ## Build, push, and redeploy noti-service

# storage-service-all: storage-service-load storage-service-redeploy ## Build, push, and redeploy storage-service

# user-service-all: user-service-load user-service-redeploy ## Build, push, and redeploy user-service

# video-service-all: video-service-load video-service-redeploy ## Build, push, and redeploy video-service

# ##@ Utility

# redeploy-all: ## Restart all deployments (pick up new secrets)
# 	@echo "Restarting all deployments in $(K8S_NAMESPACE)..."
# 	kubectl rollout restart deployment -n $(K8S_NAMESPACE)
# 	@echo "All deployments restarted"

# status: ## Show pods status
# 	@echo "=== Kubernetes Pods (ENV=$(ENV), NS=$(K8S_NAMESPACE)) ==="
# 	@kubectl get pods -n $(K8S_NAMESPACE) 2>/dev/null || echo "Namespace not found"

# clean: ## Clean up
# 	./docker/scripts/clean.sh
