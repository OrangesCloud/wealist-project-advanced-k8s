# =============================================================================
# Helm 차트 명령어
# =============================================================================

##@ Helm 차트 (권장)

.PHONY: helm-deps-build helm-lint helm-validate
.PHONY: helm-install-cert-manager helm-install-infra helm-install-services helm-install-frontend helm-install-istio-config helm-install-istio-addons helm-install-monitoring
.PHONY: helm-install-all helm-install-all-init helm-upgrade-all helm-uninstall-all
.PHONY: helm-setup-route53-secret helm-check-secrets helm-check-db
.PHONY: helm-localhost helm-local-ubuntu helm-dev helm-staging helm-prod

# Helm으로 배포할 서비스 목록 (백엔드만)
# 프론트엔드는 별도 배포 (CDN/S3 또는 npm run dev)
HELM_SERVICES = $(BACKEND_SERVICES)

helm-deps-build: ## 모든 Helm 의존성 빌드
	@echo "모든 Helm 의존성 빌드 중..."
	@helm dependency update ./k8s/helm/charts/wealist-common 2>/dev/null || true
	@for chart in $(HELM_SERVICES); do \
		echo "$$chart 의존성 업데이트 중..."; \
		helm dependency update ./k8s/helm/charts/$$chart; \
	done
	@helm dependency update ./k8s/helm/charts/frontend 2>/dev/null || true
	@helm dependency update ./k8s/helm/charts/wealist-infrastructure
	@helm dependency update ./k8s/helm/charts/cert-manager-config 2>/dev/null || true
	@echo "모든 의존성 빌드 완료!"

helm-lint: ## 모든 Helm 차트 린트
	@echo "모든 Helm 차트 린트 중..."
	@helm lint ./k8s/helm/charts/wealist-common
	@helm lint ./k8s/helm/charts/wealist-infrastructure
	@helm lint ./k8s/helm/charts/istio-config
	@helm lint ./k8s/helm/charts/cert-manager-config 2>/dev/null || echo "cert-manager-config: 먼저 'helm dependency update' 실행 필요"
	@for service in $(HELM_SERVICES); do \
		echo "$$service 린트 중..."; \
		helm lint ./k8s/helm/charts/$$service; \
	done
	@echo "모든 차트 린트 성공!"

helm-validate: ## Helm 종합 검증 실행
	@echo "Helm 종합 검증 실행 중..."
	@./k8s/helm/scripts/validate-all-charts.sh
	@echo ""
	@echo "ArgoCD Applications 검증 실행 중..."
	@./k8s/argocd/scripts/validate-applications.sh

##@ Helm 설치

# -----------------------------------------------------------------------------
# secrets 파일 체크 (로컬 환경 공통)
# -----------------------------------------------------------------------------
helm-check-secrets: ## secrets.yaml 파일 존재 여부 확인
	@echo "=============================================="
	@echo "  시크릿 파일 확인 중"
	@echo "=============================================="
	@if [ "$(ENV)" = "dev" ] || [ "$(ENV)" = "localhost" ] || [ "$(ENV)" = "staging" ] || [ "$(ENV)" = "prod" ]; then \
		if [ ! -f "./k8s/helm/environments/secrets.yaml" ]; then \
			echo ""; \
			echo "❌ 오류: 시크릿 파일이 없습니다!"; \
			echo ""; \
			echo "다음 명령어로 시크릿 파일을 생성하세요:"; \
			echo ""; \
			echo "  cp ./k8s/helm/environments/secrets.example.yaml ./k8s/helm/environments/secrets.yaml"; \
			echo ""; \
			echo "그 후 secrets.yaml 파일을 열어 다음 값들을 설정하세요:"; \
			echo "  - GOOGLE_CLIENT_ID: Google OAuth 클라이언트 ID"; \
			echo "  - GOOGLE_CLIENT_SECRET: Google OAuth 클라이언트 시크릿"; \
			echo "  - JWT_SECRET: JWT 서명용 비밀키 (64자 이상 권장)"; \
			echo ""; \
			echo "※ 시크릿 파일은 .gitignore에 포함되어 있어 커밋되지 않습니다."; \
			echo ""; \
			exit 1; \
		else \
			echo "✅ 시크릿 파일 확인됨: ./k8s/helm/environments/secrets.yaml"; \
		fi; \
	else \
		echo "ℹ️  $(ENV) 환경은 시크릿 파일이 선택사항입니다."; \
	fi

# -----------------------------------------------------------------------------
# DB 연결 체크 (외부 DB 사용 시 필수, localhost는 내부 Pod 사용으로 스킵)
# -----------------------------------------------------------------------------
helm-check-db: ## PostgreSQL/Redis 실행 상태 확인 (외부 DB 사용 환경)
ifeq ($(ENV),localhost)
	@echo "=============================================="
	@echo "  데이터베이스 확인 (localhost)"
	@echo "=============================================="
	@echo ""
	@echo "ℹ️  localhost 환경은 내부 PostgreSQL/Redis Pod를 사용합니다."
	@echo "   외부 DB 체크를 건너뜁니다."
	@echo ""
