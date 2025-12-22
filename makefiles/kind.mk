# =============================================================================
# Kubernetes (Kind) 명령어
# =============================================================================

##@ Kubernetes (Kind)

.PHONY: kind-setup kind-setup-simple kind-setup-db kind-check-db kind-check-db-setup kind-localhost-setup kind-delete kind-recover
.PHONY: kind-load-images kind-load-images-ex-db kind-load-images-all kind-load-images-mono
.PHONY: kind-load-infra kind-load-monitoring kind-load-services
.PHONY: _setup-db-macos _setup-db-debian _check-db-installed

# =============================================================================
# 통합 설정 명령어 (권장)
# =============================================================================

kind-check-db-setup: ## 🚀 통합 설정: Secrets → DB 확인 → 클러스터 생성 → 이미지 로드 (DB 제외)
	@echo "=============================================="
	@echo "  weAlist Kind 클러스터 통합 설정"
	@echo "=============================================="
	@echo ""
	@echo "이 명령어는 다음을 순서대로 실행합니다:"
	@echo "  0. 필수 도구 확인 (istioctl)"
	@echo "  1. Secrets 파일 확인/생성"
	@echo "  2. PostgreSQL/Redis 설치 상태 확인 [Y/N]"
	@echo "  3. Kind 클러스터 생성 + Istio Ambient"
	@echo "  4. 서비스 이미지 로드 (DB 이미지 제외)"
	@echo ""
	@echo "----------------------------------------------"
	@echo "  0단계: 필수 도구 확인"
	@echo "----------------------------------------------"
	@echo ""
	@# kubectl 확인 및 설치
	@if ! command -v kubectl >/dev/null 2>&1; then \
		echo "❌ kubectl: 미설치"; \
		echo ""; \
		echo "kubectl을 자동 설치하시겠습니까? [Y/n]"; \
		read -r answer; \
		if [ "$$answer" != "n" ] && [ "$$answer" != "N" ]; then \
			echo ""; \
			echo "kubectl 설치 중..."; \
			if [ "$$(uname)" = "Darwin" ]; then \
				brew install kubectl; \
			else \
				curl -LO "https://dl.k8s.io/release/$$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"; \
				chmod +x kubectl; \
				sudo mv kubectl /usr/local/bin/kubectl; \
			fi; \
			echo ""; \
			echo "✅ kubectl 설치 완료!"; \
		else \
			echo ""; \
			echo "kubectl 없이는 진행할 수 없습니다."; \
			exit 1; \
		fi; \
	else \
		echo "✅ kubectl: $$(kubectl version --client --short 2>/dev/null || echo '설치됨')"; \
	fi
	@echo ""
	@# Kind 확인 및 설치
	@if ! command -v kind >/dev/null 2>&1; then \
		echo "❌ kind: 미설치"; \
		echo ""; \
		echo "kind를 자동 설치하시겠습니까? [Y/n]"; \
		read -r answer; \
		if [ "$$answer" != "n" ] && [ "$$answer" != "N" ]; then \
			echo ""; \
			echo "kind 설치 중..."; \
			if [ "$$(uname)" = "Darwin" ]; then \
				brew install kind; \
			else \
				curl -Lo /tmp/kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64; \
				chmod +x /tmp/kind; \
				sudo mv /tmp/kind /usr/local/bin/kind; \
			fi; \
			echo ""; \
			echo "✅ kind 설치 완료!"; \
		else \
			echo ""; \
			echo "kind 없이는 진행할 수 없습니다."; \
			exit 1; \
		fi; \
	else \
		echo "✅ kind: $$(kind version 2>/dev/null || echo '설치됨')"; \
	fi
	@echo ""
	@# Helm 확인 및 설치
	@if ! command -v helm >/dev/null 2>&1; then \
		echo "❌ helm: 미설치"; \
		echo ""; \
		echo "helm을 자동 설치하시겠습니까? [Y/n]"; \
		read -r answer; \
		if [ "$$answer" != "n" ] && [ "$$answer" != "N" ]; then \
			echo ""; \
			echo "helm 설치 중..."; \
			curl -fsSL -o /tmp/get_helm.sh https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3; \
			chmod 700 /tmp/get_helm.sh; \
			/tmp/get_helm.sh; \
			rm -f /tmp/get_helm.sh; \
			echo ""; \
			echo "✅ helm 설치 완료!"; \
		else \
			echo ""; \
			echo "helm 없이는 진행할 수 없습니다."; \
			exit 1; \
		fi; \
	else \
		echo "✅ helm: $$(helm version --short 2>/dev/null || echo '설치됨')"; \
	fi
	@echo ""
	@# istioctl 확인 및 설치
	@if ! command -v istioctl >/dev/null 2>&1; then \
		if [ -f "./istio-1.24.0/bin/istioctl" ]; then \
			echo "✅ istioctl: 로컬 설치됨 (./istio-1.24.0/bin/istioctl)"; \
		else \
			echo "❌ istioctl: 미설치"; \
			echo ""; \
			echo "istioctl을 자동 설치하시겠습니까? [Y/n]"; \
			read -r answer; \
			if [ "$$answer" != "n" ] && [ "$$answer" != "N" ]; then \
				echo ""; \
				echo "istioctl 설치 중..."; \
				curl -L https://istio.io/downloadIstio | ISTIO_VERSION=1.24.0 sh -; \
				echo ""; \
				echo "✅ istioctl 설치 완료!"; \
			else \
				echo ""; \
				echo "istioctl 없이는 진행할 수 없습니다."; \
				exit 1; \
			fi; \
		fi; \
	else \
		echo "✅ istioctl: $$(istioctl version --short 2>/dev/null || echo '설치됨')"; \
	fi
	@echo ""
	@echo "----------------------------------------------"
	@echo "  1단계: Secrets 파일 확인"
	@echo "----------------------------------------------"
	@echo ""
	@if [ ! -f "./k8s/helm/environments/secrets.yaml" ]; then \
		echo "⚠️  secrets.yaml 파일이 없습니다."; \
		echo "   secrets.example.yaml에서 자동 생성합니다..."; \
		echo ""; \
		cp ./k8s/helm/environments/secrets.example.yaml ./k8s/helm/environments/secrets.yaml; \
		echo "✅ secrets.yaml 생성 완료!"; \
		echo ""; \
		echo "📝 주의: 배포 전 아래 파일을 편집하여 실제 값을 입력하세요:"; \
		echo "   k8s/helm/environments/secrets.yaml"; \
		echo ""; \
	else \
		echo "✅ secrets.yaml 파일 존재 확인"; \
	fi
	@echo ""
	@echo "----------------------------------------------"
	@echo "  2단계: DB 설치 상태 확인"
	@echo "----------------------------------------------"
	@echo ""
	@# DB 확인 및 설치
	@POSTGRES_OK=false; \
	REDIS_OK=false; \
	if command -v psql >/dev/null 2>&1; then \
		echo "✅ PostgreSQL: 설치됨"; \
		if pg_isready >/dev/null 2>&1 || systemctl is-active postgresql >/dev/null 2>&1 2>&1; then \
			echo "   └─ 상태: 실행 중"; \
			POSTGRES_OK=true; \
		else \
			echo "   └─ 상태: 설치되었으나 실행 중이 아님"; \
		fi; \
	else \
		echo "❌ PostgreSQL: 미설치"; \
	fi; \
	echo ""; \
	if command -v redis-cli >/dev/null 2>&1; then \
		echo "✅ Redis: 설치됨"; \
		if redis-cli ping >/dev/null 2>&1; then \
			echo "   └─ 상태: 실행 중"; \
			REDIS_OK=true; \
		else \
			echo "   └─ 상태: 설치되었으나 실행 중이 아님"; \
		fi; \
	else \
		echo "❌ Redis: 미설치"; \
	fi; \
	echo ""; \
	if [ "$$POSTGRES_OK" = "false" ] || [ "$$REDIS_OK" = "false" ]; then \
		echo "⚠️  일부 DB가 설치되지 않았거나 실행 중이 아닙니다."; \
		echo ""; \
		echo "DB 설치 및 설정을 진행하시겠습니까? [Y/n]"; \
		read -r answer; \
		if [ "$$answer" != "n" ] && [ "$$answer" != "N" ]; then \
			$(MAKE) kind-setup-db; \
		else \
			echo ""; \
			echo "⚠️  DB 없이 진행합니다. 서비스 실행 시 오류가 발생할 수 있습니다."; \
		fi; \
	else \
		echo "✅ 모든 DB가 정상적으로 실행 중입니다!"; \
	fi
	@echo ""
	@echo "----------------------------------------------"
	@echo "  3단계: Kind 클러스터 생성"
	@echo "----------------------------------------------"
	@$(MAKE) kind-setup
	@echo ""
	@echo "----------------------------------------------"
	@echo "  4단계: 서비스 이미지 로드 (DB 제외)"
	@echo "----------------------------------------------"
	@$(MAKE) kind-load-images-ex-db
	@echo ""
	@echo "=============================================="
	@echo "  🎉 통합 설정 완료!"
	@echo "=============================================="
	@echo ""
	@echo "  다음 단계:"
	@echo "    1. (선택) secrets.yaml 편집 (API 키, 비밀번호 등 입력):"
	@echo "       vi k8s/helm/environments/secrets.yaml"
	@echo ""
	@echo "    2. Helm 배포:"
	@echo "       make helm-install-all ENV=dev"
	@echo ""
	@echo "=============================================="

