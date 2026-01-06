# weAlist Configuration Reference

> 이 문서는 Claude 또는 다른 개발자가 프로젝트 설정을 빠르게 파악할 수 있도록 작성되었습니다.
> 새 세션에서 이 파일을 먼저 읽으면 프로젝트 context를 빠르게 파악할 수 있습니다.

## 핵심 참조 파일

| 용도 | 파일 경로 |
|------|-----------|
| Docker-compose 환경변수 | `docker/env/.env.dev.example` |
| Docker-compose 메인 설정 | `docker/compose/docker-compose.yml` |
| Docker-compose DB 초기화 | `docker/init/postgres/init-db.sh` |
| K8s DB 초기화 | `infrastructure/base/postgres/configmap.yaml` |
| K8s 서비스 설정 | `services/{service}/k8s/base/` |
| 서비스별 환경변수 예시 | `services/{service}/.env.example` |

---

## 서비스 포트 (고정)

### 백엔드 서비스

| 서비스 | 포트 | 언어 | 비고 |
|--------|------|------|------|
| auth-service | **8080** | Spring Boot | OAuth2, JWT |
| user-service | **8081** | Go | 사용자 관리 |
| board-service | **8000** | Go | 게시판 |
| chat-service | **8001** | Go | WebSocket 채팅 |
| noti-service | **8002** | Go | SSE 알림 |
| storage-service | **8003** | Go | 파일 저장 (S3/MinIO) |
| frontend | **3000** | React + Vite | 개발: 5173 (내부) |

### 인프라 서비스

| 서비스 | 포트 | 비고 |
|--------|------|------|
| nginx | **80** | API Gateway |
| postgres | **5432** | PostgreSQL 17 |
| redis | **6379** | Redis 7.2 |
| minio | **9000** | S3 호환 API |
| minio-console | **9001** | MinIO 관리 콘솔 |
| livekit | **7880** | HTTP/WebSocket |
| livekit-rtc | **7881** | RTC TCP |
| coturn | **3478** | STUN/TURN (UDP/TCP) |

### 모니터링 서비스

| 서비스 | 포트 | 비고 |
|--------|------|------|
| prometheus | **9090** | 메트릭 수집 |
| loki | **3100** | 로그 수집 |
| grafana | **3001** | 대시보드 |
| redis-exporter | **9121** | Redis 메트릭 |
| postgres-exporter | **9187** | PostgreSQL 메트릭 |

---

## 데이터베이스 설정 (고정)

### 패턴
- **DB 이름**: `wealist_{service}_db` (docker-compose) / `{service}_db` (K8s)
- **유저**: `{service}_service`
- **비밀번호**: `{service}_service_password`

### 서비스별 DB 정보

| 서비스 | DB 이름 (docker-compose) | DB 이름 (K8s) | 유저 | 비밀번호 |
|--------|--------------------------|---------------|------|----------|
| user | wealist_user_db | user_db | user_service | user_service_password |
| board | wealist_board_db | board_db | board_service | board_service_password |
| chat | wealist_chat_db | chat_db | chat_service | chat_service_password |
| noti | wealist_noti_db | noti_db | noti_service | noti_service_password |
| storage | wealist_storage_db | storage_db | storage_service | storage_service_password |
| auth | *(DB 미사용)* | auth_db (미사용) | - | - |

> **Note**: auth-service는 DB를 사용하지 않음 (Redis만 사용)

---

## 서비스 간 통신 URL

### Docker-compose 환경 (컨테이너 네트워크)
```bash
AUTH_SERVICE_URL=http://auth-service:8080
USER_SERVICE_URL=http://user-service:8081
BOARD_SERVICE_URL=http://board-service:8000
CHAT_SERVICE_URL=http://chat-service:8001
NOTI_SERVICE_URL=http://noti-service:8002
STORAGE_SERVICE_URL=http://storage-service:8003
```

### K8s 환경 (ClusterIP)
```bash
AUTH_SERVICE_URL=http://auth-service:8080
USER_SERVICE_URL=http://user-service:8081
BOARD_SERVICE_URL=http://board-service:8000
CHAT_SERVICE_URL=http://chat-service:8001
NOTI_SERVICE_URL=http://noti-service:8002
STORAGE_SERVICE_URL=http://storage-service:8003
```

### 로컬 개발 환경 (localhost)
```bash
AUTH_SERVICE_URL=http://localhost:8080
USER_SERVICE_URL=http://localhost:8081
BOARD_SERVICE_URL=http://localhost:8000
CHAT_SERVICE_URL=http://localhost:8001
NOTI_SERVICE_URL=http://localhost:8002
STORAGE_SERVICE_URL=http://localhost:8003
```

---

## 환경변수 패턴

