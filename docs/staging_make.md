# Staging 환경 Make 명령어 가이드

> Kind 클러스터 기반 Staging 환경 (ArgoCD + ECR + Istio)

## 목차
- [개요](#개요)
- [명령어 요약](#명령어-요약)
- [상세 사용법](#상세-사용법)
- [문제 해결](#문제-해결)

---

## 개요

### GitOps 구조
```
argo-develop 브랜치 (Git)
        ↓ ArgoCD가 감시
Kind 클러스터 (로컬)
        ↓ 자동 배포
Services, Infrastructure, Monitoring
```

### 핵심 개념
- **Git = Source of Truth**: 모든 설정은 Git에 저장
- **ArgoCD selfHeal**: 수동 변경해도 Git 상태로 자동 복원
- **클러스터 삭제해도 복원 가능**: Git에서 다시 읽어서 재생성

---

## 명령어 요약

| 명령어 | 용도 | 언제 사용? |
|--------|------|-----------|
| `make kind-staging-setup` | 전체 셋업 | 처음 환경 구축 |
| `make kind-staging-reset` | 완전 리셋 | 심각한 문제, Kind 설정 변경 |
| `make kind-staging-clean` | 클러스터 삭제만 | 임시 정리 |
| `make argo-reset-apps` | ArgoCD 앱 리셋 | 앱 배포 이상 (가장 자주 사용) |
| `make argo-status` | 상태 확인 | 현재 상태 확인 |
| `make status ENV=staging` | Pod 상태 | 서비스 상태 확인 |

---

## 상세 사용법

### 1. 처음 환경 구축

```bash
make kind-staging-setup
```

**수행 내용:**
1. Kind 클러스터 생성 (3노드)
2. Istio Ambient 모드 설치
3. Gateway API + HTTPRoute 설정
4. ArgoCD 설치
5. Git 레포 등록 (GitHub 토큰 필요)
6. 모든 Staging 앱 배포

**소요 시간:** 약 3-5분

---

### 2. 앱 배포 문제 시 (가장 자주 사용)

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

### 3. 완전 리셋 (클러스터 포함)

```bash
make kind-staging-reset
```

**사용 상황:**
- Kind 설정 변경 필요 (포트, 노드 수)
- Istio/CNI 문제
- 모든게 꼬여서 처음부터 시작하고 싶을 때

**동작:**
1. 확인 프롬프트 (y/N)
2. Kind 클러스터 삭제
3. 로컬 파일 변경 정리 (`git checkout -- .`)
4. `make kind-staging-setup` 실행

---

### 4. 클러스터만 삭제

```bash
make kind-staging-clean
```

**사용 상황:**
- 클러스터 임시 정리
- 리소스 확보 필요
- 나중에 다시 생성 예정

---

### 5. 상태 확인

```bash
# ArgoCD 전체 상태
make argo-status

# Pod 상태
make status ENV=staging

# ArgoCD 앱 목록
kubectl get applications -n argocd

# 특정 앱 상세
kubectl describe application <앱이름> -n argocd
```

---

## argo-status 출력 설명

```
📦 ArgoCD Pods: (ArgoCD 시스템 컴포넌트)
  Running: 7 pods
  → ArgoCD 자체 Pod (server, repo-server, redis, controller 등)

🔐 Sealed Secrets: (암호화 Secret용 컨트롤러)
  Controller: 0 pod(s)
  → Bitnami SealedSecrets 컨트롤러 (현재 미사용)

🎯 Applications: (ArgoCD가 관리하는 앱)
  Total: 14
  Synced: 12 (Git 동기화 완료)
  → OutOfSync = Git과 클러스터 상태 불일치

🔒 SealedSecrets: (암호화된 Secret, Git 저장 가능)
  Total: 0
  → kubeseal로 암호화된 Secret (현재 미사용)

🗝️  Secrets: (일반 Secret, 암호화 안됨)
  Total: 5
  → base64 인코딩만 됨, Git 저장 비권장
```

---

## 문제 해결

### Q: Pod가 CrashLoopBackOff

```bash
# 1. 로그 확인
kubectl logs -n wealist-staging <pod-name> --tail=50

# 2. 앱 리셋
make argo-reset-apps

# 3. 그래도 안되면 완전 리셋
make kind-staging-reset
```

### Q: ArgoCD 앱이 Unknown 상태

```bash
# 원인 확인
kubectl describe application <앱이름> -n argocd | tail -30

# 흔한 원인: repo not permitted
# 해결: project.yaml에 sourceRepos 확인
kubectl apply -f k8s/argocd/apps/staging/project.yaml
```

### Q: DB 연결 실패 (connection refused)

```bash
# 1. PostgreSQL 서비스 확인
sudo systemctl status postgresql

# 2. pg_hba.conf에 Kind 네트워크 허용 확인
# 10.244.0.0/16, 172.17.0.1 등

# 3. ConfigMap에 DB_HOST 확인
kubectl get configmap wealist-shared-config -n wealist-staging -o yaml | grep DB_HOST
```

### Q: Git 변경이 반영 안됨

```bash
# 1. argo-develop 브랜치에 push 했는지 확인
git log origin/argo-develop --oneline -5

# 2. ArgoCD 강제 refresh
kubectl patch application <앱이름> -n argocd \
  --type merge -p '{"metadata":{"annotations":{"argocd.argoproj.io/refresh":"hard"}}}'
```

---

## 브랜치 전략

```
작업 브랜치 (예: claude/xxx)
        ↓ merge
argo-develop (ArgoCD가 바라봄)
        ↓ ArgoCD sync
Kind 클러스터
```

**작업 흐름:**
```bash
# 1. 작업 브랜치에서 개발
git checkout claude/argocd-auto-deploy-dev-tWRSt
# ... 작업 ...
git add . && git commit -m "feat: xxx"
git push

# 2. argo-develop에 반영
git checkout argo-develop
git merge claude/argocd-auto-deploy-dev-tWRSt
git push origin argo-develop

# 3. ArgoCD가 자동으로 배포 (selfHeal)
```

---

## 접속 정보

| 서비스 | URL |
|--------|-----|
| ArgoCD | http://localhost:8080/api/argo |
| Grafana | http://localhost:8080/api/monitoring/grafana |
| Prometheus | http://localhost:8080/api/monitoring/prometheus |
| Kiali | http://localhost:8080/api/monitoring/kiali |
| Jaeger | http://localhost:8080/api/monitoring/jaeger |

**ArgoCD 로그인:**
```bash
# 비밀번호 확인
kubectl -n argocd get secret argocd-initial-admin-secret \
  -o jsonpath='{.data.password}' | base64 -d
# Username: admin
```

---

## 관련 문서

- [KIND_DEV_GUIDE.md](./KIND_DEV_GUIDE.md) - Dev 환경 가이드
- [TROUBLESHOOTING-KIND-SETUP.md](./TROUBLESHOOTING-KIND-SETUP.md) - 문제 해결
- [CONFIGURATION.md](./CONFIGURATION.md) - 설정 가이드