else
	@echo "=============================================="
	@echo "  데이터베이스 연결 확인 중 ($(ENV))"
	@echo "=============================================="
	@echo ""
	@POSTGRES_OK=false; \
	REDIS_OK=false; \
	if command -v psql >/dev/null 2>&1; then \
		if pg_isready >/dev/null 2>&1 || (command -v systemctl >/dev/null 2>&1 && systemctl is-active postgresql >/dev/null 2>&1) || (command -v brew >/dev/null 2>&1 && brew services list 2>/dev/null | grep -q "postgresql.*started"); then \
			echo "✅ PostgreSQL: 실행 중"; \
			POSTGRES_OK=true; \
		else \
			echo "❌ PostgreSQL: 설치되었으나 실행 중이 아님"; \
		fi; \
	else \
		echo "❌ PostgreSQL: 미설치"; \
	fi; \
	if command -v redis-cli >/dev/null 2>&1; then \
		if redis-cli ping >/dev/null 2>&1; then \
			echo "✅ Redis: 실행 중"; \
			REDIS_OK=true; \
		else \
			echo "❌ Redis: 설치되었으나 실행 중이 아님"; \
		fi; \
	else \
		echo "❌ Redis: 미설치"; \
	fi; \
	echo ""; \
	if [ "$$POSTGRES_OK" = "false" ] || [ "$$REDIS_OK" = "false" ]; then \
		echo "============================================"; \
		echo "❌ 오류: 데이터베이스가 준비되지 않았습니다!"; \
		echo "============================================"; \
		echo ""; \
		echo "$(ENV) 환경은 호스트 PC의 PostgreSQL/Redis를 사용합니다."; \
		echo ""; \
		echo "해결 방법:"; \
		echo "  1. DB 설치 및 시작:"; \
		echo "     make kind-setup-db"; \
		echo ""; \
		echo "  2. 또는 수동으로 시작:"; \
		echo "     (macOS)  brew services start postgresql redis"; \
		echo "     (Ubuntu) sudo systemctl start postgresql redis"; \
		echo ""; \
		exit 1; \
	else \
		echo "✅ 모든 데이터베이스 연결 확인 완료!"; \
	fi
endif

helm-setup-route53-secret: ## Route53 인증 시크릿 설정 (cert-manager용)
	@echo "Route53 인증 시크릿 설정 중..."
	@kubectl create namespace cert-manager 2>/dev/null || true
	@if [ -z "$$AWS_SECRET_ACCESS_KEY" ]; then \
		echo "오류: AWS_SECRET_ACCESS_KEY 환경변수가 설정되지 않았습니다"; \
		echo "사용법: AWS_SECRET_ACCESS_KEY=xxx make helm-setup-route53-secret"; \
		exit 1; \
	fi
	@kubectl create secret generic route53-credentials \
		--namespace cert-manager \
		--from-literal=secret-access-key=$$AWS_SECRET_ACCESS_KEY \
		--dry-run=client -o yaml | kubectl apply -f -
	@echo "Route53 인증 시크릿 생성/업데이트 완료!"

helm-install-cert-manager: ## cert-manager 설치 (환경에서 활성화된 경우)
	@echo "cert-manager 설정 확인 중 (ENV=$(ENV))..."
	@if grep -q "certManager:" "$(HELM_ENV_VALUES)" 2>/dev/null && \
	   grep -A1 "certManager:" "$(HELM_ENV_VALUES)" | grep -q "enabled: true"; then \
		echo "cert-manager-config 설치 중..."; \
		cd ./k8s/helm/charts/cert-manager-config && helm dependency update && cd -; \
		helm upgrade --install cert-manager-config ./k8s/helm/charts/cert-manager-config \
			-f $(HELM_BASE_VALUES) \
			-f $(HELM_ENV_VALUES) $(HELM_SECRETS_FLAG) \
			-n cert-manager --create-namespace --wait --timeout 5m; \
		echo "cert-manager 설치 완료!"; \
		echo "cert-manager 웹훅 준비 대기 중..."; \
		sleep 10; \
	else \
		echo "cert-manager 건너뜀 ($(ENV) 환경에서 비활성화됨)"; \
	fi

helm-install-infra: ## 인프라 차트 설치 (EXTERNAL_DB가 DB 배포 결정)
	@echo "인프라 설치 중 (ENV=$(ENV), NS=$(K8S_NAMESPACE), EXTERNAL_DB=$(EXTERNAL_DB))..."
ifeq ($(EXTERNAL_DB),true)
	@if [ -f /tmp/kind_db_host.env ]; then \
		. /tmp/kind_db_host.env; \
		echo "외부 DB 사용 (Host: $$DB_HOST)"; \
		helm upgrade --install wealist-infrastructure ./k8s/helm/charts/wealist-infrastructure \
			-f $(HELM_BASE_VALUES) \
			-f $(HELM_ENV_VALUES) $(HELM_SECRETS_FLAG) \
			--set postgres.enabled=false \
			--set postgres.external.enabled=true \
			--set postgres.external.host=$$DB_HOST \
			--set redis.enabled=false \
			--set redis.external.enabled=true \
			--set redis.external.host=$$DB_HOST \
			--set shared.config.DB_HOST=$$DB_HOST \
			--set shared.config.POSTGRES_HOST=$$DB_HOST \
			--set shared.config.REDIS_HOST=$$DB_HOST \
			-n $(K8S_NAMESPACE) --create-namespace; \
	else \
		echo "⚠️  /tmp/kind_db_host.env 없음 - 기본값 사용"; \
		helm upgrade --install wealist-infrastructure ./k8s/helm/charts/wealist-infrastructure \
			-f $(HELM_BASE_VALUES) \
			-f $(HELM_ENV_VALUES) $(HELM_SECRETS_FLAG) \
			--set postgres.enabled=false \
			--set postgres.external.enabled=true \
			--set redis.enabled=false \
			--set redis.external.enabled=true \
			-n $(K8S_NAMESPACE) --create-namespace; \
	fi
else
	@echo "내부 데이터베이스 사용 중 (클러스터 내 PostgreSQL/Redis 파드)"
	helm upgrade --install wealist-infrastructure ./k8s/helm/charts/wealist-infrastructure \
		-f $(HELM_BASE_VALUES) \
		-f $(HELM_ENV_VALUES) $(HELM_SECRETS_FLAG) \
		--set postgres.enabled=true \
		--set postgres.external.enabled=false \
		--set redis.enabled=true \
		--set redis.external.enabled=false \
		-n $(K8S_NAMESPACE) --create-namespace
