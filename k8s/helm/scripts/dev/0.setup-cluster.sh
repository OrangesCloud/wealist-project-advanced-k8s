#!/bin/bash
# =============================================================================
# Kind 클러스터 + Istio Ambient 설정 (dev 환경)
# =============================================================================
# - 레지스트리: AWS ECR
# - Istio Ambient: Service Mesh (sidecar-less)
# - Gateway API: Kubernetes 표준 (NodePort 30080 → hostPort 8080)
# - ArgoCD: GitOps 배포

set -e

CLUSTER_NAME="wealist"
ISTIO_VERSION="1.24.0"
GATEWAY_API_VERSION="v1.2.0"
AWS_REGION="ap-northeast-2"
NAMESPACE="wealist-dev"

# 스크립트 디렉토리 및 kind-config.yaml 경로
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HELM_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
KIND_CONFIG="${SCRIPT_DIR}/kind-config.yaml"

echo "🚀 Kind 클러스터 + Istio Ambient 설정 (dev - AWS ECR)"
echo "   - Istio: ${ISTIO_VERSION}"
echo "   - Gateway API: ${GATEWAY_API_VERSION}"
echo "   - Registry: AWS ECR (ap-northeast-2)"
echo "   - Namespace: ${NAMESPACE}"
echo "   - Kind Config: ${KIND_CONFIG}"
echo ""

# Kind 설정 파일 확인
if [ ! -f "${KIND_CONFIG}" ]; then
    echo "❌ kind-config.yaml 파일이 없습니다: ${KIND_CONFIG}"
    exit 1
fi

# =============================================================================
# AWS 로그인 확인
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

# 1. 기존 클러스터 삭제 (있으면)
if kind get clusters 2>/dev/null | grep -q "^${CLUSTER_NAME}$"; then
    echo "기존 클러스터 삭제 중..."
    kind delete cluster --name "$CLUSTER_NAME"
fi

# 2. Kind 클러스터 생성
echo "🚀 Kind 클러스터 생성 중..."
kind create cluster --name "$CLUSTER_NAME" --config "${KIND_CONFIG}"

# 3. Gateway API CRDs 설치
echo "⏳ Gateway API CRDs 설치 중..."
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/${GATEWAY_API_VERSION}/standard-install.yaml
echo "✅ Gateway API CRDs 설치 완료"

# 4. Istio Ambient 모드 설치
echo "⏳ Istio Ambient 모드 설치 중..."

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

# Istio Ambient 프로필 설치
${ISTIOCTL} install --set profile=ambient --skip-confirmation

echo "⏳ Istio 컴포넌트 준비 대기 중..."
kubectl wait --namespace istio-system \
  --for=condition=ready pod \
  --selector=app=istiod \
  --timeout=120s || echo "WARNING: istiod not ready yet"

kubectl wait --namespace istio-system \
  --for=condition=ready pod \
  --selector=app=ztunnel \
  --timeout=120s || echo "WARNING: ztunnel not ready yet"

echo "✅ Istio Ambient 설치 완료"

# 4-1. Istio 관측성 애드온 설치 (Kiali, Jaeger)
echo "⏳ Istio 관측성 애드온 설치 중 (Kiali, Jaeger)..."
kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.24/samples/addons/kiali.yaml 2>/dev/null || \
    kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.20/samples/addons/kiali.yaml
kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.24/samples/addons/jaeger.yaml 2>/dev/null || \
    kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.20/samples/addons/jaeger.yaml

# 4-2. Kiali/Jaeger subpath 설정
echo "⏳ Kiali/Jaeger subpath 설정 중..."
kubectl get configmap kiali -n istio-system -o yaml | \
    sed 's|web_root: /kiali|web_root: /monitoring/kiali|g' | \
    kubectl apply -f - 2>/dev/null || true

kubectl set env deployment/jaeger -n istio-system QUERY_BASE_PATH=/monitoring/jaeger 2>/dev/null || true
kubectl rollout restart deployment/kiali -n istio-system 2>/dev/null || true
kubectl rollout restart deployment/jaeger -n istio-system 2>/dev/null || true

