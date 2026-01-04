#!/bin/bash
# =============================================================================
# 팀원용 제한된 kubeconfig 생성
# =============================================================================
# wealist-dev 네임스페이스만 접근 가능한 kubeconfig 파일 생성
#
# 사용법:
#   ./scripts/create-team-kubeconfig.sh [username]
#
# 예시:
#   ./scripts/create-team-kubeconfig.sh member1
#   → ~/.kube/config-member1 생성
#
# 팀원 사용법:
#   export KUBECONFIG=~/.kube/config-member1
#   kubectl get pods  # wealist-dev 네임스페이스만 접근 가능
# =============================================================================

set -e

USERNAME="${1:-team-developer}"
NAMESPACE="wealist-dev"
SERVICE_ACCOUNT="team-developer"
CLUSTER_NAME="wealist"
OUTPUT_DIR="${HOME}/.kube"
OUTPUT_FILE="${OUTPUT_DIR}/config-${USERNAME}"

echo "🔐 팀원용 kubeconfig 생성: ${USERNAME}"
echo "   - Namespace: ${NAMESPACE}"
echo "   - ServiceAccount: ${SERVICE_ACCOUNT}"
echo ""

# RBAC 리소스가 있는지 확인
if ! kubectl get serviceaccount ${SERVICE_ACCOUNT} -n ${NAMESPACE} &>/dev/null; then
    echo "⚠️  ServiceAccount '${SERVICE_ACCOUNT}'가 없습니다. RBAC 먼저 적용하세요:"
    echo "   kubectl apply -f k8s/rbac/team-developer.yaml"
    exit 1
fi

# 출력 디렉토리 생성
mkdir -p "${OUTPUT_DIR}"

# ServiceAccount 토큰 시크릿 생성 (K8s 1.24+에서는 자동 생성 안 됨)
echo "🔑 ServiceAccount 토큰 생성 중..."
kubectl apply -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: ${SERVICE_ACCOUNT}-token
  namespace: ${NAMESPACE}
  annotations:
    kubernetes.io/service-account.name: ${SERVICE_ACCOUNT}
type: kubernetes.io/service-account-token
EOF

# 토큰이 생성될 때까지 대기
echo "⏳ 토큰 준비 대기 중..."
for i in {1..30}; do
    TOKEN=$(kubectl get secret ${SERVICE_ACCOUNT}-token -n ${NAMESPACE} -o jsonpath='{.data.token}' 2>/dev/null | base64 -d)
    if [ -n "${TOKEN}" ]; then
        break
    fi
    sleep 1
done

if [ -z "${TOKEN}" ]; then
    echo "❌ 토큰 생성 실패"
    exit 1
fi

# 클러스터 정보 가져오기
CLUSTER_SERVER=$(kubectl config view --minify -o jsonpath='{.clusters[0].cluster.server}')
CLUSTER_CA=$(kubectl config view --minify --raw -o jsonpath='{.clusters[0].cluster.certificate-authority-data}')

# kubeconfig 생성
echo "📝 kubeconfig 파일 생성 중..."
cat > "${OUTPUT_FILE}" <<EOF
apiVersion: v1
kind: Config
current-context: ${USERNAME}@${CLUSTER_NAME}
clusters:
  - name: ${CLUSTER_NAME}
    cluster:
      server: ${CLUSTER_SERVER}
      certificate-authority-data: ${CLUSTER_CA}
contexts:
  - name: ${USERNAME}@${CLUSTER_NAME}
    context:
      cluster: ${CLUSTER_NAME}
      user: ${USERNAME}
      namespace: ${NAMESPACE}
users:
  - name: ${USERNAME}
    user:
      token: ${TOKEN}
EOF

# 파일 권한 설정
chmod 600 "${OUTPUT_FILE}"

echo ""
echo "=============================================="
echo "  ✅ kubeconfig 생성 완료!"
echo "=============================================="
echo ""
echo "📁 파일: ${OUTPUT_FILE}"
echo ""
echo "🔧 사용 방법:"
echo ""
echo "   # 방법 1: 환경변수로 설정"
echo "   export KUBECONFIG=${OUTPUT_FILE}"
echo "   kubectl get pods"
echo ""
echo "   # 방법 2: --kubeconfig 옵션 사용"
echo "   kubectl --kubeconfig=${OUTPUT_FILE} get pods"
echo ""
echo "   # 방법 3: .bashrc에 추가 (영구 설정)"
echo "   echo 'export KUBECONFIG=${OUTPUT_FILE}' >> ~/.bashrc"
echo ""
echo "🔒 권한 범위:"
echo "   - ✅ wealist-dev 네임스페이스: 읽기, 로그, exec"
echo "   - ❌ 다른 네임스페이스: 접근 불가"
echo "   - ❌ Docker 컨테이너: 접근 불가"
echo "=============================================="