endif
	@echo "인프라 설치 완료!"

helm-install-services: ## 모든 서비스 차트 설치
	@echo "서비스 설치 중 (ENV=$(ENV), NS=$(K8S_NAMESPACE), EXTERNAL_DB=$(EXTERNAL_DB))..."
	@echo "설치할 서비스: $(HELM_SERVICES)"
	@# dev 환경: AWS Account ID 자동 확인 및 설정
ifeq ($(ENV),dev)
	@if grep -q "<AWS_ACCOUNT_ID>" "$(HELM_ENV_VALUES)" 2>/dev/null; then \
		echo "⚠️  dev.yaml에 <AWS_ACCOUNT_ID> 플레이스홀더가 남아있습니다."; \
		if command -v aws >/dev/null 2>&1 && aws sts get-caller-identity >/dev/null 2>&1; then \
			AWS_ACCOUNT_ID=$$(aws sts get-caller-identity --query Account --output text); \
			echo "🔧 AWS Account ID 자동 업데이트 중: $$AWS_ACCOUNT_ID"; \
			if [ "$$(uname)" = "Darwin" ]; then \
				sed -i '' "s/<AWS_ACCOUNT_ID>/$$AWS_ACCOUNT_ID/g" "$(HELM_ENV_VALUES)"; \
			else \
				sed -i "s/<AWS_ACCOUNT_ID>/$$AWS_ACCOUNT_ID/g" "$(HELM_ENV_VALUES)"; \
			fi; \
			echo "✅ dev.yaml 업데이트 완료!"; \
		else \
			echo "❌ AWS CLI 로그인이 필요합니다."; \
			echo "   aws sso login 또는 aws configure 후 다시 시도하세요."; \
			exit 1; \
		fi; \
	fi
endif
ifeq ($(EXTERNAL_DB),true)
	@echo "EXTERNAL_DB=true: 외부 DB 사용"
	@if [ -f /tmp/kind_db_host.env ]; then \
		. /tmp/kind_db_host.env; \
		echo "  DB/Redis Host: $$DB_HOST"; \
		for service in $(HELM_SERVICES); do \
			echo "$$service 설치 중..."; \
			db_user=$$(echo $$service | sed 's/-/_/g'); \
			db_name="wealist_$${db_user}_db"; \
			db_url="postgresql://$${db_user}:postgres@$${DB_HOST}:5432/$${db_name}?sslmode=disable"; \
			echo "  DATABASE_URL: $$db_url"; \
			helm upgrade --install $$service ./k8s/helm/charts/$$service \
				-f $(HELM_BASE_VALUES) \
				-f $(HELM_ENV_VALUES) $(HELM_SECRETS_FLAG) \
				--set shared.config.DB_HOST=$$DB_HOST \
				--set shared.config.POSTGRES_HOST=$$DB_HOST \
				--set shared.config.REDIS_HOST=$$DB_HOST \
				--set config.DATABASE_URL=$$db_url \
				-n $(K8S_NAMESPACE); \
		done; \
	else \
		echo "⚠️  /tmp/kind_db_host.env 없음. 기본값 사용"; \
		for service in $(HELM_SERVICES); do \
			echo "$$service 설치 중..."; \
			helm upgrade --install $$service ./k8s/helm/charts/$$service \
				-f $(HELM_BASE_VALUES) \
				-f $(HELM_ENV_VALUES) $(HELM_SECRETS_FLAG) \
				-n $(K8S_NAMESPACE); \
		done; \
	fi
else
	@echo "EXTERNAL_DB=false: 내부 DB 파드 사용, auto-migrate 활성화"
	@for service in $(HELM_SERVICES); do \
		echo "$$service (DB auto-migrate 포함) 설치 중..."; \
		helm upgrade --install $$service ./k8s/helm/charts/$$service \
			-f $(HELM_BASE_VALUES) \
			-f $(HELM_ENV_VALUES) $(HELM_SECRETS_FLAG) \
			--set shared.config.DB_AUTO_MIGRATE=true \
			-n $(K8S_NAMESPACE); \
	done
endif
	@echo "모든 서비스 설치 완료!"
	@echo ""
	@echo "다음: make status"

helm-install-frontend: ## 프론트엔드 설치 (localhost.yaml에서 frontend.enabled=true인 경우)
	@echo "프론트엔드 설정 확인 중 (ENV=$(ENV))..."
	@if grep -A1 "^frontend:" "$(HELM_ENV_VALUES)" 2>/dev/null | grep -q "enabled: true"; then \
		echo "frontend 설치 중..."; \
		helm upgrade --install frontend ./k8s/helm/charts/frontend \
			-f $(HELM_BASE_VALUES) \
			-f $(HELM_ENV_VALUES) $(HELM_SECRETS_FLAG) \
			-n $(K8S_NAMESPACE); \
		echo "프론트엔드 설치 완료!"; \
	else \
		echo "프론트엔드 건너뜀 ($(ENV) 환경에서 비활성화됨)"; \
	fi

helm-install-monitoring: ## 모니터링 스택 설치 (Prometheus, Loki, Grafana)
	@echo "모니터링 스택 설치 중 (ENV=$(ENV), NS=$(K8S_NAMESPACE), EXTERNAL_DB=$(EXTERNAL_DB))..."
