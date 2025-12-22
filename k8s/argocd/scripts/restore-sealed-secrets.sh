#!/bin/bash
set -e

echo "🔐 Re-sealing secrets for new cluster..."
echo ""

# 색상
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

ENVIRONMENT=${1:-dev}
NAMESPACE="wealist-${ENVIRONMENT}"

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}  Wealist Secrets Re-sealing${NC}"
echo -e "${BLUE}  Environment: ${ENVIRONMENT}${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# 1. Controller 확인
echo -e "${YELLOW}1️⃣ Verifying Sealed Secrets Controller...${NC}"
if ! kubectl get deployment sealed-secrets -n kube-system &> /dev/null; then
    echo -e "${RED}❌ Controller not found!${NC}"
    echo "Installing controller..."
    helm repo add sealed-secrets https://bitnami-labs.github.io/sealed-secrets 2>/dev/null || true
    helm repo update
    helm upgrade --install sealed-secrets sealed-secrets/sealed-secrets \
      -n kube-system \
      --set fullnameOverride=sealed-secrets \
      --wait --timeout=300s
fi
echo -e "${GREEN}✅ Controller ready${NC}"
echo ""

# 2. 현재 클러스터의 공개키 확인
echo -e "${YELLOW}2️⃣ Fetching current cluster certificate...${NC}"
kubeseal --fetch-cert \
  --controller-namespace=kube-system \
  --controller-name=sealed-secrets \
  > /tmp/current-cluster-cert.pem

echo "Current cluster public key fingerprint:"
cat /tmp/current-cluster-cert.pem | head -5
echo "..."
echo -e "${GREEN}✅ Certificate fetched${NC}"
echo ""

# 3. 평문 Secret 값 입력
echo -e "${YELLOW}3️⃣ Enter secret values:${NC}"
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

read -p "GIT_ACCESS (GitHub Personal Access Token): " GIT_ACCESS
read -p "GIT_NAME (Your Git username or name): " GIT_NAME
echo ""
read -p "GOOGLE_CLIENT_ID: " GOOGLE_CLIENT_ID
read -sp "GOOGLE_CLIENT_SECRET: " GOOGLE_CLIENT_SECRET
echo ""
echo ""
read -sp "JWT_SECRET (for JWT token signing): " JWT_SECRET
echo ""
echo ""

# 4. 임시 평문 Secret 생성
echo -e "${YELLOW}4️⃣ Creating temporary plain secret...${NC}"
TEMP_SECRET="/tmp/wealist-argocd-secret-${ENVIRONMENT}-$(date +%s).yaml"

cat > $TEMP_SECRET <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: wealist-argocd-secret
  namespace: ${NAMESPACE}
  labels:
    app.kubernetes.io/component: shared-secret
    app.kubernetes.io/name: wealist-argocd-secrets
type: Opaque
stringData:
  GIT_ACCESS: "${GIT_ACCESS}"
  GIT_NAME: "${GIT_NAME}"
  GOOGLE_CLIENT_ID: "${GOOGLE_CLIENT_ID}"
  GOOGLE_CLIENT_SECRET: "${GOOGLE_CLIENT_SECRET}"
  JWT_SECRET: "${JWT_SECRET}"
EOF

echo -e "${GREEN}✅ Temporary secret created${NC}"
echo ""

# 5. SealedSecret으로 암호화
echo -e "${YELLOW}5️⃣ Sealing secret with current cluster key...${NC}"

# 프로젝트 루트 찾기
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
OUTPUT_FILE="${PROJECT_ROOT}/k8s/argocd/sealed-secrets/wealist-argocd-secret.yaml"

echo "Project root: ${PROJECT_ROOT}"
echo "Output file: ${OUTPUT_FILE}"
echo ""

mkdir -p "$(dirname "$OUTPUT_FILE")"

kubeseal -f $TEMP_SECRET \
  -w $OUTPUT_FILE \
  --controller-namespace=kube-system \
  --controller-name=sealed-secrets \
  --format yaml

echo -e "${GREEN}✅ Secret sealed and saved to: ${OUTPUT_FILE}${NC}"
echo ""

# 6. 임시 파일 안전하게 삭제
echo -e "${YELLOW}6️⃣ Cleaning up temporary files...${NC}"
shred -u $TEMP_SECRET 2>/dev/null || rm -f $TEMP_SECRET
rm -f /tmp/current-cluster-cert.pem
echo -e "${GREEN}✅ Temporary files deleted${NC}"
echo ""

# 7. Namespace 생성 (없으면)
echo -e "${YELLOW}7️⃣ Creating namespace...${NC}"
kubectl create namespace $NAMESPACE --dry-run=client -o yaml | kubectl apply -f -
echo -e "${GREEN}✅ Namespace ready${NC}"
echo ""

