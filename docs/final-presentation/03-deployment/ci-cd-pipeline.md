# CI/CD 파이프라인

> **목표**: 코드 변경 → 자동 빌드 → 이미지 배포 → ArgoCD 동기화

---

## 파이프라인 개요

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        CI/CD PIPELINE FLOW                              │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐              │
│  │  Push   │───▶│ GitHub  │───▶│  Build  │───▶│   ECR   │              │
│  │  Code   │    │ Actions │    │  Image  │    │  Push   │              │
│  └─────────┘    └─────────┘    └─────────┘    └─────────┘              │
│       │                                            │                    │
│       │                                            ▼                    │
│       │         ┌─────────┐    ┌─────────┐    ┌─────────┐              │
│       │         │  Sync   │◀───│ ArgoCD  │◀───│ Detect  │              │
│       │         │  Apply  │    │  Watch  │    │ Change  │              │
│       │         └─────────┘    └─────────┘    └─────────┘              │
│       │              │                                                  │
│       │              ▼                                                  │
│       │         ┌─────────┐                                            │
│       └────────▶│  Live   │                                            │
│                 │  Pods   │                                            │
│                 └─────────┘                                            │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 빌드 단계

### Docker 이미지 빌드

각 서비스별 Dockerfile이 `services/{service}/docker/Dockerfile`에 위치합니다.

#### Go 서비스 빌드 패턴

```dockerfile
# Multi-stage build (Go services)
FROM golang:1.24-bookworm AS builder
WORKDIR /workspace

# 공통 패키지 복사
COPY packages/wealist-advanced-go-pkg/ ./packages/wealist-advanced-go-pkg/
WORKDIR /workspace/services/{service}
COPY services/{service}/ ./

# 로컬 패키지 연결
RUN echo 'replace github.com/OrangesCloud/wealist-advanced-go-pkg => ../../packages/wealist-advanced-go-pkg' >> go.mod
RUN GOWORK=off go mod tidy && GOWORK=off go mod download

# 빌드
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOWORK=off go build -o app ./cmd/api

# Runtime stage
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /workspace/services/{service}/app /app
ENTRYPOINT ["/app"]
```

#### Spring Boot 서비스 빌드 패턴

```dockerfile
# Multi-stage build (auth-service)
FROM eclipse-temurin:21-jdk AS builder
WORKDIR /workspace
COPY . .
RUN ./gradlew bootJar -x test

# Runtime stage
FROM eclipse-temurin:21-jre
COPY --from=builder /workspace/build/libs/*.jar app.jar
ENTRYPOINT ["java", "-jar", "/app.jar"]
```

### 빌드 명령어

```bash
# 로컬 빌드 (Kind 클러스터용)
make {service}-build      # Docker 이미지 빌드
make {service}-load       # Kind 클러스터에 로드
make {service}-redeploy   # 배포 재시작

# 전체 서비스 빌드
make build-all            # 모든 서비스 빌드
make redeploy-all         # 모든 서비스 재시작
```

---

## 이미지 레지스트리

### 환경별 레지스트리

| 환경 | Registry | 비고 |
|------|----------|------|
| localhost | Kind 내부 | `kind load docker-image` |
| dev | AWS ECR | `{account}.dkr.ecr.ap-northeast-2.amazonaws.com` |
| prod | AWS ECR | `{account}.dkr.ecr.ap-northeast-2.amazonaws.com` |

### ECR 이미지 태그 전략

```
{repository}/{service}:{build-number}-{short-commit-hash}

예시:
290008131187.dkr.ecr.ap-northeast-2.amazonaws.com/prod/auth-service:96-a04f072
```

| 구성요소 | 설명 |
|----------|------|
| `{repository}` | ECR 레지스트리 URL |
| `{service}` | 서비스명 |
| `{build-number}` | CI 빌드 번호 |
| `{short-commit-hash}` | Git 커밋 해시 (7자리) |

---

## Git 브랜치 전략

### 브랜치 구조

```
main (개발 완료)
  ↓ PR merge
prod (스테이징)
  ↓ PR merge
k8s-deploy-prod (ArgoCD 감시)
```

### 배포 플로우

```
1. feature/* → main     # 개발 완료 후 PR 머지
2. main → prod          # 스테이징 검증 후 PR 머지
3. prod → k8s-deploy-prod  # 프로덕션 배포 (ArgoCD 감지)
```

### ArgoCD 감시 브랜치

| 환경 | 감시 브랜치 | 트리거 |
|------|------------|--------|
| dev | `main` | 자동 (Auto Sync) |
| prod | `k8s-deploy-prod` | 자동 (Auto Sync) |

---

## 배포 자동화

### ArgoCD Sync Policy

```yaml
syncPolicy:
  automated:
    prune: true      # 삭제된 리소스 자동 제거
    selfHeal: true   # drift 자동 복구
```

### 수동 배포 트리거

```bash
# ArgoCD CLI로 동기화
argocd app sync {app-name}

# 특정 서비스만 재배포
kubectl rollout restart deployment/{service} -n wealist-prod
```

---

## 검증 단계

### Pre-deployment 검증

```bash
# 전체 검증 (Terraform + Helm + Go)
make validate-all

# 개별 검증
make validate-terraform    # Terraform validate
make validate-helm         # Helm lint
make validate-go           # Go 서비스 빌드 검증
```

### Post-deployment 검증

```bash
# Pod 상태 확인
make status

# 서비스 로그 확인
kubectl logs -f deploy/{service} -n wealist-prod

# Health check
curl https://api.wealist.co.kr/svc/{service}/health/live
```

---

## 롤백 전략

### 이미지 태그 롤백

```yaml
# ArgoCD Application에서 이전 태그로 변경
parameters:
  - name: image.tag
    value: "95-b23e456"  # 이전 버전 태그
```

### ArgoCD History 롤백

```bash
# 히스토리 조회
argocd app history {app-name}

# 특정 버전으로 롤백
argocd app rollback {app-name} {revision-id}
```

### Kubernetes 롤백

```bash
# 롤아웃 히스토리 조회
kubectl rollout history deployment/{service} -n wealist-prod

# 이전 버전으로 롤백
kubectl rollout undo deployment/{service} -n wealist-prod
```

---

## 관련 문서

- [ArgoCD 설정](./argocd-setup.md)
- [환경 설정](./environment-config.md)