echo "✅ Kiali, Jaeger 설치 완료"

# 5. Istio Ingress Gateway 설치
echo "⏳ Istio Ingress Gateway 설치 중..."
kubectl apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: istio-ingressgateway
  namespace: istio-system
spec:
  gatewayClassName: istio
  listeners:
  - name: http
    port: 80
    protocol: HTTP
    allowedRoutes:
      namespaces:
        from: All
EOF

echo "⏳ Istio Gateway Pod 준비 대기 중..."
sleep 5
kubectl wait --namespace istio-system \
  --for=condition=ready pod \
  --selector=gateway.networking.k8s.io/gateway-name=istio-ingressgateway \
  --timeout=120s || echo "WARNING: Istio gateway not ready yet"

# 6. Istio Gateway Service를 NodePort로 노출
echo "⚙️ Istio Gateway NodePort 설정 중..."
echo "⏳ Istio Gateway 서비스 대기 중..."
kubectl wait --namespace istio-system \
  --for=jsonpath='{.spec.type}'=LoadBalancer \
  svc/istio-ingressgateway-istio \
  --timeout=60s 2>/dev/null || true

kubectl patch service istio-ingressgateway-istio -n istio-system --type='json' -p='[
  {"op": "replace", "path": "/spec/type", "value": "NodePort"},
  {"op": "replace", "path": "/spec/ports/1/nodePort", "value": 30080}
]' 2>/dev/null || \
kubectl patch service istio-ingressgateway-istio -n istio-system --type='json' -p='[
  {"op": "replace", "path": "/spec/type", "value": "NodePort"},
  {"op": "add", "path": "/spec/ports/1/nodePort", "value": 30080}
]' 2>/dev/null || echo "⚠️ NodePort 패치 실패 - 수동 설정 필요"

echo "📋 Gateway 서비스 상태:"
kubectl get svc -n istio-system istio-ingressgateway-istio -o wide

echo "✅ Istio Gateway 설정 완료"

# 7. 애플리케이션 네임스페이스 생성 (Ambient 모드 라벨 포함)
echo "📦 ${NAMESPACE} 네임스페이스 생성 (Ambient 모드)..."
kubectl create namespace ${NAMESPACE} 2>/dev/null || true
kubectl label namespace ${NAMESPACE} istio.io/dataplane-mode=ambient --overwrite

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

echo "✅ 네임스페이스에 Ambient 모드 + Git 정보 라벨 적용 완료"

# 8. ECR 인증 Secret 생성
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

# 8-1. External Secrets Operator (ESO) 설치 및 설정
echo "🔐 External Secrets Operator (ESO) 설치 중..."

# external-secrets 네임스페이스 생성
kubectl create namespace external-secrets 2>/dev/null || true

# ESO Helm 레포 추가 및 설치
helm repo add external-secrets https://charts.external-secrets.io 2>/dev/null || true
helm repo update

# ESO 설치 (이미 있으면 업그레이드)
helm upgrade --install external-secrets external-secrets/external-secrets \
    --namespace external-secrets \
    --set installCRDs=true \
    --wait --timeout 5m
echo "✅ External Secrets Operator 설치 완료"

# CRD가 준비될 때까지 대기
echo "⏳ ESO CRDs 준비 대기 중..."
sleep 5
kubectl wait --for=condition=established --timeout=60s crd/clustersecretstores.external-secrets.io 2>/dev/null || true
kubectl wait --for=condition=established --timeout=60s crd/externalsecrets.external-secrets.io 2>/dev/null || true
echo "✅ ESO CRDs 준비 완료"

# 8-2. AWS 자격증명 Secret 생성 (ESO가 AWS Secrets Manager 접근용)
echo "🔐 AWS 자격증명 Secret 생성 중..."

# AWS 자격증명 가져오기 (환경변수 → AWS CLI → CLI 입력 순서)
AWS_ACCESS_KEY="${AWS_ACCESS_KEY_ID:-}"
AWS_SECRET_KEY="${AWS_SECRET_ACCESS_KEY:-}"

