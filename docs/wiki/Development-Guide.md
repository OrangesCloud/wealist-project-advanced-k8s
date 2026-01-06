# Development Guide

weAlist 개발 환경 설정 가이드입니다.

---

## Project Structure

```
wealist-project-advanced-k8s/
├── services/                  # 애플리케이션 서비스
│   ├── auth-service/          # Spring Boot
│   ├── user-service/          # Go
│   ├── board-service/         # Go
│   ├── chat-service/          # Go
│   ├── noti-service/          # Go
│   ├── storage-service/       # Go
│   └── frontend/              # React
│
├── packages/                  # 공유 패키지
│   └── wealist-advanced-go-pkg/
│
├── helm/                      # Helm 차트
│   ├── charts/
│   └── environments/
│
├── docker/                    # Docker 관련
│   ├── compose/
│   ├── env/
│   └── nginx/
│
└── docs/                      # 문서
```

---

## Go Service Development

### Setup

```bash
# Go workspace 활성화
cd wealist-project-advanced-k8s
go work sync

# 특정 서비스 개발
cd services/board-service
go mod tidy
```

### Service Structure (Clean Architecture)

```
services/board-service/
├── cmd/api/main.go           # Entry point
├── internal/
│   ├── config/               # Configuration
│   ├── domain/               # Entity models
│   ├── dto/                  # Request/Response DTOs
│   ├── handler/              # HTTP handlers (Gin)
│   ├── service/              # Business logic
│   ├── repository/           # Data access (GORM)
│   ├── middleware/           # JWT auth, CORS
│   ├── client/               # HTTP clients to other services
│   └── metrics/              # Prometheus
├── migrations/               # SQL migrations
└── docker/Dockerfile
```

### Run Locally

```bash
# 환경 변수 설정
export DB_HOST=localhost
export DB_PORT=5432
export REDIS_HOST=localhost

# 실행
cd services/board-service
go run cmd/api/main.go
```

---

## Frontend Development

### Setup

```bash
cd services/frontend
npm install
```

### Run Development Server

```bash
npm run dev
# http://localhost:3000
```

### Build

```bash
npm run build
```

---

## Docker Build

### Single Service

```bash
# board-service 빌드
docker build -t localhost:5001/board-service:latest \
  -f services/board-service/docker/Dockerfile .

# 레지스트리 푸시
docker push localhost:5001/board-service:latest
```

### All Services

```bash
make kind-load-images
```

---

## Database

### Local PostgreSQL

```bash
# Docker로 실행
docker run -d \
  --name postgres \
  -e POSTGRES_PASSWORD=postgres \
  -p 5432:5432 \
  postgres:17
```

### Database Naming Convention

| Environment | Database | User |
|-------------|----------|------|
| Docker | wealist_board_db | board_service |
| K8s | board_db | board_service |

---

## Testing

### Go Unit Tests

```bash
cd services/board-service
go test ./...
```

### Integration Tests

```bash
# Docker Compose 환경에서
make dev-up
./docker/scripts/test-health.sh
```

### Helm Validation

```bash
make helm-validate  # 156개 테스트
```

---

## API Documentation

### Swagger 생성

```bash
# 전체 서비스
./docker/scripts/generate-swagger.sh all

# 특정 서비스
cd services/board-service
swag init -g cmd/api/main.go
```

### Swagger URLs

| Service | URL |
|---------|-----|
| board-service | http://localhost:8000/swagger/index.html |
| user-service | http://localhost:8081/swagger/index.html |
| auth-service | http://localhost:8080/swagger-ui/index.html |

---

## Debugging

### Pod Shell Access

```bash
kubectl exec -it deploy/board-service -n wealist-kind-local -- sh
```

### Port Forward

```bash
# 특정 서비스 직접 접근
kubectl port-forward deploy/board-service 8000:8000 -n wealist-kind-local
```

### Logs

```bash
# 실시간 로그
kubectl logs -f deploy/board-service -n wealist-kind-local

# 이전 컨테이너 로그
kubectl logs deploy/board-service -n wealist-kind-local --previous
```

---

## Related Pages

- [Getting Started](Getting-Started.md)
- [Architecture Overview](Architecture.md)
