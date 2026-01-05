#!/bin/bash
# =============================================================================
# Kind 클러스터 + Istio Sidecar 설정 (dev 환경)
# =============================================================================
# - 레지스트리: AWS ECR
# - Istio Sidecar: Service Mesh with Envoy sidecar proxy
# - Gateway API: Kubernetes 표준 (NodePort 30080 → hostPort 8080)
# - ArgoCD: GitOps 배포
# - 포트 대역: oranges 전용 9000-9999
# - 데이터 저장: ${WEALIST_DATA_PATH}/db_data

set -e

CLUSTER_NAME="wealist"
ISTIO_VERSION="1.28.2"
GATEWAY_API_VERSION="v1.2.0"
AWS_REGION="ap-northeast-2"
NAMESPACE="wealist-dev"

# =============================================================================
# 환경 변수 설정 (wealist-oranges 전용)
# =============================================================================
export WEALIST_DATA_PATH="${WEALIST_DATA_PATH:-/home/wealist-oranges/data}"
export POSTGRES_USER="${POSTGRES_USER:-wealist}"
export POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-wealist-dev-2024}"
export POSTGRES_DB="${POSTGRES_DB:-wealist}"

# 스크립트 디렉토리 및 kind-config.yaml 경로
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HELM_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
KIND_CONFIG_TEMPLATE="${SCRIPT_DIR}/kind-config.yaml"
KIND_CONFIG_RENDERED="/tmp/kind-config-rendered.yaml"
DOCKER_COMPOSE_DB="${SCRIPT_DIR}/../../docker/dev/docker-compose.dev-db.yaml"

echo "🚀 Kind 클러스터 + Istio Sidecar 설정 (dev - AWS ECR)"
echo "   - Istio: ${ISTIO_VERSION} (Sidecar mode)"
echo "   - Gateway API: ${GATEWAY_API_VERSION}"
echo "   - Registry: AWS ECR (ap-northeast-2)"
echo "   - Namespace: ${NAMESPACE}"
echo "   - Data Path: ${WEALIST_DATA_PATH}"
echo "   - Ports: 9080 (HTTP), 9443 (HTTPS)"
echo ""

# =============================================================================
# 1. 데이터 디렉토리 생성
# =============================================================================
echo "📁 데이터 디렉토리 생성 중..."
mkdir -p "${WEALIST_DATA_PATH}/db_data/postgres"
mkdir -p "${WEALIST_DATA_PATH}/db_data/redis"
mkdir -p "${WEALIST_DATA_PATH}/minio"
mkdir -p "${WEALIST_DATA_PATH}/prometheus"
mkdir -p "${WEALIST_DATA_PATH}/grafana"
mkdir -p "${WEALIST_DATA_PATH}/loki"
echo "✅ 데이터 디렉토리 생성 완료: ${WEALIST_DATA_PATH}"

# =============================================================================
# 1-1. Prometheus/Grafana/Loki 권한 설정 (hostPath 사용 시 필수)
# =============================================================================
# Prometheus: UID 65534 (nobody), Grafana: UID 472, Loki: UID 10001
echo ""
echo "🔐 모니터링 디렉토리 권한 설정이 필요합니다."
echo "   - Prometheus: UID 65534 (nobody)"
echo "   - Grafana: UID 472"
echo "   - Loki: UID 10001"
echo ""
read -p "sudo로 권한을 설정할까요? (Y/n): " SETUP_PERMS
SETUP_PERMS=${SETUP_PERMS:-Y}

if [[ "$SETUP_PERMS" =~ ^[Yy]$ ]]; then
    echo "⏳ 모니터링 디렉토리 권한 설정 중..."

    # Prometheus (UID 65534)
    sudo chown -R 65534:65534 "${WEALIST_DATA_PATH}/prometheus"
    sudo chmod 770 "${WEALIST_DATA_PATH}/prometheus"

    # Grafana (UID 472)
    sudo chown -R 472:472 "${WEALIST_DATA_PATH}/grafana"
    sudo chmod 770 "${WEALIST_DATA_PATH}/grafana"

    # Loki (UID 10001)
    sudo chown -R 10001:10001 "${WEALIST_DATA_PATH}/loki"
    sudo chmod 770 "${WEALIST_DATA_PATH}/loki"

    echo "✅ 모니터링 디렉토리 권한 설정 완료"