# 환경변수 없으면 AWS CLI 설정에서 가져오기
if [ -z "$AWS_ACCESS_KEY" ] || [ -z "$AWS_SECRET_KEY" ]; then
    echo "  → 환경변수에서 AWS 자격증명을 찾을 수 없어 AWS CLI에서 가져옵니다..."
    AWS_ACCESS_KEY=$(aws configure get aws_access_key_id 2>/dev/null || echo "")
    AWS_SECRET_KEY=$(aws configure get aws_secret_access_key 2>/dev/null || echo "")
fi

# 여전히 없으면 CLI로 입력받기
if [ -z "$AWS_ACCESS_KEY" ] || [ -z "$AWS_SECRET_KEY" ]; then
    echo ""
    echo "  AWS 자격증명이 필요합니다. (ESO가 AWS Secrets Manager 접근용)"
    echo "  AWS Access Key와 Secret Key를 입력하세요."
    echo "  (건너뛰려면 Enter를 누르세요)"
    echo ""
    read -p "  AWS Access Key ID: " AWS_ACCESS_KEY
    if [ -n "$AWS_ACCESS_KEY" ]; then
        read -sp "  AWS Secret Access Key: " AWS_SECRET_KEY
        echo ""
    fi
fi

if [ -z "$AWS_ACCESS_KEY" ] || [ -z "$AWS_SECRET_KEY" ]; then
    echo ""
    echo "⚠️  AWS 자격증명이 설정되지 않았습니다."
    echo "   ESO 없이 진행합니다. 나중에 다음 명령어로 설정할 수 있습니다:"
    echo ""
    echo "   make eso-setup-aws"
    echo "   make eso-apply-dev"
    echo ""
else
    # AWS 자격증명 Secret 생성
    kubectl delete secret aws-credentials -n external-secrets 2>/dev/null || true
    kubectl create secret generic aws-credentials \
        --from-literal=access-key="${AWS_ACCESS_KEY}" \
        --from-literal=secret-access-key="${AWS_SECRET_KEY}" \
        -n external-secrets
    echo "✅ AWS 자격증명 Secret 생성 완료"

    # 8-3. ClusterSecretStore 적용
    echo "🔐 ClusterSecretStore 적용 중..."
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    kubectl apply -f "${SCRIPT_DIR}/../../../argocd/base/external-secrets/dev/cluster-secret-store-dev.yaml"
    echo "✅ ClusterSecretStore 적용 완료"

    # 8-4. ExternalSecret 적용 (wealist-shared-secret 자동 생성)
    echo "🔐 ExternalSecret 적용 중 (wealist-shared-secret 자동 생성)..."
    # 기존 수동 생성 secret 삭제
    kubectl delete secret wealist-shared-secret -n ${NAMESPACE} 2>/dev/null || true
    kubectl apply -f "${SCRIPT_DIR}/../../../argocd/base/external-secrets/dev/external-secret-shared.yaml"
    echo "✅ ExternalSecret 적용 완료"

    # ESO sync 상태 확인
    echo "⏳ ExternalSecret sync 대기 중..."
    sleep 5
    kubectl get externalsecret wealist-shared-secret -n ${NAMESPACE} 2>/dev/null || echo "  (ArgoCD가 나중에 생성)"
fi

echo "✅ ESO 설정 완료"

# 9. 호스트 PostgreSQL/Redis 설정 (Kind 네트워크 허용)
echo "🔐 호스트 PostgreSQL 설정 중 (Kind 네트워크 허용)..."

