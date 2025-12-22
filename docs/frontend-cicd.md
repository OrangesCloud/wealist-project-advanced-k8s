# Frontend CI/CD 가이드

이 문서는 프론트엔드 CI/CD 파이프라인의 구조, 버전 관리 전략, 롤백 방법을 설명합니다.

## 목차

- [개요](#개요)
- [아키텍처](#아키텍처)
- [Docker 이미지 태그 전략](#docker-이미지-태그-전략)
- [워크플로우 상세](#워크플로우-상세)
- [S3 버전 관리](#s3-버전-관리)
- [롤백 가이드](#롤백-가이드)
- [필수 Secrets 설정](#필수-secrets-설정)

---

## 개요

프론트엔드는 두 가지 방식으로 배포됩니다:

| 배포 대상 | 용도 | 버전 관리 |
|-----------|------|-----------|
| **S3 + CloudFront** | 정적 웹 호스팅 (주요) | 버전별 폴더 (`/v1.0.0/`) |
| **GHCR + Kubernetes** | 컨테이너 배포 | Docker 이미지 태그 |

### 핵심 원칙

1. **이미지는 Dev에서만 빌드** - Prod는 검증된 이미지를 그대로 사용
2. **환경 차이는 런타임 설정으로** - `config.js`로 환경별 설정 주입
3. **버전별 보관** - S3와 GHCR 모두 이전 버전 영구 보관

---

## 아키텍처

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         Development Flow                                 │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│   dev 브랜치 push                                                        │
│         │                                                                │
│         ▼                                                                │
│   ┌─────────────────────────────────────────────────────┐               │
│   │           frontend-dev.yaml                          │               │
│   │                                                      │               │
│   │   1. pnpm install & type-check                      │               │
│   │   2. pnpm build                                      │               │
│   │   3. Docker build & push to GHCR                    │               │
│   │      - :sha-abc1234                                 │               │
│   │      - :v1.0.0                                      │               │
│   │      - :latest-dev                                  │               │
│   │   4. S3 배포 (dev 버킷)                              │               │
│   │      - /v1.0.0-abc1234/ (버전 폴더)                 │               │
│   │      - / (루트, 현재 서빙)                           │               │
│   │   5. CloudFront 캐시 무효화                          │               │
│   └─────────────────────────────────────────────────────┘               │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────┐
│                         Production Flow                                  │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│   v* 태그 push 또는 수동 트리거                                           │
│         │                                                                │
│         ▼                                                                │
│   ┌─────────────────────────────────────────────────────┐               │
│   │           frontend-prod.yaml                         │               │
│   │                                                      │               │
│   │   1. GHCR에서 이미지 존재 확인                        │               │
│   │   2. 이미지에 :latest 태그 추가                       │               │
│   │   3. S3 배포 (prod 버킷)                             │               │
│   │      - /v1.0.0/ (버전 폴더)                          │               │
│   │      - / (루트, 현재 서빙)                           │               │
│   │   4. CloudFront 캐시 무효화                          │               │
│   │   5. GitHub Release 생성                             │               │
│   └─────────────────────────────────────────────────────┘               │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Docker 이미지 태그 전략

### 태그 종류

| 태그 | 예시 | 동작 | 용도 |
|------|------|------|------|
| **SHA 태그** | `:sha-abc1234` | 고정 (불변) | 정확한 커밋 추적 |
| **버전 태그** | `:v1.0.0` | 고정 (불변) | 릴리스 버전 |
| **Dev 최신** | `:latest-dev` | 이동 (매 배포) | Dev 환경 최신 |
| **Prod 최신** | `:latest` | 이동 (매 배포) | Prod 환경 최신 |
| **안정화 버전** | `:stable` 🏆 | 이동 (수동) | 검증 완료된 안정 버전 |

### Stable 태그 사용법

`:stable` 태그는 **충분히 검증된 안정화 버전**을 나타냅니다.

**Stable로 지정하는 방법:**

1. **Prod 배포 시 옵션으로**
   - Actions → Frontend CI/CD (Prod) → Run workflow
   - `mark_as_stable` 체크박스 선택

2. **나중에 별도로 지정**
   - Actions → Frontend Mark as Stable → Run workflow
   - 버전 입력 (예: `v1.0.0`)

**권장 플로우:**
```
v1.0.0 배포 → 3-7일 모니터링 → 문제 없으면 stable로 지정
```

**Stable 버전 사용:**
```bash
# K8s에서 stable 버전 사용
helm upgrade frontend ./helm/charts/frontend --set image.tag=stable

# S3에서 stable로 롤백
aws s3 sync s3://${BUCKET}/stable/ s3://${BUCKET}/ --delete
```

### 태그 동작 예시

```
[1] dev 브랜치 push (커밋 abc1234, 버전 1.0.0)
    ┌─────────────────────────────────────┐
    │ sha256:abc123...                    │
    │   ├── :sha-abc1234  (영구)          │
    │   ├── :v1.0.0       (영구)          │
    │   └── :latest-dev   (현재 dev 최신) │
    └─────────────────────────────────────┘

[2] prod 배포 (v1.0.0을 프로덕션으로)
    ┌─────────────────────────────────────┐
    │ sha256:abc123...                    │
    │   ├── :sha-abc1234                  │
    │   ├── :v1.0.0                       │
    │   ├── :latest-dev                   │
    │   └── :latest  ← 추가됨!            │  ← 4개 태그가 같은 이미지
    └─────────────────────────────────────┘

[3] 새 dev 배포 (커밋 def5678, 버전 1.0.1)
    ┌─────────────────────────────────────┐
    │ sha256:abc123... (이전)             │
    │   ├── :sha-abc1234                  │
    │   ├── :v1.0.0                       │
    │   └── :latest      ← prod는 그대로  │
    └─────────────────────────────────────┘
    ┌─────────────────────────────────────┐
    │ sha256:def456... (새 이미지)        │
    │   ├── :sha-def5678                  │
    │   ├── :v1.0.1                       │
    │   └── :latest-dev  ← 이동!          │  ← dev만 새 이미지
    └─────────────────────────────────────┘
```

**핵심 포인트**: 태그는 이미지의 "별명"일 뿐, 실제 이미지는 SHA digest로 식별됩니다.

---

## 자동 버전업 (Semantic Release)

커밋 메시지 기반으로 **자동으로 버전이 올라갑니다!**

### 커밋 메시지 형식

```bash
# 패치 버전 (1.0.0 → 1.0.1)
front-fix: 로그인 버튼 클릭 안되는 문제 수정
front-perf: 이미지 로딩 성능 개선

# 마이너 버전 (1.0.0 → 1.1.0)
front-feat: 다크모드 기능 추가
front-feat(auth): 소셜 로그인 추가

# 메이저 버전 (1.0.0 → 2.0.0) - Breaking Change
front-feat!: API 응답 구조 변경
front-fix!: 인증 토큰 형식 변경

# 버전업 없음 (문서, 스타일, 테스트 등)
front-docs: README 업데이트
front-style: 코드 포맷팅
front-refactor: 컴포넌트 구조 개선
front-test: 단위 테스트 추가
front-chore: 의존성 업데이트
```

### 자동화 흐름

```
1. front-feat: 새 기능 추가 (커밋)
2. dev 브랜치 push
3. semantic-release 분석
   └→ "front-feat 발견! minor 버전업 필요"
4. 빌드 & 배포 성공 후
   └→ package.json 수정 (0.1.0 → 0.2.0)
   └→ CHANGELOG.md 생성
   └→ Git 태그: frontend-v0.2.0
   └→ GitHub Release 생성
```

### 주의사항

- 프론트엔드 관련 커밋만 `front-` prefix 사용
- 다른 서비스 커밋은 기존대로 (`fix:`, `feat:` 등)
- Breaking Change는 `!` 추가 (예: `front-feat!:`)

---

## 워크플로우 상세

### frontend-dev.yaml

**트리거:**
- `dev` 또는 `dev-frontend` 브랜치 push
- `services/frontend/**` 경로 변경 시

**실행 단계:**

```yaml
Jobs:
  1. build-and-test:
     - pnpm install
     - pnpm type-check
     - pnpm build (검증용)
     - 버전 정보 추출

  2. docker-build-push:
     - Docker Buildx 설정
     - GHCR 로그인
     - 이미지 빌드 & 푸시
     - 태그: :sha-xxx, :v1.0.0, :latest-dev

  3. deploy-s3-dev:
     - pnpm build (배포용)
     - config.js 주입 (dev 설정)
     - S3 버전 폴더 배포
     - S3 루트 배포
     - CloudFront 무효화
```

### frontend-prod.yaml

**트리거:**
- `v*` 태그 push (예: `v1.0.0`)
- 수동 트리거 (워크플로우 디스패치)

**실행 단계:**

```yaml
Jobs:
  1. validate:
     - GHCR에서 이미지 존재 확인
     - 버전 정보 준비

  2. tag-image-prod:
     - 기존 이미지 pull
     - :latest 태그 추가
     - :v{version} 태그 추가
     - GHCR에 push

  3. deploy-s3-prod:
     - 소스 체크아웃 (해당 태그)
     - pnpm build
     - config.js 주입 (prod 설정)
     - S3 버전 폴더 배포
     - S3 루트 배포
     - CloudFront 무효화

  4. create-release:
     - GitHub Release 생성
     - 릴리스 노트 자동 생성
```

---

## S3 버전 관리

### 폴더 구조

```
s3://your-bucket-dev/
├── v1.0.0-abc1234/        # 버전 + 커밋 (영구 보관)
│   ├── index.html
│   ├── config.js
│   └── assets/
├── v1.0.1-def5678/
│   └── ...
├── index.html              # 현재 서빙 중인 버전
├── config.js
└── assets/

s3://your-bucket-prod/
├── v1.0.0/                 # 버전별 폴더 (영구 보관)
│   ├── index.html
│   ├── config.js
│   └── assets/
├── v1.0.1/
│   └── ...
├── index.html              # 현재 서빙 중인 버전
├── config.js
└── assets/
```

### 캐시 전략

| 파일 | Cache-Control | 이유 |
|------|---------------|------|
| `index.html` | `no-cache` | 항상 최신 버전 체크 |
| `config.js` | `no-cache` | 런타임 설정, 즉시 반영 |
| `assets/*` | `max-age=31536000, immutable` | 해시된 파일명, 영구 캐시 |

---

## 롤백 가이드

### 🚀 GitHub Actions 롤백 (추천!)

**가장 쉬운 방법!** 웹에서 버튼 클릭만으로 롤백할 수 있습니다.

1. GitHub Repository → **Actions** 탭
2. 왼쪽 메뉴에서 **Frontend Rollback** 선택
3. **Run workflow** 버튼 클릭
4. 옵션 입력:
   - **environment**: `dev` 또는 `prod`
   - **version**: 롤백할 버전 (예: `v1.0.0`, `v1.0.0-abc1234`)
   - **rollback_type**: `s3`, `k8s`, `both` 중 선택
   - **dry_run**: 실제 롤백 없이 확인만 하려면 체크
5. **Run workflow** 클릭!

```
┌─────────────────────────────────────────────────────────┐
│  Frontend Rollback                    [Run workflow ▼]  │
├─────────────────────────────────────────────────────────┤
│  environment:     [dev     ▼]                           │
│  version:         [v1.0.0        ]                      │
│  rollback_type:   [s3      ▼]                           │
│  dry_run:         [ ] Check to preview only             │
│                                                         │
│                              [Cancel] [Run workflow]    │
└─────────────────────────────────────────────────────────┘
```

---

### S3 롤백 (CLI, ~1분)

```bash
# 환경 변수 설정
export BUCKET="your-s3-bucket"
export DIST_ID="your-cloudfront-distribution-id"
export ROLLBACK_VERSION="v1.0.0"  # 롤백할 버전

# 1. 이전 버전을 루트로 복사
aws s3 sync "s3://${BUCKET}/${ROLLBACK_VERSION}/" "s3://${BUCKET}/" --delete

# 2. CloudFront 캐시 무효화
aws cloudfront create-invalidation \
  --distribution-id "${DIST_ID}" \
  --paths "/*"

# 3. 무효화 상태 확인
aws cloudfront get-invalidation \
  --distribution-id "${DIST_ID}" \
  --id <invalidation-id>
```

### Kubernetes 롤백 (Helm)

```bash
# 이전 버전 이미지로 롤백
helm upgrade frontend ./helm/charts/frontend \
  --set image.tag=v1.0.0 \
  -f ./helm/environments/prod.yaml

# 또는 kubectl로 직접 롤백
kubectl rollout undo deployment/frontend -n wealist-prod

# 롤백 상태 확인
kubectl rollout status deployment/frontend -n wealist-prod
```

### Docker 이미지로 롤백 (수동)

```bash
# 1. 이전 버전 이미지 pull
docker pull ghcr.io/your-org/frontend:v1.0.0

# 2. latest 태그로 재지정
docker tag ghcr.io/your-org/frontend:v1.0.0 ghcr.io/your-org/frontend:latest
docker push ghcr.io/your-org/frontend:latest

# 3. K8s 재배포 (이미지 pull)
kubectl rollout restart deployment/frontend -n wealist-prod
```

---

## 필수 Secrets 설정

GitHub Repository Settings > Secrets and variables > Actions에서 설정:

### Dev 환경

| Secret | 설명 | 예시 |
|--------|------|------|
| `AWS_ROLE_ARN` | AWS OIDC 역할 ARN | `arn:aws:iam::123456789:role/github-actions-frontend` |
| `AWS_S3_BUCKET_DEV` | Dev S3 버킷명 | `wealist-frontend-dev` |
| `AWS_CLOUDFRONT_DISTRIBUTION_ID_DEV` | Dev CloudFront ID | `E1234567890ABC` |
| `API_DOMAIN_DEV` | Dev API 도메인 (선택) | `api.dev.wealist.co.kr` |

### Prod 환경

| Secret | 설명 | 예시 |
|--------|------|------|
| `AWS_S3_BUCKET_PROD` | Prod S3 버킷명 | `wealist-frontend-prod` |
| `AWS_CLOUDFRONT_DISTRIBUTION_ID_PROD` | Prod CloudFront ID | `E0987654321XYZ` |
| `API_DOMAIN_PROD` | Prod API 도메인 (선택) | `api.wealist.co.kr` |

### 공통 (선택)

| Secret | 설명 | 예시 |
|--------|------|------|
| `VITE_API_BASE_URL` | 빌드 타임 API URL | `` (빈 문자열 권장) |

---

## 배포 체크리스트

### Dev 배포

- [ ] `dev` 또는 `dev-frontend` 브랜치에 코드 push
- [ ] GitHub Actions 워크플로우 실행 확인
- [ ] GHCR에 이미지 푸시 확인
- [ ] S3 배포 확인
- [ ] CloudFront 캐시 무효화 완료 확인
- [ ] 실제 사이트에서 기능 테스트

### Prod 배포

- [ ] Dev에서 충분히 테스트 완료
- [ ] Git 태그 생성 (`git tag v1.0.0 && git push origin v1.0.0`)
- [ ] GitHub Actions 워크플로우 실행 확인
- [ ] GHCR 이미지 `:latest` 태그 확인
- [ ] S3 prod 배포 확인
- [ ] GitHub Release 생성 확인
- [ ] 프로덕션 사이트 기능 테스트

---

## 트러블슈팅

### 이미지가 GHCR에서 안 보여요

```bash
# GHCR 로그인 확인
echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin

# 이미지 목록 확인
docker search ghcr.io/your-org/frontend
```

### CloudFront 캐시가 안 풀려요

```bash
# 무효화 상태 확인
aws cloudfront list-invalidations --distribution-id $DIST_ID

# 강제 전체 무효화
aws cloudfront create-invalidation \
  --distribution-id $DIST_ID \
  --paths "/*"
```

### S3 권한 오류

AWS OIDC 역할에 다음 권한이 있는지 확인:
- `s3:PutObject`
- `s3:GetObject`
- `s3:DeleteObject`
- `s3:ListBucket`
- `cloudfront:CreateInvalidation`

---

## 관련 문서

- [Helm Chart 가이드](./helm-guide.md)
- [Kubernetes 배포 가이드](./k8s-deployment.md)
- [모니터링 설정](./monitoring.md)
