# 기술 스택

## Backend

### Go 서비스 (5개)

| 기술 | 버전 | 용도 | 선정 이유 |
|------|------|------|----------|
| **Go** | 1.24 | 프로그래밍 언어 | 고성능, 간결한 문법, 컴파일 타입 안전성 |
| **Gin** | 1.10 | HTTP 프레임워크 | 빠른 라우팅, 미들웨어 지원 |
| **GORM** | 1.25 | ORM | 자동 마이그레이션, 쿼리 빌더 |
| **Zap** | 1.27 | 로깅 | 구조화된 로깅, 고성능 |
| **golang-jwt** | 5.2 | JWT 처리 | 표준 준수, 검증된 라이브러리 |

### Spring Boot 서비스 (2개: auth, ops)

| 기술 | 버전 | 용도 | 선정 이유 |
|------|------|------|----------|
| **Spring Boot** | 3.4 | 프레임워크 | OAuth2 표준 구현, 검증된 보안 |
| **Spring Security** | 6.4 | 보안 | OAuth2/JWT 표준 지원 |
| **Spring Data JPA** | 3.4 | ORM | 타입 안전한 쿼리 |
| **Gradle** | 8.x | 빌드 도구 | 빠른 빌드, 캐싱 |

### 공통 패키지

```
packages/wealist-advanced-go-pkg/
├── auth/       # JWT 파서, 검증기
├── errors/     # 타입화된 에러 처리
├── health/     # Health check handlers
├── logger/     # Zap 기반 로거
├── metrics/    # Prometheus 메트릭
├── middleware/ # CORS, logging, recovery
├── ratelimit/  # Redis 기반 Rate Limiting
└── response/   # HTTP 응답 헬퍼
```

---

## Frontend

### 메인 애플리케이션

| 기술 | 버전 | 용도 | 선정 이유 |
|------|------|------|----------|
| **React** | 18.3 | UI 프레임워크 | 컴포넌트 기반, 대규모 생태계 |
| **TypeScript** | 5.6 | 타입 시스템 | 타입 안전성, IDE 지원 |
| **Vite** | 6.0 | 빌드 도구 | 빠른 HMR, 최적화된 번들링 |
| **TailwindCSS** | 3.4 | 스타일링 | 유틸리티 클래스, 빠른 개발 |
| **React Query** | 5.x | 서버 상태 관리 | 캐싱, 자동 재검증 |
| **Zustand** | 4.x | 클라이언트 상태 | 간단한 API, 작은 번들 |

### 주요 라이브러리

| 라이브러리 | 용도 |
|-----------|------|
| react-router-dom | 라우팅 |
| axios | HTTP 클라이언트 |
| react-beautiful-dnd | 드래그 앤 드롭 |
| date-fns | 날짜 처리 |
| lucide-react | 아이콘 |

---

## Database

### PostgreSQL

| 항목 | 값 | 설명 |
|------|---|------|
| **버전** | 17 | 최신 안정 버전 |
| **Production** | AWS RDS | 관리형, Multi-AZ |
| **Development** | Docker/Kind | 로컬 개발용 |
| **확장** | uuid-ossp | UUID 생성 |

### Redis

| 항목 | 값 | 설명 |
|------|---|------|
| **버전** | 7.2 | 최신 안정 버전 |
| **Production** | AWS ElastiCache | 관리형, 클러스터 |
| **용도** | 캐시, 세션, Rate Limiting | |

---

## Infrastructure

### Kubernetes

| 컴포넌트 | 버전 | 설명 |
|---------|------|------|
| **EKS** | 1.34 | AWS 관리형 쿠버네티스 |
| **Node Group** | t3.medium | 2-4 노드 (Karpenter) |
| **Namespace** | wealist-prod | 서비스 격리 |

### Istio Service Mesh

| 컴포넌트 | 버전 | 용도 |
|---------|------|------|
| **Istio** | 1.24 | 서비스 메시 |
| **Envoy Sidecar** | - | L4/L7 프록시 |
| **istiod** | - | 컨트롤 플레인 |
| **Ingress Gateway** | - | 외부 트래픽 진입점 |

#### Istio 기능 활용

| 기능 | 설정 | 용도 |
|------|------|------|
| **mTLS** | STRICT | 서비스 간 암호화 |
| **Circuit Breaker** | outlierDetection | 장애 격리 |
| **Traffic Management** | VirtualService | 라우팅, 카나리 |
| **Telemetry** | 10% sampling | 분산 트레이싱 |