else
    echo "⚠️  권한 설정 건너뜀. 나중에 수동으로 설정하세요:"
    echo "   sudo chown -R 65534:65534 ${WEALIST_DATA_PATH}/prometheus"
    echo "   sudo chown -R 472:472 ${WEALIST_DATA_PATH}/grafana"
    echo "   sudo chown -R 10001:10001 ${WEALIST_DATA_PATH}/loki"
fi

# =============================================================================
# 2. Kind 설정 파일 렌더링 (환경변수 치환)
# =============================================================================
echo "📝 Kind 설정 파일 렌더링 중..."
envsubst < "${KIND_CONFIG_TEMPLATE}" > "${KIND_CONFIG_RENDERED}"
echo "✅ Kind 설정 파일 렌더링 완료: ${KIND_CONFIG_RENDERED}"

# Kind 설정 파일 확인
if [ ! -f "${KIND_CONFIG_RENDERED}" ]; then
    echo "❌ kind-config.yaml 렌더링 실패"
    exit 1
fi

# =============================================================================
# 3. AWS 로그인 확인
# =============================================================================
echo "🔐 AWS 로그인 확인 중..."
if ! aws sts get-caller-identity &>/dev/null; then
    echo "❌ AWS 로그인이 필요합니다!"
    echo ""
    echo "   다음 중 하나로 로그인하세요:"
    echo ""
    echo "   1. AWS SSO 로그인:"
    echo "      aws sso login --profile <your-profile>"
    echo ""
    echo "   2. AWS 자격증명 설정:"
    echo "      aws configure"
    echo ""
    exit 1
fi

AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
ECR_REGISTRY="${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com"
echo "✅ AWS 로그인 확인 완료"
echo "   - Account: ${AWS_ACCOUNT_ID}"
echo "   - ECR: ${ECR_REGISTRY}"
echo ""

# =============================================================================
# 4. DB 안내 (클러스터 내부 Deployment로 배포됨)
# =============================================================================
echo "🐘 PostgreSQL + Redis는 클러스터 내부에서 실행됩니다."
echo "   - 데이터 저장: ${WEALIST_DATA_PATH}/db_data/postgres, redis"
echo "   - ArgoCD가 wealist-infrastructure 차트를 배포하면 자동 시작됩니다."
echo ""

# =============================================================================
# 5. 기존 클러스터 삭제 (있으면)
# =============================================================================
if kind get clusters 2>/dev/null | grep -q "^${CLUSTER_NAME}$"; then
    echo "기존 클러스터 삭제 중..."
    kind delete cluster --name "$CLUSTER_NAME"
fi

# =============================================================================
# 6. Kind 클러스터 생성
# =============================================================================
echo "🚀 Kind 클러스터 생성 중..."
kind create cluster --name "$CLUSTER_NAME" --config "${KIND_CONFIG_RENDERED}"

# =============================================================================
# 7. Gateway API CRDs 설치
# =============================================================================
echo "⏳ Gateway API CRDs 설치 중..."
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/${GATEWAY_API_VERSION}/standard-install.yaml
echo "✅ Gateway API CRDs 설치 완료"

# 4. Istio Sidecar 모드 설치
echo "⏳ Istio Sidecar 모드 설치 중..."

# istioctl 설치 확인 및 경로 설정
ISTIOCTL=""
if command -v istioctl &> /dev/null; then
    ISTIOCTL="istioctl"
    echo "✅ istioctl 발견: $(which istioctl)"
elif [ -f "${HELM_DIR}/../../istio-${ISTIO_VERSION}/bin/istioctl" ]; then
    ISTIOCTL="${HELM_DIR}/../../istio-${ISTIO_VERSION}/bin/istioctl"
    echo "✅ 로컬 istioctl 사용: ${ISTIOCTL}"
elif [ -f "./istio-${ISTIO_VERSION}/bin/istioctl" ]; then
    ISTIOCTL="./istio-${ISTIO_VERSION}/bin/istioctl"
    echo "✅ 로컬 istioctl 사용: ${ISTIOCTL}"
