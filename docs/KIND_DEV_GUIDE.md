# Kind 클러스터 개발 가이드

팀원을 위한 로컬 Kubernetes 개발 환경 설정 및 사용 가이드입니다.

---

## 목차

- [사전 요구사항](#사전-요구사항)
- [빠른 시작 (3단계)](#빠른-시작-3단계)
- [일상적인 개발 워크플로우](#일상적인-개발-워크플로우)
- [서비스별 재배포](#서비스별-재배포)
- [Helm 배포 (권장)](#helm-배포-권장)
- [유용한 명령어](#유용한-명령어)
- [SonarQube 코드 분석](#sonarqube-코드-분석)
- [트러블슈팅](#트러블슈팅)

---

## 사전 요구사항

### 필수 설치

```bash
# macOS
brew install kind kubectl helm

# Docker Desktop 필요 (또는 Rancher Desktop, Colima 등)
```

### 설치 확인

```bash
docker --version      # Docker 24+ 권장
kind --version        # v0.20+ 권장
kubectl version       # v1.28+ 권장
helm version          # v3.12+ 권장
```

### Secrets 설정 (필수)

OAuth 로그인과 JWT 토큰이 정상 동작하려면 secrets 파일이 필요합니다:

```bash
# 1. secrets 파일 생성
cp helm/environments/secrets.example.yaml helm/environments/localhost-secrets.yaml

# 2. 파일 편집하여 실제 값 입력
vi helm/environments/localhost-secrets.yaml

# 필수 항목:
# - GOOGLE_CLIENT_ID: Google Cloud Console에서 발급
# - GOOGLE_CLIENT_SECRET: 위와 동일
# - JWT_SECRET: 자동 생성됨 (기본값 사용 가능)
```

> ⚠️ **secrets 파일이 없으면:**
>
> - Google OAuth 로그인 실패 (invalid_client 에러)
> - JWT 토큰 생성 실패 (WeakKeyException)
> - API 호출 시 401/500 에러

---

## 빠른 시작 (4단계)

처음 환경을 설정하거나, 클러스터를 새로 만들 때 사용합니다.

```bash
# 1단계: Kind 클러스터 + 로컬 레지스트리 생성
make kind-setup

# 2단계: 모든 이미지 빌드 및 로드
make kind-load-images

# 3단계: Secrets 설정 (위 "Secrets 설정" 섹션 참고)
cp helm/environments/secrets.example.yaml helm/environments/localhost-secrets.yaml
vi helm/environments/localhost-secrets.yaml  # GOOGLE_CLIENT_ID 등 입력

# 4단계: Helm으로 전체 배포
make helm-install-all ENV=localhost
```

**완료 후 확인:**

```bash
make status                    # Pod 상태 확인
kubectl get pods -n wealist-kind-local
```

**접속 URL:**

- Frontend: http://localhost
- API Gateway: http://localhost/api/...

---

## 일상적인 개발 워크플로우

### 아침에 출근했을 때

```bash
# 클러스터 상태 확인
make status

# Pod이 안 보이면 (재부팅 후)
make kind-recover
```

### 코드 수정 후 테스트

```bash
# 특정 서비스만 재빌드 + 재배포
make board-service-all
make user-service-all
make chat-service-all
# ... 등등

# 전체 서비스 재시작 (설정 변경 시)
make redeploy-all
```

### 퇴근할 때

```bash
# 클러스터는 그대로 두고 퇴근해도 됨 (Docker Desktop 켜둔 상태)
# 또는 리소스 절약을 위해 중지
docker stop wealist-control-plane wealist-worker wealist-worker2
```

---

## 서비스별 재배포

코드를 수정한 서비스만 빠르게 재배포할 수 있습니다.

### 명령어 패턴

| 명령어                   | 설명                                  |
| ------------------------ | ------------------------------------- |
| `make {서비스}-build`    | Docker 이미지만 빌드                  |
| `make {서비스}-load`     | 빌드 + 레지스트리에 푸시              |
| `make {서비스}-redeploy` | Pod 재시작 (이미지 다시 풀)           |
| `make {서비스}-all`      | 빌드 + 푸시 + 재시작 (가장 많이 사용) |

### 사용 예시

```bash
# board-service 코드 수정 후
make board-service-all

# 여러 서비스 수정 후
make user-service-all
make chat-service-all

# ConfigMap/Secret 변경 후 전체 재시작
make redeploy-all
```

### 사용 가능한 서비스

- `auth-service` - 인증 (Spring Boot)
- `user-service` - 사용자/워크스페이스
- `board-service` - 프로젝트/보드
- `chat-service` - 실시간 채팅
- `noti-service` - 알림
- `storage-service` - 파일 저장
- `frontend` - React 웹 UI

---

## Helm 배포 (권장)

### 환경별 배포

```bash
# 로컬 Kind 클러스터 (기본)
make helm-install-all ENV=localhost

# 로컬 Ubuntu 서버 (외부 DB 사용)
make helm-install-all ENV=local-ubuntu

# 개발/스테이징/프로덕션
make helm-install-all ENV=dev
make helm-install-all ENV=staging
make helm-install-all ENV=prod
```

### 환경별 네임스페이스

| ENV          | Namespace          | 도메인                |
| ------------ | ------------------ | --------------------- |
| localhost    | wealist-kind-local | localhost             |
| local-ubuntu | wealist-dev        | local.wealist.co.kr   |
| dev          | wealist-dev        | dev.wealist.co.kr     |
| staging      | wealist-staging    | staging.wealist.co.kr |
| prod         | wealist-prod       | wealist.co.kr         |

### 업그레이드 / 삭제

```bash
# 설정 변경 후 업그레이드
make helm-upgrade-all ENV=localhost

# 전체 삭제
make helm-uninstall-all ENV=localhost
```

### 최초 설치 (DB 마이그레이션 포함)

처음 배포하거나 DB 스키마가 변경되었을 때:

```bash
make helm-install-all-init ENV=localhost
```

---

## 유용한 명령어

### 상태 확인

```bash
# Pod 상태
make status
kubectl get pods -n wealist-kind-local

# 특정 Pod 로그
kubectl logs -f deployment/board-service -n wealist-kind-local

# Pod 상세 정보 (에러 확인)
kubectl describe pod -l app=board-service -n wealist-kind-local
```

### 클러스터 관리

```bash
# 클러스터 복구 (재부팅 후)
make kind-recover

# 클러스터 완전 삭제
make kind-delete

# 클러스터 새로 생성
make kind-setup
```

### Helm 차트 관리

```bash
# 차트 문법 검사
make helm-lint

# 의존성 업데이트
make helm-deps-build

# 전체 검증
make helm-validate
```

### 디버깅

```bash
# Pod 안에서 직접 명령 실행
kubectl exec -it deployment/board-service -n wealist-kind-local -- sh

# DB 연결 테스트
kubectl exec -it deployment/board-service -n wealist-kind-local -- \
  env | grep DB

# 서비스 간 통신 테스트
kubectl run -it --rm debug --image=curlimages/curl -n wealist-kind-local -- \
  curl http://user-service:8081/health/live
```

---

## SonarQube 코드 분석

### SonarQube 서버 시작

```bash
make sonar-up
# 접속: http://localhost:9000
# 기본 로그인: admin / admin
```

### 토큰 생성

1. http://localhost:9000 접속
2. 로그인 후 우상단 프로필 > My Account > Security > Tokens
3. Generate Token > 복사

### 코드 분석 실행

```bash
# 토큰 설정
export SONAR_TOKEN="sqa_xxxxx"

# 개별 서비스 분석
make sonar-user
make sonar-board
make sonar-chat
make sonar-auth
make sonar-frontend

# Go 서비스 전체
make sonar-go

# 모든 서비스
make sonar-all
```

### SonarQube 환경 관리

```bash
make sonar-up       # 시작
make sonar-down     # 중지
make sonar-status   # 상태 확인
make sonar-logs     # 로그 확인
make sonar-clean    # 데이터 초기화 (주의!)
```

---

## 트러블슈팅

### Pod이 계속 재시작됨

```bash
# 1. 에러 확인
kubectl describe pod -l app=board-service -n wealist-kind-local

# 2. 로그 확인
kubectl logs deployment/board-service -n wealist-kind-local --previous

# 3. Health check 경로 문제인 경우가 많음
# Helm values에서 healthCheck 설정 확인
```

### 이미지가 업데이트 안 됨

```bash
# 이미지 다시 빌드 + 푸시 + Pod 재시작
make board-service-all

# 그래도 안 되면 Pod 강제 삭제
kubectl delete pod -l app=board-service -n wealist-kind-local
```

### 클러스터 접속 안 됨 (재부팅 후)

```bash
# Kind 컨테이너 재시작
make kind-recover

# 그래도 안 되면 클러스터 재생성
make kind-delete
make kind-setup
make kind-load-images
make helm-install-all ENV=localhost
```

### DB 연결 실패

```bash
# 1. PostgreSQL Pod 상태 확인
kubectl get pods -l app=postgres -n wealist-kind-local

# 2. 서비스 환경변수 확인
kubectl exec deployment/board-service -n wealist-kind-local -- env | grep DB

# 3. DB 직접 연결 테스트
kubectl exec -it statefulset/postgres -n wealist-kind-local -- \
  psql -U postgres -c "\l"
```

### Secret/ConfigMap 변경이 적용 안 됨

```bash
# 모든 Pod 재시작 (새 설정 로드)
make redeploy-all ENV=localhost
```

---

## 참고 문서

| 문서                                                      | 설명                       |
| --------------------------------------------------------- | -------------------------- |
| [CLAUDE.md](../CLAUDE.md)                                 | 프로젝트 전체 구조 및 규칙 |
| [CONFIGURATION.md](./CONFIGURATION.md)                    | 포트, URL, 환경변수 설정   |
| [docker/SONARQUBE_GUIDE.md](../docker/SONARQUBE_GUIDE.md) | SonarQube 상세 가이드      |
| [helm/README.md](../helm/README.md)                       | Helm 차트 구조             |

---

## 자주 쓰는 명령어 요약

```bash
# === 매일 쓰는 명령어 ===
make status                    # Pod 상태 확인
make board-service-all         # 특정 서비스 재배포
make redeploy-all              # 전체 재시작
make kind-recover              # 재부팅 후 복구

# === 처음 설정할 때 ===
make kind-setup                # 클러스터 생성
make kind-load-images          # 이미지 로드
make helm-install-all ENV=localhost  # 배포

# === Helm 관련 ===
make helm-upgrade-all ENV=localhost  # 업그레이드
make helm-uninstall-all ENV=localhost # 삭제
make helm-deps-build           # 의존성 빌드

# === SonarQube ===
make sonar-up                  # 서버 시작
export SONAR_TOKEN="..."       # 토큰 설정
make sonar-all                 # 전체 분석

# === 디버깅 ===
kubectl logs -f deployment/board-service -n wealist-kind-local
kubectl describe pod -l app=board-service -n wealist-kind-local
kubectl exec -it deployment/board-service -n wealist-kind-local -- sh
```
