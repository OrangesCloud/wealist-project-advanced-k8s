#!/bin/bash
# =============================================================================
# Kind 클러스터 + Istio Ambient 설정 (dev 환경)
# =============================================================================
# - 레지스트리: AWS ECR (<AWS_ACCOUNT_ID>.dkr.ecr.ap-northeast-2.amazonaws.com)
# - Istio Ambient: Service Mesh (sidecar-less)
# - Gateway API: Kubernetes 표준 (NodePort 30080 → hostPort 8080)
# - ArgoCD: GitOps 배포

set -e

CLUSTER_NAME="wealist"
ISTIO_VERSION="1.24.0"
GATEWAY_API_VERSION="v1.2.0"
AWS_REGION="ap-northeast-2"

# 스크립트 디렉토리 및 kind-config.yaml 경로
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HELM_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
KIND_CONFIG="${SCRIPT_DIR}/kind-config.yaml"  # 환경별 분리된 설정 사용

echo "🚀 Kind 클러스터 + Istio Ambient 설정 (dev - AWS ECR)"
echo "   - Istio: ${ISTIO_VERSION}"
echo "   - Gateway API: ${GATEWAY_API_VERSION}"
echo "   - Registry: AWS ECR (ap-northeast-2)"
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
    echo "   3. 환경변수 설정:"
    echo "      export AWS_ACCESS_KEY_ID=<your-key>"
    echo "      export AWS_SECRET_ACCESS_KEY=<your-secret>"
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
    echo "⚠️  istioctl이 설치되어 있지 않습니다."
    echo "   다음 명령어로 설치하세요:"
    echo "   curl -L https://istio.io/downloadIstio | ISTIO_VERSION=${ISTIO_VERSION} sh -"
    exit 1
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

# 4-2. Kiali/Jaeger subpath 설정 (HTTPRoute /monitoring/* 경로용)
echo "⏳ Kiali/Jaeger subpath 설정 중..."

# Kiali ConfigMap 패치 - web_root 설정
kubectl get configmap kiali -n istio-system -o yaml | \
    sed 's|web_root: ""|web_root: "/monitoring/kiali"|g' | \
    sed '/server:/a\      web_root: "/monitoring/kiali"' | \
    kubectl apply -f - 2>/dev/null || true

# Kiali ConfigMap이 web_root 없으면 직접 패치
kubectl patch configmap kiali -n istio-system --type=json \
    -p='[{"op": "replace", "path": "/data/config.yaml", "value": "'"$(kubectl get configmap kiali -n istio-system -o jsonpath='{.data.config\.yaml}' | sed 's/server:/server:\n      web_root: "\/monitoring\/kiali"/')"'"}]' 2>/dev/null || true

# Jaeger 환경변수 설정 (QUERY_BASE_PATH)
kubectl set env deployment/jaeger -n istio-system QUERY_BASE_PATH=/monitoring/jaeger 2>/dev/null || true

# Kiali, Jaeger 재시작 (설정 적용)
kubectl rollout restart deployment/kiali -n istio-system 2>/dev/null || true
kubectl rollout restart deployment/jaeger -n istio-system 2>/dev/null || true

echo "✅ Kiali, Jaeger 설치 완료 (subpath: /monitoring/kiali, /monitoring/jaeger)"

# 5. Istio Ingress Gateway 설치 (외부 트래픽용)
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

# 6. Istio Gateway Service를 NodePort로 노출 (Kind hostPort 80/443 사용)
echo "⚙️ Istio Gateway NodePort 설정 중..."
# HTTP (port 80) → NodePort 30080 → hostPort 80
# 서비스 포트 구조: ports[0]=15021(status), ports[1]=80(http)

# 서비스가 생성될 때까지 대기
echo "⏳ Istio Gateway 서비스 대기 중..."
kubectl wait --namespace istio-system \
  --for=jsonpath='{.spec.type}'=LoadBalancer \
  svc/istio-ingressgateway-istio \
  --timeout=60s 2>/dev/null || true

