# Kubernetes Architecture

weAlist의 Kubernetes 플랫폼 아키텍처입니다.

---

## Cluster Overview

![EKS Cluster Architecture](https://raw.githubusercontent.com/OrangesCloud/wealist-project-advanced-k8s/main/docs/images/wealist_k8s_cluster.png)

### EKS Configuration

| 항목 | 값 | 비고 |
|------|------|------|
| **Kubernetes Version** | 1.34 | 최신 안정 버전 |
| **Region** | ap-northeast-2 | Seoul |
| **VPC** | Dedicated | 3 AZs (a, b, c) |

### Node Configuration

| Node Group | Instance | Strategy | 용도 |
|------------|----------|----------|------|
| **wealist-spot** | m5.large | 100% Spot | 애플리케이션 워크로드 |

> **비용 최적화**: 100% Spot Instance로 월 ~$50 수준 유지. On-demand 대비 70-90% 절감.

### EKS Add-ons

| Add-on | 버전 | 용도 |
|--------|------|------|
| **vpc-cni** | Latest | Pod 네트워킹 |
| **coredns** | Latest | 클러스터 DNS |
| **kube-proxy** | Latest | 서비스 프록시 |
| **aws-ebs-csi-driver** | Latest | EBS 볼륨 |
| **eks-pod-identity-agent** | Latest | Pod Identity (IRSA 대체) |

### Security Groups

| Port Range | 용도 |
|------------|------|
| 15001-15021 | Istio Envoy Proxy |
| 53 (TCP/UDP) | CoreDNS |

---

## Scheduled Scaling (비용 최적화)

업무 시간 외에는 노드를 0으로 스케일링하여 비용을 절감합니다.

### 스케줄

| 요일 | 활성 시간 (KST) | 비활성 시간 (KST) |
|------|----------------|------------------|
| **Weekday** | 08:00 - 01:00 (+1) | 01:00 - 08:00 |
| **Weekend** | 09:00 - 03:00 (+1) | 03:00 - 09:00 |

### 구현

```hcl
# terraform/prod/compute/scheduled-scaling.tf
resource "aws_autoscaling_schedule" "weekday_scale_up" {
  scheduled_action_name  = "weekday-scale-up"
  recurrence             = "0 23 * * 0-4"  # UTC (= KST 08:00)
  desired_capacity       = 1
  min_size              = 1
}

resource "aws_autoscaling_schedule" "weekday_scale_down" {
  scheduled_action_name  = "weekday-scale-down"
  recurrence             = "0 16 * * 1-5"  # UTC (= KST 01:00)
  desired_capacity       = 0
  min_size              = 0
}
```

---

## Namespace Strategy

```
┌──────────────────────────────────────────────────────────────┐
│                       EKS Cluster                              │
├──────────────────────────────────────────────────────────────┤
│  ┌──────────────────┐            ┌──────────────────┐        │
│  │   wealist-dev    │            │   wealist-prod   │        │
│  │                  │            │                  │        │
│  │ - 9 Services     │            │ - 9 Services     │        │
│  │ - In-cluster DB  │            │ - AWS RDS/Redis  │        │
│  │ - ConfigMaps     │            │ - ExternalSecret │        │
│  │ - PERMISSIVE mTLS│            │ - STRICT mTLS    │        │
│  └──────────────────┘            └──────────────────┘        │
│                                                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐   │
│  │    argocd    │  │  monitoring  │  │   istio-system   │   │
│  │              │  │              │  │                  │   │
│  │ - Argo CD    │  │ - Prometheus │  │ - istiod 1.28.2  │   │
│  │ - App of Apps│  │ - Grafana    │  │ - Ingress Gateway│   │
│  │              │  │ - Loki/Tempo │  │ - Kiali/Jaeger   │   │
│  └──────────────┘  └──────────────┘  └──────────────────┘   │
│                                                               │
│  ┌──────────────────┐  ┌──────────────────┐                  │
│  │ external-secrets │  │ argo-rollouts    │                  │
│  │                  │  │                  │                  │
│  │ - ESO Controller │  │ - Rollouts Ctrl  │                  │
│  │ - ClusterStore   │  │ - AnalysisRuns   │                  │
│  └──────────────────┘  └──────────────────┘                  │
└──────────────────────────────────────────────────────────────┘
```

| Namespace | 용도 | 비고 |
|-----------|------|------|
| `wealist-dev` | 개발/테스트 | In-cluster DB, PERMISSIVE mTLS |
| `wealist-prod` | 운영 | AWS Managed Services, STRICT mTLS |
| `argocd` | GitOps CD | App of Apps 패턴 |
| `monitoring` | Observability | Prometheus, Grafana, Loki, Tempo |
| `istio-system` | Service Mesh | Istio 1.28.2 Sidecar |
| `external-secrets` | Secrets Management | AWS Secrets Manager 연동 |
| `argo-rollouts` | Progressive Delivery | Canary 배포 |

> **Note**: `wealist-staging` 네임스페이스는 사용하지 않습니다. Dev와 Prod만 운영합니다.

---

## Workload Architecture

![Kubernetes Workloads](https://raw.githubusercontent.com/OrangesCloud/wealist-project-advanced-k8s/main/docs/images/wealist_k8s_workloads.png)

### Service List (9 Services)

| Service | Tech | Port | 용도 | 비고 |
|---------|------|------|------|------|
| **auth-service** | Spring Boot 3 | 8080 | JWT 인증, OAuth2 | Redis only |
| **user-service** | Go + Gin | 8081 | 사용자, 워크스페이스 | PostgreSQL |
| **board-service** | Go + Gin | 8000 | 프로젝트, 보드, 댓글 | PostgreSQL |
| **chat-service** | Go + Gin | 8001 | 실시간 메시징 | WebSocket |
| **noti-service** | Go + Gin | 8002 | 푸시 알림 | SSE |
| **storage-service** | Go + Gin | 8003 | 파일 저장소 | S3/MinIO |
| **ops-service** | Go + Gin | 8005 | 운영 API | Admin 전용 |
| **ops-portal** | React | 80 | 운영 대시보드 | Admin UI |
| **frontend** | React + Vite | 80 | 웹 UI | SPA |

> **Production Frontend**: CloudFront + S3로 배포되며 EKS에는 배포하지 않습니다.

---

## Service Mesh (Istio 1.28.2 Sidecar)

![Istio Service Mesh](https://raw.githubusercontent.com/OrangesCloud/wealist-project-advanced-k8s/main/docs/images/wealist_k8s_istio.png)

### 컴포넌트

| Component | 역할 | 위치 |
|-----------|------|------|
| **istiod** | Control Plane (Pilot, Citadel, Galley) | istio-system |
| **Envoy Sidecar** | L4/L7 프록시, mTLS, Telemetry | 각 Pod 내부 |
| **Ingress Gateway** | 외부 트래픽 진입점 | istio-system |
| **Kiali** | 서비스 메시 시각화 | istio-system |
| **Jaeger/Tempo** | 분산 추적 | istio-system |

### Sidecar 리소스 설정

```yaml
# 환경 파일 (prod.yaml)
istio:
  sidecar:
    resources:
      requests:
        cpu: 100m
        memory: 128Mi
      limits:
        cpu: 1000m
        memory: 512Mi
```

### Zero-Trust Security Model

Production 환경에서는 **denyAll baseline + allowList** 정책을 적용합니다.

```yaml
# 1. Deny All (baseline)
apiVersion: security.istio.io/v1
kind: AuthorizationPolicy
metadata:
  name: deny-all
  namespace: wealist-prod
spec: {}  # 모든 트래픽 거부

# 2. Allow Specific (per-service)
apiVersion: security.istio.io/v1
kind: AuthorizationPolicy
metadata:
  name: board-service-policy
  namespace: wealist-prod
spec:
  selector:
    matchLabels:
      app: board-service
  action: ALLOW
  rules:
  - from:
    - source:
        principals:
        - cluster.local/ns/wealist-prod/sa/frontend
        - cluster.local/ns/wealist-prod/sa/user-service
        - cluster.local/ns/istio-system/sa/istio-ingressgateway
```

### PeerAuthentication (mTLS)

```yaml
apiVersion: security.istio.io/v1
kind: PeerAuthentication
metadata:
  name: default
  namespace: wealist-prod
spec:
  mtls:
    mode: STRICT  # Production: 필수 mTLS
```

### RequestAuthentication (JWT)

```yaml
apiVersion: security.istio.io/v1
kind: RequestAuthentication
metadata:
  name: jwt-auth
  namespace: wealist-prod
spec:
  selector:
    matchLabels:
      app: board-service
  jwtRules:
  - issuer: "https://wealist.co.kr"
    jwksUri: "http://auth-service.wealist-prod.svc.cluster.local:8080/.well-known/jwks.json"
    forwardOriginalToken: true
```

### 환경별 정책 비교

| 설정 | Dev | Prod |
|------|-----|------|
| **mTLS Mode** | PERMISSIVE | STRICT |
| **AuthorizationPolicy** | 없음 (allow all) | denyAll + allowList |
| **RequestAuthentication** | 없음 | JWT 검증 |
| **Telemetry Sampling** | 100% | 10% |

---

## Traffic Management (VirtualService)

![Traffic Flow](https://raw.githubusercontent.com/OrangesCloud/wealist-project-advanced-k8s/main/docs/images/wealist_k8s_traffic.png)

### VirtualService 라우팅

```yaml
apiVersion: networking.istio.io/v1
kind: VirtualService
metadata:
  name: wealist-routes
  namespace: wealist-prod
spec:
  hosts:
  - "wealist.co.kr"
  - "api.wealist.co.kr"
  gateways:
  - istio-system/istio-ingressgateway
  http:
  # API Service Routes (prefix 제거)
  - name: auth-service-primary
    match:
    - uri:
        prefix: /api/svc/auth
    rewrite:
      uriRegexRewrite:
        match: "^/api/svc/auth(.*)$"
        rewrite: '\1'
    route:
    - destination:
        host: auth-service
        port:
          number: 8080
    timeout: 30s
    retries:
      attempts: 3
      perTryTimeout: 10s
      retryOn: "5xx,reset,connect-failure"

  # Frontend (catch-all)
  - match:
    - uri:
        prefix: /
    route:
    - destination:
        host: frontend
        port:
          number: 80
```

### 라우팅 테이블

| Path | Target | Port | 특수 설정 |
|------|--------|------|-----------|
| `/oauth2/*` | auth-service | 8080 | - |
| `/api/svc/auth/*` | auth-service | 8080 | prefix 제거 |
| `/api/svc/user/*` | user-service | 8081 | prefix 제거 |
| `/api/svc/board/*` | board-service | 8000 | WebSocket, no timeout |
| `/api/svc/chat/*` | chat-service | 8001 | WebSocket, no timeout |
| `/api/svc/noti/*` | noti-service | 8002 | SSE, no timeout |
| `/api/svc/storage/*` | storage-service | 8003 | 120s timeout |
| `/api/ops-portal/*` | ops-portal | 80 | Admin UI |
| `/api/ops/*` | ops-service | 8005 | Admin API |
| `/api/monitoring/*` | Grafana/Prometheus/Loki | - | Observability |
| `/*` | frontend | 80 | catch-all |

### Canary 배포 지원

```yaml
# VirtualService에서 weight-based routing
route:
- destination:
    host: board-service
    subset: stable
  weight: 90
- destination:
    host: board-service
    subset: canary
  weight: 10
```

---

## Argo Rollouts (Canary Deployment)

Progressive Delivery를 위한 Argo Rollouts를 사용합니다.

### Rollout 설정

```yaml
# 서비스 values.yaml
rollout:
  enabled: true
  canary:
    steps:
    - setWeight: 10
    - pause: { duration: 2m }
    - setWeight: 30
    - pause: { duration: 2m }
    - setWeight: 50
    - pause: { duration: 2m }
    - setWeight: 100
```

### AnalysisTemplate

```yaml
apiVersion: argoproj.io/v1alpha1
kind: AnalysisTemplate
metadata:
  name: success-rate
spec:
  metrics:
  - name: success-rate
    interval: 1m
    successCondition: result[0] >= 0.95
    provider:
      prometheus:
        address: http://prometheus:9090
        query: |
          sum(rate(istio_requests_total{
            destination_service=~"{{args.service-name}}.*",
            response_code!~"5.*"
          }[5m])) / sum(rate(istio_requests_total{
            destination_service=~"{{args.service-name}}.*"
          }[5m]))
```

---

## Secrets Management (External Secrets Operator)

### 아키텍처

```
┌─────────────────────┐      ┌─────────────────────┐
│  AWS Secrets Manager │      │  ClusterSecretStore │
│                     │◄─────│                     │
│  wealist/prod/*     │      │  aws-secrets-store  │
└─────────────────────┘      └─────────────────────┘
                                       │
                                       ▼
                             ┌─────────────────────┐
                             │   ExternalSecret    │
                             │                     │
                             │ - shared-secrets    │
                             │ - db-credentials    │
                             │ - redis-credentials │
                             └─────────────────────┘
                                       │
                                       ▼
                             ┌─────────────────────┐
                             │   K8s Secret        │
                             │                     │
                             │ (자동 생성/동기화)  │
                             └─────────────────────┘
```

### ClusterSecretStore

```yaml
apiVersion: external-secrets.io/v1
kind: ClusterSecretStore
metadata:
  name: aws-secrets-store
spec:
  provider:
    aws:
      service: SecretsManager
      region: ap-northeast-2
      auth:
        jwt:
          serviceAccountRef:
            name: external-secrets-sa
            namespace: external-secrets
```

### ExternalSecret

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: shared-secrets
  namespace: wealist-prod
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: aws-secrets-store
    kind: ClusterSecretStore
  target:
    name: wealist-shared-secret
  dataFrom:
  - extract:
      key: wealist/prod/shared
```

---

## Pod Identity

IRSA를 대체하는 EKS Pod Identity를 사용합니다.

### 구성

```hcl
# terraform/prod/compute/eks.tf
resource "aws_eks_addon" "pod_identity_agent" {
  cluster_name = aws_eks_cluster.main.name
  addon_name   = "eks-pod-identity-agent"
}

resource "aws_eks_pod_identity_association" "eso" {
  cluster_name    = aws_eks_cluster.main.name
  namespace       = "external-secrets"
  service_account = "external-secrets-sa"
  role_arn        = aws_iam_role.eso_role.arn
}
```

### 장점

| 항목 | IRSA | Pod Identity |
|------|------|--------------|
| **설정 복잡도** | 높음 (OIDC, annotation) | 낮음 (EKS 네이티브) |
| **ServiceAccount** | IAM Role ARN annotation 필요 | association만 필요 |
| **관리** | Terraform + K8s 분리 | Terraform 통합 |

---

## Resource Management

### Requests & Limits

| Service | CPU Request | CPU Limit | Memory Request | Memory Limit |
|---------|-------------|-----------|----------------|--------------|
| auth-service | 100m | 500m | 384Mi | 768Mi |
| Go 서비스 (6개) | 25m | 500m | 64Mi | 256Mi |
| ops-portal | 50m | 100m | 64Mi | 128Mi |
| frontend | 50m | 100m | 64Mi | 128Mi |

> **Go 서비스**: user, board, chat, noti, storage, ops-service

### HPA (Horizontal Pod Autoscaler)

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
spec:
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

---

## Health Checks

| Probe | Path | 용도 |
|-------|------|------|
| **Liveness** | `/health/live` | Pod 재시작 여부 |
| **Readiness** | `/health/ready` | 트래픽 수신 여부 |

```yaml
livenessProbe:
  httpGet:
    path: /health/live
    port: 8000
  initialDelaySeconds: 10
  periodSeconds: 10
  failureThreshold: 3

readinessProbe:
  httpGet:
    path: /health/ready
    port: 8000
  initialDelaySeconds: 5
  periodSeconds: 5
  failureThreshold: 3
```

---

## Helm Chart Structure

```
k8s/helm/
├── environments/
│   ├── base.yaml           # 공통 설정
│   ├── local-kind.yaml     # Kind 로컬
│   ├── dev.yaml            # AWS Dev
│   └── prod.yaml           # AWS Production
│
└── charts/
    ├── wealist-common/         # 공통 템플릿 라이브러리
    ├── wealist-infrastructure/ # 인프라 (DB, Redis)
    ├── istio-config/           # Istio VirtualService, Gateway
    ├── istio-addons/           # Kiali, Jaeger
    ├── auth-service/
    ├── user-service/
    ├── board-service/
    ├── chat-service/
    ├── noti-service/
    ├── storage-service/
    ├── ops-service/
    └── ops-portal/
```

### Values Merge 순서

```
chart/values.yaml (기본값)
    ↓
base.yaml (공통 설정)
    ↓
{environment}.yaml (환경별 설정 - 최종 우선)
```

---

## GitOps (ArgoCD)

### App of Apps 패턴

```
┌─────────────────────────────────────────────────────────────┐
│                    k8s-deploy-prod Branch                     │
├─────────────────────────────────────────────────────────────┤
│  k8s/argocd/apps/prod/                                       │
│  ├── root-app.yaml          (App of Apps)                    │
│  ├── external-secrets.yaml  (Bootstrap 첫 번째)              │
│  ├── wealist-infrastructure.yaml                             │
│  ├── auth-service.yaml                                       │
│  ├── user-service.yaml                                       │
│  ├── board-service.yaml                                      │
│  └── ...                                                     │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼ sync
┌─────────────────────────────────────────────────────────────┐
│                         ArgoCD                                │
├─────────────────────────────────────────────────────────────┤
│  Applications:                                                │
│  ├── wealist-prod-root      (App of Apps)                    │
│  ├── external-secrets                                        │
│  ├── wealist-infrastructure                                  │
│  ├── auth-service                                            │
│  └── ...                                                     │
└─────────────────────────────────────────────────────────────┘
```

### Sync Policy

```yaml
syncPolicy:
  automated:
    prune: true
    selfHeal: true
  syncOptions:
  - CreateNamespace=true
  - ServerSideApply=true
```

---

## Environment Comparison

| 항목 | Dev | Prod |
|------|-----|------|
| **Database** | In-cluster PostgreSQL | AWS RDS Multi-AZ |
| **Cache** | In-cluster Redis | AWS ElastiCache Serverless |
| **Storage** | MinIO | AWS S3 + CloudFront |
| **Secrets** | K8s Secret (직접) | AWS Secrets Manager (ESO) |
| **mTLS** | PERMISSIVE | STRICT |
| **AuthorizationPolicy** | 없음 | Zero-Trust (denyAll + allowList) |
| **Telemetry Sampling** | 100% | 10% |
| **Frontend** | EKS 배포 | CloudFront + S3 |
| **Node Type** | Kind (local) | EKS Spot Instance |

---

## Related Pages

- [Architecture Overview](Architecture)
- [AWS Architecture](Architecture-AWS)
- [CI/CD Pipeline](Architecture-CICD)
- [Monitoring Stack](Architecture-Monitoring)
- [Getting Started](Getting-Started)
