# =============================================================================
# Kubernetes (Kind) 명령어
# =============================================================================

##@ Kubernetes (Kind)

.PHONY: kind-setup kind-setup-simple kind-setup-db kind-check-db kind-load-images kind-load-images-mono kind-delete kind-recover
.PHONY: _setup-db-macos _setup-db-debian _check-db-installed

kind-setup: ## 클러스터 생성 + Nginx Ingress (권장)
	@echo "=== Kind 클러스터 + Nginx Ingress 생성 ==="
	@echo ""
	@echo "외부 DB 사용 시 먼저 'make kind-check-db'로 DB 상태를 확인하세요."
	@echo ""
	./k8s/installShell/0.setup-cluster.sh
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

kind-load-images: ## 모든 이미지 빌드/풀 (인프라 + 백엔드 서비스)
	@echo "=== 모든 이미지 로드 ==="
	@echo ""
	@echo "--- 인프라 이미지 로드 중 ---"
	./docker/scripts/dev/1.load_infra_images.sh
	@echo ""
	@echo "--- 백엔드 서비스 이미지 빌드 중 ---"
	SKIP_FRONTEND=true ./docker/scripts/dev/2.build_services_and_load.sh
	@echo ""
	@echo "모든 이미지 로드 완료!"
	@echo ""
	@echo "다음: make helm-install-all ENV=dev"

kind-load-images-mono: ## Go 서비스를 모노레포 패턴으로 빌드 (더 빠른 리빌드)
	@echo "=== 모노레포 빌드로 이미지 로드 (BuildKit 캐시) ==="
	@echo ""
	@echo "--- 인프라 이미지 로드 중 ---"
	./docker/scripts/dev/1.load_infra_images.sh
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
