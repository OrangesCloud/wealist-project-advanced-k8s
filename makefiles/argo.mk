# ============================================
# ArgoCD Makefile
# ============================================
.PHONY: argo-help cluster-up cluster-down bootstrap deploy argo-clean argo-status helm-install-infra all
.PHONY: setup-local-argocd kind-setup-ecr load-infra-images-ecr
.PHONY: argo-deploy-dev argo-deploy-dev argo-deploy-prod

# 색상
GREEN  := \033[0;32m
YELLOW := \033[1;33m
RED    := \033[0;31m
NC     := \033[0m

# 변수
CLUSTER_NAME ?= wealist-dev
SEALED_SECRETS_KEY ?= k8s/argocd/scripts/sealed-secrets-dev-20251218-152119.key
ENVIRONMENT ?= dev
ENV ?= dev

argo-help: ## [ArgoCD] 도움말 표시
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "  Wealist Platform - Make Commands"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo ""
	@echo "빠른 시작:"
	@echo "  make kind-dev-setup  - Dev 환경 전체 설정"
	@echo ""
	@echo "단계별 실행:"
	@echo "  make cluster-up          - Kind 클러스터 생성"
	@echo "  make argo-install-simple - ArgoCD 설치"
	@echo "  make argo-deploy-dev - Applications 배포"
	@echo ""
	@echo "관리:"
	@echo "  make argo-status      - 전체 상태 확인"
	@echo "  make logs             - ArgoCD 로그 확인"
	@echo "  make ui               - ArgoCD UI 열기"
	@echo "  make argo-clean       - 모든 리소스 삭제"
	@echo "  make cluster-down     - 클러스터 삭제"
	@echo ""
	@echo "ESO (External Secrets):"
	@echo "  make eso-status       - ESO 상태 확인"
	@echo "  make eso-sync         - Secret 강제 동기화"
	@echo "  make verify-secrets   - Secret 확인"
	@echo ""
	@echo "변수:"
	@echo "  ENVIRONMENT=$(ENVIRONMENT)"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

argo-setup: ## ArgoCD 설치 (인터랙티브)
	@echo ""
	@echo -e "$(YELLOW)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo -e "$(YELLOW)  ArgoCD 설치 옵션 선택$(NC)"
	@echo -e "$(YELLOW)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@if [ -f "$(SEALED_SECRETS_KEY)" ]; then \
		echo -e "$(GREEN)✅ Sealed Secrets 키 발견: $(SEALED_SECRETS_KEY)$(NC)"; \
		echo ""; \
		echo "1) 키 사용해서 설치 (Sealed Secrets 포함)"; \
		echo "2) ArgoCD만 설치 (Sealed Secrets 없이) - 권장"; \
		echo "3) 새 키 생성해서 설치"; \
		echo ""; \
		read -p "선택 [1/2/3] (기본: 2): " choice; \
		case $$choice in \
			1) $(MAKE) bootstrap ;; \
			3) $(MAKE) bootstrap-without-key ;; \
			*) $(MAKE) argo-install-simple ;; \
		esac; \
	else \
		echo -e "$(YELLOW)⚠️  Sealed Secrets 키 없음$(NC)"; \
		echo ""; \
		echo "1) ArgoCD만 설치 (Sealed Secrets 없이) - 권장"; \
		echo "2) 새 키 생성해서 설치 (Sealed Secrets 포함)"; \
		echo "3) 키 파일 경로 직접 입력"; \
		echo ""; \
		read -p "선택 [1/2/3] (기본: 1): " choice; \
		case $$choice in \
			2) $(MAKE) bootstrap-without-key ;; \
			3) read -p "키 파일 경로: " keypath; $(MAKE) bootstrap SEALED_SECRETS_KEY=$$keypath ;; \
			*) $(MAKE) argo-install-simple ;; \
		esac; \
	fi
	@echo ""
	@echo -e "$(GREEN)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo -e "$(GREEN)✅ ArgoCD 설치 완료!$(NC)"
	@echo -e "$(GREEN)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@echo "ArgoCD UI: https://localhost:8079"
	@echo "Username: admin"
	@echo "Password: $$(kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d)"
	@echo ""
	@echo "다음 명령어로 포트 포워딩:"
	@echo "  make ui"

# ============================================
# 클러스터 관리
# ============================================