ifeq ($(EXTERNAL_DB),true)
	@if [ -f /tmp/kind_db_host.env ]; then \
		. /tmp/kind_db_host.env; \
		echo "외부 DB exporter 사용 (host: $$DB_HOST)"; \
		helm upgrade --install wealist-monitoring ./k8s/helm/charts/wealist-monitoring \
			-f $(HELM_BASE_VALUES) \
			-f $(HELM_ENV_VALUES) $(HELM_SECRETS_FLAG) \
			--set global.namespace=$(K8S_NAMESPACE) \
			--set postgresExporter.config.host=$$DB_HOST \
			--set redisExporter.config.host=$$DB_HOST \
			-n $(K8S_NAMESPACE); \
	else \
		echo "외부 DB exporter 사용 (host: 기본값)"; \
		helm upgrade --install wealist-monitoring ./k8s/helm/charts/wealist-monitoring \
			-f $(HELM_BASE_VALUES) \
			-f $(HELM_ENV_VALUES) $(HELM_SECRETS_FLAG) \
			--set global.namespace=$(K8S_NAMESPACE) \
			-n $(K8S_NAMESPACE); \
	fi
else
	@echo "내부 데이터베이스 exporter 사용 (host: postgres/redis 서비스)"
	helm upgrade --install wealist-monitoring ./k8s/helm/charts/wealist-monitoring \
		-f $(HELM_BASE_VALUES) \
		-f $(HELM_ENV_VALUES) $(HELM_SECRETS_FLAG) \
		--set global.namespace=$(K8S_NAMESPACE) \
		--set postgresExporter.config.host=postgres \
		--set redisExporter.config.host=redis \
		-n $(K8S_NAMESPACE)
endif
	@echo ""
	@echo "=============================================="
	@echo "  모니터링 스택 설치 성공!"
	@echo "=============================================="
	@echo ""
	@echo "  📊 모니터링 URL (Ingress 경유):"
	@echo "    - Grafana:    $(PROTOCOL)://$(DOMAIN)/api/monitoring/grafana"
	@echo "    - Prometheus: $(PROTOCOL)://$(DOMAIN)/api/monitoring/prometheus"
	@echo "    - Loki:       $(PROTOCOL)://$(DOMAIN)/api/monitoring/loki"
	@echo ""
	@echo "  🌐 Istio 관측성 (setup 시 자동 설치됨):"
	@echo "    - Kiali:      $(PROTOCOL)://$(DOMAIN)/api/monitoring/kiali"
	@echo "    - Jaeger:     $(PROTOCOL)://$(DOMAIN)/api/monitoring/jaeger"
	@echo ""
	@echo "  🔐 Grafana 로그인: admin / admin"
	@echo "=============================================="

helm-install-istio-config: ## Istio 설정 설치 (HTTPRoute, DestinationRules 등)
	@echo "Istio 설정 설치 중 (ENV=$(ENV), NS=$(K8S_NAMESPACE))..."
	@helm upgrade --install istio-config ./k8s/helm/charts/istio-config \
		-f $(HELM_BASE_VALUES) \
		-f $(HELM_ENV_VALUES) \
		-n $(K8S_NAMESPACE) --wait
	@echo ""
	@echo "Istio 설정 설치 완료! (HTTPRoute, PeerAuthentication, DestinationRules)"

helm-install-istio-addons: ## Istio Addons 설치 (Kiali, Jaeger - istio-system 네임스페이스)
	@echo "Istio Addons 설치 중 (Kiali, Jaeger)..."
	@if grep -q "kiali:" "$(HELM_ENV_VALUES)" 2>/dev/null && grep -A1 "kiali:" "$(HELM_ENV_VALUES)" | grep -q "enabled: true"; then \
		echo "기존 Kiali/Jaeger/Zipkin 리소스 정리 중 (setup 스크립트로 설치된 경우)..."; \
		kubectl delete deployment,service,serviceaccount,configmap -l app=kiali -n istio-system --ignore-not-found 2>/dev/null || true; \
		kubectl delete clusterrole,clusterrolebinding kiali --ignore-not-found 2>/dev/null || true; \
		kubectl delete clusterrole,clusterrolebinding kiali-viewer --ignore-not-found 2>/dev/null || true; \
		kubectl delete deployment,service -l app=jaeger -n istio-system --ignore-not-found 2>/dev/null || true; \
		kubectl delete deployment,service tracing zipkin jaeger-query jaeger-collector -n istio-system --ignore-not-found 2>/dev/null || true; \
		echo "Helm으로 Istio Addons 설치 중..."; \
		helm upgrade --install istio-addons ./k8s/helm/charts/istio-addons \
			-f $(HELM_BASE_VALUES) \
			-f $(HELM_ENV_VALUES) \
			--set prometheus.enabled=false \
			--set grafana.enabled=false \
			-n istio-system; \
		echo "Istio Addons 설치 완료! (Kiali, Jaeger)"; \
	else \
		echo "Istio Addons 건너뜀 ($(ENV) 환경에서 Kiali 비활성화됨)"; \
	fi

# -----------------------------------------------------------------------------
# helm-install-all: secrets 체크 → 의존성 → 인프라 → 서비스 → Istio → 모니터링
# -----------------------------------------------------------------------------
# Note: Istio Gateway는 0.setup-cluster.sh에서 생성, HTTPRoute는 여기서 설치
helm-install-all: helm-check-secrets helm-check-db helm-deps-build helm-install-cert-manager helm-install-infra ## 전체 차트 설치 (인프라 + 서비스 + 프론트엔드 + Istio + 모니터링)
	@sleep 5
	@$(MAKE) helm-install-services ENV=$(ENV)
	@sleep 2
	@$(MAKE) helm-install-frontend ENV=$(ENV)
	@sleep 3
	@$(MAKE) helm-install-istio-config ENV=$(ENV)
	@sleep 2
	@$(MAKE) helm-install-istio-addons ENV=$(ENV)
	@sleep 2
	@$(MAKE) helm-install-monitoring ENV=$(ENV)
	@echo ""
	@echo "=============================================="
	@echo "  전체 설치 완료!"
	@echo "=============================================="
	@echo ""
	@echo "  상태 확인: make status"
	@echo "  Pod 로그:  kubectl logs -n $(K8S_NAMESPACE) <pod-name>"
	@echo "=============================================="

