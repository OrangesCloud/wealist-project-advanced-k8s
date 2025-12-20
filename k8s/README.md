# wealist-argo-helm

이 프로젝트는 Helm Chart와 ArgoCD를 사용하여 Wealist 서비스를 배포하는 저장소입니다.

## 📁 프로젝트 구조

- **charts/** - 각 서비스별 개별 Helm Chart
- **environments/** - 환경별 설정 파일 (local, dev, staging)
  - Chart 템플릿은 변경하지 않고, 3개 브랜치에서 공통으로 사용합니다
  - 환경별로 다른 값들만 이 디렉토리에서 관리합니다

> ⚠️ **중요**: 배포 전에 각 환경의 `secret` 파일에 환경변수 값을 반드시 설정해야 합니다.

---

## 🚀 로컬 환경 (local-kind) 배포 가이드

### 1. 사전 준비

먼저 각 서비스의 Docker 이미지를 빌드해야 합니다:
- 목표: `localhost:5001/service:latest` 형식의 이미지 생성
- 각 서비스 디렉토리에서 Docker 이미지 빌드를 수행하세요

### 2. 로컬 레지스트리 설정

다음 스크립트들을 순서대로 실행합니다:

```bash
# k8s/installShell/ 디렉토리에서 실행
./00-*.sh
./01-*.sh
./02-*.sh
```

### 3. 이미지 업로드 확인

로컬 레지스트리에 이미지가 정상적으로 업로드되었는지 확인:

```bash
curl -s http://localhost:5001/v2/_catalog | jq
```

### 4. Helm 배포

```bash
make helm-install-all ENV=local-kind
```

### 5. ArgoCD 배포

```bash
./k8s/argocd/scripts/deploy-argocd.sh
```

✅ 로컬 환경 배포 완료!

---

## 🌐 Dev 환경 배포 가이드

### 1. Sealed Secrets 키 설정

Dev 환경의 암호화된 시크릿을 복호화하기 위한 키가 필요합니다.

#### 키 파일 준비
- **키 이름**: `sealed-secrets-dev-20251218-152119.key`
- **저장 위치**: `k8s/argocd/scripts/sealed-secrets-dev-20251218-152119.key`
- **키 복사**: xaczx 폴더에서 해당 키 파일을 복사하여 위 경로에 생성

### 2. GitHub Access Token 발급

ArgoCD가 GitHub 저장소에 접근하기 위한 토큰이 필요합니다.

#### 토큰 생성 방법

1. GitHub 계정 → **Settings** 이동
2. **Developer Settings** → **Personal access tokens** → **Tokens (classic)**
3. **Generate new token** 클릭
4. 다음 권한을 선택:
   - ✅ `read:org` - 조직 정보 읽기
   - ✅ `repo` - 저장소 전체 접근
   - ✅ `workflow` - GitHub Actions 워크플로우 접근
   - ✅ 'write:package - 이미지 패키지 읽기 권한
5. 생성된 토큰 값을 복사 (한 번만 표시됩니다!)

### 3. 배포 실행

```bash
make all-simple
```

실행 중 다음 정보를 입력하라는 프롬프트가 나타납니다:
- **GitHub 계정 이름** (username)
- **GitHub Access Token** (위에서 생성한 토큰)

✅ Dev 환경 배포 완료!


### postgre 설정을 하기 위해
```
#k8s/helm/charts/wealist-infrastructure/templates/postgres/secret.yaml 밑에꺼를 생성해야됨
{{- if .Values.postgres.enabled }}
apiVersion: v1
kind: Secret
metadata:
  name: postgres-secret
  namespace: {{ .Values.global.namespace }}
  labels:
    {{- include "wealist-infrastructure.labels" . | nindent 4 }}
    app.kubernetes.io/component: postgres
type: Opaque
data:
  POSTGRES_PASSWORD: {{ .Values.postgres.config.superuserPassword | b64enc | quote }}
{{- end }}
```
---

## 💡 주요 참고사항

- Chart 템플릿은 수정하지 말고, `environments/` 디렉토리의 값만 수정하세요
- 환경별 시크릿 설정을 잊지 마세요
- GitHub Access Token은 안전하게 보관하세요