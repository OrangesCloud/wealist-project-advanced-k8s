#!/bin/bash
# =============================================================================
# Kind 클러스터 + Istio Ambient 설정 (localhost 환경)
# =============================================================================
# - 로컬 레지스트리: localhost:5001
# - Istio Ambient: Service Mesh (sidecar-less)
# - Gateway API: Kubernetes 표준 + hostPort 80

set -e

CLUSTER_NAME="wealist"
REG_NAME="kind-registry"
REG_PORT="5001"
ISTIO_VERSION="1.24.0"
GATEWAY_API_VERSION="v1.2.0"

# 스크립트 디렉토리 및 kind-config.yaml 경로
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HELM_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
KIND_CONFIG="${HELM_DIR}/kind-config.yaml"

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

# 9. Istio Gateway를 hostPort 80으로 설정 (localhost:80 접근)
# Gateway API가 생성하는 deployment 이름: <gateway-name>-istio
echo "⚙️ Istio Gateway hostPort 80 설정 중..."
GATEWAY_DEPLOY=$(kubectl get deployment -n istio-system -l gateway.networking.k8s.io/gateway-name=istio-ingressgateway -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)

if [ -n "$GATEWAY_DEPLOY" ]; then
  echo "   Gateway Deployment: $GATEWAY_DEPLOY"
  kubectl patch deployment "$GATEWAY_DEPLOY" -n istio-system --type='json' -p='[
    {
      "op": "replace",
      "path": "/spec/template/spec/containers/0/ports",
      "value": [
        {"containerPort": 80, "hostPort": 80, "protocol": "TCP", "name": "http"},
        {"containerPort": 443, "hostPort": 443, "protocol": "TCP", "name": "https"},
        {"containerPort": 15020, "protocol": "TCP", "name": "metrics"},
        {"containerPort": 15021, "protocol": "TCP", "name": "status-port"}
      ]
    },
    {
      "op": "add",
      "path": "/spec/template/spec/nodeSelector",
      "value": {"ingress-ready": "true"}
    }
  ]'

  # Gateway Pod 재시작 대기
  echo "⏳ Gateway Pod 재시작 대기 중..."
  sleep 3
  kubectl rollout status deployment/"$GATEWAY_DEPLOY" -n istio-system --timeout=120s || true
else
  echo "⚠️ Gateway deployment를 찾을 수 없습니다. 수동 패치 필요."
fi

echo ""
echo "=============================================="
echo "  ✅ localhost 클러스터 준비 완료!"
echo "=============================================="
echo ""
echo "📦 로컬 레지스트리: localhost:${REG_PORT}"
echo "🌐 Istio Gateway: localhost (hostPort 80)"
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
echo "      http://localhost/"
echo "      http://localhost/svc/auth/api/..."
echo "=============================================="