# -----------------------------------------------------------------------------
# kind-localhost-setup: 통합 환경 (DB내장 + 프론트내장 + Istio)
# -----------------------------------------------------------------------------
kind-localhost-setup: ## 🏠 통합 환경: 클러스터 생성 → 모든 이미지 로드 (DB + Frontend 포함)
	@echo "=============================================="
	@echo "  weAlist Kind 로컬 통합 환경 설정"
	@echo "=============================================="
	@echo ""
	@echo "이 명령어는 다음을 순서대로 실행합니다:"
	@echo "  0. 필수 도구 확인 (istioctl)"
	@echo "  1. Secrets 파일 확인/생성"
	@echo "  2. Kind 클러스터 생성 + Istio Ambient"
	@echo "  3. 모든 이미지 로드 (DB + Backend + Frontend)"
	@echo ""
	@echo "※ 이 환경은 모든 컴포넌트가 클러스터 내부에서 실행됩니다."
	@echo "  - PostgreSQL: Pod로 실행"
	@echo "  - Redis: Pod로 실행"
	@echo "  - Frontend: Pod로 실행"
	@echo ""
	@echo "----------------------------------------------"
	@echo "  0단계: 필수 도구 확인"
	@echo "----------------------------------------------"
	@echo ""
	@# kubectl 확인 및 설치
	@if ! command -v kubectl >/dev/null 2>&1; then \
		echo "❌ kubectl: 미설치"; \
		echo ""; \
		echo "kubectl을 자동 설치하시겠습니까? [Y/n]"; \
		read -r answer; \
		if [ "$$answer" != "n" ] && [ "$$answer" != "N" ]; then \
			echo ""; \
			echo "kubectl 설치 중..."; \
			if [ "$$(uname)" = "Darwin" ]; then \
				brew install kubectl; \
			else \
				curl -LO "https://dl.k8s.io/release/$$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"; \
				chmod +x kubectl; \
				sudo mv kubectl /usr/local/bin/kubectl; \
			fi; \
			echo ""; \
			echo "✅ kubectl 설치 완료!"; \
		else \
			echo ""; \
			echo "kubectl 없이는 진행할 수 없습니다."; \
			exit 1; \
		fi; \
	else \
		echo "✅ kubectl: $$(kubectl version --client --short 2>/dev/null || echo '설치됨')"; \
	fi
	@echo ""
	@# Kind 확인 및 설치
	@if ! command -v kind >/dev/null 2>&1; then \
		echo "❌ kind: 미설치"; \
		echo ""; \
		echo "kind를 자동 설치하시겠습니까? [Y/n]"; \
		read -r answer; \
		if [ "$$answer" != "n" ] && [ "$$answer" != "N" ]; then \
			echo ""; \
			echo "kind 설치 중..."; \
			if [ "$$(uname)" = "Darwin" ]; then \
				brew install kind; \
			else \
				curl -Lo /tmp/kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64; \
				chmod +x /tmp/kind; \
				sudo mv /tmp/kind /usr/local/bin/kind; \
			fi; \
			echo ""; \
			echo "✅ kind 설치 완료!"; \
		else \
			echo ""; \
			echo "kind 없이는 진행할 수 없습니다."; \
			exit 1; \
		fi; \
	else \
		echo "✅ kind: $$(kind version 2>/dev/null || echo '설치됨')"; \
	fi
	@echo ""
	@# Helm 확인 및 설치
	@if ! command -v helm >/dev/null 2>&1; then \
		echo "❌ helm: 미설치"; \
		echo ""; \
		echo "helm을 자동 설치하시겠습니까? [Y/n]"; \
		read -r answer; \
		if [ "$$answer" != "n" ] && [ "$$answer" != "N" ]; then \
			echo ""; \
			echo "helm 설치 중..."; \
			curl -fsSL -o /tmp/get_helm.sh https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3; \
			chmod 700 /tmp/get_helm.sh; \
			/tmp/get_helm.sh; \
			rm -f /tmp/get_helm.sh; \
			echo ""; \
			echo "✅ helm 설치 완료!"; \
		else \
			echo ""; \
			echo "helm 없이는 진행할 수 없습니다."; \
			exit 1; \
		fi; \
	else \
		echo "✅ helm: $$(helm version --short 2>/dev/null || echo '설치됨')"; \
	fi
	@echo ""
	@# istioctl 확인 및 설치
	@if ! command -v istioctl >/dev/null 2>&1; then \
		if [ -f "./istio-1.24.0/bin/istioctl" ]; then \
			echo "✅ istioctl: 로컬 설치됨 (./istio-1.24.0/bin/istioctl)"; \
		else \
			echo "❌ istioctl: 미설치"; \
			echo ""; \
			echo "istioctl을 자동 설치하시겠습니까? [Y/n]"; \
			read -r answer; \
			if [ "$$answer" != "n" ] && [ "$$answer" != "N" ]; then \
				echo ""; \
				echo "istioctl 설치 중..."; \
				curl -L https://istio.io/downloadIstio | ISTIO_VERSION=1.24.0 sh -; \
				echo ""; \
				echo "✅ istioctl 설치 완료!"; \
			else \
				echo ""; \
				echo "istioctl 없이는 진행할 수 없습니다."; \
				exit 1; \
			fi; \
		fi; \
	else \
		echo "✅ istioctl: $$(istioctl version --short 2>/dev/null || echo '설치됨')"; \
	fi
	@echo ""
	@echo "----------------------------------------------"
	@echo "  1단계: Secrets 파일 확인"
	@echo "----------------------------------------------"
	@echo ""
	@if [ ! -f "./k8s/helm/environments/secrets.yaml" ]; then \
		echo "⚠️  secrets.yaml 파일이 없습니다."; \
		echo "   secrets.example.yaml에서 자동 생성합니다..."; \
		echo ""; \
		cp ./k8s/helm/environments/secrets.example.yaml ./k8s/helm/environments/secrets.yaml; \
		echo "✅ secrets.yaml 생성 완료!"; \
		echo ""; \
	else \
		echo "✅ secrets.yaml 파일 존재 확인"; \
	fi
	@echo ""
	@echo "----------------------------------------------"
	@echo "  2단계: Kind 클러스터 생성"
	@echo "----------------------------------------------"
	@$(MAKE) kind-setup ENV=localhost
	@echo ""
	@echo "----------------------------------------------"
	@echo "  3단계: 모든 이미지 로드 (DB + Backend + Frontend)"
	@echo "----------------------------------------------"
	@$(MAKE) kind-load-images-all
	@echo ""
	@echo "=============================================="
	@echo "  🎉 통합 환경 설정 완료!"
	@echo "=============================================="
	@echo ""
	@echo "  다음 단계:"
	@echo "    1. (선택) secrets.yaml 편집:"
	@echo "       vi k8s/helm/environments/secrets.yaml"
	@echo ""
	@echo "    2. Helm 배포:"
	@echo "       make helm-install-all ENV=localhost"
	@echo ""
	@echo "=============================================="