cluster-up: ## Kind 클러스터 + 로컬 레지스트리 + 이미지 준비
	@echo -e "$(YELLOW)📦 Kind 클러스터 + 로컬 환경 설정 중...$(NC)"
	@echo -e "$(YELLOW)ℹ️  'make kind-dev-setup' 사용을 권장합니다.$(NC)"
	@if kind get clusters | grep -q "$(CLUSTER_NAME)"; then \
		echo -e "$(YELLOW)⚠️  클러스터가 이미 존재합니다: $(CLUSTER_NAME)$(NC)"; \
		read -p "삭제하고 다시 만들까요? (y/N): " answer; \
		if [ "$$answer" = "y" ] || [ "$$answer" = "Y" ]; then \
			$(MAKE) cluster-down; \
		else \
			echo "기존 클러스터를 사용합니다."; \
			$(MAKE) load-images-only; \
			exit 0; \
		fi; \
	fi
	@echo -e "$(YELLOW)🏗️  Step 1: 클러스터 생성...$(NC)"
	@$(MAKE) kind-dev-setup
	@kubectl cluster-info
	@echo -e "$(GREEN)✅ 클러스터 + 로컬 환경 준비 완료$(NC)"

load-images-only: ## 인프라 이미지만 로드 (기존 클러스터용)
	@echo -e "$(YELLOW)📦 인프라 이미지 로드...$(NC)"
	@if [ -f "k8s/helm/scripts/dev/1.load_infra_images.sh" ]; then \
		chmod +x k8s/helm/scripts/dev/1.load_infra_images.sh; \
		./k8s/helm/scripts/dev/1.load_infra_images.sh; \
	else \
		echo -e "$(RED)❌ 1.load_infra_images.sh not found$(NC)"; \
	fi
	@echo -e "$(GREEN)✅ 이미지 로드 완료$(NC)"
	@echo -e "$(YELLOW)ℹ️  서비스 이미지는 AWS ECR에서 직접 pull됩니다.$(NC)"

cluster-down: ## Kind 클러스터 삭제
	@echo -e "$(YELLOW)🗑️  클러스터 삭제 중...$(NC)"
	@kind delete cluster --name $(CLUSTER_NAME) || true
	@echo -e "$(GREEN)✅ 클러스터 삭제 완료$(NC)"

# ============================================
# Bootstrap
# ============================================

bootstrap: check-key ## ArgoCD & Sealed Secrets 설치 (키 복원 포함)
	@echo -e "$(YELLOW)🚀 Bootstrap 시작...$(NC)"
	@chmod +x k8s/argocd/scripts/deploy-argocd.sh
	@./k8s/argocd/scripts/deploy-argocd.sh $(SEALED_SECRETS_KEY)

check-key: ## Sealed Secrets 키 파일 확인
	@if [ ! -f "$(SEALED_SECRETS_KEY)" ]; then \
		echo -e "$(RED)❌ 키 파일을 찾을 수 없습니다: $(SEALED_SECRETS_KEY)$(NC)"; \
		echo ""; \
		echo "옵션:"; \
		echo "  1. 키 파일을 현재 디렉토리에 배치"; \
		echo "  2. SEALED_SECRETS_KEY 변수로 경로 지정:"; \
		echo "     make bootstrap SEALED_SECRETS_KEY=path/to/key.yaml"; \
		echo "  3. 키 없이 진행 (새 키 생성):"; \
		echo "     make bootstrap-without-key"; \
		echo ""; \
		exit 1; \
	fi
	@echo -e "$(GREEN)✅ 키 파일 확인: $(SEALED_SECRETS_KEY)$(NC)"

bootstrap-without-key: ## 키 없이 Bootstrap (새 키 생성)
	@echo -e "$(YELLOW)⚠️  키 없이 진행 - 새 키가 생성됩니다$(NC)"
	@chmod +x k8s/argocd/scripts/deploy-argocd.sh
	@./k8s/argocd/scripts/deploy-argocd.sh