# pg_hba.conf 찾기
PG_HBA=$(sudo find /etc/postgresql -name pg_hba.conf 2>/dev/null | head -1)
if [ -n "${PG_HBA}" ]; then
    PG_CHANGED=false

    # Kind Pod 네트워크 (10.244.0.0/16) - 필수!
    if ! sudo grep -q "10.244.0.0/16" "${PG_HBA}" 2>/dev/null; then
        echo "  → pg_hba.conf: Kind Pod 네트워크 (10.244.0.0/16) 허용 추가..."
        echo "# Kind cluster Pod network" | sudo tee -a "${PG_HBA}" > /dev/null
        echo "host    all    all    10.244.0.0/16     md5" | sudo tee -a "${PG_HBA}" > /dev/null
        PG_CHANGED=true
    fi

    # Docker bridge 네트워크 (172.17.0.0/16, 172.18.0.0/16)
    if ! sudo grep -q "172.16.0.0/12" "${PG_HBA}" 2>/dev/null; then
        echo "  → pg_hba.conf: Docker 네트워크 (172.16.0.0/12) 허용 추가..."
        echo "host    all    all    172.16.0.0/12     md5" | sudo tee -a "${PG_HBA}" > /dev/null
        PG_CHANGED=true
    fi

    # WSL/Host 네트워크 (192.168.x.x)
    if ! sudo grep -q "192.168.0.0/16" "${PG_HBA}" 2>/dev/null; then
        echo "  → pg_hba.conf: WSL/Host 네트워크 (192.168.0.0/16) 허용 추가..."
        echo "host    all    all    192.168.0.0/16    md5" | sudo tee -a "${PG_HBA}" > /dev/null
        PG_CHANGED=true
    fi

    # 넓은 범위 (fallback)
    if ! sudo grep -q "10.0.0.0/8" "${PG_HBA}" 2>/dev/null; then
        echo "  → pg_hba.conf: 10.0.0.0/8 네트워크 허용 추가..."
        echo "host    all    all    10.0.0.0/8        md5" | sudo tee -a "${PG_HBA}" > /dev/null
        PG_CHANGED=true
    fi

    if [ "$PG_CHANGED" = false ]; then
        echo "  → pg_hba.conf: Kind 네트워크 이미 허용됨"
    fi

    # postgresql.conf에서 listen_addresses 확인
    PG_CONF=$(dirname "${PG_HBA}")/postgresql.conf
    if [ -f "${PG_CONF}" ]; then
        if ! sudo grep -q "listen_addresses = '\*'" "${PG_CONF}" 2>/dev/null; then
            echo "  → postgresql.conf: listen_addresses = '*' 설정..."
            sudo sed -i "s/#listen_addresses = 'localhost'/listen_addresses = '*'/" "${PG_CONF}"
            sudo sed -i "s/listen_addresses = 'localhost'/listen_addresses = '*'/" "${PG_CONF}"
            PG_CHANGED=true
        fi
    fi

    # PostgreSQL 재시작 (변경사항 있으면)
    if [ "$PG_CHANGED" = true ]; then
        echo "  → PostgreSQL 재시작..."
        sudo systemctl restart postgresql 2>/dev/null || sudo service postgresql restart 2>/dev/null || true
    fi
    echo "✅ PostgreSQL 설정 완료"
else
    echo "⚠️  PostgreSQL 설정 파일을 찾을 수 없습니다. 수동 설정 필요."
fi

# 10. DB_HOST 동적 감지 (WSL/Linux/macOS)
echo "🔍 호스트 IP 감지 중..."
if [ "$(uname)" = "Darwin" ]; then
    DB_HOST="host.docker.internal"
    echo "  🖥️  macOS 감지 → DB_HOST: host.docker.internal"
elif grep -qi microsoft /proc/version 2>/dev/null; then
    # WSL2: Docker bridge gateway IP 사용 (Kind 노드에서 호스트 접근용)
    DB_HOST=$(docker network inspect bridge -f '{{range .IPAM.Config}}{{.Gateway}}{{end}}' 2>/dev/null || echo "172.17.0.1")
    echo "  🖥️  WSL 감지 → DB_HOST: ${DB_HOST} (Docker bridge gateway)"
else
    DB_HOST="172.17.0.1"
    echo "  🖥️  Linux 감지 → DB_HOST: 172.17.0.1 (Docker bridge gateway)"
fi

