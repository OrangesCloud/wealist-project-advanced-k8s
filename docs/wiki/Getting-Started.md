# Getting Started

weAlist 프로젝트 시작 가이드입니다.

---

## Prerequisites

```bash
# Required
docker --version       # Docker 20.10+
docker-compose --version  # Docker Compose 2.0+
kubectl version        # kubectl 1.28+
helm version           # Helm 3.12+
kind version           # Kind 0.20+

# Optional
go version             # Go 1.24+ (개발 시)
node --version         # Node 18+ (프론트엔드 개발 시)
```

---

## Quick Start

### Option 1: Kind + Helm (권장)

```bash
# 1. 저장소 클론
git clone https://github.com/OrangesCloud/wealist-project-advanced-k8s.git
cd wealist-project-advanced-k8s

# 2. Kind 클러스터 생성
make kind-setup

# 3. 이미지 빌드 및 로드
make kind-load-images

# 4. Helm으로 전체 배포
make helm-install-all ENV=local-kind

# 5. 상태 확인
make status

# 6. 접속
# http://localhost
```

### Option 2: Docker Compose (간단 테스트)

```bash
# 1. 환경 변수 설정
cp docker/env/.env.dev.example docker/env/.env.dev

# 2. 전체 서비스 시작
make dev-up

# 3. 접속
# Frontend: http://localhost:3000
# API Gateway: http://localhost:80
```

---

## Access Points

| Service | URL |
|---------|-----|
| Frontend | http://localhost:3000 (Compose) / http://localhost (K8s) |
| API Gateway | http://localhost |
| Grafana | http://localhost:3001 (admin/admin) |
| Prometheus | http://localhost:9090 |
| MinIO Console | http://localhost:9001 |

---

## Common Commands

```bash
# 상태 확인
make status

# 로그 확인
make dev-logs                    # Docker Compose
kubectl logs -f deploy/board-service -n wealist-kind-local  # K8s

# 서비스 재배포
make board-service-all           # 빌드 + 로드 + 재배포

# 전체 재시작
make redeploy-all ENV=local-kind
```

---

## Troubleshooting

### Pod가 시작되지 않음
```bash
# Pod 상태 확인
kubectl describe pod -l app=board-service -n wealist-kind-local

# 로그 확인
kubectl logs -f deploy/board-service -n wealist-kind-local
```

### 이미지 빌드 실패
```bash
# 캐시 없이 재빌드
docker build --no-cache -t localhost:5001/board-service:latest \
  -f services/board-service/docker/Dockerfile .
```

### DB 연결 실패
```bash
# Pod에서 DB 연결 테스트
kubectl exec -it deploy/board-service -n wealist-kind-local -- \
  env | grep DB
```

---

## Next Steps

1. [Development Guide](Development-Guide.md) - 개발 환경 설정
2. [Architecture Overview](Architecture.md) - 시스템 이해
3. [ADR](ADR.md) - 아키텍처 결정 배경
