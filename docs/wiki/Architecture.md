# Architecture Overview

weAlist의 전체 시스템 아키텍처입니다.

---

## AWS Architecture (대표)

![AWS Architecture](https://raw.githubusercontent.com/OrangesCloud/wealist-project-advanced-k8s/main/docs/images/wealist_aws_arch_v2.png)

> AWS 인프라 상세: [Architecture-AWS](Architecture-AWS)

---

## Microservices Architecture

![Microservices](https://raw.githubusercontent.com/OrangesCloud/wealist-project-advanced-k8s/main/docs/images/wealist_microservices.png)

### 서비스 구성

| Service | Tech | Port | Description |
|---------|------|------|-------------|
| **auth-service** | Spring Boot 3 | 8080 | JWT 토큰 관리, OAuth2 인증 (Redis only) |
| **user-service** | Go + Gin | 8081 | 사용자, 워크스페이스 관리 |
| **board-service** | Go + Gin | 8000 | 프로젝트, 보드, 댓글 관리 |
| **chat-service** | Go + Gin | 8001 | 실시간 메시징 (WebSocket) |
| **noti-service** | Go + Gin | 8002 | 푸시 알림 (SSE) |
| **storage-service** | Go + Gin | 8003 | 파일 스토리지 (S3/MinIO) |
| **ops-service** | Go + Gin | 8004 | 운영 대시보드 (메트릭, 로그 조회) |

---

## Kubernetes Workloads

![K8s Workloads](https://raw.githubusercontent.com/OrangesCloud/wealist-project-advanced-k8s/main/docs/images/wealist_k8s_workloads.png)

> K8s 플랫폼 상세: [Architecture-K8s](Architecture-K8s)

---

## Infrastructure

| Component | Technology | Description |
|-----------|------------|-------------|
| **Database** | PostgreSQL 17 | 6개 DB (서비스별 분리) |
| **Cache** | Redis 7.2 | 캐시, 토큰 저장소 |
| **Object Storage** | S3 / MinIO | 파일 스토리지 |
| **API Gateway** | Istio Gateway API | HTTPRoute 기반 라우팅 |
| **Service Mesh** | Istio 1.24.0 | mTLS, AuthorizationPolicy |
| **Monitoring** | Prometheus + Grafana + Loki + OTEL + Tempo | 메트릭/로그/트레이싱 |

---

## Service Communication

![Service Communication](https://raw.githubusercontent.com/OrangesCloud/wealist-project-advanced-k8s/main/docs/images/wealist_service_communication.png)

| Path | Service | Port |
|------|---------|------|
| `/svc/auth/*` | auth-service | 8080 |
| `/svc/user/*` | user-service | 8081 |
| `/svc/board/*` | board-service | 8000 |
| `/svc/chat/*` | chat-service | 8001 |
| `/svc/noti/*` | noti-service | 8002 |
| `/svc/storage/*` | storage-service | 8003 |
| `/svc/ops/*` | ops-service | 8004 |
| `/*` | frontend (CloudFront) | - |

### Internal Communication

- **External**: JWT Bearer token in `Authorization` header (Istio RequestAuthentication 검증)
- **Internal**: mTLS로 암호화, AuthorizationPolicy로 접근 제어
- **Service Discovery**: Kubernetes DNS (`{service}.{namespace}.svc.cluster.local`)

---

## Related Pages

- [AWS Architecture](Architecture-AWS)
- [Kubernetes Architecture](Architecture-K8s)
- [CI/CD Pipeline](Architecture-CICD)
- [Security (VPC)](Architecture-VPC)
- [Monitoring Stack](Architecture-Monitoring)