# 8. SealedSecret 적용
echo -e "${YELLOW}8️⃣ Applying SealedSecret to cluster...${NC}"
kubectl apply -f $OUTPUT_FILE
echo -e "${GREEN}✅ SealedSecret applied${NC}"
echo ""

# 9. 복호화 대기 및 확인
echo -e "${YELLOW}9️⃣ Waiting for decryption...${NC}"
echo "This may take up to 30 seconds..."

SUCCESS=false
for i in {1..30}; do
    if kubectl get secret wealist-argocd-secret -n $NAMESPACE &> /dev/null; then
        echo ""
        echo -e "${GREEN}✅ SUCCESS! Secret decrypted successfully!${NC}"
        echo ""
        kubectl get secret wealist-argocd-secret -n $NAMESPACE
        SUCCESS=true
        break
    fi
    echo -n "."
    sleep 1
done

if [ "$SUCCESS" = false ]; then
    echo ""
    echo -e "${RED}❌ Secret not created after 30 seconds${NC}"
    echo ""
    echo "Debugging information:"
    echo ""
    echo "SealedSecret status:"
    kubectl describe sealedsecret wealist-argocd-secret -n $NAMESPACE 2>/dev/null || echo "Not found"
    echo ""
    echo "Controller logs:"
    kubectl logs -n kube-system -l app.kubernetes.io/name=sealed-secrets --tail=20
    exit 1
fi

echo ""

# 10. 현재 클러스터 키 백업
echo -e "${YELLOW}🔟 Backing up current cluster key...${NC}"
BACKUP_FILE="sealed-secrets-${ENVIRONMENT}-$(date +%Y%m%d-%H%M%S).key"
kubectl get secret -n kube-system \
  -l sealedsecrets.bitnami.com/sealed-secrets-key \
  -o yaml > $BACKUP_FILE

echo -e "${GREEN}✅ Key backed up to: ${BACKUP_FILE}${NC}"
echo ""

# 11. 완료 및 다음 단계
echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✅ Re-sealing Complete!${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "${BLUE}📝 Files Created:${NC}"
echo "  - ${OUTPUT_FILE} (SealedSecret for cluster)"
echo "  - ${BACKUP_FILE} (Private key backup - KEEP SECURE!)"
echo ""
echo -e "${BLUE}📝 Next Steps:${NC}"
echo ""
echo "1. Review the sealed secret:"
echo "   cat ${OUTPUT_FILE}"
echo ""
echo "2. Commit to Git:"
echo "   git add ${OUTPUT_FILE}"
echo "   git commit -m 're-seal secrets for ${ENVIRONMENT} with new cluster key'"
echo "   git push"
echo ""
echo "3. ${RED}CRITICAL: Backup the private key securely!${NC}"
echo "   File: ${BACKUP_FILE}"
echo ""
echo "   Recommended secure storage options:"
echo "   a) Password Manager (1Password/LastPass/Bitwarden)"
echo "   b) AWS Secrets Manager / Google Secret Manager"
echo "   c) Encrypted backup:"
echo "      gpg --symmetric --cipher-algo AES256 ${BACKUP_FILE}"
echo "      # Store ${BACKUP_FILE}.gpg in secure private location"
echo "      # Delete unencrypted ${BACKUP_FILE}"
echo ""
echo "4. Add key files to .gitignore:"
echo "   echo '*.key' >> .gitignore"
echo "   echo 'sealed-secrets-*.key' >> .gitignore"
echo "   git add .gitignore"
echo "   git commit -m 'ignore sealed-secrets key files'"
echo ""
echo -e "${YELLOW}⚠️  CRITICAL WARNINGS:${NC}"
echo "   - ${RED}NEVER commit ${BACKUP_FILE} to Git in plain text!${NC}"
echo "   - Without this key, you cannot decrypt existing SealedSecrets"
echo "   - You'll need this key when recreating the cluster"
echo "   - Store it in at least 2 secure locations"
echo ""
echo -e "${BLUE}🔍 Verify Everything:${NC}"
echo "   kubectl get secrets -n ${NAMESPACE}"
echo "   kubectl get sealedsecrets -n ${NAMESPACE}"
echo "   kubectl describe sealedsecret wealist-argocd-secret -n ${NAMESPACE}"
echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "${BLUE}💡 For future cluster migrations:${NC}"
echo "   Use this key file: ${BACKUP_FILE}"
echo "   Run: kubectl apply -f ${BACKUP_FILE}"
echo "   Then restart sealed-secrets controller"
echo ""