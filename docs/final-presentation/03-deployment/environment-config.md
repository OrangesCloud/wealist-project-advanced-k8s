# 환경별 설정 가이드

> **환경**: localhost (Kind) / dev / prod

---

## 환경 개요

| 환경 | 네임스페이스 | 도메인 | 인프라 |
|------|-------------|--------|--------|
| localhost | wealist-kind-local | localhost:8080 | Kind 내부 DB/Redis |
| dev | wealist-dev | dev.wealist.co.kr | AWS RDS/ElastiCache |
| prod | wealist-prod | wealist.co.kr | AWS RDS/ElastiCache |

---

## 환경 아키텍처

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          ENVIRONMENT COMPARISON                          │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌─────────────────────┐ ┌─────────────────────┐ ┌────────────────────┐│
│  │     LOCALHOST       │ │        DEV          │ │       PROD         ││
│  │    (Kind Cluster)   │ │   (EKS + AWS)       │ │   (EKS + AWS)      ││
│  ├─────────────────────┤ ├─────────────────────┤ ├────────────────────┤│
│  │                     │ │                     │ │                    ││
│  │  ┌───────────────┐  │ │  ┌───────────────┐  │ │  ┌──────────────┐ ││
│  │  │  PostgreSQL   │  │ │  │   AWS RDS     │  │ │  │   AWS RDS    │ ││
│  │  │  (In-cluster) │  │ │  │ (PostgreSQL)  │  │ │  │(PostgreSQL)  │ ││
│  │  └───────────────┘  │ │  └───────────────┘  │ │  └──────────────┘ ││
│  │                     │ │                     │ │                    ││
│  │  ┌───────────────┐  │ │  ┌───────────────┐  │ │  ┌──────────────┐ ││
│  │  │    Redis      │  │ │  │ ElastiCache   │  │ │  │ ElastiCache  │ ││
│  │  │  (In-cluster) │  │ │  │   (Redis)     │  │ │  │   (Redis)    │ ││
│  │  └───────────────┘  │ │  └───────────────┘  │ │  └──────────────┘ ││
│  │                     │ │                     │ │                    ││
│  │  ┌───────────────┐  │ │  ┌───────────────┐  │ │  ┌──────────────┐ ││
│  │  │    MinIO      │  │ │  │    AWS S3     │  │ │  │   AWS S3     │ ││
│  │  │  (S3 호환)    │  │ │  │               │  │ │  │ + CloudFront │ ││
│  │  └───────────────┘  │ │  └───────────────┘  │ │  └──────────────┘ ││
│  │                     │ │                     │ │                    ││
│  │  NodePort: 30080    │ │  ALB Ingress       │ │  NLB + Istio GW   ││
│  │                     │ │                     │ │                    ││
│  └─────────────────────┘ └─────────────────────┘ └────────────────────┘│
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Helm Values 파일 구조

```
k8s/helm/environments/
├── base.yaml              # 공통 설정 (모든 환경)
├── local-kind.yaml        # Kind 클러스터
├── local-ubuntu.yaml      # Ubuntu 개발 서버
├── dev.yaml               # AWS 개발 환경
├── prod.yaml              # AWS 프로덕션
├── prod-registry.yaml     # ECR URL (gitignored)
└── prod-secrets.yaml      # DB/Redis 호스트 (gitignored)
```

---

## 환경별 주요 설정

### Localhost (Kind)

```yaml
# local-kind.yaml
global:
  namespace: wealist-kind-local
  environment: localhost
  domain: localhost

# 내부 인프라 사용
postgres:
  enabled: true

redis:
  enabled: true

minio:
  enabled: true

# 프론트엔드 Pod 배포
frontend:
  enabled: true

# Istio 설정
istio:
  mtls:
    mode: PERMISSIVE  # mTLS 선택적
  authorizationPolicy:
    denyAll: false    # 모든 트래픽 허용
```

### Production (AWS)

```yaml
# prod.yaml
global:
  namespace: wealist-prod
  environment: production
  domain: wealist.co.kr
  imageRegistry: "{account}.dkr.ecr.ap-northeast-2.amazonaws.com"

# 외부 인프라 사용 (AWS)
postgres:
  enabled: false  # AWS RDS 사용

redis:
  enabled: false  # AWS ElastiCache 사용

minio:
  enabled: false  # AWS S3 사용

# 프론트엔드는 CloudFront 사용
frontend:
  enabled: false

# External Secrets (AWS Secrets Manager)
externalSecrets:
  enabled: true

# Istio 설정
istio:
  mtls:
    mode: STRICT    # mTLS 강제
  authorizationPolicy:
    denyAll: true   # Zero Trust

# HPA 활성화
autoscaling:
  enabled: true
  minReplicas: 1
  maxReplicas: 10
```