argo-install-simple: ## ArgoCD만 간단 설치 (Sealed Secrets 없이)
	@echo "ArgoCD 설치 중..."
	@kubectl create namespace argocd 2>/dev/null || true
	@kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
	@echo "ArgoCD 설치 완료, Pod 준비 대기 중..."
	@kubectl wait --for=condition=available --timeout=300s deployment/argocd-server -n argocd || echo "WARNING: ArgoCD server not ready yet"
	@echo ""
	@echo "ArgoCD sub-path 설정 중 (/api/argo)..."
	@# ArgoCD 2.0+: argocd-cmd-params-cm에서 server 설정 관리
	@kubectl patch configmap argocd-cmd-params-cm -n argocd --type merge \
		-p '{"data":{"server.insecure":"true","server.rootpath":"/api/argo","server.basehref":"/api/argo"}}' 2>/dev/null || true
	@# 기존 argocd-cm도 설정 (호환성)
	@kubectl patch configmap argocd-cm -n argocd --type merge \
		-p '{"data":{"server.rootpath":"/api/argo","server.insecure":"true"}}' 2>/dev/null || true
	@kubectl rollout restart deployment argocd-server -n argocd 2>/dev/null || true
	@kubectl rollout status deployment argocd-server -n argocd --timeout=120s 2>/dev/null || true
	@echo ""
	@echo "ReferenceGrant 적용 중 (cross-namespace routing)..."
	@kubectl apply -f k8s/argocd/base/referencegrant-argocd.yaml 2>/dev/null || true
	@echo ""
	@echo "=============================================="
	@echo "  ✅ ArgoCD 설치 완료!"
	@echo "=============================================="
	@echo ""
	@echo "  웹 접속 (Istio Gateway 통해):"
	@echo "    http://localhost:8080/api/argo"
	@echo "    https://dev.wealist.co.kr/api/argo"
	@echo ""
	@echo "  포트 포워딩 (직접 접속):"
	@echo "    kubectl port-forward svc/argocd-server -n argocd 8079:443"
	@echo "    https://localhost:8079"
	@echo ""
	@echo "  로그인 정보:"
	@echo "    User: admin"
	@echo "    Password: $$(kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath='{.data.password}' 2>/dev/null | base64 -d || echo '(아직 생성 안됨)')"
	@echo ""
	@echo "  Git 레포 연결:"
	@echo "    make argo-add-repo"
	@echo "=============================================="

argo-add-repo: ## Git 레포지토리 ArgoCD에 등록
	@echo "Git 레포지토리를 ArgoCD에 등록합니다."
	@echo ""
	@echo "GitHub Personal Access Token이 필요합니다."
	@echo "Token 생성: https://github.com/settings/tokens (repo 권한 필요)"
	@echo ""
	@read -p "GitHub Username: " gh_user; \
	read -p "GitHub Token: " gh_token; \
	read -p "Repository URL (예: https://github.com/org/repo.git): " repo_url; \
	kubectl -n argocd create secret generic repo-creds \
		--from-literal=url=$$repo_url \
		--from-literal=username=$$gh_user \
		--from-literal=password=$$gh_token \
		--dry-run=client -o yaml | kubectl apply -f -; \
	echo ""; \
	echo "✅ Git 레포 등록 완료: $$repo_url"

argo-ui: ## ArgoCD UI 포트 포워딩
	@echo "ArgoCD UI 포트 포워딩: https://localhost:8079"
	@echo "종료하려면 Ctrl+C"
	@kubectl port-forward svc/argocd-server -n argocd 8079:443

# ============================================
# 배포
# ============================================

