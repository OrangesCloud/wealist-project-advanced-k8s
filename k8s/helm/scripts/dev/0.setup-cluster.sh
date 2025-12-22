#!/bin/bash
# =============================================================================
# Kind 클러스터 + Istio Ambient 설정 (dev 환경)
# =============================================================================
# - 레지스트리: GHCR (ghcr.io/orangescloud)
# - Istio Ambient: Service Mesh (sidecar-less)
# - Gateway API: Kubernetes 표준 (NodePort 30080 → hostPort 8080)
# - ArgoCD: GitOps 배포

set -e

CLUSTER_NAME="wealist"
ISTIO_VERSION="1.24.0"
GATEWAY_API_VERSION="v1.2.0"

# 스크립트 디렉토리 및 kind-config.yaml 경로
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HELM_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
KIND_CONFIG="${SCRIPT_DIR}/kind-config.yaml"  # 환경별 분리된 설정 사용

echo "🚀 Kind 클러스터 + Istio Ambient 설정 (dev - GHCR)"
echo "   - Istio: ${ISTIO_VERSION}"
echo "   - Gateway API: ${GATEWAY_API_VERSION}"
echo "   - Registry: ghcr.io/orangescloud (GHCR)"
echo "   - Kind Config: ${KIND_CONFIG}"
echo ""

# Kind 설정 파일 확인
if [ ! -f "${KIND_CONFIG}" ]; then
    echo "❌ kind-config.yaml 파일이 없습니다: ${KIND_CONFIG}"
    exit 1
fi

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
echo "✅ Kiali, Jaeger 설치 완료"

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
# HTTPS (port 443) → NodePort 30443 → hostPort 443
kubectl patch service istio-ingressgateway-istio -n istio-system --type='json' -p='[
  {
    "op": "replace",
    "path": "/spec/type",
    "value": "NodePort"
  },
  {
    "op": "add",
    "path": "/spec/ports/1/nodePort",
    "value": 30080
  }
]' || echo "INFO: Service 이미 NodePort로 설정됨"

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

# 8. GHCR 인증 Secret 생성
echo "🔐 GHCR 인증 Secret 설정 중..."
if [ -n "${GHCR_TOKEN}" ] && [ -n "${GHCR_USERNAME}" ]; then
    kubectl create secret docker-registry ghcr-secret \
        --docker-server=ghcr.io \
        --docker-username="${GHCR_USERNAME}" \
        --docker-password="${GHCR_TOKEN}" \
        -n wealist-dev 2>/dev/null || \
    kubectl delete secret ghcr-secret -n wealist-dev 2>/dev/null && \
    kubectl create secret docker-registry ghcr-secret \
        --docker-server=ghcr.io \
        --docker-username="${GHCR_USERNAME}" \
        --docker-password="${GHCR_TOKEN}" \
        -n wealist-dev
    echo "✅ GHCR Secret 생성 완료"
else
    echo "⚠️  GHCR_TOKEN 또는 GHCR_USERNAME 환경변수가 없습니다."
    echo "   나중에 다음 명령어로 생성하세요:"
    echo "   kubectl create secret docker-registry ghcr-secret \\"
    echo "     --docker-server=ghcr.io \\"
    echo "     --docker-username=<github-username> \\"
    echo "     --docker-password=<github-token> \\"
    echo "     -n wealist-dev"
fi

echo ""
echo "=============================================="
echo "  ✅ dev 클러스터 준비 완료!"
echo "=============================================="
echo ""
echo "🔐 Registry: ghcr.io/orangescloud (GHCR)"
echo "🌐 Istio Gateway: localhost:80 (또는 :8080)"
echo ""
echo "📊 모니터링 (helm-install-all 후 접근 가능):"
echo "   - Grafana:    http://dev.wealist.co.kr/monitoring/grafana"
echo "   - Prometheus: http://dev.wealist.co.kr/monitoring/prometheus"
echo "   - Kiali:      http://dev.wealist.co.kr/monitoring/kiali"
echo "   - Jaeger:     http://dev.wealist.co.kr/monitoring/jaeger"
echo "   ※ hosts 파일에 127.0.0.1 dev.wealist.co.kr 추가 필요"
echo ""
echo "📝 다음 단계:"
echo "   1. GHCR 로그인 (이미지 푸시/풀 위해):"
echo "      echo \$GHCR_TOKEN | docker login ghcr.io -u \$GHCR_USERNAME --password-stdin"
echo ""
echo "   2. 이미지 빌드 및 GHCR 푸시:"
echo "      ./2.build_and_push_ghcr.sh"
echo ""
echo "   3. ArgoCD 배포 (선택사항):"
echo "      make bootstrap && make deploy"
echo ""
echo "   4. 또는 Helm 직접 배포:"
echo "      make helm-install-all ENV=dev"
echo ""
echo "   5. 접근:"
echo "      http://localhost:8080/"
echo "=============================================="