# -----------------------------------------------------------------------------
# kind-dev-setup: 개발 환경 (외부 DB + Istio)
# -----------------------------------------------------------------------------
kind-dev-setup: ## 🔧 개발 환경: 클러스터 생성 → 서비스 이미지 로드 (외부 DB 사용)
	@echo "=============================================="
	@echo "  weAlist Kind 개발 환경 설정"
	@echo "=============================================="
	@echo ""
	@echo "이 명령어는 다음을 순서대로 실행합니다:"
	@echo "  1. 필수 도구 확인 (kubectl, kind, helm, istioctl)"
	@echo "  2. Secrets 파일 확인/생성"
	@echo "  3. GHCR 로그인"
	@echo "  4. Kind 클러스터 생성 + Istio Ambient"
	@echo "  5. 외부 DB 확인 + 연결 테스트 (172.18.0.1)"
	@echo "  6. GHCR Secret + 인프라 이미지 로드"
	@echo "  7. GHCR 서비스 이미지 확인"
	@echo "  8. ArgoCD 설치 (선택)"
	@echo ""
	@echo "※ dev 환경은 호스트 PC의 PostgreSQL/Redis를 사용합니다."
	@echo "  - PostgreSQL: 호스트 머신 (172.18.0.1:5432)"
	@echo "  - Redis: 호스트 머신 (172.18.0.1:6379)"
	@echo ""
	@echo "----------------------------------------------"
	@echo "  [1/8] 필수 도구 확인"
	@echo "----------------------------------------------"
	@echo ""
	@# kubectl 확인 및 설치
	@if ! command -v kubectl >/dev/null 2>&1; then \
		echo "❌ kubectl: 미설치"; \
		echo ""; \
		echo "kubectl을 자동 설치하시겠습니까? [Y/n]"; \
		read -r answer; \
		if [ "$$answer" != "n" ] && [ "$$answer" != "N" ]; then \
			echo ""; \
			echo "kubectl 설치 중..."; \
			if [ "$$(uname)" = "Darwin" ]; then \
				brew install kubectl; \
			else \
				curl -LO "https://dl.k8s.io/release/$$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"; \
				chmod +x kubectl; \
				sudo mv kubectl /usr/local/bin/kubectl; \
			fi; \
			echo ""; \
			echo "✅ kubectl 설치 완료!"; \
		else \
			echo ""; \
			echo "kubectl 없이는 진행할 수 없습니다."; \
			exit 1; \
		fi; \
	else \
		echo "✅ kubectl: $$(kubectl version --client --short 2>/dev/null || echo '설치됨')"; \
	fi
	@echo ""
	@# Kind 확인 및 설치
	@if ! command -v kind >/dev/null 2>&1; then \
		echo "❌ kind: 미설치"; \
		echo ""; \
		echo "kind를 자동 설치하시겠습니까? [Y/n]"; \
		read -r answer; \
		if [ "$$answer" != "n" ] && [ "$$answer" != "N" ]; then \
			echo ""; \
			echo "kind 설치 중..."; \
			if [ "$$(uname)" = "Darwin" ]; then \
				brew install kind; \
			else \
				curl -Lo /tmp/kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64; \
				chmod +x /tmp/kind; \
				sudo mv /tmp/kind /usr/local/bin/kind; \
			fi; \
			echo ""; \
			echo "✅ kind 설치 완료!"; \
		else \
			echo ""; \
			echo "kind 없이는 진행할 수 없습니다."; \
			exit 1; \
		fi; \
	else \
		echo "✅ kind: $$(kind version 2>/dev/null || echo '설치됨')"; \
	fi
	@echo ""
	@# Helm 확인 및 설치
	@if ! command -v helm >/dev/null 2>&1; then \
		echo "❌ helm: 미설치"; \
		echo ""; \
		echo "helm을 자동 설치하시겠습니까? [Y/n]"; \
		read -r answer; \
		if [ "$$answer" != "n" ] && [ "$$answer" != "N" ]; then \
			echo ""; \
			echo "helm 설치 중..."; \
			curl -fsSL -o /tmp/get_helm.sh https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3; \
			chmod 700 /tmp/get_helm.sh; \
			/tmp/get_helm.sh; \
			rm -f /tmp/get_helm.sh; \
			echo ""; \
			echo "✅ helm 설치 완료!"; \
		else \
			echo ""; \
			echo "helm 없이는 진행할 수 없습니다."; \
			exit 1; \
		fi; \
	else \
		echo "✅ helm: $$(helm version --short 2>/dev/null || echo '설치됨')"; \
	fi
	@echo ""
	@# istioctl 확인 및 설치
	@if ! command -v istioctl >/dev/null 2>&1; then \
		if [ -f "./istio-1.24.0/bin/istioctl" ]; then \
			echo "✅ istioctl: 로컬 설치됨 (./istio-1.24.0/bin/istioctl)"; \
		else \
			echo "❌ istioctl: 미설치"; \
			echo ""; \
			echo "istioctl을 자동 설치하시겠습니까? [Y/n]"; \
			read -r answer; \
			if [ "$$answer" != "n" ] && [ "$$answer" != "N" ]; then \
				echo ""; \
				echo "istioctl 설치 중..."; \
				curl -L https://istio.io/downloadIstio | ISTIO_VERSION=1.24.0 sh -; \
				echo ""; \
				echo "✅ istioctl 설치 완료!"; \
			else \
				echo ""; \
				echo "istioctl 없이는 진행할 수 없습니다."; \
				exit 1; \
			fi; \
		fi; \
	else \
		echo "✅ istioctl: $$(istioctl version --short 2>/dev/null || echo '설치됨')"; \
	fi
	@echo ""
	@echo "----------------------------------------------"
	@echo "  [2/8] Secrets 파일 확인"
	@echo "----------------------------------------------"
	@echo ""
	@if [ ! -f "./k8s/helm/environments/secrets.yaml" ]; then \
		echo "⚠️  secrets.yaml 파일이 없습니다."; \
		echo "   secrets.example.yaml에서 자동 생성합니다..."; \
		echo ""; \
		cp ./k8s/helm/environments/secrets.example.yaml ./k8s/helm/environments/secrets.yaml; \
		echo "✅ secrets.yaml 생성 완료!"; \
		echo ""; \
	else \
		echo "✅ secrets.yaml 파일 존재 확인"; \
	fi
	@echo ""
	@echo "----------------------------------------------"
	@echo "  [3/8] GHCR (GitHub Container Registry) 로그인"
	@echo "----------------------------------------------"
	@echo ""
	@echo "dev 환경은 ghcr.io/orangescloud에서 이미지를 pull합니다."
	@echo "GHCR 로그인이 필요합니다."
	@echo ""
	@# GHCR 로그인 확인 및 credential 저장
	@rm -f /tmp/ghcr_credentials.env; \
	GHCR_LOGGED_IN=false; \
	if docker login ghcr.io --get-login 2>/dev/null | grep -q .; then \
		echo "✅ GHCR: Docker에 로그인됨"; \
		echo ""; \
		echo "⚠️  Kubernetes Secret 생성을 위해 토큰 재입력이 필요합니다."; \
		echo "   (Docker credential helper는 토큰을 직접 저장하지 않음)"; \
	else \
		echo "❌ GHCR: 로그인 필요"; \
	fi; \
	echo ""; \
	echo "GitHub Personal Access Token (PAT)이 필요합니다."; \
	echo "  - GitHub → Settings → Developer settings → Personal access tokens"; \
	echo "  - 권한: read:packages (최소)"; \
	echo ""; \
	printf "GitHub 사용자명: "; \
	read GITHUB_USER; \
	printf "GitHub Personal Access Token: "; \
	stty -echo 2>/dev/null || true; \
	read GITHUB_TOKEN; \
	stty echo 2>/dev/null || true; \
	echo ""; \
	echo "GHCR 로그인 중..."; \
	if echo "$$GITHUB_TOKEN" | docker login ghcr.io -u "$$GITHUB_USER" --password-stdin; then \
		echo ""; \
		echo "✅ GHCR 로그인 성공!"; \
		echo "GITHUB_USER=$$GITHUB_USER" > /tmp/ghcr_credentials.env; \
		echo "GITHUB_TOKEN=$$GITHUB_TOKEN" >> /tmp/ghcr_credentials.env; \
		chmod 600 /tmp/ghcr_credentials.env; \
		echo "✅ Credentials 저장됨 (Secret 생성에 사용)"; \
	else \
		echo ""; \
		echo "❌ GHCR 로그인 실패"; \
		echo "   토큰 권한을 확인하세요 (read:packages 필요)"; \
		exit 1; \
	fi
	@echo ""
	@echo "----------------------------------------------"
	@echo "  [4/8] Kind 클러스터 생성"
	@echo "----------------------------------------------"
	@$(MAKE) kind-setup ENV=dev
	@echo ""
	@echo "----------------------------------------------"
	@echo "  [5/8] 외부 DB 확인 + 연결 테스트"
	@echo "----------------------------------------------"
	@echo ""
	@echo "dev 환경은 호스트 PC의 PostgreSQL/Redis를 사용합니다."
	@echo ""
	@# OS 감지 및 DB 호스트 설정
	@if [ "$$(uname)" = "Darwin" ]; then \
		DB_HOST="host.docker.internal"; \
		echo "🖥️  macOS 감지 → DB 호스트: host.docker.internal"; \
	elif grep -qi microsoft /proc/version 2>/dev/null; then \
		DB_HOST=$$(hostname -I | awk '{print $$1}'); \
		echo "🖥️  WSL 감지 → DB 호스트: $$DB_HOST (WSL IP)"; \
		echo "   ⚠️  WSL IP는 재부팅 시 변경될 수 있습니다."; \
	else \
		DB_HOST="172.18.0.1"; \
		echo "🖥️  Linux 감지 → DB 호스트: 172.18.0.1"; \
	fi; \
	echo ""; \
	echo "DB_HOST=$$DB_HOST" > /tmp/kind_db_host.env
	@echo ""
	@# PostgreSQL 확인
	@echo "🔍 PostgreSQL 확인 중..."
	@if command -v psql >/dev/null 2>&1; then \
		if pg_isready >/dev/null 2>&1 || (command -v systemctl >/dev/null 2>&1 && systemctl is-active postgresql >/dev/null 2>&1) || (command -v brew >/dev/null 2>&1 && brew services list 2>/dev/null | grep -q "postgresql.*started"); then \
			echo "  ✅ 호스트: PostgreSQL 실행 중"; \
		else \
			echo "  ❌ 호스트: PostgreSQL 실행 중이 아님"; \
			echo "     PostgreSQL을 시작하세요: brew services start postgresql (macOS)"; \
			echo "     또는: sudo systemctl start postgresql (Linux)"; \
			exit 1; \
		fi; \
	else \
		echo "  ❌ 호스트: PostgreSQL 미설치"; \
		echo "     설치 후 다시 시도하세요."; \
		exit 1; \
	fi
	@# Redis 확인
	@echo "🔍 Redis 확인 중..."
	@if command -v redis-cli >/dev/null 2>&1; then \
		if redis-cli ping >/dev/null 2>&1; then \
			echo "  ✅ 호스트: Redis 실행 중"; \
		else \
			echo "  ❌ 호스트: Redis 실행 중이 아님"; \
			echo "     Redis를 시작하세요: brew services start redis (macOS)"; \
			echo "     또는: sudo systemctl start redis (Linux)"; \
			exit 1; \
		fi; \
	else \
		echo "  ❌ 호스트: Redis 미설치"; \
		echo "     설치 후 다시 시도하세요."; \
		exit 1; \
	fi
	@echo ""
	@# Kind에서 DB 연결 테스트
	@. /tmp/kind_db_host.env && \
	echo "🔗 Kind 클러스터 → 호스트 DB 연결 테스트..." && \
	echo "  PostgreSQL ($$DB_HOST:5432)..." && \
	if kubectl run pg-test --rm -i --restart=Never --image=postgres:15-alpine -- \
		pg_isready -h $$DB_HOST -p 5432 -t 5 2>/dev/null; then \
		echo "  ✅ PostgreSQL 연결 성공!"; \
		echo ""; \
		echo "🔧 PostgreSQL 데이터베이스 초기화 중..."; \
		sudo ./scripts/init-local-postgres.sh; \
	else \
		echo "  ❌ PostgreSQL 연결 실패"; \
		echo ""; \
		echo "PostgreSQL 외부 연결을 자동 설정하시겠습니까? [Y/n]"; \
		read -r answer; \
		if [ "$$answer" != "n" ] && [ "$$answer" != "N" ]; then \
			echo ""; \
			echo "🔧 PostgreSQL 외부 연결 설정 중..."; \
			PG_CONF=""; PG_HBA=""; \
			PG_CONF=$$(sudo -u postgres psql -t -c "SHOW config_file" 2>/dev/null | tr -d ' '); \
			PG_HBA=$$(sudo -u postgres psql -t -c "SHOW hba_file" 2>/dev/null | tr -d ' '); \
			if [ -z "$$PG_CONF" ] || [ ! -f "$$PG_CONF" ]; then \
				echo "  🔍 postgresql.conf 경로 검색 중..."; \
				for path in /etc/postgresql/*/main/postgresql.conf /var/lib/pgsql/*/data/postgresql.conf /var/lib/pgsql/data/postgresql.conf /usr/local/var/postgres/postgresql.conf /opt/homebrew/var/postgres/postgresql.conf; do \
					if [ -f "$$path" ]; then PG_CONF="$$path"; break; fi; \
				done; \
			fi; \
			if [ -z "$$PG_HBA" ] || [ ! -f "$$PG_HBA" ]; then \
				for path in /etc/postgresql/*/main/pg_hba.conf /var/lib/pgsql/*/data/pg_hba.conf /var/lib/pgsql/data/pg_hba.conf /usr/local/var/postgres/pg_hba.conf /opt/homebrew/var/postgres/pg_hba.conf; do \
					if [ -f "$$path" ]; then PG_HBA="$$path"; break; fi; \
				done; \
			fi; \
			if [ -n "$$PG_CONF" ] && [ -f "$$PG_CONF" ]; then \
				echo "  📄 postgresql.conf: $$PG_CONF"; \
				sudo sed -i "s/^#*listen_addresses.*/listen_addresses = '*'/" "$$PG_CONF"; \
				echo "  ✅ listen_addresses = '*' 설정 완료"; \
			else \
				echo "  ❌ postgresql.conf를 찾을 수 없습니다"; \
				echo "     수동으로 listen_addresses = '*' 설정이 필요합니다"; \
			fi; \
			if [ -n "$$PG_HBA" ] && [ -f "$$PG_HBA" ]; then \
				echo "  📄 pg_hba.conf: $$PG_HBA"; \
				. /tmp/kind_db_host.env; \
				DB_SUBNET=$$(echo "$$DB_HOST" | sed 's/\.[0-9]*\.[0-9]*$/.0.0\/16/'); \
				echo "  🔗 DB 서브넷: $$DB_SUBNET"; \
				if ! sudo grep -q "$$DB_SUBNET" "$$PG_HBA"; then \
					echo "host    all    all    $$DB_SUBNET    md5" | sudo tee -a "$$PG_HBA" > /dev/null; \
					echo "  ✅ $$DB_SUBNET 접근 허용 추가"; \
				else \
					echo "  ✅ $$DB_SUBNET 접근 이미 설정됨"; \
				fi; \
			else \
				echo "  ❌ pg_hba.conf를 찾을 수 없습니다"; \
				echo "     수동으로 host all all <subnet>/16 md5 설정이 필요합니다"; \
			fi; \
			echo ""; \
			echo "🔄 PostgreSQL 재시작 중..."; \
			IS_WSL=false; \
			if grep -qi microsoft /proc/version 2>/dev/null; then \
				IS_WSL=true; \
				echo "  🖥️  WSL 환경 감지 (systemd 대신 직접 실행)"; \
			fi; \
			if [ "$$IS_WSL" = "true" ]; then \
				PG_DATA_DIR=$$(dirname "$$PG_CONF"); \
				PG_VERSION=$$(ls /usr/lib/postgresql/ 2>/dev/null | sort -rn | head -1); \
				echo "  📂 PostgreSQL Data: $$PG_DATA_DIR"; \
				echo "  📦 PostgreSQL Version: $$PG_VERSION"; \
				sudo -u postgres /usr/lib/postgresql/$$PG_VERSION/bin/pg_ctl restart -D "$$PG_DATA_DIR" -l /var/log/postgresql/postgresql.log 2>/dev/null || \
				sudo pg_ctlcluster $$PG_VERSION main restart 2>/dev/null || \
				{ sudo -u postgres /usr/lib/postgresql/$$PG_VERSION/bin/pg_ctl stop -D "$$PG_DATA_DIR" -m fast 2>/dev/null; \
				  sleep 2; \
				  sudo -u postgres /usr/lib/postgresql/$$PG_VERSION/bin/pg_ctl start -D "$$PG_DATA_DIR" -l /var/log/postgresql/postgresql.log; }; \
				echo "  ✅ PostgreSQL 재시작 완료 (WSL)"; \
			else \
				sudo systemctl restart postgresql 2>/dev/null || sudo service postgresql restart 2>/dev/null; \
				echo "  ✅ PostgreSQL 재시작 완료"; \
			fi; \
			sleep 3; \
			echo ""; \
			echo "🔗 연결 재테스트..."; \
			. /tmp/kind_db_host.env; \
			if kubectl run pg-test2 --rm -i --restart=Never --image=postgres:15-alpine -- \
				pg_isready -h $$DB_HOST -p 5432 -t 5 2>/dev/null; then \
				echo "  ✅ PostgreSQL 연결 성공!"; \
				echo ""; \
				echo "🔧 PostgreSQL 데이터베이스 초기화 중..."; \
				sudo ./scripts/init-local-postgres.sh; \
			else \
				echo "  ❌ 여전히 연결 실패"; \
				echo ""; \
				echo "  수동 확인 필요:"; \
				echo "    1. postgresql.conf: listen_addresses = '*'"; \
				echo "    2. pg_hba.conf: host all all $$DB_SUBNET md5"; \
				if [ "$$IS_WSL" = "true" ]; then \
					echo "    3. sudo pg_ctlcluster <version> main restart"; \
					echo "       또는: sudo -u postgres /usr/lib/postgresql/<version>/bin/pg_ctl restart -D <data_dir>"; \
				else \
					echo "    3. sudo systemctl restart postgresql"; \
				fi; \
				echo ""; \
				echo "계속 진행하시겠습니까? (DB 연결 없이) [y/N]"; \
				read -r skip; \
				if [ "$$skip" != "y" ] && [ "$$skip" != "Y" ]; then \
					exit 1; \
				fi; \
			fi; \
		else \
			echo ""; \
			echo "수동 설정이 필요합니다:"; \
			echo "  - listen_addresses = '*' (postgresql.conf)"; \
			echo "  - host all all 172.18.0.0/16 md5 (pg_hba.conf)"; \
			exit 1; \
		fi; \
	fi
	@. /tmp/kind_db_host.env && \
	echo "  Redis ($$DB_HOST:6379)..." && \
	if kubectl run redis-test --rm -i --restart=Never --image=redis:7-alpine -- \
		redis-cli -h $$DB_HOST -p 6379 ping 2>/dev/null | grep -q PONG; then \
		echo "  ✅ Redis 연결 성공!"; \
	else \
		echo "  ❌ Redis 연결 실패"; \
		echo ""; \
		echo "Redis 외부 연결을 자동 설정하시겠습니까? [Y/n]"; \
		read -r answer; \
		if [ "$$answer" != "n" ] && [ "$$answer" != "N" ]; then \
			echo ""; \
			echo "🔧 Redis 외부 연결 설정 중..."; \
			REDIS_CONF=""; \
			IS_WSL=false; \
			if grep -qi microsoft /proc/version 2>/dev/null; then \
				IS_WSL=true; \
				echo "  🖥️  WSL 환경 감지 (systemd 대신 직접 실행)"; \
			fi; \
			echo "  🔍 redis.conf 경로 검색 중..."; \
			for path in /etc/redis/redis.conf /etc/redis.conf /usr/local/etc/redis.conf /opt/homebrew/etc/redis.conf; do \
				if sudo test -f "$$path" 2>/dev/null; then REDIS_CONF="$$path"; echo "  📄 redis.conf: $$path"; break; fi; \
			done; \
			if [ -n "$$REDIS_CONF" ]; then \
				echo "  📄 redis.conf: $$REDIS_CONF"; \
				sudo sed -i 's/^bind 127\.0\.0\.1.*$$/bind 0.0.0.0/' "$$REDIS_CONF"; \
				sudo sed -i 's/^protected-mode yes$$/protected-mode no/' "$$REDIS_CONF"; \
				if ! sudo grep -q "^bind 0.0.0.0" "$$REDIS_CONF"; then \
					echo "bind 0.0.0.0" | sudo tee -a "$$REDIS_CONF" > /dev/null; \
				fi; \
				if ! sudo grep -q "^protected-mode no" "$$REDIS_CONF"; then \
					echo "protected-mode no" | sudo tee -a "$$REDIS_CONF" > /dev/null; \
				fi; \
				echo "  ✅ bind 0.0.0.0, protected-mode no 설정 완료"; \
			else \
				echo "  ❌ redis.conf를 찾을 수 없습니다"; \
				echo "     수동으로 bind 0.0.0.0, protected-mode no 설정이 필요합니다"; \
			fi; \
			echo ""; \
			echo "🔄 Redis 재시작 중..."; \
			if [ "$$IS_WSL" = "true" ]; then \
				sudo pkill redis-server 2>/dev/null || true; \
				sleep 1; \
				sudo redis-server "$$REDIS_CONF" --daemonize yes; \
				echo "  ✅ Redis 직접 시작 완료 (WSL)"; \
			else \
				sudo systemctl restart redis 2>/dev/null || \
				sudo systemctl restart redis-server 2>/dev/null || \
				sudo service redis restart 2>/dev/null || \
				sudo service redis-server restart 2>/dev/null || \
				{ sudo pkill redis-server 2>/dev/null; sleep 1; sudo redis-server "$$REDIS_CONF" --daemonize yes; }; \
				echo "  ✅ Redis 재시작 완료"; \
			fi; \
			sleep 2; \
			echo ""; \
			echo "🔗 연결 재테스트..."; \
			. /tmp/kind_db_host.env; \
			if kubectl run redis-test2 --rm -i --restart=Never --image=redis:7-alpine -- \
				redis-cli -h $$DB_HOST -p 6379 ping 2>/dev/null | grep -q PONG; then \
				echo "  ✅ Redis 연결 성공!"; \
			else \
				echo "  ❌ 여전히 연결 실패"; \
				echo ""; \
				echo "  수동 확인 필요:"; \
				echo "    1. redis.conf: bind 0.0.0.0"; \
				echo "    2. redis.conf: protected-mode no"; \
				if [ "$$IS_WSL" = "true" ]; then \
					echo "    3. sudo pkill redis-server && sudo redis-server /etc/redis/redis.conf --daemonize yes"; \
				else \
					echo "    3. sudo systemctl restart redis"; \
				fi; \
				echo ""; \
				echo "계속 진행하시겠습니까? (DB 연결 없이) [y/N]"; \
				read -r skip; \
				if [ "$$skip" != "y" ] && [ "$$skip" != "Y" ]; then \
					exit 1; \
				fi; \
			fi; \
		else \
			echo ""; \
			echo "수동 설정이 필요합니다:"; \
			echo "  - bind 0.0.0.0 (redis.conf)"; \
			echo "  - protected-mode no"; \
			exit 1; \
		fi; \
	fi
	@echo ""
	@echo "✅ DB 연결 테스트 완료!"
	@echo ""
	@echo "----------------------------------------------"
	@echo "  [6/8] GHCR Secret + 인프라 이미지 로드"
	@echo "----------------------------------------------"
	@echo ""
	@echo "Kubernetes에서 GHCR 이미지를 pull하려면 Secret이 필요합니다."
	@# ghcr-secret 생성 (저장된 credentials 사용)
	@if kubectl get secret ghcr-secret -n wealist-dev >/dev/null 2>&1; then \
		echo "⚠️  ghcr-secret 이미 존재 - 재생성합니다..."; \
		kubectl delete secret ghcr-secret -n wealist-dev; \
	fi; \
	if [ -f /tmp/ghcr_credentials.env ]; then \
		. /tmp/ghcr_credentials.env; \
		echo "ghcr-secret 생성 중 (user: $$GITHUB_USER)..."; \
		kubectl create secret docker-registry ghcr-secret \
			--docker-server=ghcr.io \
			--docker-username="$$GITHUB_USER" \
			--docker-password="$$GITHUB_TOKEN" \
			-n wealist-dev; \
		echo "✅ ghcr-secret 생성 완료!"; \
		rm -f /tmp/ghcr_credentials.env; \
	else \
		echo "❌ GHCR credentials를 찾을 수 없습니다."; \
		echo "   다시 setup을 실행하세요."; \
		exit 1; \
	fi
	@echo ""
	@# 인프라 이미지 로드
	@./k8s/helm/scripts/dev/1.load_infra_images.sh
	@echo ""
	@echo "----------------------------------------------"
	@echo "  [7/8] GHCR 서비스 이미지 확인"
	@echo "----------------------------------------------"
	@echo ""
	@echo "GHCR에 서비스 이미지가 있는지 확인합니다..."
	@echo ""
	@# 서비스 이미지 존재 여부 확인
	@MISSING_IMAGES=""; \
	for svc in auth-service user-service board-service chat-service noti-service storage-service video-service; do \
		if docker manifest inspect ghcr.io/orangescloud/$$svc:latest >/dev/null 2>&1; then \
			echo "✅ $$svc: 존재"; \
		else \
			echo "❌ $$svc: 없음"; \
			MISSING_IMAGES="$$MISSING_IMAGES $$svc"; \
		fi; \
	done; \
	echo ""; \
	if [ -n "$$MISSING_IMAGES" ]; then \
		echo "⚠️  일부 이미지가 GHCR에 없습니다:$$MISSING_IMAGES"; \
		echo ""; \
		echo "이미지를 빌드하고 GHCR에 푸시하시겠습니까? [Y/n]"; \
		read -r answer; \
		if [ "$$answer" != "n" ] && [ "$$answer" != "N" ]; then \
			echo ""; \
			echo "이미지 빌드 및 푸시 중... (시간이 걸릴 수 있습니다)"; \
			$(MAKE) ghcr-push-all ENV=dev || { \
				echo ""; \
				echo "❌ 이미지 빌드/푸시 실패"; \
				echo "   수동으로 실행: make ghcr-push-all ENV=dev"; \
				echo ""; \
				echo "계속 진행하시겠습니까? (이미지 없이) [y/N]"; \
				read -r cont; \
				if [ "$$cont" != "y" ] && [ "$$cont" != "Y" ]; then \
					exit 1; \
				fi; \
			}; \
		else \
			echo ""; \
			echo "⚠️  이미지 없이 진행합니다."; \
			echo "   helm-install-all 시 ImagePullBackOff 발생할 수 있습니다."; \
			echo "   나중에 빌드: make ghcr-push-all ENV=dev"; \
		fi; \
	else \
		echo "✅ 모든 서비스 이미지가 GHCR에 존재합니다!"; \
	fi
	@echo ""
	@echo "----------------------------------------------"
	@echo "  [8/8] ArgoCD 설치 (GitOps)"
	@echo "----------------------------------------------"
	@echo ""
	@echo "ArgoCD 설치 중..."
	@$(MAKE) argo-install-simple
	@echo ""
	@echo "✅ ArgoCD 설치 완료!"
	@echo ""
	@echo "📝 ArgoCD 접속 정보:"
	@echo "   URL: https://localhost:8079"
	@echo "   User: admin"
	@echo "   Password: kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath=\"{.data.password}\" | base64 -d"
	@echo ""
	@echo "=============================================="
	@echo "  🎉 개발 환경 설정 완료!"
	@echo "=============================================="
	@echo ""
	@echo "  ✅ 설치 완료:"
	@echo "    - Kind 클러스터 + Istio Ambient"
	@echo "    - Kiali, Jaeger (Istio 관측성)"
	@echo "    - ArgoCD (GitOps)"
	@echo ""
	@echo "  🌐 Gateway: localhost:80 (또는 :8080)"
	@echo ""
	@echo "  📊 모니터링 (helm-install-all 후 접근 가능):"
	@echo "    - Grafana:    http://localhost:8080/monitoring/grafana"
	@echo "    - Prometheus: http://localhost:8080/monitoring/prometheus"
	@echo "    - Kiali:      http://localhost:8080/monitoring/kiali"
	@echo "    - Jaeger:     http://localhost:8080/monitoring/jaeger"
	@echo ""
	@echo "  다음 단계:"
	@echo "    make helm-install-all ENV=dev"
	@echo ""
	@echo "  이후 개발 사이클:"
	@echo "    git push → GitHub Actions → GHCR → ArgoCD 자동 배포"
	@echo ""
	@echo "=============================================="