else
    echo "⚠️  istioctl이 설치되어 있지 않습니다. 자동 설치 중..."
    echo ""
    curl -L https://istio.io/downloadIstio | ISTIO_VERSION=${ISTIO_VERSION} sh -
    ISTIOCTL="./istio-${ISTIO_VERSION}/bin/istioctl"
    if [ -f "${ISTIOCTL}" ]; then
        echo "✅ istioctl 설치 완료: ${ISTIOCTL}"
    else
        echo "❌ istioctl 설치 실패"
        exit 1
    fi
fi

# Istio default 프로필 설치 (Sidecar mode)
${ISTIOCTL} install --set profile=default --skip-confirmation

echo "⏳ Istio 컴포넌트 준비 대기 중..."
kubectl wait --namespace istio-system \
  --for=condition=ready pod \
  --selector=app=istiod \
  --timeout=120s || echo "WARNING: istiod not ready yet"

echo "✅ Istio Sidecar 설치 완료"

# NOTE: Kiali, Jaeger는 ArgoCD가 istio-addons 차트로 배포합니다.
# 수동 설치하면 충돌이 발생하므로 여기서는 설치하지 않습니다.

# 5. Istio Native Gateway 설치 (VirtualService용)
# NOTE: Kubernetes Gateway API가 아닌 Istio Native Gateway 사용
#       - VirtualService는 networking.istio.io/v1 Gateway 필요
#       - istio install --profile=default가 생성한 istio-ingressgateway와 연결
echo "⏳ Istio Native Gateway 설치 중..."
kubectl apply -f - <<EOF
apiVersion: networking.istio.io/v1
kind: Gateway
metadata:
  name: istio-ingressgateway
  namespace: istio-system
spec:
  selector:
    istio: ingressgateway
  servers:
  - port:
      number: 80
      name: http
      protocol: HTTP
    hosts:
    - "*"
  - port:
      number: 443
      name: https
      protocol: HTTPS
    hosts:
    - "*"
    tls:
      mode: PASSTHROUGH
EOF

echo "⏳ Istio Ingressgateway Pod 준비 대기 중..."
kubectl wait --namespace istio-system \
  --for=condition=ready pod \
  --selector=app=istio-ingressgateway \
  --timeout=120s || echo "WARNING: Istio gateway not ready yet"

# 6. Istio Gateway NodePort 서비스 생성 (Kind hostPort 30080 연결)
# NOTE: 기본 istio-ingressgateway는 LoadBalancer 타입
#       Kind에서는 NodePort 30080이 hostPort 80/8080에 매핑됨
echo "⚙️ Istio Gateway NodePort 서비스 생성 중..."
kubectl apply -f - <<EOF
apiVersion: v1
kind: Service
metadata:
  name: istio-ingressgateway-nodeport
  namespace: istio-system
  labels:
    app: istio-ingressgateway
    istio: ingressgateway
spec:
  type: NodePort
  selector:
    app: istio-ingressgateway
    istio: ingressgateway
  ports:
  - name: http
    port: 80
    targetPort: 8080
    nodePort: 30080
  - name: https
    port: 443
    targetPort: 8443
    nodePort: 30443
EOF

echo "📋 Gateway 서비스 상태:"
kubectl get svc -n istio-system -l istio=ingressgateway

echo "✅ Istio Gateway 설정 완료"

# 7. Argo Rollouts 설치 (Progressive Delivery)
echo "⏳ Argo Rollouts 설치 중..."
kubectl create namespace argo-rollouts 2>/dev/null || true
# Argo Rollouts v1.8.3 (버전 고정 - 재현성 보장)
ARGO_ROLLOUTS_VERSION="v1.8.3"
kubectl apply -n argo-rollouts -f https://github.com/argoproj/argo-rollouts/releases/download/${ARGO_ROLLOUTS_VERSION}/install.yaml

echo "⏳ Argo Rollouts 준비 대기 중..."
kubectl wait --namespace argo-rollouts \
  --for=condition=available deployment/argo-rollouts \
  --timeout=120s || echo "WARNING: Argo Rollouts not ready yet"

echo "✅ Argo Rollouts 설치 완료"

# 8. 애플리케이션 네임스페이스 생성 (Sidecar injection 라벨 포함)
echo "📦 ${NAMESPACE} 네임스페이스 생성 (Sidecar mode)..."
kubectl create namespace ${NAMESPACE} 2>/dev/null || true
kubectl label namespace ${NAMESPACE} istio-injection=enabled --overwrite

