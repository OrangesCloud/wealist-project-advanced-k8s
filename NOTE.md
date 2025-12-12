# 빠른 시작

## Docker Compose 환경

```bash
./docker/scripts/dev.sh up
```

> `.env.dev` 파일 주의! 없으면 자동 생성됨

---

## Kubernetes (Kind) 환경

### 사전 준비

`k8s/base/shared/secret-shared.yaml`, `services/auth-service/k8s/base/secret.yaml`에 Google OAuth 관련 설정 필요

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
# CNAME 등록 후 아래 명령어 실행
make kind-apply LOCAL_DOMAIN=wonny.wealist.co.kr

# 접속: https://wonny.wealist.co.kr
# (자체 서명 인증서 - 브라우저 경고 무시)
```

### mkcert로 브라우저 경고 없애기 (선택)

```bash
# mkcert 설치 후
cd docker/scripts/dev/certs/
mkcert <서버IP> wonny.wealist.co.kr local.wealist.co.kr localhost 127.0.0.1

# 생성된 .pem 파일들이 자동으로 인식됨
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
