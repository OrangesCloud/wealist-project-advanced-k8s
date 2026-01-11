# CI/CD Pipeline

weAlist의 CI/CD 파이프라인 아키텍처입니다.

---

## Overview

weAlist는 **GitOps** 기반의 CI/CD 파이프라인을 구현합니다:
- **Backend**: GitHub Actions → ECR → ArgoCD → EKS
- **Frontend**: GitHub Actions → S3 → CloudFront

---

## Architecture Diagrams

### Backend CI/CD Pipeline

![Backend CI/CD](https://raw.githubusercontent.com/OrangesCloud/wealist-project-advanced-k8s/main/docs/images/wealist_cicd_backend.png)

### Frontend CI/CD Pipeline

![Frontend CI/CD](https://raw.githubusercontent.com/OrangesCloud/wealist-project-advanced-k8s/main/docs/images/wealist_cicd_frontend.png)

### GitOps Branch Strategy

![Branch Strategy](https://raw.githubusercontent.com/OrangesCloud/wealist-project-advanced-k8s/main/docs/images/wealist_cicd_branch_strategy.png)

---

## Backend CI/CD

### Dev Environment

| 항목 | 값 |
|------|-----|
| **트리거 브랜치** | `service-deploy-dev` |
| **ECR 경로** | `{service}:latest`, `{service}:{num}-{sha}` |
| **K8s 매니페스트** | `k8s-deploy-dev` 브랜치 |
| **ArgoCD 프로젝트** | `wealist-dev` |
| **네임스페이스** | `wealist-dev` |

**파이프라인 단계:**
1. **Change Detection** - `dorny/paths-filter`로 변경된 서비스 감지
2. **Docker Build** - Matrix 빌드 (최대 8개 서비스 병렬)
3. **ECR Push** - `latest` + `{BUILD_NUM}-{SHA}` 태그
4. **GitOps Update** - `k8s-deploy-dev` 브랜치의 ArgoCD Application YAML 업데이트
5. **ArgoCD Sync** - 자동 동기화 (Auto Sync 활성화)
6. **Discord Notification** - 배포 결과 알림

### Prod Environment

| 항목 | 값 |
|------|-----|
| **트리거 브랜치** | `service-deploy-prod` |
| **ECR 경로** | `prod/{service}:{num}-{sha}` (IMMUTABLE) |
| **K8s 매니페스트** | `k8s-deploy-prod` 브랜치 |
| **ArgoCD 프로젝트** | `wealist-prod` |
| **네임스페이스** | `wealist-prod` |

**주요 차이점:**
- ECR 레포지토리가 **IMMUTABLE** (태그 덮어쓰기 불가)
- `latest` 태그 없음 (버전 태그만 사용)
- Secrets는 AWS Secrets Manager + ESO 사용
- DB/Redis는 AWS RDS/ElastiCache 사용

---

## Frontend CI/CD

### Dev Environment

| 항목 | 값 |
|------|-----|
| **트리거 브랜치** | `dev`, `dev-frontend` |
| **워크플로우** | `frontend-dev.yaml` |
| **Docker Registry** | GHCR (GitHub Container Registry) |
| **배포 대상** | S3 + CloudFront |
| **도메인** | `dev.wealist.co.kr` |

**파이프라인 단계:**
1. **Semantic Release** - 커밋 메시지 기반 자동 버전 범프
2. **pnpm build** - TypeScript 체크 포함
3. **Docker → GHCR** - `sha-{hash}`, `v{version}`, `latest-dev` 태그
4. **S3 Deploy** - Versioned 폴더 + Root 배포
5. **CloudFront Invalidation** - `/*` 캐시 무효화
6. **Discord Notification** - 배포 결과 알림

**캐시 전략:**
```
Assets (js, css, images): max-age=31536000, immutable
index.html, config.js: no-cache, no-store, must-revalidate
```

### Prod Environment

| 항목 | 값 |
|------|-----|
| **트리거** | `frontend-v*` 태그 push |
| **워크플로우** | `frontend-prod.yaml` |
| **배포 대상** | S3 + CloudFront |
| **도메인** | `wealist.co.kr` |

**파이프라인 단계:**
1. **Extract Version** - 태그명에서 버전 추출 (`frontend-v1.0.0` → `1.0.0`)
2. **pnpm build** - Runtime config 주입 (API_DOMAIN)
3. **S3 Versioned Deploy** - `/v1.0.0/` 폴더에 배포
4. **S3 Root Deploy** - immutable assets만 root에 복사
5. **CloudFront Invalidation** - 캐시 무효화
6. **GitHub Release** - CHANGELOG 생성

---

## Branch Strategy

### 브랜치 역할

| 브랜치 | 용도 | 자동 업데이트 |
|--------|------|--------------|
| `main` | Feature 개발 | - |
| `service-deploy-dev` | Dev 서비스 빌드 트리거 | - |
| `service-deploy-prod` | Prod 서비스 빌드 트리거 | - |
| `k8s-deploy-dev` | Dev K8s 매니페스트 | CI가 image.tag 업데이트 |
| `k8s-deploy-prod` | Prod K8s 매니페스트 | CI가 image.tag 업데이트 |

### 배포 플로우

```
main → PR merge → service-deploy-{env} → CI Build → k8s-deploy-{env} → ArgoCD → EKS
```

---

## Security Scanning

### PR Validation (`ci-pr-validation.yaml`)

| 단계 | 도구 | 대상 |
|------|------|------|
| **Lint** | golangci-lint v2.0.2 | Go 서비스 |
| **Test** | go test | Go 서비스 (커버리지 포함) |
| **Auth Test** | Gradle | Spring Boot (auth-service) |
| **Container Scan** | Trivy | Docker 이미지 |

**Trivy 스캔 결과:**
- Table 형식으로 콘솔 출력
- SARIF 형식으로 GitHub Security 탭에 업로드

---

## Deployment Strategies

| 전략 | 용도 | 구현 |
|------|------|------|
| **Rolling Update** | 일반적인 배포 | Kubernetes 기본 |
| **Canary** | 점진적 롤아웃 | Argo Rollouts |

### Canary 배포 (Argo Rollouts)

```yaml
# 10% → 30% → 50% → 100%
rollout:
  enabled: true
  canary:
    steps:
      - setWeight: 10
      - pause: { duration: 2m }
      - setWeight: 30
      - pause: { duration: 2m }
```

---

## Rollback

### Frontend Rollback

`frontend-rollback.yaml` 워크플로우로 이전 버전 복원:

1. **Version 선택** - S3의 versioned 폴더 목록에서 선택
2. **S3 Restore** - `/v{version}/` → root로 복사
3. **CloudFront Invalidation** - 캐시 무효화

### Backend Rollback

ArgoCD UI에서 이전 revision으로 Rollback 또는:

1. `k8s-deploy-{env}` 브랜치에서 이전 image.tag로 수정
2. ArgoCD 자동 동기화

---

## Notifications

### Discord 알림

| 이벤트 | 알림 채널 |
|--------|----------|
| 빌드 시작 | #deploy-notifications |
| 빌드 성공/실패 | #deploy-notifications |
| ArgoCD Sync 완료 | ArgoCD Discord Integration |

---

## GitHub Actions Workflows

| 워크플로우 | 파일 | 용도 |
|-----------|------|------|
| **Backend CI/CD** | `ci-build-images.yaml` | 서비스 빌드 및 배포 |
| **PR Validation** | `ci-pr-validation.yaml` | PR 검증 (lint, test, scan) |
| **Frontend Dev** | `frontend-dev.yaml` | Dev 프론트엔드 배포 |
| **Frontend Prod** | `frontend-prod.yaml` | Prod 프론트엔드 배포 |
| **Frontend Rollback** | `frontend-rollback.yaml` | 프론트엔드 롤백 |
| **Frontend Mark Stable** | `frontend-mark-stable.yaml` | 안정 버전 마킹 |
| **Prod Image Update** | `discord-update-production-image.yaml` | 수동 이미지 업데이트 |

---

## Environment Comparison

| 항목 | Dev | Prod |
|------|-----|------|
| ECR 경로 | `{service}` | `prod/{service}` |
| 이미지 태그 | `latest` + 버전 | 버전만 (IMMUTABLE) |
| Secrets | Helm values | AWS Secrets Manager (ESO) |
| DB | Helm-deployed PostgreSQL | AWS RDS |
| Redis | Helm-deployed | AWS ElastiCache |
| mTLS | PERMISSIVE | STRICT |
| Prometheus 샘플링 | 100% | 10% |

---

## Related Pages

- [Architecture Overview](Architecture)
- [AWS Architecture](Architecture-AWS)
- [Kubernetes Architecture](Architecture-K8s)
- [Getting Started](Getting-Started)