helm-install-all-init: helm-check-secrets helm-deps-build helm-install-cert-manager helm-install-infra ## DB 마이그레이션 포함 전체 설치 (최초 설치용)
	@sleep 5
	@echo "DB 마이그레이션 활성화하여 서비스 설치 중 (최초 설정)..."
	@echo "설치할 서비스: $(HELM_SERVICES)"
ifeq ($(EXTERNAL_DB),true)
	@if [ -f /tmp/kind_db_host.env ]; then \
		. /tmp/kind_db_host.env; \
		echo "  외부 DB Host: $$DB_HOST"; \
		for service in $(HELM_SERVICES); do \
			echo "$$service (DB 마이그레이션 포함) 설치 중..."; \
			helm install $$service ./k8s/helm/charts/$$service \
				-f $(HELM_BASE_VALUES) \
				-f $(HELM_ENV_VALUES) $(HELM_SECRETS_FLAG) \
				--set shared.config.DB_AUTO_MIGRATE=true \
				--set shared.config.DB_HOST=$$DB_HOST \
				--set shared.config.POSTGRES_HOST=$$DB_HOST \
				--set shared.config.REDIS_HOST=$$DB_HOST \
				-n $(K8S_NAMESPACE); \
		done; \
	else \
		for service in $(HELM_SERVICES); do \
			echo "$$service (DB 마이그레이션 포함) 설치 중..."; \
			helm install $$service ./k8s/helm/charts/$$service \
				-f $(HELM_BASE_VALUES) \
				-f $(HELM_ENV_VALUES) $(HELM_SECRETS_FLAG) \
				--set shared.config.DB_AUTO_MIGRATE=true \
				-n $(K8S_NAMESPACE); \
		done; \
	fi
else
	@for service in $(HELM_SERVICES); do \
		echo "$$service (DB 마이그레이션 포함) 설치 중..."; \
		helm install $$service ./k8s/helm/charts/$$service \
			-f $(HELM_BASE_VALUES) \
			-f $(HELM_ENV_VALUES) $(HELM_SECRETS_FLAG) \
			--set shared.config.DB_AUTO_MIGRATE=true \
			-n $(K8S_NAMESPACE); \
	done
endif
	@echo "최초 설정 완료! 이후 배포는 DB 마이그레이션이 건너뜁니다."

##@ Helm 업그레이드/삭제

helm-upgrade-all: helm-deps-build ## 전체 차트 업그레이드
	@echo "전체 차트 업그레이드 중 (ENV=$(ENV), NS=$(K8S_NAMESPACE), EXTERNAL_DB=$(EXTERNAL_DB))..."
	@echo "업그레이드할 서비스: $(HELM_SERVICES)"
ifeq ($(EXTERNAL_DB),true)
	@if [ -f /tmp/kind_db_host.env ]; then \
		. /tmp/kind_db_host.env; \
		echo "  외부 DB Host: $$DB_HOST"; \
		helm upgrade wealist-infrastructure ./k8s/helm/charts/wealist-infrastructure \
			-f $(HELM_BASE_VALUES) \
			-f $(HELM_ENV_VALUES) $(HELM_SECRETS_FLAG) \
			--set postgres.enabled=false \
			--set postgres.external.enabled=true \
			--set postgres.external.host=$$DB_HOST \
			--set redis.enabled=false \
			--set redis.external.enabled=true \
			--set redis.external.host=$$DB_HOST \
			-n $(K8S_NAMESPACE); \
	else \
		echo "⚠️  /tmp/kind_db_host.env 없음"; \
		helm upgrade wealist-infrastructure ./k8s/helm/charts/wealist-infrastructure \
			-f $(HELM_BASE_VALUES) \
			-f $(HELM_ENV_VALUES) $(HELM_SECRETS_FLAG) \
			--set postgres.enabled=false \
			--set postgres.external.enabled=true \
			--set redis.enabled=false \
			--set redis.external.enabled=true \
			-n $(K8S_NAMESPACE); \
	fi
else
	@helm upgrade wealist-infrastructure ./k8s/helm/charts/wealist-infrastructure \
		-f $(HELM_BASE_VALUES) \
		-f $(HELM_ENV_VALUES) $(HELM_SECRETS_FLAG) \
		--set postgres.enabled=true \
		--set postgres.external.enabled=false \
		--set redis.enabled=true \
		--set redis.external.enabled=false \
		-n $(K8S_NAMESPACE)
endif
ifeq ($(EXTERNAL_DB),true)
	@if [ -f /tmp/kind_db_host.env ]; then \
		. /tmp/kind_db_host.env; \
		for service in $(HELM_SERVICES); do \
			echo "$$service 업그레이드 중..."; \
			helm upgrade $$service ./k8s/helm/charts/$$service \
				-f $(HELM_BASE_VALUES) \
				-f $(HELM_ENV_VALUES) $(HELM_SECRETS_FLAG) \
				--set shared.config.DB_HOST=$$DB_HOST \
				--set shared.config.POSTGRES_HOST=$$DB_HOST \
				--set shared.config.REDIS_HOST=$$DB_HOST \
				-n $(K8S_NAMESPACE); \
		done; \
	else \
		for service in $(HELM_SERVICES); do \
			echo "$$service 업그레이드 중..."; \
			helm upgrade $$service ./k8s/helm/charts/$$service \
				-f $(HELM_BASE_VALUES) \
				-f $(HELM_ENV_VALUES) $(HELM_SECRETS_FLAG) \
				-n $(K8S_NAMESPACE); \
		done; \
	fi