argo-deploy-dev: ## [ArgoCD] Dev 환경 Applications 배포 (Root App 생성)
	@echo -e "$(YELLOW)🎯 Dev Applications 배포 중...$(NC)"
	@echo ""
	@echo "1. AppProject 생성..."
	@kubectl apply -f k8s/argocd/apps/dev/project.yaml || true
	@kubectl apply -f k8s/argocd/projects/wealist-dev.yaml || true
	@echo ""
	@echo "2. Root Application 생성..."
	@kubectl apply -f k8s/argocd/apps/dev/root-app.yaml || true
	@echo ""
	@echo "3. 모든 Dev Apps 적용 중..."
	@for file in k8s/argocd/apps/dev/*.yaml; do \
		if [ -f "$$file" ]; then \
			kubectl apply -f $$file 2>/dev/null || true; \
		fi; \
	done
	@echo ""
	@echo "4. ArgoCD Sync 대기 중..."
	@sleep 5
	@echo ""
	@echo -e "$(GREEN)✅ Dev 배포 완료$(NC)"
	@echo ""
	@echo "Applications 확인:"
	@kubectl get applications -n argocd
	@echo ""
	@echo -e "$(YELLOW)📝 ArgoCD가 자동으로 모든 앱을 Sync합니다.$(NC)"
	@echo "   상태 확인: make argo-status"

argo-deploy-dev: ## [ArgoCD] Dev 환경 Applications 배포
	@echo -e "$(YELLOW)🎯 Dev Applications 배포 중...$(NC)"
	@kubectl apply -f k8s/argocd/apps/dev/project.yaml || true
	@kubectl apply -f k8s/argocd/projects/wealist-dev.yaml || true
	@kubectl apply -f k8s/argocd/apps/dev/root-app.yaml || true
	@echo -e "$(GREEN)✅ Dev 배포 완료$(NC)"

argo-deploy-prod: ## [ArgoCD] Prod 환경 Applications 배포
	@echo -e "$(YELLOW)🎯 Prod Applications 배포 중...$(NC)"
	@kubectl apply -f k8s/argocd/projects/wealist-prod.yaml || true
	@kubectl apply -f k8s/argocd/apps/prod/root-app.yaml || true
	@echo -e "$(GREEN)✅ Prod 배포 완료$(NC)"

# ============================================
# 상태 확인
# ============================================

# argo-status 각 항목 설명:
# - ArgoCD Pods: ArgoCD 시스템 컴포넌트 (server, repo-server, redis, controller 등)
# - ESO: External Secrets Operator - AWS Secrets Manager에서 시크릿 동기화
# - Applications: ArgoCD Application CRD 개수 (Git에서 읽어 배포할 앱 정의)
#   - Synced = Git과 클러스터 상태 일치
#   - OutOfSync = Git과 클러스터 상태 불일치 (sync 필요)
argo-status: ## [ArgoCD] 전체 상태 확인
	@echo -e "$(YELLOW)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo -e "$(YELLOW)📊 시스템 상태$(NC)"
	@echo -e "$(YELLOW)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@echo "🏗️  클러스터:"
	@kubectl cluster-info | head -1 || echo "클러스터 없음"
	@echo ""
	@echo "📦 ArgoCD Pods:"
	@kubectl get pods -n argocd --no-headers 2>/dev/null | grep -E "Running|Ready" | wc -l | xargs -I {} echo "  Running: {} pods"
	@echo ""
	@echo "🔐 ESO (External Secrets Operator):"
	@kubectl get pods -n external-secrets --no-headers 2>/dev/null | grep -E "Running" | wc -l | xargs -I {} echo "  Running: {} pods"
	@kubectl get externalsecret -n wealist-$(ENVIRONMENT) --no-headers 2>/dev/null | wc -l | xargs -I {} echo "  ExternalSecrets: {}"
	@kubectl get externalsecret -n wealist-$(ENVIRONMENT) --no-headers 2>/dev/null | grep -i "SecretSynced" | wc -l | xargs -I {} echo "  Synced: {}"
	@echo ""
	@echo "🎯 Applications:"
	@kubectl get applications -n argocd --no-headers 2>/dev/null | wc -l | xargs -I {} echo "  Total: {}"
	@kubectl get applications -n argocd --no-headers 2>/dev/null | grep Synced | wc -l | xargs -I {} echo "  Synced: {}"
	@echo ""
	@echo "🗝️  Secrets (wealist-$(ENVIRONMENT)):"
	@kubectl get secrets -n wealist-$(ENVIRONMENT) --no-headers 2>/dev/null | wc -l | xargs -I {} echo "  Total: {}"
	@echo ""
	@echo -e "$(YELLOW)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"

status-detail: ## 상세 상태 확인
	@echo "📦 ArgoCD Pods:"
	@kubectl get pods -n argocd
	@echo ""
	@echo "🔐 ESO Pods:"
	@kubectl get pods -n external-secrets
	@echo ""
	@echo "🎯 Applications:"
	@kubectl get applications -n argocd
	@echo ""
	@echo "🔒 ExternalSecrets:"
	@kubectl get externalsecrets -A
	@echo ""
	@echo "🗝️  Secrets:"
	@kubectl get secrets -n wealist-$(ENVIRONMENT)

# ============================================
# UI 및 로그
# ============================================

ui: ## ArgoCD UI 접속 (포트 포워딩)
	@echo -e "$(GREEN)🌐 ArgoCD UI 접속...$(NC)"
	@echo ""
	@echo "URL: https://localhost:8079"
	@echo "Username: admin"
	@echo "Password: $$(kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d)"
	@echo ""
	@echo "브라우저에서 https://localhost:8079 를 열어주세요"
	@echo "(Ctrl+C로 중지)"
	@echo ""
	@kubectl port-forward svc/argocd-server -n argocd 8079:443

logs: ## ArgoCD 로그 확인
	@echo "ArgoCD Application Controller 로그:"
	@kubectl logs -n argocd -l app.kubernetes.io/name=argocd-application-controller --tail=50

logs-eso: ## ESO Controller 로그
	@echo "ESO Controller 로그:"
	@kubectl logs -n external-secrets -l app.kubernetes.io/name=external-secrets --tail=50

# ============================================
# 정리
# ============================================

argo-clean: ## [ArgoCD] 모든 리소스 삭제 (클러스터는 유지)
	@echo -e "$(YELLOW)🗑️  리소스 삭제 중...$(NC)"
	@kubectl delete namespace wealist-$(ENVIRONMENT) --ignore-not-found=true
	@kubectl delete namespace argocd --ignore-not-found=true
	@echo -e "$(GREEN)✅ 리소스 삭제 완료$(NC)"

argo-clean-all: cluster-down ## [ArgoCD] 클러스터 포함 모든 것 삭제
	@echo -e "$(GREEN)✅ 전체 정리 완료$(NC)"

# ============================================
# 개발 편의 기능
# ============================================

restart-argocd: ## ArgoCD 재시작
	@echo -e "$(YELLOW)🔄 ArgoCD 재시작...$(NC)"
	@kubectl rollout restart deployment -n argocd
	@kubectl rollout status deployment -n argocd

restart-eso: ## ESO Controller 재시작
	@echo -e "$(YELLOW)🔄 ESO Controller 재시작...$(NC)"
	@kubectl rollout restart deployment -n external-secrets
	@kubectl rollout status deployment -n external-secrets --timeout=120s
	@echo -e "$(GREEN)✅ 재시작 완료$(NC)"

sync-all: ## 모든 Applications Sync
	@echo -e "$(YELLOW)🔄 전체 Sync...$(NC)"
	@kubectl get applications -n argocd -o name | xargs -I {} kubectl patch {} -n argocd --type merge -p '{"operation":{"initiatedBy":{"username":"admin"},"sync":{"revision":"HEAD"}}}'
	@echo -e "$(GREEN)✅ Sync 완료$(NC)"

# ============================================
# 트러블슈팅
# ============================================

debug: ## 디버깅 정보 출력
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "🔍 디버깅 정보"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo ""
	@echo "클러스터 정보:"
	@kubectl cluster-info
	@echo ""
	@echo "Nodes:"
	@kubectl get nodes
	@echo ""
	@echo "Namespaces:"
	@kubectl get namespaces
	@echo ""
	@echo "ArgoCD Applications:"
	@kubectl get applications -n argocd
	@echo ""
	@echo "ExternalSecrets 상태:"
	@kubectl get externalsecrets -A
	@echo ""
	@echo "ESO Controller 로그 (last 20):"
	@kubectl logs -n external-secrets -l app.kubernetes.io/name=external-secrets --tail=20 2>/dev/null || echo "ESO 미설치"
	@echo ""
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

verify-secrets: ## Secrets 확인 (ESO 동기화 상태)
	@echo -e "$(YELLOW)🔐 Secrets 확인...$(NC)"
	@echo ""
	@echo "ExternalSecrets:"
	@kubectl get externalsecrets -n wealist-$(ENVIRONMENT)
	@echo ""
	@echo "Secrets:"
	@kubectl get secrets -n wealist-$(ENVIRONMENT)
	@echo ""
	@if kubectl get secret wealist-shared-secret -n wealist-$(ENVIRONMENT) &> /dev/null; then \
		echo -e "$(GREEN)✅ wealist-shared-secret 존재$(NC)"; \
		kubectl describe secret wealist-shared-secret -n wealist-$(ENVIRONMENT) | grep -A 20 "Data:"; \
	else \
		echo -e "$(RED)❌ wealist-shared-secret 없음$(NC)"; \
		echo ""; \
		echo "ExternalSecret 상태:"; \
		kubectl describe externalsecret wealist-shared-secret -n wealist-$(ENVIRONMENT) 2>/dev/null || echo "ExternalSecret도 없음"; \
	fi
# ... (기존 내용 유지) ...

# ============================================
# 로컬 개발 (Kind + Registry) - ArgoCD용
# ============================================
# NOTE: kind-setup은 kind.mk에서 정의됨 (Istio Ambient + 로컬 레지스트리)
# 아래는 ECR 직접 연결이 필요한 ArgoCD 환경용

setup-local-argocd: ## [ArgoCD] 로컬 개발 환경 전체 설정 (ECR + Bootstrap)
	$(MAKE) kind-setup-ecr
	$(MAKE) load-infra-images-ecr
	$(MAKE) bootstrap
	$(MAKE) deploy

kind-setup-ecr: ## [ArgoCD] Kind 클러스터 + ECR 직접 연결 (dev)
	@echo -e "$(YELLOW)🏗️  Kind 클러스터 + ECR 설정...$(NC)"
	@if [ -f "k8s/helm/scripts/dev/0.setup-cluster.sh" ]; then \
		chmod +x k8s/helm/scripts/dev/0.setup-cluster.sh; \
		./k8s/helm/scripts/dev/0.setup-cluster.sh; \
	else \
		echo -e "$(RED)❌ 0.setup-cluster.sh not found$(NC)"; \
		exit 1; \
	fi
	@echo -e "$(GREEN)✅ Kind 클러스터 + ECR 준비 완료$(NC)"

kind-dev-setup: ## [ArgoCD] Kind 클러스터 + ECR + ArgoCD + 앱 배포 (dev 환경)
	@echo -e "$(YELLOW)🏗️  Kind 클러스터 + ECR 설정 (dev)...$(NC)"
	@if [ -f "k8s/helm/scripts/dev/0.setup-cluster.sh" ]; then \
		chmod +x k8s/helm/scripts/dev/0.setup-cluster.sh; \
		./k8s/helm/scripts/dev/0.setup-cluster.sh; \
	else \
		echo -e "$(RED)❌ dev/0.setup-cluster.sh not found$(NC)"; \
		exit 1; \
	fi
	@echo -e "$(GREEN)✅ Kind 클러스터 (dev) 준비 완료$(NC)"
	@echo ""
	@echo -e "$(YELLOW)🐘 Host PostgreSQL 초기화 (dev)...$(NC)"
	@if [ -f "scripts/init-local-postgres.sh" ]; then \
		chmod +x scripts/init-local-postgres.sh; \
		if [ "$$(uname)" = "Darwin" ]; then \
			DEV_DB_PASSWORD=$${DEV_DB_PASSWORD:-wealist-dev-password} ./scripts/init-local-postgres.sh dev; \
		else \
			sudo DEV_DB_PASSWORD=$${DEV_DB_PASSWORD:-wealist-dev-password} ./scripts/init-local-postgres.sh dev; \
		fi; \
	else \
		echo -e "$(YELLOW)⚠️  init-local-postgres.sh not found, skipping DB init$(NC)"; \
	fi
	@echo ""
	@echo -e "$(YELLOW)🚀 ArgoCD 설치 중...$(NC)"
	$(MAKE) argo-install-simple
	@echo ""
	@echo -e "$(YELLOW)🔐 Git 레포지토리 등록 중...$(NC)"
	$(MAKE) argo-add-repo-auto
	@echo ""
	@echo -e "$(YELLOW)🎯 Dev Applications 배포 중...$(NC)"
	$(MAKE) argo-deploy-dev
	@echo ""
	@echo -e "$(GREEN)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo -e "$(GREEN)✅ Dev 환경 전체 설정 완료!$(NC)"
	@echo -e "$(GREEN)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@echo "ArgoCD UI: https://dev.wealist.co.kr/api/argo"
	@echo "Username: admin"
	@echo "Password: $$(kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath='{.data.password}' 2>/dev/null | base64 -d || echo '(생성 중...)')"
	@echo ""
	@echo "상태 확인: make argo-status"

# ============================================
# 리셋 명령어
# ============================================

# kind-dev-reset: 클러스터 완전 리셋 (삭제 + 재생성)
# - Kind 클러스터 삭제 (ArgoCD, Helm, Pod 전부 삭제)
# - 로컬 변경사항 제거 (git checkout)
# - 클러스터 + ArgoCD + 앱 전부 새로 생성
kind-dev-reset: ## [Reset] Dev 클러스터 완전 리셋 (삭제 후 재생성)
	@echo -e "$(RED)⚠️  Dev 클러스터를 완전히 리셋합니다...$(NC)"
	@echo ""
	@read -p "정말 리셋하시겠습니까? (y/N): " confirm; \
	if [ "$$confirm" = "y" ] || [ "$$confirm" = "Y" ]; then \
		echo ""; \
		echo -e "$(YELLOW)1. Kind 클러스터 삭제 중...$(NC)"; \
		kind delete cluster --name wealist 2>/dev/null || true; \
		echo ""; \
		echo -e "$(YELLOW)2. 로컬 변경사항 정리 중...$(NC)"; \
		git checkout -- . 2>/dev/null || true; \
		echo ""; \
		echo -e "$(YELLOW)3. Dev 클러스터 재생성 중...$(NC)"; \
		$(MAKE) kind-dev-setup; \
	else \
		echo "리셋 취소됨"; \
	fi

kind-dev-clean: ## [Reset] Dev 클러스터만 삭제 (재생성 없음)
	@echo -e "$(RED)🗑️  Dev 클러스터 삭제 중...$(NC)"
	kind delete cluster --name wealist 2>/dev/null || echo "클러스터 없음"
	@echo -e "$(GREEN)✅ 클러스터 삭제 완료$(NC)"
	@echo ""
	@echo "재생성: make kind-dev-setup"

argo-reset-apps: ## [Reset] ArgoCD 앱만 리셋 (클러스터 유지)
	@echo -e "$(YELLOW)🔄 ArgoCD 앱 리셋 중...$(NC)"
	kubectl delete applications --all -n argocd 2>/dev/null || true
	@echo ""
	@echo -e "$(YELLOW)📦 앱 재생성 중...$(NC)"
	$(MAKE) argo-deploy-dev
	@echo -e "$(GREEN)✅ ArgoCD 앱 리셋 완료$(NC)"

# GitHub 토큰: 환경변수 또는 CLI 입력
argo-add-repo-auto: ## Git 레포 자동 등록 (CLI 입력 또는 환경변수 GITHUB_TOKEN)
	@GITHUB_USER=$${GITHUB_USER:-212clab}; \
	REPO_URL="https://github.com/212clab/wealist-project-advanced-k8s-forked.git"; \
	if [ -z "$$GITHUB_TOKEN" ]; then \
		echo ""; \
		echo "GitHub Personal Access Token이 필요합니다."; \
		echo "Token 생성: https://github.com/settings/tokens (repo 권한)"; \
		echo ""; \
		read -p "GitHub Token: " GITHUB_TOKEN; \
	fi; \
	echo "Git 레포 등록: $$REPO_URL (User: $$GITHUB_USER)"; \
	kubectl -n argocd create secret generic repo-creds \
		--from-literal=url=$$REPO_URL \
		--from-literal=username=$$GITHUB_USER \
		--from-literal=password=$$GITHUB_TOKEN \
		--dry-run=client -o yaml | kubectl apply -f -; \
	echo -e "$(GREEN)✅ Git 레포 등록 완료$(NC)"

load-infra-images-ecr: ## [ArgoCD] 인프라 이미지 로드
	@echo -e "$(YELLOW)📦 인프라 이미지 로드 중...$(NC)"
	@if [ -f "k8s/helm/scripts/dev/1.load_infra_images.sh" ]; then \
		chmod +x k8s/helm/scripts/dev/1.load_infra_images.sh; \
		./k8s/helm/scripts/dev/1.load_infra_images.sh; \
	else \
		echo -e "$(RED)❌ 1.load_infra_images.sh not found$(NC)"; \
		exit 1; \
	fi
	@echo -e "$(GREEN)✅ 인프라 이미지 로드 완료$(NC)"

check-images: ## 로컬 레지스트리 이미지 확인
	@echo -e "$(YELLOW)🔍 로컬 레지스트리 이미지 확인...$(NC)"
	@echo ""
	@echo "Registry catalog:"
	@curl -s http://localhost:5001/v2/_catalog | jq -r '.repositories[]' || echo "No images found"
	@echo ""
	@echo "서비스 이미지 확인:"
	@for svc in auth-service user-service board-service chat-service noti-service storage-service video-service; do \
		echo -n "  $$svc: "; \
		if curl -sf "http://localhost:5001/v2/$$svc/tags/list" > /dev/null 2>&1; then \
			echo -e "$(GREEN)✅$(NC)"; \
		else \
			echo -e "$(RED)❌$(NC)"; \
		fi; \
	done

# ============================================
# External Secrets Operator (ESO)
# ============================================

eso-install: ## [ESO] External Secrets Operator 설치
	@echo -e "$(YELLOW)🔐 External Secrets Operator 설치 중...$(NC)"
	@kubectl create namespace external-secrets 2>/dev/null || true
	@helm repo add external-secrets https://charts.external-secrets.io 2>/dev/null || true
	@helm repo update
	@helm upgrade --install external-secrets external-secrets/external-secrets \
		--namespace external-secrets \
		--set installCRDs=true \
		--wait --timeout 5m
	@echo -e "$(GREEN)✅ ESO 설치 완료$(NC)"

eso-setup-aws: ## [ESO] AWS 자격증명 Secret 생성 (ESO가 AWS Secrets Manager 접근용)
	@echo -e "$(YELLOW)🔐 AWS 자격증명 설정 중...$(NC)"
	@echo ""
	@ACCESS_KEY="$$AWS_ACCESS_KEY_ID"; \
	SECRET_KEY="$$AWS_SECRET_ACCESS_KEY"; \
	if [ -z "$$ACCESS_KEY" ] || [ -z "$$SECRET_KEY" ]; then \
		ACCESS_KEY=$$(aws configure get aws_access_key_id 2>/dev/null || echo ""); \
		SECRET_KEY=$$(aws configure get aws_secret_access_key 2>/dev/null || echo ""); \
	fi; \
	if [ -z "$$ACCESS_KEY" ] || [ -z "$$SECRET_KEY" ]; then \
		echo "AWS 자격증명을 입력하세요:"; \
		echo ""; \
		printf "AWS Access Key ID: "; \
		read ACCESS_KEY; \
		printf "AWS Secret Access Key: "; \
		read -s SECRET_KEY; \
		echo ""; \
	fi; \
	if [ -z "$$ACCESS_KEY" ] || [ -z "$$SECRET_KEY" ]; then \
		echo ""; \
		echo -e "$(RED)❌ AWS 자격증명이 입력되지 않았습니다$(NC)"; \
		exit 1; \
	fi; \
	kubectl create namespace external-secrets 2>/dev/null || true; \
	kubectl delete secret aws-credentials -n external-secrets 2>/dev/null || true; \
	kubectl create secret generic aws-credentials \
		--from-literal=access-key="$$ACCESS_KEY" \
		--from-literal=secret-access-key="$$SECRET_KEY" \
		-n external-secrets; \
	echo -e "$(GREEN)✅ AWS 자격증명 Secret 생성 완료$(NC)"

eso-apply-dev: ## [ESO] Dev용 ClusterSecretStore + ExternalSecret 적용
	@echo -e "$(YELLOW)🔐 ESO Dev 설정 적용 중...$(NC)"
	@kubectl apply -f k8s/argocd/base/external-secrets/dev/cluster-secret-store-dev.yaml
	@kubectl apply -f k8s/argocd/base/external-secrets/dev/external-secret-shared.yaml
	@echo ""
	@echo "ExternalSecret 상태 확인 중..."
	@sleep 3
	@kubectl get externalsecret -n wealist-dev
	@echo -e "$(GREEN)✅ ESO Dev 설정 완료$(NC)"

eso-status: ## [ESO] ExternalSecret 상태 확인
	@echo -e "$(YELLOW)🔐 External Secrets 상태$(NC)"
	@echo ""
	@echo "ClusterSecretStore:"
	@kubectl get clustersecretstores 2>/dev/null || echo "  없음"
	@echo ""
	@echo "ExternalSecrets:"
	@kubectl get externalsecrets -A 2>/dev/null || echo "  없음"
	@echo ""
	@echo "ESO Pods:"
	@kubectl get pods -n external-secrets 2>/dev/null || echo "  ESO 미설치"

eso-sync: ## [ESO] ExternalSecret 강제 sync (wealist-shared-secret 재생성)
	@echo -e "$(YELLOW)🔄 ExternalSecret sync 중...$(NC)"
	@kubectl delete secret wealist-shared-secret -n wealist-dev 2>/dev/null || true
	@kubectl annotate externalsecret wealist-shared-secret -n wealist-dev force-sync=$$(date +%s) --overwrite 2>/dev/null || true
	@echo "⏳ Sync 대기 중..."
	@sleep 5
	@kubectl get secret wealist-shared-secret -n wealist-dev 2>/dev/null && echo -e "$(GREEN)✅ wealist-shared-secret 재생성 완료$(NC)" || echo -e "$(RED)❌ Secret 생성 실패$(NC)"

# ============================================
# 수정된 all 타겟
# ============================================

all: setup-local ## 전체 프로세스 (Registry + 이미지 + Bootstrap + 배포)
	@echo ""
	@echo -e "$(GREEN)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo -e "$(GREEN)✅ 전체 배포 완료!$(NC)"
	@echo -e "$(GREEN)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@echo "ArgoCD UI: https://localhost:8079"
	@echo "Username: admin"
	@echo "Password: $$(kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d)"
	@echo ""
	@echo "로컬 Registry: http://localhost:5001"
	@echo "이미지 확인: make check-images"
	@echo ""
	@echo "다음 명령어로 포트 포워딩:"
	@echo "  make ui"

# ============================================
# 기존 cluster-up 타겟 수정 (Registry 포함)
# ============================================

cluster-up-simple: ## Kind 클러스터만 생성 (Registry 없이)
	@echo -e "$(YELLOW)📦 Kind 클러스터 생성 중...$(NC)"
	@if kind get clusters | grep -q "$(CLUSTER_NAME)"; then \
		echo -e "$(YELLOW)⚠️  클러스터가 이미 존재합니다: $(CLUSTER_NAME)$(NC)"; \
		read -p "삭제하고 다시 만들까요? (y/N): " answer; \
		if [ "$$answer" = "y" ] || [ "$$answer" = "Y" ]; then \
			$(MAKE) cluster-down; \
		else \
			echo "기존 클러스터를 사용합니다."; \
			exit 0; \
		fi; \
	fi
	@if [ -f "k8s/helm/scripts/dev/kind-config.yaml" ]; then \
		kind create cluster --name $(CLUSTER_NAME) --config k8s/helm/scripts/dev/kind-config.yaml; \
	else \
		kind create cluster --name $(CLUSTER_NAME); \
	fi
	@kubectl cluster-info
	@echo -e "$(GREEN)✅ 클러스터 생성 완료$(NC)"