# Git 정보 라벨 추가
GIT_REPO=$(git config --get remote.origin.url 2>/dev/null | sed 's/.*github.com[:/]\(.*\)\.git/\1/' || echo "unknown")
GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_USER=$(git config --get user.name 2>/dev/null || echo "unknown")
GIT_EMAIL=$(git config --get user.email 2>/dev/null || echo "unknown")
DEPLOY_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

kubectl annotate namespace ${NAMESPACE} \
  "wealist.io/git-repo=${GIT_REPO}" \
  "wealist.io/git-branch=${GIT_BRANCH}" \
  "wealist.io/git-commit=${GIT_COMMIT}" \
  "wealist.io/deployed-by=${GIT_USER}" \
  "wealist.io/deployed-by-email=${GIT_EMAIL}" \
  "wealist.io/deploy-time=${DEPLOY_TIME}" \
  --overwrite

echo "✅ 네임스페이스에 Sidecar injection + Git 정보 라벨 적용 완료"

# 9. ECR 인증 Secret 생성
echo "🔐 ECR 인증 Secret 설정 중..."
ECR_PASSWORD=$(aws ecr get-login-password --region ${AWS_REGION})

if [ -n "${ECR_PASSWORD}" ]; then
    kubectl delete secret ecr-secret -n ${NAMESPACE} 2>/dev/null || true
    kubectl create secret docker-registry ecr-secret \
        --docker-server="${ECR_REGISTRY}" \
        --docker-username=AWS \
        --docker-password="${ECR_PASSWORD}" \
        -n ${NAMESPACE}
    echo "✅ ECR Secret 생성 완료"
else
    echo "❌ ECR 로그인 실패. AWS 자격증명을 확인하세요."
    exit 1
fi

# =============================================================================
# 12. External Secrets Operator (ESO) 설치 및 설정
# =============================================================================
echo "🔐 External Secrets Operator (ESO) 설치 중..."

kubectl create namespace external-secrets 2>/dev/null || true

helm repo add external-secrets https://charts.external-secrets.io 2>/dev/null || true
helm repo update

helm upgrade --install external-secrets external-secrets/external-secrets \
    --namespace external-secrets \
    --set installCRDs=true \
    --wait --timeout 5m
echo "✅ External Secrets Operator 설치 완료"

echo "⏳ ESO CRDs 준비 대기 중..."
sleep 5
kubectl wait --for=condition=established --timeout=60s crd/clustersecretstores.external-secrets.io 2>/dev/null || true
kubectl wait --for=condition=established --timeout=60s crd/externalsecrets.external-secrets.io 2>/dev/null || true
echo "✅ ESO CRDs 준비 완료"

# AWS 자격증명 Secret 생성 (ESO가 AWS Secrets Manager 접근용)
echo "🔐 AWS 자격증명 Secret 생성 중..."

AWS_ACCESS_KEY="${AWS_ACCESS_KEY_ID:-}"
AWS_SECRET_KEY="${AWS_SECRET_ACCESS_KEY:-}"

if [ -z "$AWS_ACCESS_KEY" ] || [ -z "$AWS_SECRET_KEY" ]; then
    echo "  → 환경변수에서 AWS 자격증명을 찾을 수 없어 AWS CLI에서 가져옵니다..."
    AWS_ACCESS_KEY=$(aws configure get aws_access_key_id 2>/dev/null || echo "")
    AWS_SECRET_KEY=$(aws configure get aws_secret_access_key 2>/dev/null || echo "")
fi

if [ -z "$AWS_ACCESS_KEY" ] || [ -z "$AWS_SECRET_KEY" ]; then
    echo ""
    echo "  AWS 자격증명이 필요합니다. (ESO가 AWS Secrets Manager 접근용)"
    read -p "  AWS Access Key ID: " AWS_ACCESS_KEY
    if [ -n "$AWS_ACCESS_KEY" ]; then
        read -sp "  AWS Secret Access Key: " AWS_SECRET_KEY
        echo ""
    fi
fi

if [ -z "$AWS_ACCESS_KEY" ] || [ -z "$AWS_SECRET_KEY" ]; then
    echo ""
    echo "⚠️  AWS 자격증명이 설정되지 않았습니다."
    echo "   ESO 없이 진행합니다. 나중에 make eso-setup-aws로 설정할 수 있습니다."