# NodePort로 변경하고 포트 80의 nodePort를 30080으로 설정
# replace를 사용하여 기존 nodePort 값을 덮어씀
kubectl patch service istio-ingressgateway-istio -n istio-system --type='json' -p='[
  {"op": "replace", "path": "/spec/type", "value": "NodePort"},
  {"op": "replace", "path": "/spec/ports/1/nodePort", "value": 30080}
]' 2>/dev/null || \
kubectl patch service istio-ingressgateway-istio -n istio-system --type='json' -p='[
  {"op": "replace", "path": "/spec/type", "value": "NodePort"},
  {"op": "add", "path": "/spec/ports/1/nodePort", "value": 30080}
]' 2>/dev/null || echo "⚠️ NodePort 패치 실패 - 수동 설정 필요"

# 설정 확인
echo "📋 Gateway 서비스 상태:"
kubectl get svc -n istio-system istio-ingressgateway-istio -o wide

echo "✅ Istio Gateway 설정 완료"
echo "   - HTTP:  localhost:80 (또는 :8080)"
echo "   - HTTPS: localhost:443"

# 7. 애플리케이션 네임스페이스 생성 (Ambient 모드 라벨 포함)
echo "📦 wealist-dev 네임스페이스 생성 (Ambient 모드)..."
kubectl create namespace wealist-dev 2>/dev/null || true
kubectl label namespace wealist-dev istio.io/dataplane-mode=ambient --overwrite

# Git 정보 라벨 추가 (배포 추적용)
GIT_REPO=$(git config --get remote.origin.url 2>/dev/null | sed 's/.*github.com[:/]\(.*\)\.git/\1/' || echo "unknown")
GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_USER=$(git config --get user.name 2>/dev/null || echo "unknown")
GIT_EMAIL=$(git config --get user.email 2>/dev/null || echo "unknown")
DEPLOY_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

kubectl annotate namespace wealist-dev \
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
    kubectl delete secret ecr-secret -n wealist-dev 2>/dev/null || true
    kubectl create secret docker-registry ecr-secret \
        --docker-server="${ECR_REGISTRY}" \
        --docker-username=AWS \
        --docker-password="${ECR_PASSWORD}" \
        -n wealist-dev
    echo "✅ ECR Secret 생성 완료"
else
    echo "❌ ECR 로그인 실패. AWS 자격증명을 확인하세요."
    exit 1
fi

# 9. dev.yaml에 AWS Account ID 자동 업데이트
DEV_YAML="${HELM_DIR}/environments/dev.yaml"
if grep -q "<AWS_ACCOUNT_ID>" "${DEV_YAML}" 2>/dev/null; then
    echo "🔧 dev.yaml에 AWS Account ID 자동 업데이트 중..."
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS (BSD sed)
        sed -i '' "s/<AWS_ACCOUNT_ID>/${AWS_ACCOUNT_ID}/g" "${DEV_YAML}"
    else
        # Linux (GNU sed)
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
echo ""
echo "📊 모니터링 (helm-install-all 후 접근 가능):"
echo "   - Grafana:    https://api.dev.wealist.co.kr/monitoring/grafana"
echo "   - Prometheus: https://api.dev.wealist.co.kr/monitoring/prometheus"
echo "   - Kiali:      https://api.dev.wealist.co.kr/monitoring/kiali"
echo "   - Jaeger:     https://api.dev.wealist.co.kr/monitoring/jaeger"
echo ""
echo "📝 다음 단계:"
echo "   1. Helm 배포:"
echo "      make helm-install-all ENV=dev"
echo ""
echo "   2. 또는 ArgoCD로 GitOps 배포:"
echo "      make argo-bootstrap && make argo-deploy"
echo ""
echo "   3. 접근:"
echo "      https://api.dev.wealist.co.kr/"
echo "=============================================="
