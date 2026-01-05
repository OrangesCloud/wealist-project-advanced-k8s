# Dev 환경 Make 명령어 가이드 (wealist-oranges)

> Kind 클러스터 기반 Dev 환경 (ArgoCD + ECR + Istio Ambient)

---

## 빠른 시작 (Quick Start)

```bash
# 1. 전체 환경 설정 (클러스터 + ArgoCD + 앱 배포)
make kind-dev-setup

# 2. 팀원 RBAC 설정 (팀원 접근 권한)
make kind-dev-rbac

# 3. 팀원별 kubeconfig 생성
make kind-dev-kubeconfig USERNAME=team-oranges
```

---

## 명령어 순서 (Execution Order)

### 관리자 (Admin)

```
make kind-dev-setup       # 1. 클러스터 + ArgoCD + 앱 배포
        ↓
make kind-dev-rbac        # 2. RBAC 설정 (팀원 권한)
        ↓
make kind-dev-kubeconfig  # 3. 팀원 kubeconfig 생성
```

### 팀원 (Developer)

```bash
# 관리자에게 받은 kubeconfig 사용
export KUBECONFIG=/path/to/team-oranges-kubeconfig.yaml

# 상태 확인
kubectl get pods -n wealist-dev
kubectl logs -n wealist-dev <pod-name>
```

---

## 명령어 요약

| 명령어                       | 용도                        | 언제 사용?               |
| ---------------------------- | --------------------------- | ------------------------ |
| `make kind-dev-setup`        | 전체 환경 설정              | 처음 환경 구축           |
| `make kind-dev-rbac`         | 팀원 RBAC 설정              | setup 후 팀원 추가 시    |
| `make kind-dev-kubeconfig`   | 팀원 kubeconfig 생성        | 새 팀원 추가 시          |
| `make kind-dev-env-status`   | 환경 상태 확인              | 클러스터+DB 상태 확인    |
| `make argo-status`           | ArgoCD 상태 확인            | 앱 배포 상태 확인        |
| `make kind-dev-reset`        | 완전 리셋 (클러스터 재생성) | 심각한 문제, 처음부터    |
| `make kind-dev-clean`        | 클러스터 삭제 (데이터 보존) | 임시 정리                |
| `make argo-reset-apps`       | ArgoCD 앱만 리셋            | 앱 배포 문제 시          |

---

## 상세 사용법

### 1. `make kind-dev-setup` - 전체 환경 설정

```bash
make kind-dev-setup
```

**수행 내용:**

1. Kind 클러스터 생성 (3노드)
2. Istio Sidecar 모드 설치
3. Gateway API + HTTPRoute 설정
4. **PostgreSQL/Redis 클러스터 내부 배포** (hostPath로 데이터 영속화)
5. ArgoCD 설치
6. Git 레포 등록 (GitHub 토큰 필요)
7. 모든 Dev 앱 배포

**소요 시간:** 약 5-10분

**데이터 저장 위치:**
- 호스트: `/home/wealist-oranges/wealist-project-data/db_data/`
- 컨테이너(Kind 노드): `/data/db_data/`

---

### 2. `make kind-dev-rbac` - 팀원 RBAC 설정

```bash
make kind-dev-rbac
```

**수행 내용:**
1. `team-developer` ServiceAccount 생성
2. Role + RoleBinding 생성 (wealist-dev 네임스페이스용)
3. ClusterRole + ClusterRoleBinding 생성 (노드/네임스페이스 조회용)

**팀원 권한:**
- wealist-dev 네임스페이스만 접근 가능
- Pod 조회, 로그, exec 가능
- Pod 생성/삭제/수정 불가
- 다른 네임스페이스 접근 불가
- Docker 명령어 접근 불가 (Linux 사용자 권한으로 별도 제어)

---

### 3. `make kind-dev-kubeconfig` - 팀원 kubeconfig 생성

```bash
make kind-dev-kubeconfig USERNAME=team-oranges
```

**수행 내용:**
1. ServiceAccount 토큰 Secret 생성
2. 제한된 권한의 kubeconfig 파일 생성
3. 출력 위치: `./team-oranges-kubeconfig.yaml`

**팀원에게 전달:**
```bash
# 팀원이 사용할 kubeconfig
export KUBECONFIG=/path/to/team-oranges-kubeconfig.yaml
kubectl get pods -n wealist-dev
```

---

### 4. `make kind-dev-env-status` - 환경 상태 확인

```bash
make kind-dev-env-status
```

**확인 내용:**
- Kind 클러스터 상태
- PostgreSQL Pod 상태 (클러스터 내부)
- Redis Pod 상태 (클러스터 내부)
- 접속 정보 표시

---

### 5. `make kind-dev-reset` - 완전 리셋

```bash
make kind-dev-reset
```

**사용 상황:**
- Kind 설정 변경 필요 (포트, 노드 수)
- Istio/CNI 문제
- 처음부터 다시 시작하고 싶을 때

**동작:**
1. 확인 프롬프트 (y/N)
2. Kind 클러스터 삭제
3. `make kind-dev-setup` 재실행

**데이터 보존:**
- DB 데이터는 `wealist-project-data/db_data/`에 보존됨
- 클러스터 재생성 후 기존 데이터 자동 복원