# =============================================================================
# 개별 설정 명령어
# =============================================================================

kind-setup: ## 클러스터 생성 + Istio Ambient (ENV에 따라 스크립트 선택)
	@echo "=== Kind 클러스터 + Istio Ambient 생성 (ENV=$(ENV)) ==="
	@echo ""
ifeq ($(ENV),localhost)
	./k8s/helm/scripts/localhost/0.setup-cluster.sh
else ifeq ($(ENV),dev)
	./k8s/helm/scripts/dev/0.setup-cluster.sh
else
	@echo "ENV를 지정하세요: make kind-setup ENV=localhost 또는 ENV=dev"
	@echo "기본값으로 localhost 스크립트 실행..."
	./k8s/helm/scripts/localhost/0.setup-cluster.sh
endif
	@echo ""
	@echo "클러스터 준비 완료! 다음: make kind-load-images"

kind-setup-simple: ## 클러스터 생성 + nginx ingress (Istio 없음, 단순 테스트용)
	@echo "=== Kind 클러스터 + nginx ingress (simple 모드) 생성 ==="
	./k8s/installShell/0.setup-cluster-simple.sh
	@echo ""
	@echo "클러스터 준비 완료! 다음: make kind-load-images"

# -----------------------------------------------------------------------------
# DB 설치 확인 (Y/N 프롬프트)
# -----------------------------------------------------------------------------
kind-check-db: ## PostgreSQL/Redis 설치 상태 확인 및 설치 안내
	@echo "=============================================="
	@echo "  데이터베이스 설치 상태 확인"
	@echo "=============================================="
	@echo ""
	@POSTGRES_OK=false; \
	REDIS_OK=false; \
	if command -v psql >/dev/null 2>&1; then \
		echo "✅ PostgreSQL: 설치됨"; \
		if pg_isready >/dev/null 2>&1 || systemctl is-active postgresql >/dev/null 2>&1; then \
			echo "   └─ 상태: 실행 중"; \
			POSTGRES_OK=true; \
		else \
			echo "   └─ 상태: 설치되었으나 실행 중이 아님"; \
			echo "   └─ 시작: brew services start postgresql / sudo systemctl start postgresql"; \
		fi; \
	else \
		echo "❌ PostgreSQL: 미설치"; \
	fi; \
	echo ""; \
	if command -v redis-cli >/dev/null 2>&1; then \
		echo "✅ Redis: 설치됨"; \
		if redis-cli ping >/dev/null 2>&1; then \
			echo "   └─ 상태: 실행 중"; \
			REDIS_OK=true; \
		else \
			echo "   └─ 상태: 설치되었으나 실행 중이 아님"; \
			echo "   └─ 시작: brew services start redis / sudo systemctl start redis"; \
		fi; \
	else \
		echo "❌ Redis: 미설치"; \
	fi; \
	echo ""; \
	echo "----------------------------------------------"; \
	if [ "$$POSTGRES_OK" = "false" ] || [ "$$REDIS_OK" = "false" ]; then \
		echo ""; \
		echo "⚠️  일부 DB가 설치되지 않았거나 실행 중이 아닙니다."; \
		echo ""; \
		echo "DB 설치 및 설정을 진행하시겠습니까? [Y/n]"; \
		read -r answer; \
		if [ "$$answer" != "n" ] && [ "$$answer" != "N" ]; then \
			$(MAKE) kind-setup-db; \
		else \
			echo ""; \
			echo "DB 설치를 건너뜁니다."; \
			echo "나중에 'make kind-setup-db'로 설치할 수 있습니다."; \
		fi; \
	else \
		echo "✅ 모든 DB가 정상적으로 실행 중입니다!"; \
		echo ""; \
		echo "다음 단계: make kind-setup"; \
	fi

