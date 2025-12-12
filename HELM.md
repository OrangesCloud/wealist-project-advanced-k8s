# Helm Migration 가이드 (Test Environment)

이 문서는 `wealist-project-advanced-k8s` 프로젝트의 Helm Chart 도입 및 테스트 방법에 대해 기술합니다.
테스트 편의를 위해 별도의 메이크파일(`Makefile.helm`)과 클러스터(`wealist-helm`)를 사용합니다.

## 1. 사전 준비
- Docker Desktop 실행 중일 것
- `kubectl`, `kind`, `helm` 설치됨

## 2. 테스트 워크플로우
아래 명령어들을 순서대로 실행하여 로컬 환경에서 전체 서비스를 배포할 수 있습니다.

### 2.1. 테스트 클러스터 생성
기존 `wealist` 클러스터와 충돌하지 않도록 `wealist-helm`이라는 이름으로 생성됩니다.
```bash
make -f Makefile.helm setup
```

### 2.2. 이미지 빌드 및 로컬 레지스트리 전송
모든 마이크로서비스 및 인프라 이미지를 빌드하여 `localhost:5001` 레지스트리에 푸시합니다.
```bash
make -f Makefile.helm load-images
```

### 2.3. 인프라 배포 (DB, Redis)
Helm 배포 전, 필수 의존성(PostgreSQL, Redis 등)을 먼저 배포합니다.
```bash
make -f Makefile.helm deploy-infra
```

### 2.4. 설정 파일 배포 (ConfigMap/Secret)
**[중요]** Helm 차트가 참조하는 `wealist-shared-config` 등을 생성해야 Pod가 정상 실행됩니다.
```bash
make -f Makefile.helm deploy-configs
```

### 2.5. Helm Chart 배포 (전체 서비스)
작성된 `charts/microservice-chart`와 각 서비스의 `values.yaml`을 사용하여 배포합니다.
이미지 경로는 자동으로 `localhost:5001/<service>`로 설정됩니다.
```bash
make -f Makefile.helm deploy-helm
```

## 3. 검증 및 접속
배포 상태를 확인합니다.
```bash
make -f Makefile.helm status
```

### 접속 주소
- **Frontend**: http://local.wealist.co.kr (로컬 DNS 설정 필요: `127.0.0.1 local.wealist.co.kr`)
- 또는 포트 포워딩:
  ```bash
  kubectl port-forward svc/frontend 8080:80 -n wealist-dev
  # 접속: http://localhost:8080
  ```

## 4. 정리 (Clean up)
테스트가 끝나면 클러스터를 삭제합니다.
```bash
make -f Makefile.helm clean
```
