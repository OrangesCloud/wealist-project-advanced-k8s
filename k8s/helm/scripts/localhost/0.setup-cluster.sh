#!/bin/bash
# =============================================================================
# Kind 클러스터 + Istio Ambient 설정 (localhost 환경)
# =============================================================================
# - 로컬 레지스트리: localhost:5001
# - Istio Ambient: Service Mesh (sidecar-less)
# - Gateway API: Kubernetes 표준 (NodePort 30080 → hostPort 8080)

set -e

CLUSTER_NAME="wealist"
REG_NAME="kind-registry"
REG_PORT="5001"
ISTIO_VERSION="1.24.0"
GATEWAY_API_VERSION="v1.2.0"

# 스크립트 디렉토리 및 kind-config.yaml 경로
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HELM_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
KIND_CONFIG="${SCRIPT_DIR}/kind-config.yaml"  # 환경별 분리된 설정 사용

echo "🚀 Kind 클러스터 + Istio Ambient 설정 (localhost)"
echo "   - Istio: ${ISTIO_VERSION}"
echo "   - Gateway API: ${GATEWAY_API_VERSION}"
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

# 2. 로컬 레지스트리 시작 (없으면)
if [ "$(docker inspect -f '{{.State.Running}}' "${REG_NAME}" 2>/dev/null || true)" != 'true' ]; then
    echo "📦 로컬 레지스트리 시작 (localhost:${REG_PORT})"
    docker run -d --restart=always -p "127.0.0.1:${REG_PORT}:5000" --network bridge --name "${REG_NAME}" registry:2
fi

# 3. Kind 클러스터 생성
echo "🚀 Kind 클러스터 생성 중..."
kind create cluster --name "$CLUSTER_NAME" --config "${KIND_CONFIG}"

# 4. 레지스트리를 Kind 네트워크에 연결
if [ "$(docker inspect -f='{{json .NetworkSettings.Networks.kind}}' "${REG_NAME}" 2>/dev/null)" = 'null' ]; then
    echo "레지스트리를 Kind 네트워크에 연결..."
    docker network connect "kind" "${REG_NAME}"
fi

# 5. 레지스트리 ConfigMap 생성
kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: local-registry-hosting
  namespace: kube-public
data:
  localRegistryHosting.v1: |
    host: "localhost:${REG_PORT}"
    help: "https://kind.sigs.k8s.io/docs/user/local-registry/"
EOF

# 6. Gateway API CRDs 설치
echo "⏳ Gateway API CRDs 설치 중..."
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/${GATEWAY_API_VERSION}/standard-install.yaml
echo "✅ Gateway API CRDs 설치 완료"

# 7. Istio Ambient 모드 설치
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

# 7-1. Istio 관측성 애드온 설치 (Kiali, Jaeger)
echo "⏳ Istio 관측성 애드온 설치 중 (Kiali, Jaeger)..."
kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.24/samples/addons/kiali.yaml 2>/dev/null || \
    kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.20/samples/addons/kiali.yaml
kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.24/samples/addons/jaeger.yaml 2>/dev/null || \
    kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.20/samples/addons/jaeger.yaml

# 7-2. Kiali/Jaeger subpath 설정 (HTTPRoute /monitoring/* 경로용)
echo "⏳ Kiali/Jaeger subpath 설정 중..."

# Kiali ConfigMap 패치 - web_root를 /monitoring/kiali로 변경
# 기본 Istio addon은 web_root: /kiali로 설정됨 → /monitoring/kiali로 변경 필요
kubectl get configmap kiali -n istio-system -o yaml | \
    sed 's|web_root: /kiali|web_root: /monitoring/kiali|g' | \
    kubectl apply -f - 2>/dev/null || true

# Jaeger 환경변수 설정 (QUERY_BASE_PATH)
kubectl set env deployment/jaeger -n istio-system QUERY_BASE_PATH=/monitoring/jaeger 2>/dev/null || true

# Kiali, Jaeger 재시작 (설정 적용)
kubectl rollout restart deployment/kiali -n istio-system 2>/dev/null || true
kubectl rollout restart deployment/jaeger -n istio-system 2>/dev/null || true

echo "✅ Kiali, Jaeger 설치 완료 (subpath: /monitoring/kiali, /monitoring/jaeger)"

# 8. Istio Ingress Gateway 설치 (외부 트래픽용)
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

# 9. Istio Gateway Service를 NodePort로 노출 (Kind hostPort 8080 사용)
# ports[0]=status-port(15021), ports[1]=http(80) → http에 NodePort 30080 할당
echo "⚙️ Istio Gateway NodePort 설정 중..."
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

# 10. 애플리케이션 네임스페이스 생성 (Ambient 모드 라벨 포함)
echo "📦 wealist-localhost 네임스페이스 생성 (Ambient 모드)..."
kubectl create namespace wealist-localhost 2>/dev/null || true
kubectl label namespace wealist-localhost istio.io/dataplane-mode=ambient --overwrite

# Git 정보 라벨 추가 (배포 추적용)
GIT_REPO=$(git config --get remote.origin.url 2>/dev/null | sed 's/.*github.com[:/]\(.*\)\.git/\1/' || echo "unknown")
GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_USER=$(git config --get user.name 2>/dev/null || echo "unknown")
GIT_EMAIL=$(git config --get user.email 2>/dev/null || echo "unknown")
DEPLOY_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

kubectl annotate namespace wealist-localhost \
  "wealist.io/git-repo=${GIT_REPO}" \
  "wealist.io/git-branch=${GIT_BRANCH}" \
  "wealist.io/git-commit=${GIT_COMMIT}" \
  "wealist.io/deployed-by=${GIT_USER}" \
  "wealist.io/deployed-by-email=${GIT_EMAIL}" \
  "wealist.io/deploy-time=${DEPLOY_TIME}" \
  --overwrite

echo "✅ 네임스페이스에 Ambient 모드 + Git 정보 라벨 적용 완료"

echo ""
echo "=============================================="
echo "  ✅ localhost 클러스터 준비 완료!"
echo "=============================================="
echo ""
echo "📦 로컬 레지스트리: localhost:${REG_PORT}"
echo "🌐 Istio Gateway: localhost:80 (또는 :8080)"
echo ""
echo "📊 모니터링 (helm-install-all 후 접근 가능):"
echo "   - Grafana:    http://localhost:8080/api/monitoring/grafana"
echo "   - Prometheus: http://localhost:8080/api/monitoring/prometheus"
echo "   - Kiali:      http://localhost:8080/api/monitoring/kiali"
echo "   - Jaeger:     http://localhost:8080/api/monitoring/jaeger"
echo ""
echo "📝 다음 단계:"
echo "   1. 이미지 로드:"
echo "      ./1.load_infra_images.sh"
echo "      ./2.build_all_and_load.sh"
echo ""
echo "   2. Helm 배포:"
echo "      make helm-install-all ENV=localhost"
echo ""
echo "   3. 접근:"
echo "      http://localhost:8080/"
echo "      http://localhost:8080/svc/auth/api/..."
echo "=============================================="