# dev.yaml에 DB_HOST 동적 업데이트
DEV_YAML="${HELM_DIR}/environments/dev.yaml"
echo "  → dev.yaml에 DB_HOST 업데이트: ${DB_HOST}"
if [[ "$OSTYPE" == "darwin"* ]]; then
    sed -i '' "s/DB_HOST: .*/DB_HOST: \"${DB_HOST}\"/" "${DEV_YAML}"
    sed -i '' "s/POSTGRES_HOST: .*/POSTGRES_HOST: \"${DB_HOST}\"/" "${DEV_YAML}"
    sed -i '' "s/REDIS_HOST: .*/REDIS_HOST: \"${DB_HOST}\"/" "${DEV_YAML}"
    sed -i '' "s/SPRING_REDIS_HOST: .*/SPRING_REDIS_HOST: \"${DB_HOST}\"/" "${DEV_YAML}"
    # postgres/redis external host 업데이트
    sed -i '' "s/^    host: .*/    host: \"${DB_HOST}\"/" "${DEV_YAML}"
else
    sed -i "s/DB_HOST: .*/DB_HOST: \"${DB_HOST}\"/" "${DEV_YAML}"
    sed -i "s/POSTGRES_HOST: .*/POSTGRES_HOST: \"${DB_HOST}\"/" "${DEV_YAML}"
    sed -i "s/REDIS_HOST: .*/REDIS_HOST: \"${DB_HOST}\"/" "${DEV_YAML}"
    sed -i "s/SPRING_REDIS_HOST: .*/SPRING_REDIS_HOST: \"${DB_HOST}\"/" "${DEV_YAML}"
    # postgres/redis external host 업데이트
    sed -i "s/^    host: .*/    host: \"${DB_HOST}\"/" "${DEV_YAML}"
fi
echo "✅ DB_HOST 설정 완료"

# ArgoCD Application 파일들에 DB_HOST 업데이트
ARGOCD_APPS_DIR="${HELM_DIR}/../argocd/apps/dev"
if [ -d "${ARGOCD_APPS_DIR}" ]; then
    echo "  → ArgoCD Application 파일들 업데이트 중..."
    for file in "${ARGOCD_APPS_DIR}"/*-service.yaml; do
        if [ -f "$file" ]; then
            # host.docker.internal을 실제 DB_HOST로 교체
            sed -i "s|value: \"host.docker.internal\"|value: \"${DB_HOST}\"|g" "$file"
        fi
    done
    echo "✅ ArgoCD Application 파일들 업데이트 완료 (DB_HOST: ${DB_HOST})"
fi

# 11. dev.yaml에 AWS Account ID 자동 업데이트
DEV_YAML="${HELM_DIR}/environments/dev.yaml"
if grep -q "<AWS_ACCOUNT_ID>" "${DEV_YAML}" 2>/dev/null; then
    echo "🔧 dev.yaml에 AWS Account ID 자동 업데이트 중..."
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' "s/<AWS_ACCOUNT_ID>/${AWS_ACCOUNT_ID}/g" "${DEV_YAML}"
    else
        sed -i "s/<AWS_ACCOUNT_ID>/${AWS_ACCOUNT_ID}/g" "${DEV_YAML}"
    fi
    echo "✅ dev.yaml 업데이트 완료: imageRegistry → ${ECR_REGISTRY}"
else
    echo "✅ dev.yaml: AWS Account ID 이미 설정됨"
fi

echo ""
echo "=============================================="
echo "  ✅ dev 클러스터 준비 완료!"
echo "=============================================="
echo ""
echo "🔐 Registry: ${ECR_REGISTRY} (AWS ECR)"
echo "🌐 Istio Gateway: localhost:80 (또는 :8080)"
echo "📦 Namespace: ${NAMESPACE}"
echo ""
echo "📊 모니터링 (배포 후 접근 가능):"
echo "   - Grafana:    https://dev.wealist.co.kr/api/monitoring/grafana"
echo "   - Prometheus: https://dev.wealist.co.kr/api/monitoring/prometheus"
echo "   - Kiali:      https://dev.wealist.co.kr/api/monitoring/kiali"
echo "   - Jaeger:     https://dev.wealist.co.kr/api/monitoring/jaeger"
echo ""
echo "📝 다음 단계:"
echo "   1. ArgoCD 설치:"
echo "      make argo-install-simple"
echo ""
echo "   2. ArgoCD로 GitOps 배포:"
echo "      make argo-deploy-dev"
echo ""
echo "   3. 상태 확인:"
echo "      kubectl get pods -n ${NAMESPACE}"
echo "=============================================="