else
	@for service in $(HELM_SERVICES); do \
		echo "$$service (DB auto-migrate 포함) 업그레이드 중..."; \
		helm upgrade $$service ./k8s/helm/charts/$$service \
			-f $(HELM_BASE_VALUES) \
			-f $(HELM_ENV_VALUES) $(HELM_SECRETS_FLAG) \
			--set shared.config.DB_AUTO_MIGRATE=true \
			-n $(K8S_NAMESPACE); \
	done
endif
ifeq ($(EXTERNAL_DB),true)
	@if [ -f /tmp/kind_db_host.env ]; then \
		. /tmp/kind_db_host.env; \
		helm upgrade wealist-monitoring ./k8s/helm/charts/wealist-monitoring \
			-f $(HELM_BASE_VALUES) \
			-f $(HELM_ENV_VALUES) $(HELM_SECRETS_FLAG) \
			--set global.namespace=$(K8S_NAMESPACE) \
			--set postgresExporter.config.host=$$DB_HOST \
			--set redisExporter.config.host=$$DB_HOST \
			-n $(K8S_NAMESPACE) 2>/dev/null || echo "모니터링 미설치, 업그레이드 건너뜀"; \
	else \
		helm upgrade wealist-monitoring ./k8s/helm/charts/wealist-monitoring \
			-f $(HELM_BASE_VALUES) \
			-f $(HELM_ENV_VALUES) $(HELM_SECRETS_FLAG) \
			--set global.namespace=$(K8S_NAMESPACE) \
			-n $(K8S_NAMESPACE) 2>/dev/null || echo "모니터링 미설치, 업그레이드 건너뜀"; \
	fi
else
	@helm upgrade wealist-monitoring ./k8s/helm/charts/wealist-monitoring \
		-f $(HELM_BASE_VALUES) \
		-f $(HELM_ENV_VALUES) $(HELM_SECRETS_FLAG) \
		--set global.namespace=$(K8S_NAMESPACE) \
		--set postgresExporter.config.host=postgres \
		--set redisExporter.config.host=redis \
		-n $(K8S_NAMESPACE) 2>/dev/null || echo "모니터링 미설치, 업그레이드 건너뜀"
endif
	@echo "전체 차트 업그레이드 완료!"

helm-uninstall-all: ## 전체 차트 삭제
	@echo "전체 차트 삭제 중 (ENV=$(ENV), NS=$(K8S_NAMESPACE))..."
	@# 모니터링 먼저 삭제
	@helm uninstall wealist-monitoring -n $(K8S_NAMESPACE) 2>/dev/null || true
	@# 모든 서비스 삭제 (프론트엔드 포함)
	@for service in $(SERVICES); do \
		echo "$$service 삭제 중..."; \
		helm uninstall $$service -n $(K8S_NAMESPACE) 2>/dev/null || true; \
	done
	@helm uninstall wealist-infrastructure -n $(K8S_NAMESPACE) 2>/dev/null || true
	@echo "cert-manager 삭제 필요 여부 확인 중..."
	@if helm list -n cert-manager 2>/dev/null | grep -q cert-manager-config; then \
		echo "cert-manager-config 삭제 중..."; \
		helm uninstall cert-manager-config -n cert-manager 2>/dev/null || true; \
	fi
	@echo "전체 차트 삭제 완료!"

##@ 환경별 빠른 배포

helm-localhost: ## 로컬 Kind 클러스터에 배포
	@$(MAKE) helm-install-all ENV=localhost

# (레거시) helm-local-ubuntu - helm-staging 또는 helm-dev 사용 권장
helm-local-ubuntu:
	@$(MAKE) helm-install-all ENV=local-ubuntu

helm-dev: ## dev 환경에 배포
	@$(MAKE) helm-install-all ENV=dev

helm-staging: ## staging 환경에 배포
	@$(MAKE) helm-install-all ENV=staging

helm-prod: ## production 환경에 배포
	@$(MAKE) helm-install-all ENV=prod

##@ 포트 포워딩 (모니터링)

.PHONY: port-forward-grafana port-forward-prometheus port-forward-loki port-forward-monitoring

port-forward-grafana: ## Grafana 포트 포워딩 (localhost:3001 -> 3000)
	@echo "Grafana 포워딩: http://localhost:3001"
	@echo "중지하려면 Ctrl+C"
	kubectl port-forward svc/grafana -n $(K8S_NAMESPACE) 3001:3000

port-forward-prometheus: ## Prometheus 포트 포워딩 (localhost:9090 -> 9090)
	@echo "Prometheus 포워딩: http://localhost:9090"
	@echo "중지하려면 Ctrl+C"
	kubectl port-forward svc/prometheus -n $(K8S_NAMESPACE) 9090:9090

port-forward-loki: ## Loki 포트 포워딩 (localhost:3100 -> 3100)
	@echo "Loki 포워딩: http://localhost:3100"
	@echo "중지하려면 Ctrl+C"
	kubectl port-forward svc/loki -n $(K8S_NAMESPACE) 3100:3100

port-forward-monitoring: ## 모든 모니터링 서비스 포트 포워딩 (백그라운드)
	@echo "모든 모니터링 서비스 포트 포워딩 시작 중..."
	@echo ""
	@kubectl port-forward svc/grafana -n $(K8S_NAMESPACE) 3001:3000 &
	@kubectl port-forward svc/prometheus -n $(K8S_NAMESPACE) 9090:9090 &
	@kubectl port-forward svc/loki -n $(K8S_NAMESPACE) 3100:3100 &
	@echo ""
	@echo "=============================================="
	@echo "  모니터링 서비스 포트 포워딩 활성화됨"
	@echo "=============================================="
	@echo "  Grafana:    http://localhost:3001"
	@echo "  Prometheus: http://localhost:9090"
	@echo "  Loki:       http://localhost:3100"
	@echo "=============================================="
	@echo ""
	@echo "중지: pkill -f 'kubectl port-forward'"

