# =============================================================================
# Docker Compose Commands
# =============================================================================

##@ Development (Docker Compose)

.PHONY: dev-up dev-down dev-restart dev-logs dev-build dev-clean

dev-up: ## Start all services
	./docker/scripts/dev.sh up

dev-down: ## Stop all services
	./docker/scripts/dev.sh down

dev-restart: ## Restart all services
	./docker/scripts/dev.sh restart

dev-logs: ## View logs
	./docker/scripts/dev.sh logs

dev-build: ## Rebuild all Docker images
	./docker/scripts/dev.sh build

dev-clean: ## Stop and remove all containers, volumes (destructive)
	./docker/scripts/dev.sh clean

##@ Monorepo Build (BuildKit Cache - Fast)

.PHONY: dev-mono-up dev-mono-down dev-mono-build dev-mono-build-parallel

dev-mono-up: ## Start with monorepo build (shared package 1회 컴파일)
	./docker/scripts/dev-mono.sh up

dev-mono-down: ## Stop monorepo dev environment
	./docker/scripts/dev-mono.sh down

dev-mono-build: ## Build Go services only (sequential, uses BuildKit cache)
	./docker/scripts/dev-mono.sh build

dev-mono-build-parallel: ## Build Go services in parallel (faster)
	./docker/scripts/dev-mono.sh build-parallel

##@ SonarQube (Code Quality)

.PHONY: sonar-up sonar-down sonar-logs sonar-status sonar-restart sonar-clean

sonar-up: ## Start SonarQube only (lightweight)
	@echo "Starting SonarQube standalone environment..."
	./docker/scripts/sonar.sh up

sonar-down: ## Stop SonarQube environment
	@echo "Stopping SonarQube standalone environment..."
	./docker/scripts/sonar.sh down

sonar-logs: ## View SonarQube logs
	./docker/scripts/sonar.sh logs

sonar-status: ## Check SonarQube status
	./docker/scripts/sonar.sh status

sonar-restart: ## Restart SonarQube environment
	@echo "Restarting SonarQube standalone environment..."
	./docker/scripts/sonar.sh restart

sonar-clean: ## Clean SonarQube data (destructive)
	@echo "Cleaning SonarQube standalone environment..."
	./docker/scripts/sonar.sh clean
