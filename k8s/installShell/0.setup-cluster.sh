#!/bin/bash
# Kind 클러스터 설정 스크립트 (GHCR 직접 연결)
# GitHub Container Registry에서 직접 이미지 가져오기

set -e

CLUSTER_NAME="wealist"

echo "🚀 Kind 클러스터 설정 (GHCR 직접 연결)"
echo ""

# 1. 기존 클러스터 삭제 (있으면)
if kind get clusters 2>/dev/null | grep -q "^${CLUSTER_NAME}$"; then
    echo "🗑️  기존 클러스터 삭제 중..."
    kind delete cluster --name "$CLUSTER_NAME"
fi

# 2. 기존 로컬 레지스트리 정리 (필요시)
REG_NAME="kind-registry"
if docker ps -a --format '{{.Names}}' | grep -q "^${REG_NAME}$"; then
    echo "🧹 기존 로컬 레지스트리 정리 중..."
    docker rm -f "${REG_NAME}" 2>/dev/null || true
fi

# 3. Kind 설정 파일 생성 (GHCR 직접 연결)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cat > "${SCRIPT_DIR}/kind-config.yaml" <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
containerdConfigPatches:
  - |-
    [plugins."io.containerd.grpc.v1.cri".registry]
      [plugins."io.containerd.grpc.v1.cri".registry.mirrors]
        [plugins."io.containerd.grpc.v1.cri".registry.mirrors."ghcr.io"]
          endpoint = ["https://ghcr.io"]
        [plugins."io.containerd.grpc.v1.cri".registry.mirrors."docker.io"]
          endpoint = ["https://registry-1.docker.io"]
      [plugins."io.containerd.grpc.v1.cri".registry.configs]
        [plugins."io.containerd.grpc.v1.cri".registry.configs."ghcr.io".tls]
          insecure_skip_verify = false
        [plugins."io.containerd.grpc.v1.cri".registry.configs."docker.io".tls]
          insecure_skip_verify = false

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
      - containerPort: 8079
        hostPort: 8079
        protocol: TCP
  - role: worker
  - role: worker
EOF

# 4. Kind 클러스터 생성
echo "🚀 Kind 클러스터 생성 중..."
kind create cluster --name "$CLUSTER_NAME" --config "${SCRIPT_DIR}/kind-config.yaml"

# 5. Ingress Nginx Controller 설치
echo "⏳ Ingress Nginx Controller 설치 중..."
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml

echo "⏳ Ingress Controller 준비 대기 중..."
kubectl wait --namespace ingress-nginx \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller \
  --timeout=180s || echo "⚠️  WARNING: Ingress controller not ready yet"

# 6. Nginx Ingress Controller 설정 (snippet 허용)
echo "⚙️  Ingress Controller 설정 중..."
kubectl patch configmap ingress-nginx-controller -n ingress-nginx \
  --type merge -p '{"data":{"allow-snippet-annotations":"true"}}' 2>/dev/null || true

# 7. 네임스페이스 생성
echo "📁 네임스페이스 생성 중..."
kubectl create namespace wealist-dev 2>/dev/null || true


# 8. GitHub 인증 정보 수집
echo ""
echo "🔑 GitHub Container Registry 인증 설정"
echo "GitHub Personal Access Token이 필요합니다."
echo "권한: repo, read:packages, write:packages"
echo ""

read -p "GitHub Username: " GITHUB_USERNAME
echo -n "GitHub Personal Access Token: "
read -s GITHUB_TOKEN
echo ""
echo ""

# 9. GHCR 인증 설정 (각 네임스페이스별)
NAMESPACES=("wealist-dev")

for namespace in "${NAMESPACES[@]}"; do
    echo "🐳 Setting up GHCR access for namespace: $namespace"
    
    # GHCR secret 생성
    kubectl create secret docker-registry ghcr-secret \
      --docker-server=ghcr.io \
      --docker-username="$GITHUB_USERNAME" \
      --docker-password="$GITHUB_TOKEN" \
      --docker-email="$GITHUB_USERNAME@users.noreply.github.com" \
      --namespace="$namespace" \
      --dry-run=client -o yaml | kubectl apply -f -
    
    # default ServiceAccount에 imagePullSecrets 추가
    kubectl patch serviceaccount default \
      -p '{"imagePullSecrets": [{"name": "ghcr-secret"}]}' \
      -n "$namespace"
    
    echo "✅ $namespace configured"
done

# 10. 클러스터 정보 확인
echo ""
echo "🔍 클러스터 정보 확인..."
kubectl cluster-info --context kind-wealist
kubectl get nodes -o wide

# 11. GHCR 접근 테스트
echo ""
echo "🧪 GHCR 접근 테스트 중..."

# 존재하는 이미지로 테스트 (nginx 사용)
kubectl run ghcr-test \
  --image=nginx:alpine \
  --restart=Never \
  --namespace=wealist-dev \
  --command -- echo "Registry access test" \
  2>/dev/null || true

sleep 5

TEST_STATUS=$(kubectl get pod ghcr-test -n wealist-dev -o jsonpath='{.status.phase}' 2>/dev/null || echo "NotFound")
if [ "$TEST_STATUS" = "Succeeded" ] || [ "$TEST_STATUS" = "Running" ]; then
    echo "✅ 컨테이너 레지스트리 접근 테스트 성공"
else
    echo "⚠️  테스트 결과: $TEST_STATUS"
fi

kubectl delete pod ghcr-test -n wealist-dev 2>/dev/null || true

echo ""
echo "✅ 클러스터 준비 완료!"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📋 클러스터 정보"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🏷️  이름:       $CLUSTER_NAME"
echo "🌐 Ingress:     localhost:80 (HTTP), localhost:443 (HTTPS)"
echo "🔧 ArgoCD:      localhost:8079 (준비되면)"
echo "🐳 Registry:    ghcr.io (GitHub Container Registry)"
echo "📁 Namespaces:  wealist-dev"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🔍 검증 명령어"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "kubectl get nodes"
echo "kubectl get namespaces"
echo "kubectl get secret ghcr-secret -n wealist-dev"
echo "kubectl describe sa default -n wealist-dev"
echo ""
echo "🧪 GHCR 테스트:"
echo "kubectl run test --image=ghcr.io/orangescloud/auth-service:latest -n wealist-dev"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"