### Argo Rollouts

| 기능 | 설정 | 설명 |
|------|------|------|
| **Canary** | 10% → 30% → 50% → 100% | 점진적 배포 |
| **Analysis** | Prometheus metrics | 자동 롤백 조건 |

---

## CI/CD

### GitHub Actions

| 워크플로우 | 트리거 | 작업 |
|-----------|--------|------|
| **Build & Push** | push to main | Docker build, ECR push |
| **Terraform Plan** | PR | 인프라 변경 검토 |
| **Helm Validate** | PR | 차트 검증 |

### ArgoCD

| 설정 | 값 | 설명 |
|------|---|------|
| **Sync Policy** | Automated | 자동 배포 |
| **Prune** | true | 삭제된 리소스 정리 |
| **Self Heal** | true | 드리프트 자동 수정 |

### Terraform

| 레이어 | 리소스 | 상태 저장 |
|--------|--------|----------|
| **foundation** | VPC, ECR, S3 | S3 Backend |
| **compute** | EKS, RDS, ElastiCache | S3 Backend |
| **argocd-apps** | ArgoCD Applications | S3 Backend |

---

## Observability

### Metrics (Prometheus)

| 설정 | 값 | 설명 |
|------|---|------|
| **버전** | 2.55 | 최신 안정 버전 |
| **Scrape Interval** | 15s | 메트릭 수집 주기 |
| **Remote Write** | enabled | Tempo 연동 |
| **Exemplars** | enabled | 트레이스 연결 |

### Logs (Loki)

| 설정 | 값 | 설명 |
|------|---|------|
| **버전** | 3.6 | 최신 안정 버전 |
| **Backend** | S3 | 로그 영속 저장 |
| **Retention** | 30 days | 보관 기간 |

### Traces (Tempo)

| 설정 | 값 | 설명 |
|------|---|------|
| **버전** | 2.9 | 최신 안정 버전 |
| **Backend** | S3 | 트레이스 저장 |
| **Retention** | 7 days | 보관 기간 |
| **Metrics Generator** | enabled | RED 메트릭 생성 |

### OTEL Collector

| 설정 | 값 | 설명 |
|------|---|------|
| **버전** | 0.114 | 최신 안정 버전 |
| **Receivers** | OTLP (gRPC/HTTP) | 트레이스/메트릭 수신 |
| **Processors** | batch, memory_limiter | 성능 최적화 |
| **Exporters** | Tempo, Prometheus | 백엔드 전송 |
| **Connectors** | spanmetrics, servicegraph | 트레이스 → 메트릭 |

### Grafana

| 설정 | 값 | 설명 |
|------|---|------|
| **버전** | 10.4 | 최신 안정 버전 |
| **Authentication** | Google OAuth | SSO |
| **Provisioning** | ConfigMap | 자동 설정 |

---

## Security

### 인증/인가

| 기술 | 용도 |
|------|------|
| **OAuth2** | Google 소셜 로그인 |
| **JWT (RS256)** | 액세스 토큰 |
| **JWKS** | 키 배포/갱신 |

### 네트워크 보안

| 기술 | 용도 |
|------|------|
| **mTLS** | 서비스 간 암호화 |
| **NetworkPolicy** | Pod 간 통신 제어 |
| **Authorization Policy** | L7 접근 제어 |

### Secrets 관리

| 기술 | 용도 |
|------|------|
| **AWS Secrets Manager** | 민감 정보 저장 |
| **External Secrets Operator** | K8s Secret 동기화 |
| **Pod Identity (IRSA)** | AWS 권한 부여 |

### Rate Limiting

| 설정 | 값 | 설명 |
|------|---|------|
| **알고리즘** | Sliding Window | Redis 기반 |
| **Production** | 60 req/min | API 제한 |
| **Development** | 1000 req/min | 개발 편의 |

---

## 버전 호환성 매트릭스

| Component | Minimum | Recommended | Notes |
|-----------|---------|-------------|-------|
| Kubernetes | 1.30 | 1.34 | EKS 버전 |
| Istio | 1.22 | 1.24 | Sidecar 모드 |
| Go | 1.22 | 1.24 | 서비스 빌드 |
| PostgreSQL | 15 | 17 | RDS 버전 |
| Redis | 7.0 | 7.2 | ElastiCache |

---

## 관련 문서

- [아키텍처 개요](./overview.md)
- [배포 가이드](../03-deployment/ci-cd-pipeline.md)
- [모니터링 가이드](../04-monitoring/observability-stack.md)