kind-setup-db: ## 로컬 PostgreSQL/Redis 설치 및 설정 (Kind용)
	@echo "=== 로컬 PostgreSQL 및 Redis 설정 ==="
	@echo ""
	@# OS 감지
	@if [ "$$(uname)" = "Darwin" ]; then \
		echo "감지됨: macOS"; \
		$(MAKE) _setup-db-macos; \
	elif [ -f /etc/debian_version ]; then \
		echo "감지됨: Debian/Ubuntu"; \
		$(MAKE) _setup-db-debian; \
	else \
		echo "지원하지 않는 OS입니다. 수동으로 PostgreSQL과 Redis를 설치해주세요."; \
		echo "  PostgreSQL: 0.0.0.0:5432에서 수신 대기"; \
		echo "  Redis: 0.0.0.0:6379에서 수신 대기"; \
	fi
	@echo ""
	@echo "=============================================="
	@echo "  로컬 DB 설정 완료!"
	@echo "=============================================="
	@echo ""
	@echo "  PostgreSQL: localhost:5432 (사용자: postgres)"
	@echo "  Redis: localhost:6379"
	@echo ""
	@echo "  Kind 클러스터에서 접근 시 172.18.0.1 사용"
	@echo ""

_setup-db-macos:
	@# PostgreSQL 설치
	@if ! command -v psql >/dev/null 2>&1; then \
		echo "PostgreSQL 설치 중..."; \
		brew install postgresql@14; \
		brew services start postgresql@14; \
	else \
		echo "PostgreSQL 이미 설치됨"; \
		brew services start postgresql@14 2>/dev/null || brew services start postgresql 2>/dev/null || true; \
	fi
	@# Redis 설치
	@if ! command -v redis-cli >/dev/null 2>&1; then \
		echo "Redis 설치 중..."; \
		brew install redis; \
		brew services start redis; \
	else \
		echo "Redis 이미 설치됨"; \
		brew services start redis 2>/dev/null || true; \
	fi
	@# wealist 데이터베이스 생성
	@echo "wealist 데이터베이스 생성 중..."
	@psql -U postgres -c "SELECT 1" 2>/dev/null || createuser -s postgres 2>/dev/null || true
	@for db in wealist wealist_auth wealist_user wealist_board wealist_chat wealist_noti wealist_storage wealist_video; do \
		psql -U postgres -tc "SELECT 1 FROM pg_database WHERE datname = '$$db'" | grep -q 1 || \
		psql -U postgres -c "CREATE DATABASE $$db" 2>/dev/null || true; \
	done
	@echo "PostgreSQL 데이터베이스 준비 완료"