---

## 공통 설정 (base.yaml)

### 서비스 간 통신 URL

```yaml
shared:
  config:
    AUTH_SERVICE_URL: "http://auth-service:8080"
    USER_SERVICE_URL: "http://user-service:8081"
    BOARD_SERVICE_URL: "http://board-service:8000"
    CHAT_SERVICE_URL: "http://chat-service:8001"
    NOTI_SERVICE_URL: "http://noti-service:8002"
    STORAGE_SERVICE_URL: "http://storage-service:8003"
```

### OpenTelemetry 설정

```yaml
shared:
  config:
    OTEL_EXPORTER_OTLP_ENDPOINT: "http://otel-collector:4318"
    OTEL_EXPORTER_OTLP_PROTOCOL: "http/protobuf"
    OTEL_TRACES_SAMPLER: "parentbased_traceidratio"
    OTEL_PROPAGATORS: "tracecontext,baggage"
```

---

## 환경별 차이점

### 데이터베이스

| 항목 | localhost | prod |
|------|-----------|------|
| 호스트 | `postgres:5432` | RDS 엔드포인트 |
| SSL | 불필요 | `sslmode=require` |
| 마이그레이션 | `DB_AUTO_MIGRATE=true` | `DB_AUTO_MIGRATE=false` |

### JWT 인증

| 항목 | localhost | prod |
|------|-----------|------|
| `ISTIO_JWT_MODE` | `true` | `true` |
| JWT 검증 | Istio + Go 파싱 | Istio + Go 파싱 |

### Rate Limiting

| 항목 | localhost | prod |
|------|-----------|------|
| 활성화 | `true` | `true` |
| 분당 요청 | 1000 | 60 |
| 버스트 | 100 | 10 |

### Istio mTLS

| 항목 | localhost | prod |
|------|-----------|------|
| 모드 | PERMISSIVE | STRICT |
| AuthorizationPolicy | 비활성화 | Zero Trust |

---

## 리소스 설정

### Go 서비스 (경량)

```yaml
resources:
  requests:
    memory: "64Mi"
    cpu: "25m"
  limits:
    memory: "256Mi"
    cpu: "500m"
```

### Spring Boot 서비스 (auth-service)

```yaml
javaResources:
  requests:
    memory: "384Mi"
    cpu: "100m"
  limits:
    memory: "768Mi"
    cpu: "500m"
```

---

## Secrets 관리

### Localhost (직접 설정)

```yaml
# local-kind-secrets.yaml
shared:
  secrets:
    DB_PASSWORD: "localpassword"
    REDIS_PASSWORD: ""
    JWT_SECRET_KEY: "local-jwt-secret"
```

### Production (AWS Secrets Manager)

```yaml
# ExternalSecret이 AWS Secrets Manager에서 자동 동기화
externalSecrets:
  enabled: true

# Secrets는 wealist-shared-secret ConfigMap으로 주입됨
```

---

## 배포 명령어

### Localhost

```bash
# Kind 클러스터 설정 + 전체 배포
./k8s/helm/scripts/localhost/0.setup-cluster.sh

# 개별 서비스 배포
make helm-upgrade-all ENV=local-kind
```

### Production

```bash
# ArgoCD가 k8s-deploy-prod 브랜치 감시
# Git push → ArgoCD 자동 sync

# 수동 동기화
argocd app sync {app-name}
```

---

## 접속 정보

| 환경 | API Gateway | 프론트엔드 |
|------|-------------|-----------|
| localhost | `http://localhost:8080/svc/*` | `http://localhost:8080` |
| prod | `https://api.wealist.co.kr/svc/*` | `https://wealist.co.kr` |

### API 라우팅 예시

```
/svc/auth/*     → auth-service:8080
/svc/user/*     → user-service:8081
/svc/board/*    → board-service:8000
/svc/chat/*     → chat-service:8001
/svc/noti/*     → noti-service:8002
/svc/storage/*  → storage-service:8003
```

---

## 관련 문서

- [CI/CD 파이프라인](./ci-cd-pipeline.md)
- [ArgoCD 설정](./argocd-setup.md)