else
    kubectl delete secret aws-credentials -n external-secrets 2>/dev/null || true
    kubectl create secret generic aws-credentials \
        --from-literal=access-key="${AWS_ACCESS_KEY}" \
        --from-literal=secret-access-key="${AWS_SECRET_KEY}" \
        -n external-secrets
    echo "✅ AWS 자격증명 Secret 생성 완료"

    echo "🔐 ClusterSecretStore 적용 중..."
    kubectl apply -f "${SCRIPT_DIR}/../../../argocd/base/external-secrets/dev/cluster-secret-store-dev.yaml"
    echo "✅ ClusterSecretStore 적용 완료"

    echo "🔐 ExternalSecret 적용 중..."
    kubectl delete secret wealist-shared-secret -n ${NAMESPACE} 2>/dev/null || true
    kubectl apply -f "${SCRIPT_DIR}/../../../argocd/base/external-secrets/dev/external-secret-shared.yaml"
    echo "✅ ExternalSecret 적용 완료"

    echo "⏳ ExternalSecret sync 대기 중..."
    sleep 5
    kubectl get externalsecret wealist-shared-secret -n ${NAMESPACE} 2>/dev/null || echo "  (ArgoCD가 나중에 생성)"
fi

echo "✅ ESO 설정 완료"

# =============================================================================
# 13. dev.yaml 업데이트 (AWS Account ID만)
# =============================================================================
echo "🔧 dev.yaml 설정 확인 중..."
DEV_YAML="${HELM_DIR}/environments/dev.yaml"

# AWS Account ID 자동 업데이트
if [ -f "${DEV_YAML}" ] && grep -q "<AWS_ACCOUNT_ID>" "${DEV_YAML}" 2>/dev/null; then
    sed -i "s/<AWS_ACCOUNT_ID>/${AWS_ACCOUNT_ID}/g" "${DEV_YAML}"
    echo "✅ dev.yaml: AWS Account ID 업데이트 완료"
fi
echo "   DB_HOST: postgres (클러스터 내부 Service)"
echo "   REDIS_HOST: redis (클러스터 내부 Service)"

# =============================================================================
# 14. ArgoCD 설치
# =============================================================================
echo ""
echo "🔧 ArgoCD 설치 중..."

kubectl create namespace argocd 2>/dev/null || true

helm repo add argo https://argoproj.github.io/argo-helm 2>/dev/null || true
helm repo update

helm upgrade --install argocd argo/argo-cd \
    --namespace argocd \
    --set server.service.type=ClusterIP \
    --set configs.params."server\.insecure"=true \
    --set configs.params."server\.rootpath"=/api/argo \
    --set configs.params."server\.basehref"=/api/argo \
    --wait --timeout 5m

echo "⏳ ArgoCD 준비 대기 중..."
kubectl wait --for=condition=available --timeout=120s deployment/argocd-server -n argocd

