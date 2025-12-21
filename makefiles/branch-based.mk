# =============================================================================
# Branch-based Helm Commands
# =============================================================================

##@ Branch-based Deployment

.PHONY: deploy-dev deploy-staging deploy-prod
.PHONY: switch-to-dev switch-to-staging switch-to-prod

# Environment to branch mapping
DEV_BRANCH = dev
STAGING_BRANCH = staging  
PROD_BRANCH = prod
MAIN_BRANCH = main

deploy-dev: ## Deploy to dev environment (switches to dev branch)
	@echo "🌿 Switching to dev branch and deploying..."
	@git stash push -m "Auto-stash before dev deploy" 2>/dev/null || true
	@git checkout $(DEV_BRANCH)
	@git pull origin $(DEV_BRANCH)
	@cd main && $(MAKE) helm-install-all K8S_NAMESPACE=wealist-dev
	@echo "✅ Dev deployment complete!"

deploy-staging: ## Deploy to staging environment (switches to staging branch)
	@echo "🌿 Switching to staging branch and deploying..."
	@git stash push -m "Auto-stash before staging deploy" 2>/dev/null || true
	@git checkout $(STAGING_BRANCH)
	@git pull origin $(STAGING_BRANCH)
	@cd main && $(MAKE) helm-install-all K8S_NAMESPACE=wealist-staging
	@echo "✅ Staging deployment complete!"

deploy-prod: ## Deploy to production (switches to prod branch)
	@echo "🌿 Switching to prod branch and deploying..."
	@echo "⚠️  WARNING: Deploying to PRODUCTION!"
	@read -p "Are you sure? (y/N): " confirm && [ "$$confirm" = "y" ]
	@git stash push -m "Auto-stash before prod deploy" 2>/dev/null || true
	@git checkout $(PROD_BRANCH)
	@git pull origin $(PROD_BRANCH)
	@cd main && $(MAKE) helm-install-all K8S_NAMESPACE=wealist-prod
	@echo "✅ Production deployment complete!"

switch-to-dev: ## Switch to dev branch
	@git stash push -m "Auto-stash before branch switch" 2>/dev/null || true
	@git checkout $(DEV_BRANCH)
	@git pull origin $(DEV_BRANCH)
	@echo "✅ Switched to dev branch"

switch-to-staging: ## Switch to staging branch  
	@git stash push -m "Auto-stash before branch switch" 2>/dev/null || true
	@git checkout $(STAGING_BRANCH)
	@git pull origin $(STAGING_BRANCH)
	@echo "✅ Switched to staging branch"

switch-to-prod: ## Switch to prod branch
	@git stash push -m "Auto-stash before branch switch" 2>/dev/null || true
	@git checkout $(PROD_BRANCH)
	@git pull origin $(PROD_BRANCH)
	@echo "✅ Switched to prod branch"

switch-to-main: ## Switch back to main branch
	@git stash push -m "Auto-stash before branch switch" 2>/dev/null || true
	@git checkout $(MAIN_BRANCH)
	@git pull origin $(MAIN_BRANCH)
	@echo "✅ Switched to main branch"

##@ Branch Management

merge-main-to-dev: ## Merge main branch changes to dev
	@echo "🔄 Merging main → dev..."
	@git checkout $(DEV_BRANCH)
	@git pull origin $(DEV_BRANCH)
	@git merge origin/$(MAIN_BRANCH)
	@echo "✅ Merged main to dev"

merge-dev-to-staging: ## Merge dev branch changes to staging
	@echo "🔄 Merging dev → staging..."
	@git checkout $(STAGING_BRANCH)
	@git pull origin $(STAGING_BRANCH)
	@git merge origin/$(DEV_BRANCH)
	@echo "✅ Merged dev to staging"

merge-staging-to-prod: ## Merge staging branch changes to prod
	@echo "🔄 Merging staging → prod..."
	@echo "⚠️  WARNING: Merging to PRODUCTION branch!"
	@read -p "Are you sure? (y/N): " confirm && [ "$$confirm" = "y" ]
	@git checkout $(PROD_BRANCH)
	@git pull origin $(PROD_BRANCH)
	@git merge origin/$(STAGING_BRANCH)
	@echo "✅ Merged staging to prod"

promote-to-prod: ## Full promotion pipeline: main → dev → staging → prod
	@echo "🚀 Starting full promotion pipeline..."
	@$(MAKE) merge-main-to-dev
	@$(MAKE) merge-dev-to-staging  
	@$(MAKE) merge-staging-to-prod
	@echo "✅ Full promotion complete!"

##@ Status & Info

branch-status: ## Show current branch and deployment status
	@echo "📊 Branch Status:"
	@echo "  Current branch: $$(git branch --show-current)"
	@echo "  Last commit: $$(git log -1 --oneline)"
	@echo ""
	@echo "📋 Available branches:"
	@git branch -a | grep -E "(dev|staging|prod|main)" | sed 's/^/  /'

deployment-info: ## Show deployment information for current branch
	@CURRENT_BRANCH=$$(git branch --show-current); \
	echo "📋 Deployment Info for branch: $$CURRENT_BRANCH"; \
	case $$CURRENT_BRANCH in \
		$(DEV_BRANCH)) echo "  Environment: Development"; echo "  Namespace: wealist-dev" ;; \
		$(STAGING_BRANCH)) echo "  Environment: Staging"; echo "  Namespace: wealist-staging" ;; \
		$(PROD_BRANCH)) echo "  Environment: Production"; echo "  Namespace: wealist-prod" ;; \
		$(MAIN_BRANCH)) echo "  Environment: Source of Truth (no deployment)" ;; \
		*) echo "  Environment: Unknown branch" ;; \
	esac