##@ Istio 서비스 메시

.PHONY: istio-install istio-install-ambient istio-install-gateway istio-install-addons istio-install-config
.PHONY: istio-label-ns istio-label-ns-ambient istio-restart-pods istio-uninstall istio-status

ISTIO_VERSION ?= 1.24.0
GATEWAY_API_VERSION ?= v1.2.0

istio-install-ambient: ## Istio Ambient 모드 설치 (권장)
	@echo "Istio Ambient $(ISTIO_VERSION) 설치 중..."
	@echo ""
	@echo "Ambient 모드 구성요소:"
	@echo "  - ztunnel (DaemonSet): 각 노드에서 L4 mTLS, 기본 인증"
	@echo "  - Waypoint Proxy: 네임스페이스별 L7 기능 (라우팅, 재시도, JWT)"
	@echo "  - 사이드카 주입 불필요"
	@echo ""
	@echo "단계 1: Kubernetes Gateway API CRDs $(GATEWAY_API_VERSION) 설치 중..."
	@kubectl get crd gateways.gateway.networking.k8s.io >/dev/null 2>&1 || \
		kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/$(GATEWAY_API_VERSION)/standard-install.yaml
	@echo ""
	@echo "단계 2: Istio Helm 저장소 추가 중..."
	@helm repo add istio https://istio-release.storage.googleapis.com/charts 2>/dev/null || true
	@helm repo update
	@echo ""
	@echo "단계 3: istio-base (CRDs) 설치 중..."
	@helm upgrade --install istio-base istio/base \
		-n istio-system --create-namespace \
		--version $(ISTIO_VERSION) --wait
	@echo ""
	@echo "단계 4: istio-cni (Ambient 필수) 설치 중..."
	@helm upgrade --install istio-cni istio/cni \
		-n istio-system \
		--version $(ISTIO_VERSION) \
		--set profile=ambient --wait
	@echo ""
	@echo "단계 5: istiod (Ambient 프로필) 설치 중..."
	@helm upgrade --install istiod istio/istiod \
		-n istio-system \
		--version $(ISTIO_VERSION) \
		--set profile=ambient --wait
	@echo ""
	@echo "단계 6: ztunnel (L4 보안 오버레이) 설치 중..."
	@helm upgrade --install ztunnel istio/ztunnel \
		-n istio-system \
		--version $(ISTIO_VERSION) --wait
	@echo ""
	@echo "Istio Ambient 코어 설치 완료!"
	@echo ""
	@echo "다음 단계:"
	@echo "  1. make istio-label-ns-ambient  # 네임스페이스에 Ambient 활성화"
	@echo "  2. make istio-install-config    # Gateway, VirtualService, Waypoint 설치"
	@echo ""
	@echo "참고: Ambient 모드는 레거시 istio-ingressgateway 대신"
	@echo "      Kubernetes Gateway API (Waypoint)를 사용합니다."
	@echo "      레거시 게이트웨이가 필요하면: make istio-install-gateway"

istio-install-gateway: ## Istio Ingress Gateway 설치 (선택사항, 레거시 지원용)
	@echo "Istio Ingress Gateway 설치 중..."
	@echo ""
	@echo "참고: Ambient 모드에서는 Kubernetes Gateway API 사용을 권장합니다."
	@echo "      이것은 레거시 사이드카 모드 또는 하이브리드 설정용입니다."
	@echo ""
	@helm upgrade --install istio-ingressgateway istio/gateway \
		-n istio-system \
		--version $(ISTIO_VERSION) \
		--set service.type=LoadBalancer --wait
	@echo ""
	@echo "Istio Ingress Gateway 설치 완료!"

# (레거시) istio-install - 사이드카 모드, istio-install-ambient 사용 권장
istio-install:
	@echo "Istio $(ISTIO_VERSION) 설치 중..."
	@echo ""
	@echo "단계 1: Istio Helm 저장소 추가 중..."
	@helm repo add istio https://istio-release.storage.googleapis.com/charts 2>/dev/null || true
	@helm repo update
	@echo ""
	@echo "단계 2: istio-base (CRDs) 설치 중..."
	@helm upgrade --install istio-base istio/base \
		-n istio-system --create-namespace \
		--version $(ISTIO_VERSION) --wait
	@echo ""
	@echo "단계 3: istiod (컨트롤 플레인) 설치 중..."
	@helm upgrade --install istiod istio/istiod \
		-n istio-system \
		--version $(ISTIO_VERSION) --wait
	@echo ""
	@echo "단계 4: istio-ingressgateway 설치 중..."
	@helm upgrade --install istio-ingressgateway istio/gateway \
		-n istio-system \
		--version $(ISTIO_VERSION) --wait
	@echo ""
	@echo "Istio 코어 설치 완료!"
	@echo ""
	@echo "다음 단계:"
	@echo "  1. make istio-label-ns       # 네임스페이스에 사이드카 주입 활성화"
	@echo "  2. make istio-install-config # Istio 라우팅 설정 설치"
	@echo "  3. make istio-restart-pods   # 사이드카 주입을 위해 파드 재시작"
	@echo "  4. make istio-install-addons # Kiali, Jaeger 설치 (선택사항)"

# (레거시) istio-label-ns - 사이드카 모드, istio-label-ns-ambient 사용 권장
istio-label-ns:
	@echo "$(K8S_NAMESPACE) 네임스페이스에 Istio 사이드카 주입 레이블 적용 중..."
	@kubectl label namespace $(K8S_NAMESPACE) istio-injection=enabled --overwrite
	@echo ""
	@echo "네임스페이스 레이블 적용됨! 재시작 시 파드에 Istio 사이드카가 주입됩니다."
	@echo "실행: make istio-restart-pods"