ARGOCD_PASSWORD=$(kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" 2>/dev/null | base64 -d)
if [ -n "$ARGOCD_PASSWORD" ]; then
    echo "✅ ArgoCD 설치 완료"
    echo "   - URL: http://localhost:9080/api/argo"
    echo "   - Username: admin"
    echo "   - Password: ${ARGOCD_PASSWORD}"
else
    echo "✅ ArgoCD 설치 완료 (비밀번호는 이미 변경됨)"
fi

# =============================================================================
# 14-1. ArgoCD Google OAuth 설정
# =============================================================================
# 우선순위:
#   1. 환경변수 (GOOGLE_OAUTH_CLIENT_ID, GOOGLE_OAUTH_CLIENT_SECRET)
#   2. AWS Secrets Manager (wealist/dev/oauth/argocd)
#   3. CLI 입력
# =============================================================================
echo ""
echo "🔐 ArgoCD Google OAuth 설정 중..."

OAUTH_CLIENT_ID="${GOOGLE_OAUTH_CLIENT_ID:-}"
OAUTH_CLIENT_SECRET="${GOOGLE_OAUTH_CLIENT_SECRET:-}"

# 환경변수 없으면 AWS Secrets Manager에서 시도
if [ -z "$OAUTH_CLIENT_ID" ] || [ -z "$OAUTH_CLIENT_SECRET" ]; then
    echo "  → 환경변수 없음, AWS Secrets Manager에서 조회 중..."

    # AWS Secrets Manager에서 가져오기 시도
    OAUTH_SECRET=$(aws secretsmanager get-secret-value \
        --secret-id "wealist/dev/oauth/argocd" \
        --region ${AWS_REGION} \
        --query SecretString \
        --output text 2>/dev/null || echo "")

    if [ -n "$OAUTH_SECRET" ]; then
        OAUTH_CLIENT_ID=$(echo "$OAUTH_SECRET" | jq -r '.client_id // empty' 2>/dev/null)
        OAUTH_CLIENT_SECRET=$(echo "$OAUTH_SECRET" | jq -r '.client_secret // empty' 2>/dev/null)
        if [ -n "$OAUTH_CLIENT_ID" ] && [ -n "$OAUTH_CLIENT_SECRET" ]; then
            echo "  ✅ AWS Secrets Manager에서 OAuth 자격증명 로드 완료"
        fi
    fi
fi

# 여전히 없으면 CLI 입력
if [ -z "$OAUTH_CLIENT_ID" ] || [ -z "$OAUTH_CLIENT_SECRET" ]; then
    echo ""
    echo "  Google OAuth 설정이 필요합니다."
    echo "  (Google Cloud Console → API 및 서비스 → 사용자 인증 정보)"
    echo ""
    read -p "  Google OAuth Client ID (Enter 건너뛰기): " OAUTH_CLIENT_ID
    if [ -n "$OAUTH_CLIENT_ID" ]; then
        read -p "  Google OAuth Client Secret: " OAUTH_CLIENT_SECRET
    fi
fi

# OAuth 설정 적용
if [ -n "$OAUTH_CLIENT_ID" ] && [ -n "$OAUTH_CLIENT_SECRET" ]; then
    echo "  → Google OAuth 설정 적용 중..."

    # Google OAuth Secret 추가
    kubectl patch secret argocd-secret -n argocd --type merge -p "{
      \"stringData\": {
        \"dex.google.clientSecret\": \"${OAUTH_CLIENT_SECRET}\"
      }
    }" 2>/dev/null || true

    # ArgoCD ConfigMap에 Dex config 추가
    DEX_CONFIG="connectors:
  - type: google
    id: google
    name: Google
    config:
      clientID: ${OAUTH_CLIENT_ID}
      clientSecret: \$dex.google.clientSecret
      redirectURI: https://dev.wealist.co.kr/api/argo/dex/callback"

    kubectl patch configmap argocd-cm -n argocd --type merge -p "$(cat <<EOF
{
  "data": {
    "url": "https://dev.wealist.co.kr/api/argo",
    "dex.config": $(echo "$DEX_CONFIG" | jq -Rs .)
  }
}
EOF
)"

    # ArgoCD 서버 재시작 (Dex 설정 적용)
    echo "⏳ ArgoCD 서버 재시작 중 (Google OAuth 적용)..."
    kubectl rollout restart deployment argocd-server argocd-dex-server -n argocd
    kubectl rollout status deployment argocd-server -n argocd --timeout=120s

    echo "✅ ArgoCD Google OAuth 설정 완료"
    echo "   - Google 로그인: https://dev.wealist.co.kr/api/argo"
else
    echo "⚠️  Google OAuth 설정 건너뜀 (admin 계정으로 로그인)"
fi

# ArgoCD RBAC 설정 (Google OAuth 사용자 권한) - OAuth 설정 후 적용해야 함
echo "🔐 ArgoCD RBAC 설정 적용 중..."
ARGOCD_RBAC="${SCRIPT_DIR}/../../../argocd/config/argocd-rbac-cm.yaml"
if [ -f "${ARGOCD_RBAC}" ]; then
    kubectl apply -f "${ARGOCD_RBAC}"
    echo "✅ ArgoCD RBAC 설정 완료 (관리자 이메일 등록됨)"
else
    echo "⚠️  ArgoCD RBAC 파일을 찾을 수 없습니다: ${ARGOCD_RBAC}"
fi

# =============================================================================
# 15. ReferenceGrant + HTTPRoute 즉시 적용 (ArgoCD 접근용)
# =============================================================================
echo ""
echo "🔐 ReferenceGrant 적용 중..."
REFERENCEGRANT="${SCRIPT_DIR}/../../../argocd/referencegrants/referencegrant-argocd.yaml"
if [ -f "${REFERENCEGRANT}" ]; then
    kubectl apply -f "${REFERENCEGRANT}"
    echo "✅ ReferenceGrant 적용 완료"