_setup-db-debian:
	@# PostgreSQL 설치
	@if ! command -v psql >/dev/null 2>&1; then \
		echo "PostgreSQL 설치 중..."; \
		sudo apt-get update && sudo apt-get install -y postgresql postgresql-contrib; \
	else \
		echo "PostgreSQL 이미 설치됨"; \
	fi
	@sudo systemctl start postgresql || true
	@# Kind 클러스터 접근용 PostgreSQL 설정
	@echo "Kind 클러스터 접근용 PostgreSQL 설정 중..."
	@PG_HBA=$$(sudo -u postgres psql -t -P format=unaligned -c "SHOW hba_file"); \
	if ! sudo grep -q "172.18.0.0/16" "$$PG_HBA" 2>/dev/null; then \
		echo "host    all    all    172.17.0.0/16    trust" | sudo tee -a "$$PG_HBA" >/dev/null; \
		echo "host    all    all    172.18.0.0/16    trust" | sudo tee -a "$$PG_HBA" >/dev/null; \
	fi
	@PG_CONF=$$(sudo -u postgres psql -t -P format=unaligned -c "SHOW config_file"); \
	sudo sed -i "s/#listen_addresses = 'localhost'/listen_addresses = '*'/" "$$PG_CONF" 2>/dev/null || true; \
	sudo sed -i "s/listen_addresses = 'localhost'/listen_addresses = '*'/" "$$PG_CONF" 2>/dev/null || true
	@sudo systemctl restart postgresql
	@# wealist 데이터베이스 생성
	@echo "wealist 데이터베이스 생성 중..."
	@for db in wealist wealist_auth wealist_user wealist_board wealist_chat wealist_noti wealist_storage wealist_video; do \
		sudo -u postgres psql -tc "SELECT 1 FROM pg_database WHERE datname = '$$db'" | grep -q 1 || \
		sudo -u postgres psql -c "CREATE DATABASE $$db" 2>/dev/null || true; \
	done
	@echo "PostgreSQL 데이터베이스 준비 완료"
	@# Redis 설치
	@if ! command -v redis-cli >/dev/null 2>&1; then \
		echo "Redis 설치 중..."; \
		sudo apt-get install -y redis-server; \
	else \
		echo "Redis 이미 설치됨"; \
	fi
	@# Kind 클러스터 접근용 Redis 설정
	@echo "Kind 클러스터 접근용 Redis 설정 중..."
	@sudo sed -i 's/^bind 127.0.0.1/bind 0.0.0.0/' /etc/redis/redis.conf 2>/dev/null || true
	@sudo sed -i 's/^protected-mode yes/protected-mode no/' /etc/redis/redis.conf 2>/dev/null || true
	@sudo systemctl restart redis-server || sudo systemctl restart redis
	@echo "Redis 준비 완료"