---

### 6. `make kind-dev-clean` - 클러스터 삭제

```bash
make kind-dev-clean
```

**사용 상황:**
- 클러스터만 임시 정리
- 리소스 확보 필요
- 나중에 재생성 예정

**보존 항목:**
- DB 데이터 (`wealist-project-data/db_data/`)

---

### 7. `make argo-reset-apps` - ArgoCD 앱만 리셋

```bash
make argo-reset-apps
```

**사용 상황:**
- Pod가 CrashLoopBackOff
- ArgoCD 앱이 OutOfSync
- Git 변경사항이 반영 안됨

**동작:**
1. 모든 ArgoCD Application 삭제
2. Git에서 다시 읽어서 재생성
3. 클러스터는 유지됨 (Istio, ArgoCD 그대로)

---

## 아키텍처

### GitOps 구조

```
argo-develop 브랜치 (Git)
        ↓ ArgoCD가 감시
Kind 클러스터 (wealist-oranges)
        ↓ 자동 배포
Services, Infrastructure, Monitoring
```

### DB 아키텍처 (클러스터 내부)

```
Kind 노드 (control-plane, worker)
    └── /data (hostPath)
          ├── db_data/
          │     ├── postgres/  ← PostgreSQL 데이터
          │     └── redis/     ← Redis 데이터
          ├── prometheus/
          ├── grafana/
          └── loki/

호스트 (wealist-oranges)
    └── /home/wealist-oranges/wealist-project-data/
          ├── db_data/
          │     ├── postgres/  ← 영속 데이터 (Kind extraMounts로 연결)
          │     └── redis/
          ├── prometheus/
          ├── grafana/
          └── loki/
```

### 포트 매핑 (oranges 전용 대역)

| 용도              | Host Port | Container Port |
| ----------------- | --------- | -------------- |
| Istio Gateway HTTP | 9080      | 30080          |
| Istio Gateway HTTPS| 9443      | 30443          |

**포트 격리:**
- wonny 전용: 8000-8999
- oranges 전용: 9000-9999

---

## 접속 정보

| 서비스     | URL                                             |
| ---------- | ----------------------------------------------- |
| ArgoCD     | http://localhost:9080/api/argo                  |
| Grafana    | http://localhost:9080/api/monitoring/grafana    |
| Prometheus | http://localhost:9080/api/monitoring/prometheus |
| Kiali      | http://localhost:9080/api/monitoring/kiali      |
| Jaeger     | http://localhost:9080/api/monitoring/jaeger     |

**ArgoCD 로그인:**
```bash
# Username: admin
# Password 확인
kubectl -n argocd get secret argocd-initial-admin-secret \
  -o jsonpath='{.data.password}' | base64 -d
```

---

## 문제 해결

### Q: Pod가 CrashLoopBackOff

```bash
# 1. 로그 확인
kubectl logs -n wealist-dev <pod-name> --tail=50

# 2. 앱 리셋
make argo-reset-apps

# 3. 그래도 안되면 완전 리셋
make kind-dev-reset
```

### Q: DB 연결 실패 (connection refused)

```bash
# 1. PostgreSQL/Redis Pod 상태 확인
kubectl get pods -n wealist-dev -l app=postgres
kubectl get pods -n wealist-dev -l app=redis

# 2. Pod 로그 확인
kubectl logs -n wealist-dev -l app=postgres
kubectl logs -n wealist-dev -l app=redis

# 3. hostPath 마운트 확인
kubectl describe pod -n wealist-dev -l app=postgres | grep -A5 Volumes
```

### Q: ArgoCD 앱이 Unknown 상태

```bash
# 원인 확인
kubectl describe application <앱이름> -n argocd | tail -30

# 흔한 원인: repo not permitted
kubectl apply -f k8s/argocd/apps/dev/project.yaml
```

### Q: 팀원이 접근 불가

```bash
# 1. ServiceAccount 확인
kubectl get serviceaccount team-developer -n wealist-dev

# 2. RBAC 재적용
make kind-dev-rbac

# 3. kubeconfig 재생성
make kind-dev-kubeconfig USERNAME=<팀원이름>
```

---

## 팀원 격리 설명

### Kubernetes RBAC 격리
- `make kind-dev-kubeconfig`로 생성한 kubeconfig 사용
- wealist-dev 네임스페이스만 접근 가능
- 읽기 + 디버그 권한만 부여 (생성/삭제 불가)

### Docker 격리
- Linux 사용자 권한으로 별도 제어
- 팀원 계정이 `docker` 그룹에 없으면 Docker 명령어 실행 불가

```bash
# 팀원이 docker 그룹에 있는지 확인
groups team-oranges

# docker 그룹에서 제거 (필요시)
sudo gpasswd -d team-oranges docker
```

---

## 관련 문서

- [KIND_DEV_GUIDE.md](./KIND_DEV_GUIDE.md) - Dev 환경 상세 가이드
- [TROUBLESHOOTING-KIND-SETUP.md](./TROUBLESHOOTING-KIND-SETUP.md) - 문제 해결
- [CONFIGURATION.md](./CONFIGURATION.md) - 설정 가이드