fi

# ArgoCD VirtualService 부트스트랩 (ArgoCD sync 전에 접근 가능하도록)
# NOTE: Istio Native Gateway + VirtualService 사용
echo "🔐 ArgoCD VirtualService 부트스트랩 적용 중..."
ARGOCD_VS="${SCRIPT_DIR}/../../../argocd/base/virtualservice-bootstrap.yaml"
if [ -f "${ARGOCD_VS}" ]; then
    kubectl apply -f "${ARGOCD_VS}"
    echo "✅ ArgoCD VirtualService 적용 완료 - /api/argo 라우팅 활성화"
else
    echo "⚠️  ArgoCD VirtualService 파일을 찾을 수 없습니다: ${ARGOCD_VS}"
fi

# =============================================================================
# 16. ArgoCD Root App 배포
# =============================================================================
echo ""
echo "🚀 ArgoCD Root App 배포 중..."

ROOT_APP="${SCRIPT_DIR}/../../../argocd/apps/dev/root-app.yaml"
if [ -f "${ROOT_APP}" ]; then
    kubectl apply -f "${ROOT_APP}"
    echo "✅ Root App 배포 완료"
else
    echo "⚠️  Root App 파일을 찾을 수 없습니다: ${ROOT_APP}"
fi

# =============================================================================
# 17. ArgoCD Discord 알림 설정
# =============================================================================
# 배포 성공/실패 시 Discord 채널로 알림 전송
# Webhook URL은 Discord 서버 설정 > 연동 > 웹후크에서 생성
echo ""
echo "🔔 ArgoCD Discord 알림 설정"
echo "   배포 성공/실패 시 Discord로 알림을 받을 수 있습니다."
echo ""

DISCORD_WEBHOOK_URL="${DISCORD_WEBHOOK_URL:-}"

# 환경변수 없으면 AWS Secrets Manager에서 시도
if [ -z "$DISCORD_WEBHOOK_URL" ]; then
    DISCORD_SECRET=$(aws secretsmanager get-secret-value \
        --secret-id "wealist/dev/discord/webhook" \
        --region ${AWS_REGION} \
        --query SecretString \
        --output text 2>/dev/null || echo "")

    if [ -n "$DISCORD_SECRET" ]; then
        DISCORD_WEBHOOK_URL=$(echo "$DISCORD_SECRET" | jq -r '.webhook_url // empty' 2>/dev/null)
        if [ -n "$DISCORD_WEBHOOK_URL" ]; then
            echo "  ✅ AWS Secrets Manager에서 Discord Webhook URL 로드 완료"
        fi
    fi
fi

# 여전히 없으면 CLI 입력
if [ -z "$DISCORD_WEBHOOK_URL" ]; then
    echo "  Discord Webhook URL을 입력하세요."
    echo "  (Discord 서버 설정 > 연동 > 웹후크에서 생성)"
    echo ""
    read -p "  Discord Webhook URL (Enter 건너뛰기): " DISCORD_WEBHOOK_URL
fi

# Discord 알림 설정 적용
if [ -n "$DISCORD_WEBHOOK_URL" ]; then
    echo "⏳ Discord 알림 설정 적용 중..."

    # ConfigMap 적용
    kubectl apply -f - <<'DISCORD_CM_EOF'
apiVersion: v1
kind: ConfigMap
metadata:
  name: argocd-notifications-cm
  namespace: argocd