_setup-db-for-kind: ## (내부) Kind 클러스터에서 호스트 DB 접근 설정
	@echo "Kind 클러스터에서 호스트 DB 접근 설정 중..."
	@# Linux에서만 필요 (macOS는 Docker Desktop이 자동 처리)
	@if [ "$$(uname)" != "Darwin" ]; then \
		echo "PostgreSQL 설정 (0.0.0.0 바인딩)..."; \
		PG_HBA=$$(sudo -u postgres psql -t -P format=unaligned -c "SHOW hba_file" 2>/dev/null) || true; \
		if [ -n "$$PG_HBA" ]; then \
			if ! sudo grep -q "172.18.0.0/16" "$$PG_HBA" 2>/dev/null; then \
				echo "host    all    all    172.17.0.0/16    trust" | sudo tee -a "$$PG_HBA" >/dev/null; \
				echo "host    all    all    172.18.0.0/16    trust" | sudo tee -a "$$PG_HBA" >/dev/null; \
			fi; \
			PG_CONF=$$(sudo -u postgres psql -t -P format=unaligned -c "SHOW config_file" 2>/dev/null) || true; \
			if [ -n "$$PG_CONF" ]; then \
				sudo sed -i "s/#listen_addresses = 'localhost'/listen_addresses = '*'/" "$$PG_CONF" 2>/dev/null || true; \
				sudo sed -i "s/listen_addresses = 'localhost'/listen_addresses = '*'/" "$$PG_CONF" 2>/dev/null || true; \
			fi; \
			sudo systemctl restart postgresql 2>/dev/null || true; \
		fi; \
		echo "Redis 설정 (0.0.0.0 바인딩)..."; \
		if [ -f /etc/redis/redis.conf ]; then \
			sudo sed -i 's/^bind 127.0.0.1/bind 0.0.0.0/' /etc/redis/redis.conf 2>/dev/null || true; \
			sudo sed -i 's/^protected-mode yes/protected-mode no/' /etc/redis/redis.conf 2>/dev/null || true; \
			sudo systemctl restart redis-server 2>/dev/null || sudo systemctl restart redis 2>/dev/null || true; \
		fi; \
		echo "✅ DB 접근 설정 완료 (Kind → Host 172.18.0.1)"; \
	else \
		echo "ℹ️  macOS: Docker Desktop이 자동으로 host.docker.internal을 제공합니다."; \
	fi

