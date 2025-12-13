# 빠른 시작

## Docker Compose 환경

```bash
./docker/scripts/dev.sh up
```

> `.env.dev` 파일 주의! 없으면 자동 생성됨

---

## Kubernetes (Kind) 환경

### 사전 준비

1. **Secret 파일 생성** (최초 1회)

   ```bash
   # 템플릿 복사
   cp k8s/base/namespace-dev/secret.yaml.example k8s/base/namespace-dev/secret.yaml

   # Google OAuth 등 실제 값으로 수정
   vi k8s/base/namespace-dev/secret.yaml
   ```

2. **Auth 서비스 Secret** (필요시)
   ```bash
   cp services/auth-service/k8s/base/secret.yaml.example services/auth-service/k8s/base/secret.yaml
   ```

> secret.yaml 파일은 .gitignore에 의해 커밋되지 않음

### 3단계 배포

```bash
# 1. 클러스터 생성
make kind-setup

# 2. 이미지 빌드/로드
make kind-load-images

# 3. 배포 (도메인 선택)
make kind-apply                                    # 기본: dev.wealist.co.kr
make kind-apply PUBLIC_DOMAIN=prod.wealist.co.kr   # 커스텀 도메인
```

### 도메인 구조 (CloudFront CDN)

- **PUBLIC_DOMAIN** (예: `dev.wealist.co.kr`): 사용자 접속용 (OAuth, 프론트엔드)
- **API_DOMAIN** (예: `api.dev.wealist.co.kr`): CloudFront → 백엔드 Origin용 (자동 생성)

```
사용자 → CloudFront (dev.wealist.co.kr)
         ├── /           → S3 (프론트엔드)
         └── /svc/*      → api.dev.wealist.co.kr (백엔드 API)
```

### 예시: dev.wealist.co.kr로 배포

```bash
# Route 53에서 설정:
# - dev.wealist.co.kr → CloudFront Alias
# - api.dev.wealist.co.kr → Kind 클러스터 IP (A 레코드)

make kind-apply PUBLIC_DOMAIN=dev.wealist.co.kr

# 접속: https://dev.wealist.co.kr (CloudFront)
```

---

## 프로젝트 구조

```
k8s/
├── base/
│   ├── namespace-dev/        # Dev 환경 전용
│   │   ├── configmap.yaml    # wealist-config
│   │   ├── secret.yaml       # wealist-secrets (gitignore)
│   │   └── secret.yaml.example
│   └── namespace-prod/       # Prod는 Vault 사용 예정
└── overlays/
    ├── develop-registry/     # Kind 로컬 개발용
    └── prod/
```

---

## 유용한 명령어

```bash
# 클러스터 확인
kind get clusters

# 네임스페이스 확인
kubectl get namespaces

# Pod 상태 확인
make status

# 클러스터 삭제 후 재설정
make kind-delete && make kind-setup && make kind-load-images && make kind-apply
```

---

## 개별 서비스 재배포

```bash
# 서비스 빌드 + 푸시 + 재배포 (한 번에)
make auth-service-all
make board-service-all
make chat-service-all
# ... 등

# 또는 단계별로
make auth-service-load      # 빌드 + 레지스트리 푸시
make auth-service-redeploy  # k8s 롤아웃 재시작
```

> 프론트엔드는 CloudFront + S3에서 호스팅되므로 K8s 배포 대상이 아님