data:
  context: |
    argocdUrl: https://dev.wealist.co.kr/api/argo

  service.webhook.discord: |
    url: $discord-webhook-url
    headers:
    - name: Content-Type
      value: application/json

  template.app-deployed: |
    webhook:
      discord:
        method: POST
        body: |
          {
            "embeds": [{
              "title": ":rocket: Dev 배포 완료",
              "color": 3066993,
              "fields": [
                {"name": "Application", "value": "{{.app.metadata.name}}", "inline": true},
                {"name": "Status", "value": "{{.app.status.health.status}}", "inline": true},
                {"name": "Sync", "value": "{{.app.status.sync.status}}", "inline": true},
                {"name": "Revision", "value": "{{.app.status.sync.revision | trunc 7}}", "inline": true}
              ],
              "timestamp": "{{.app.status.operationState.finishedAt}}"
            }]
          }

  template.app-sync-failed: |
    webhook:
      discord:
        method: POST
        body: |
          {
            "embeds": [{
              "title": ":x: Dev 배포 실패",
              "color": 15158332,
              "fields": [
                {"name": "Application", "value": "{{.app.metadata.name}}", "inline": true},
                {"name": "Error", "value": "{{.app.status.operationState.message | trunc 200}}", "inline": false}
              ],
              "timestamp": "{{.app.status.operationState.finishedAt}}"
            }]
          }

  template.app-health-degraded: |
    webhook:
      discord:
        method: POST
        body: |
          {
            "embeds": [{
              "title": ":warning: Dev 서비스 상태 이상",
              "color": 16744448,
              "fields": [
                {"name": "Application", "value": "{{.app.metadata.name}}", "inline": true},
                {"name": "Health", "value": "{{.app.status.health.status}}", "inline": true},
                {"name": "Message", "value": "{{.app.status.health.message | default `No message` | trunc 200}}", "inline": false}
              ]
            }]
          }

  trigger.on-deployed: |
    - description: 배포 완료 시 알림
      send:
      - app-deployed
      when: app.status.operationState.phase in ['Succeeded'] and app.status.health.status == 'Healthy'

  trigger.on-sync-failed: |
    - description: 배포 실패 시 알림
      send:
      - app-sync-failed
      when: app.status.operationState.phase in ['Error', 'Failed']

  trigger.on-health-degraded: |
    - description: 서비스 상태 이상 시 알림
      send:
      - app-health-degraded
      when: app.status.health.status == 'Degraded'

  subscriptions: |
    - recipients:
      - webhook:discord
      triggers:
      - on-deployed
      - on-sync-failed
      - on-health-degraded
DISCORD_CM_EOF

    # Secret에 webhook URL 추가 (Helm이 관리하는 Secret이므로 patch 사용)
    kubectl patch secret argocd-notifications-secret -n argocd \
        --type merge \
        -p "{\"stringData\":{\"discord-webhook-url\":\"$DISCORD_WEBHOOK_URL\"}}"

    # Notifications Controller 재시작
    kubectl rollout restart deployment/argocd-notifications-controller -n argocd 2>/dev/null || true

    echo "✅ Discord 알림 설정 완료"
    echo "   - 배포 성공/실패 시 Discord로 알림 전송"
else
    echo "⚠️  Discord 알림 설정 건너뜀"
    echo "   나중에 설정: ./k8s/argocd/scripts/setup-discord-notifications-dev.sh"
fi

# =============================================================================
# 완료 메시지
# =============================================================================
echo ""
echo "=============================================="
echo "  ✅ wealist-oranges dev 클러스터 준비 완료!"
echo "=============================================="
echo ""
echo "🐘 PostgreSQL: postgres.${NAMESPACE}.svc (클러스터 내부)"
echo "   - User: postgres"
echo "   - Database: wealist"
echo "   - 데이터 저장: ${WEALIST_DATA_PATH}/db_data/postgres"
echo ""
echo "📮 Redis: redis.${NAMESPACE}.svc (클러스터 내부)"
echo "   - 데이터 저장: ${WEALIST_DATA_PATH}/db_data/redis"
echo ""
echo "🌐 Istio Gateway: https://dev.wealist.co.kr"
echo "📦 Namespace: ${NAMESPACE}"
echo "📁 Data Path: ${WEALIST_DATA_PATH}"
echo ""
echo "📊 모니터링 (ArgoCD에서 monitoring-dev Sync 후):"
echo "   - Grafana:    https://dev.wealist.co.kr/api/monitoring/grafana"
echo "   - Prometheus: https://dev.wealist.co.kr/api/monitoring/prometheus"
echo "   - Kiali:      https://dev.wealist.co.kr/api/monitoring/kiali"
echo ""
echo "🔧 ArgoCD:"
echo "   - URL: https://dev.wealist.co.kr/api/argo"
echo "   - Google 로그인: LOG IN VIA GOOGLE 버튼"
echo "   - 또는 admin / ${ARGOCD_PASSWORD:-<변경됨>}"
echo ""
echo "📝 상태 확인:"
echo "   kubectl get pods -n ${NAMESPACE}"
echo "   kubectl get apps -n argocd"
echo "   make kind-dev-env-status"
echo "=============================================="