kind-load-images: ## 모든 이미지 빌드/풀 (인프라 + 백엔드 서비스)
	@echo "=== 모든 이미지 로드 ==="
	@echo ""
	@echo "--- 인프라 이미지 로드 중 ---"
	./k8s/helm/scripts/dev/1.load_infra_images.sh
	@echo ""
	@echo "--- 백엔드 서비스 이미지 빌드 중 ---"
	SKIP_FRONTEND=true ./k8s/helm/scripts/dev/2.build_services_and_load.sh
	@echo ""
	@echo "모든 이미지 로드 완료!"
	@echo ""
	@echo "다음: make helm-install-all ENV=dev"

kind-load-images-ex-db: ## 서비스 이미지만 로드 (PostgreSQL/Redis 제외 - 외부 DB 사용 시)
	@echo "=== 서비스 이미지 로드 (DB 이미지 제외) ==="
	@echo ""
	@echo "※ 외부 DB(호스트 PC의 PostgreSQL/Redis)를 사용하므로"
	@echo "  DB 이미지는 로드하지 않습니다."
	@echo ""
	@echo "--- 인프라 이미지 로드 중 (DB 제외) ---"
	SKIP_DB=true ./k8s/helm/scripts/dev/1.load_infra_images.sh
	@echo ""
	@echo "--- 백엔드 서비스 이미지 빌드 중 ---"
	SKIP_FRONTEND=true ./k8s/helm/scripts/dev/2.build_services_and_load.sh
	@echo ""
	@echo "서비스 이미지 로드 완료! (DB 제외)"
	@echo ""
	@echo "다음: make helm-install-all ENV=dev"

kind-load-images-all: ## 🏠 모든 이미지 로드 (DB + Backend + Frontend - localhost 환경용)
	@echo "=== 모든 이미지 로드 (localhost 환경) ==="
	@echo ""
	@echo "※ DB, Backend, Frontend 모든 이미지를 빌드/로드합니다."
	@echo ""
	@echo "--- 인프라 이미지 로드 중 (DB 포함) ---"
	./k8s/helm/scripts/localhost/1.load_infra_images.sh
	@echo ""
	@echo "--- 서비스 이미지 빌드 중 (Backend + Frontend) ---"
	./k8s/helm/scripts/localhost/2.build_all_and_load.sh
	@echo ""
	@echo "모든 이미지 로드 완료!"
	@echo ""
	@echo "다음: make helm-install-all ENV=localhost"

kind-load-images-mono: ## Go 서비스를 모노레포 패턴으로 빌드 (더 빠른 리빌드)
	@echo "=== 모노레포 빌드로 이미지 로드 (BuildKit 캐시) ==="
	@echo ""
	@echo "--- 인프라 이미지 로드 중 ---"
	./k8s/helm/scripts/dev/1.load_infra_images.sh
	@echo ""
	@echo "--- Go 서비스 빌드 중 (모노레포 패턴) ---"
	./docker/scripts/dev-mono.sh build
	@echo ""
	@echo "--- 로컬 레지스트리에 태그 및 푸시 중 ---"
	@for svc in user-service board-service chat-service noti-service storage-service video-service; do \
		echo "$$svc 푸시 중..."; \
		docker tag wealist/$$svc:latest $(LOCAL_REGISTRY)/$$svc:$(IMAGE_TAG); \
		docker push $(LOCAL_REGISTRY)/$$svc:$(IMAGE_TAG); \
	done
	@echo ""
	@echo "--- auth-service 빌드 중 ---"
	@$(MAKE) auth-service-load
	@echo ""
	@echo "모든 이미지 로드 완료! (모노레포 패턴)"
	@echo ""
	@echo "다음: make helm-install-all ENV=dev"

# =============================================================================
# 개별 이미지 로드 명령어 (세분화)
# =============================================================================

kind-load-infra: ## 🔧 인프라 이미지만 로드 (MinIO, LiveKit)
	@echo "=== 인프라 이미지 로드 ==="
	ONLY_INFRA=true ./k8s/helm/scripts/dev/1.load_infra_images.sh

kind-load-monitoring: ## 📊 모니터링 이미지만 로드 (Prometheus, Grafana, Loki, Exporters)
	@echo "=== 모니터링 이미지 로드 ==="
	ONLY_MONITORING=true ./k8s/helm/scripts/dev/1.load_infra_images.sh

kind-load-services: ## 🚀 서비스 이미지만 로드 (Backend 서비스)
	@echo "=== 서비스 이미지 로드 ==="
	@echo ""
	@echo "--- 백엔드 서비스 이미지 빌드 중 ---"
	SKIP_FRONTEND=true ./k8s/helm/scripts/dev/2.build_services_and_load.sh
	@echo ""
	@echo "서비스 이미지 로드 완료!"

kind-delete: ## 클러스터 삭제
	@echo "Kind 클러스터 삭제 중..."
	kind delete cluster --name $(KIND_CLUSTER)
	@docker rm -f kind-registry 2>/dev/null || true
	@echo "클러스터 삭제 완료!"

kind-recover: ## 재부팅 후 클러스터 복구
	@echo "Kind 클러스터 복구 중..."
	@docker restart $(KIND_CLUSTER)-control-plane $(KIND_CLUSTER)-worker $(KIND_CLUSTER)-worker2 kind-registry 2>/dev/null || true
	@sleep 30
	@kind export kubeconfig --name $(KIND_CLUSTER)
	@echo "API 서버 대기 중..."
	@until kubectl get nodes >/dev/null 2>&1; do sleep 5; done
	@echo "클러스터 복구 완료!"
	@kubectl get nodes

##@ 로컬 도메인 (local.wealist.co.kr)

.PHONY: local-tls-secret

local-tls-secret: ## local.wealist.co.kr용 TLS 시크릿 생성
	@echo "=== local.wealist.co.kr용 TLS 시크릿 생성 ==="
	@if kubectl get secret local-wealist-tls -n $(K8S_NAMESPACE) >/dev/null 2>&1; then \
		echo "TLS 시크릿이 이미 존재함, 건너뜀..."; \
	else \
		echo "자체 서명 인증서 생성 중..."; \
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
		echo "TLS 시크릿 생성 완료"; \
	fi

##@ 로컬 데이터베이스

.PHONY: init-local-db

init-local-db: ## 로컬 PostgreSQL/Redis 초기화 (Ubuntu, ENV=local-ubuntu)
	@echo "Wealist용 로컬 PostgreSQL 및 Redis 초기화 중..."
	@echo ""
	@echo "이 작업은 로컬 PostgreSQL과 Redis가 Kind 클러스터(Docker 네트워크)의"
	@echo "연결을 수락하도록 설정합니다."
	@echo ""
	@echo "사전 요구사항:"
	@echo "  - PostgreSQL 설치: sudo apt install postgresql postgresql-contrib"
	@echo "  - Redis 설치: sudo apt install redis-server"
	@echo ""
	@echo "sudo로 스크립트 실행 중..."
	@sudo ./scripts/init-local-postgres.sh
	@sudo ./scripts/init-local-redis.sh
	@echo ""
	@echo "로컬 데이터베이스 초기화 완료!"
	@echo ""
	@echo "다음: make helm-install-all ENV=dev"
