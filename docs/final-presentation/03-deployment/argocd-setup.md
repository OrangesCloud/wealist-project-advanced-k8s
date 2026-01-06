# ArgoCD GitOps 설정

> **목표**: Git 저장소를 Single Source of Truth로 사용하는 선언적 배포

---

## ArgoCD 개요

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        ArgoCD GitOps Architecture                        │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│    ┌──────────────────┐                                                 │
│    │  Git Repository  │ ◀──── Single Source of Truth                   │
│    │  (k8s-deploy-*)  │                                                 │
│    └────────┬─────────┘                                                 │
│             │                                                            │
│             │ Pull (3분마다)                                            │
│             ▼                                                            │
│    ┌──────────────────┐                                                 │
│    │   ArgoCD Server  │ ◀──── Application Controller                   │
│    │                  │       Repo Server                               │
│    │                  │       Notification Controller                   │
│    └────────┬─────────┘                                                 │
│             │                                                            │
│             │ Sync (자동)                                               │
│             ▼                                                            │
│    ┌──────────────────┐                                                 │
│    │  EKS Cluster     │                                                 │
│    │  ┌────────────┐  │                                                 │
│    │  │ wealist-   │  │                                                 │
│    │  │ prod       │  │◀──── Namespace                                 │
│    │  └────────────┘  │                                                 │
│    └──────────────────┘                                                 │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Application 구조

### 디렉토리 레이아웃

```
k8s/argocd/apps/
├── dev/                         # 개발 환경
│   ├── auth-service.yaml
│   ├── user-service.yaml
│   └── ...
└── prod/                        # 프로덕션 환경
    ├── auth-service.yaml        # 서비스 Applications
    ├── board-service.yaml
    ├── chat-service.yaml
    ├── noti-service.yaml
    ├── storage-service.yaml
    ├── user-service.yaml
    ├── ops-service.yaml
    ├── ops-portal.yaml
    ├── infrastructure.yaml      # 인프라 설정
    ├── monitoring.yaml          # 모니터링 스택
    ├── external-secrets.yaml    # ESO 연동
    ├── istio-config.yaml        # Istio 설정
    ├── istio-addons.yaml        # Kiali, Jaeger
    └── cluster-addons.yaml      # 클러스터 애드온
```

---

## Application 정의

### 서비스 Application 예시

```yaml
# k8s/argocd/apps/prod/auth-service.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: auth-service-prod
  namespace: argocd
  annotations:
    argocd.argoproj.io/sync-wave: "5"  # 동기화 순서
    # Discord 알림 설정
    notifications.argoproj.io/subscribe.on-deployed.discord: prod-deployment-alerts
spec:
  project: wealist-prod
  source:
    repoURL: https://github.com/OrangesCloud/wealist-project-advanced-k8s.git
    targetRevision: k8s-deploy-prod
    path: k8s/helm/charts/auth-service
    helm:
      valueFiles:
        - values.yaml
        - ../../environments/base.yaml
        - ../../environments/prod.yaml
      parameters:
        - name: image.repository
          value: "prod/auth-service"
        - name: image.tag
          value: "96-a04f072"
        - name: externalSecrets.enabled
          value: "true"
  destination:
    server: https://kubernetes.default.svc
    namespace: wealist-prod
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

---

## Sync Wave (동기화 순서)

ArgoCD는 `sync-wave` 어노테이션으로 배포 순서를 제어합니다.

### 배포 순서

| Wave | 리소스 | 설명 |
|------|--------|------|
| 0 | Namespace, CRDs | 기본 리소스 |
| 1 | External Secrets | 시크릿 동기화 |
| 2 | Infrastructure | ConfigMap, 공유 설정 |
| 3 | DB Init Jobs | 데이터베이스 초기화 |
| 4 | Monitoring | Prometheus, Grafana |
| 5 | Services | 애플리케이션 서비스 |
| 6 | Istio Config | VirtualService, DestinationRule |

### Sync Wave 설정

```yaml
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "5"
```

---

## Helm Values 계층

### Values 파일 우선순위 (후자가 덮어쓰기)

```yaml
valueFiles:
  - values.yaml                    # 1. 서비스 기본값
  - ../../environments/base.yaml   # 2. 환경 공통값
  - ../../environments/prod.yaml   # 3. 환경별 값
```

### Parameters (최우선 오버라이드)

```yaml
parameters:
  - name: image.tag
    value: "96-a04f072"           # values 파일보다 우선
```

---

## Sync Policy

### 자동 동기화

```yaml
syncPolicy:
  automated:
    prune: true       # Git에서 삭제된 리소스 자동 제거
    selfHeal: true    # 수동 변경 자동 복구 (drift detection)
```

### Sync Options

```yaml
syncPolicy:
  syncOptions:
    - CreateNamespace=true   # 네임스페이스 자동 생성
    - PruneLast=true         # 삭제는 마지막에
    - ApplyOutOfSyncOnly=true
```

---

## 프로젝트 설정

### AppProject 정의

```yaml
apiVersion: argoproj.io/v1alpha1
kind: AppProject
metadata:
  name: wealist-prod
  namespace: argocd
spec:
  description: "Wealist Production Environment"
  sourceRepos:
    - "https://github.com/OrangesCloud/*"
  destinations:
    - namespace: wealist-prod
      server: https://kubernetes.default.svc
    - namespace: istio-system
      server: https://kubernetes.default.svc
  clusterResourceWhitelist:
    - group: "*"
      kind: "*"
```

---

## 알림 설정

### Discord 알림

```yaml
metadata:
  annotations:
    # 배포 완료
    notifications.argoproj.io/subscribe.on-deployed.discord: prod-deployment-alerts
    # 동기화 실패
    notifications.argoproj.io/subscribe.on-sync-failed.discord: prod-deployment-alerts
    # 동기화 진행 중
    notifications.argoproj.io/subscribe.on-sync-running.discord: prod-deployment-alerts
```

### 알림 트리거

| 트리거 | 설명 |
|--------|------|
| `on-deployed` | 배포 성공 시 |
| `on-sync-failed` | 동기화 실패 시 |
| `on-health-degraded` | 헬스체크 실패 시 |
| `on-sync-status-unknown` | 상태 불명 시 |

---

## ArgoCD CLI 명령어

### 기본 명령어

```bash
# 로그인
argocd login argocd.wealist.co.kr

# Application 목록
argocd app list

# 동기화
argocd app sync auth-service-prod

# 상태 확인
argocd app get auth-service-prod

# 히스토리
argocd app history auth-service-prod

# 롤백
argocd app rollback auth-service-prod {revision}
```

### 트러블슈팅

```bash
# 동기화 상세 로그
argocd app sync auth-service-prod --prune --force

# 리소스 트리 확인
argocd app resources auth-service-prod

# diff 확인
argocd app diff auth-service-prod
```

---

## 접속 URL

| 환경 | URL | 비고 |
|------|-----|------|
| Production | `https://argocd.wealist.co.kr` | NLB + Istio Gateway |
| API 경로 | `https://api.wealist.co.kr/api/argo` | VirtualService |

---

## 관련 문서

- [CI/CD 파이프라인](./ci-cd-pipeline.md)
- [환경 설정](./environment-config.md)