istio-label-ns-ambient: ## Istio Ambient 모드용 네임스페이스 레이블 적용
	@echo "$(K8S_NAMESPACE) 네임스페이스에 Istio Ambient 모드 레이블 적용 중..."
	@kubectl label namespace $(K8S_NAMESPACE) istio.io/dataplane-mode=ambient --overwrite
	@kubectl label namespace $(K8S_NAMESPACE) istio-injection- 2>/dev/null || true
	@echo ""
	@echo "Ambient 모드용 네임스페이스 레이블 적용됨!"
	@echo "파드가 자동으로 등록됩니다 - 재시작 불필요."

istio-restart-pods: ## 모든 파드 재시작하여 Istio 사이드카 주입
	@echo "$(K8S_NAMESPACE)의 모든 deployment 재시작하여 사이드카 주입 중..."
	@kubectl rollout restart deployment -n $(K8S_NAMESPACE)
	@echo ""
	@echo "파드 재시작 중. 상태 확인: make status"

istio-install-config: ## Istio 설정 설치 (Gateway, VirtualService 등)
	@echo "Istio 설정 설치 중 (ENV=$(ENV), NS=$(K8S_NAMESPACE))..."
	@helm upgrade --install istio-config ./k8s/helm/charts/istio-config \
		-f $(HELM_BASE_VALUES) \
		-f $(HELM_ENV_VALUES) \
		-n $(K8S_NAMESPACE) --wait
	@echo ""
	@echo "Istio 설정 설치 완료!"
	@echo "Gateway, VirtualService, PeerAuthentication, DestinationRules, AuthorizationPolicy 배포됨."

istio-install-addons: ## Istio 애드온 설치 (Kiali, Jaeger)
	@echo "Istio 관측성 애드온 설치 중..."
	@echo ""
	@echo "Kiali (서비스 그래프) 설치 중..."
	@kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.20/samples/addons/kiali.yaml
	@echo ""
	@echo "Jaeger (분산 추적) 설치 중..."
	@kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.20/samples/addons/jaeger.yaml
	@echo ""
	@echo "Prometheus (없으면) 설치 중..."
	@kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.20/samples/addons/prometheus.yaml 2>/dev/null || true
	@echo ""
	@echo "애드온 설치 완료!"
	@echo ""
	@echo "Kiali 대시보드: kubectl port-forward svc/kiali -n istio-system 20001:20001"
	@echo "Jaeger 대시보드: kubectl port-forward svc/tracing -n istio-system 16686:80"

istio-status: ## Istio 설치 상태 확인
	@echo "=== Istio 시스템 컴포넌트 ==="
	@kubectl get pods -n istio-system
	@echo ""
	@echo "=== Istio 모드 상태 ($(K8S_NAMESPACE)) ==="
	@echo -n "Ambient 모드: "; kubectl get namespace $(K8S_NAMESPACE) -o jsonpath='{.metadata.labels.istio\.io/dataplane-mode}' 2>/dev/null && echo "" || echo "비활성화"
	@echo -n "사이드카 주입: "; kubectl get namespace $(K8S_NAMESPACE) -o jsonpath='{.metadata.labels.istio-injection}' 2>/dev/null && echo "" || echo "비활성화"
	@echo ""
	@echo "=== ztunnel 상태 (Ambient) ==="
	@kubectl get pods -n istio-system -l app=ztunnel 2>/dev/null || echo "ztunnel 미설치"
	@echo ""
	@echo "=== Waypoint Proxy ($(K8S_NAMESPACE)) ==="
	@kubectl get gateway -n $(K8S_NAMESPACE) -l istio.io/waypoint-for 2>/dev/null || echo "Waypoint 프록시 없음"
	@echo ""
	@echo "=== 파드 ($(K8S_NAMESPACE)) ==="
	@kubectl get pods -n $(K8S_NAMESPACE) -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{range .spec.containers[*]}{.name}{" "}{end}{"\n"}{end}' 2>/dev/null | grep -v "^$$" || echo "파드 없음"

istio-uninstall: ## Istio 완전 삭제
	@echo "Istio 삭제 중..."
	@echo ""
	@echo "단계 1: Istio 설정 삭제 중..."
	@helm uninstall istio-config -n $(K8S_NAMESPACE) 2>/dev/null || true
	@echo ""
	@echo "단계 2: 네임스페이스 레이블 삭제 중..."
	@kubectl label namespace $(K8S_NAMESPACE) istio-injection- 2>/dev/null || true
	@kubectl label namespace $(K8S_NAMESPACE) istio.io/dataplane-mode- 2>/dev/null || true
	@echo ""
	@echo "단계 3: Istio 애드온 삭제 중..."
	@kubectl delete -f https://raw.githubusercontent.com/istio/istio/release-$(ISTIO_VERSION)/samples/addons/kiali.yaml 2>/dev/null || true
	@kubectl delete -f https://raw.githubusercontent.com/istio/istio/release-$(ISTIO_VERSION)/samples/addons/jaeger.yaml 2>/dev/null || true
	@echo ""
	@echo "단계 4: Istio 코어 (Ambient 컴포넌트 포함) 삭제 중..."
	@helm uninstall istio-ingressgateway -n istio-system 2>/dev/null || true
	@helm uninstall ztunnel -n istio-system 2>/dev/null || true
	@helm uninstall istiod -n istio-system 2>/dev/null || true
	@helm uninstall istio-cni -n istio-system 2>/dev/null || true
	@helm uninstall istio-base -n istio-system 2>/dev/null || true
	@echo ""
	@echo "Istio 삭제 완료!"
	@echo "참고: 사이드카 모드의 경우, 사이드카 제거를 위해 파드 재시작: make istio-restart-pods"