### 공통 환경변수
```bash
# JWT
JWT_SECRET=your-super-secret-jwt-key-change-this-in-production-min-32-chars
JWT_ACCESS_TOKEN_EXPIRATION_MS=1800000      # 30분
JWT_REFRESH_TOKEN_EXPIRATION_MS=604800000   # 7일

# Redis
REDIS_HOST=redis        # K8s/docker-compose
REDIS_PORT=6379
REDIS_PASSWORD=redis_password

# S3/MinIO
S3_BUCKET=wealist-local-files
S3_REGION=ap-northeast-2
S3_ENDPOINT=http://minio:9000
S3_ACCESS_KEY=minioadmin
S3_SECRET_KEY=minioadmin
```

### Go 서비스 환경변수 패턴
```bash
ENV=dev
PORT={서비스포트}
LOG_LEVEL=debug
DATABASE_URL=postgresql://{service}_service:{service}_service_password@postgres:5432/{db_name}?sslmode=disable
SECRET_KEY={JWT_SECRET}
AUTH_SERVICE_URL=http://auth-service:8080
USER_SERVICE_URL=http://user-service:8081
```

### Spring 서비스 환경변수 패턴
```bash
SPRING_PROFILES_ACTIVE=dev
AUTH_SERVICE_PORT=8080
SPRING_REDIS_HOST=redis
SPRING_REDIS_PORT=6379
SPRING_REDIS_PASSWORD={REDIS_PASSWORD}
JWT_SECRET={JWT_SECRET}
USER_SERVICE_URL=http://user-service:8081
```

---

## K8s 구조

```
infrastructure/
├── base/
│   ├── postgres/          # PostgreSQL StatefulSet, init-db.sh
│   ├── redis/             # Redis Deployment
│   ├── livekit/           # LiveKit 서버
│   ├── coturn/            # TURN 서버
│   └── monitoring/        # Prometheus, Grafana, Loki
└── overlays/
    ├── local/             # Kind 로컬 환경
    └── eks/               # AWS EKS 환경

services/{service}/k8s/
├── base/
│   ├── configmap.yaml     # 환경변수 (민감하지 않은 정보)
│   ├── secret.yaml        # 민감한 정보 (DB 비밀번호, JWT 등)
│   ├── deployment.yaml    # Pod 정의
│   ├── service.yaml       # ClusterIP 서비스
│   └── kustomization.yaml
└── overlays/
    ├── local/             # 로컬 오버레이
    └── eks/               # EKS 오버레이
```

---

## Health Check 엔드포인트

| 서비스 타입 | Liveness | Readiness |
|-------------|----------|-----------|
| Go 서비스 | `/health` | `/ready` |
| Spring 서비스 | `/health` | `/ready` |

> Spring Security에서 `/health`, `/ready`는 `permitAll()` 설정 필요
> 참조: `services/auth-service/src/main/java/.../SecurityConfig.java`

---

## Docker-compose 실행

```bash
# 개발 환경 실행
./docker/scripts/dev.sh up

# 또는 직접 실행
cd docker/compose
cp ../env/.env.dev.example ../env/.env.dev
docker compose up -d

# 로그 확인
docker compose logs -f [service-name]

# 종료
./docker/scripts/dev.sh down
```

---

## Kind (K8s) 실행

```bash
# Kind 클러스터 생성
kind create cluster --config kind-config.yaml

# 이미지 빌드 및 로드
docker build -t user-service:latest ./services/user-service
kind load docker-image user-service:latest

# Kustomize로 배포
kubectl apply -k infrastructure/overlays/local
kubectl apply -k services/user-service/k8s/overlays/local

# 상태 확인
kubectl get pods -n wealist
kubectl logs -f deployment/user-service -n wealist
```

---

## 주의사항

1. **포트 변경 금지**: 위 포트는 모든 환경에서 고정. 변경 시 모든 참조 파일 수정 필요
2. **DB 비밀번호 패턴**: `{service}_service_password` 형식 유지
3. **Secret 파일**: `.gitignore`에 추가됨 (base는 예시용)
4. **auth-service**: DB 미사용, Redis만 사용
5. **K8s vs Docker-compose DB 이름**: K8s는 prefix 없음, docker-compose는 `wealist_` prefix

---

## 빠른 설정 확인 명령어

```bash
# 포트 설정 확인
grep -r "PORT" services/*/k8s/base/configmap.yaml

# DB 설정 확인
grep -r "DATABASE_URL\|DB_NAME" services/*/k8s/base/

# 서비스 URL 확인
grep -r "SERVICE_URL" services/*/k8s/base/configmap.yaml

# docker-compose 서비스 URL 확인
grep -r "SERVICE_URL" docker/compose/docker-compose.yml
```

---

## 변경 이력

| 날짜 | 변경 내용 |
|------|-----------|
| 2025-12-10 | 초기 문서 작성, 포트/DB 설정 통일 완료 |
| 2025-12-10 | docker-compose.yml 포트 수정, 인프라 포트 추가 |
