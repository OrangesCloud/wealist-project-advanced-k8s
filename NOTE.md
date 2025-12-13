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
make kind-apply                                   # 기본: local.wealist.co.kr
make kind-apply LOCAL_DOMAIN=wonny.wealist.co.kr  # 커스텀 도메인
```

### 예시: wonny.wealist.co.kr로 배포

```bash
# CNAME/A 레코드 등록 후 아래 명령어 실행
make kind-apply LOCAL_DOMAIN=wonny.wealist.co.kr

# 접속: http://wonny.wealist.co.kr (CDN 사용 시 https)
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
make frontend-all
make auth-service-all
make board-service-all
# ... 등

# 또는 단계별로
make frontend-load      # 빌드 + 레지스트리 푸시
make frontend-redeploy  # k8s 롤아웃 재시작
```
