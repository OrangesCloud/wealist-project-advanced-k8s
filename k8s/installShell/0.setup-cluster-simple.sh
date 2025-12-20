#!/bin/bash
# Kind 클러스터 + 로컬 레지스트리 + nginx ingress 설정 스크립트 (Simple Mode)
# - Istio 없음: 간단한 테스트용
# - nginx ingress: 표준 Kubernetes Ingress 사용
# - 로컬 레지스트리: Docker Hub rate limit 우회

set -e

CLUSTER_NAME="wealist"
REG_NAME="kind-registry"
REG_PORT="5001"

echo "🚀 Kind 클러스터 + nginx ingress 설정 (Simple Mode)"
echo "   - Istio: 없음"
echo "   - Ingress: nginx-ingress-controller"
echo ""

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

# 3. Kind 설정 파일 생성 (nginx ingress용 포트 매핑)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cat > "${SCRIPT_DIR}/kind-config-simple.yaml" <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
containerdConfigPatches:
  - |-
    [plugins."io.containerd.grpc.v1.cri".registry]
      [plugins."io.containerd.grpc.v1.cri".registry.mirrors]
        [plugins."io.containerd.grpc.v1.cri".registry.mirrors."localhost:${REG_PORT}"]
          endpoint = ["http://${REG_NAME}:5000"]
nodes:
  - role: control-plane
    kubeadmConfigPatches:
      - |
        kind: InitConfiguration
        nodeRegistration:
          kubeletExtraArgs:
            node-labels: "ingress-ready=true"
    extraPortMappings:
      - containerPort: 80
        hostPort: 80
        protocol: TCP
      - containerPort: 443
        hostPort: 443
        protocol: TCP
      - containerPort: 30080
        hostPort: 8080
        protocol: TCP
  - role: worker
  - role: worker
EOF

# 4. Kind 클러스터 생성
echo "🚀 Kind 클러스터 생성 중..."
kind create cluster --name "$CLUSTER_NAME" --config "${SCRIPT_DIR}/kind-config-simple.yaml"

# 5. 레지스트리를 Kind 네트워크에 연결
if [ "$(docker inspect -f='{{json .NetworkSettings.Networks.kind}}' "${REG_NAME}" 2>/dev/null)" = 'null' ]; then
    echo "레지스트리를 Kind 네트워크에 연결..."
    docker network connect "kind" "${REG_NAME}"
fi

# 6. 레지스트리 ConfigMap 생성
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

# 7. nginx ingress controller 설치
echo "⏳ nginx ingress controller 설치 중..."
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml

# 8. nginx ingress 준비 대기
echo "⏳ nginx ingress 준비 대기 중..."
kubectl wait --namespace ingress-nginx \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller \
  --timeout=120s || echo "WARNING: nginx ingress not ready yet"

echo ""
echo "✅ 클러스터 준비 완료! (Simple Mode)"
echo ""
echo "📦 로컬 레지스트리: localhost:${REG_PORT}"
echo "🌐 nginx ingress: localhost:80 / localhost:8080"
echo ""
echo "📝 다음 단계:"
echo "   1. make kind-load-images"
echo "   2. make helm-install-all ENV=local-kind"
echo ""
echo "⚠️  참고: Simple 모드에서는 Istio HTTPRoute 대신"
echo "   Kubernetes Ingress 리소스를 사용해야 합니다."
echo ""
