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

# RSA 키 입력 방식 선택
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}RSA Key Configuration:${NC}"
echo ""
echo "Choose RSA key input method:"
echo "  1. Use JWT_SECRET for both private and public keys (simple)"
echo "  2. Enter separate RSA private and public keys (recommended)"
echo "  3. Generate new RSA key pair automatically"
echo ""
read -p "Select option [1/2/3]: " RSA_OPTION
echo ""

case $RSA_OPTION in
    2)
        echo "Enter RSA Private Key (multi-line, end with Ctrl+D on empty line):"
        echo "Example format:"
        echo "-----BEGIN RSA PRIVATE KEY-----"
        echo "MIIEpAIBAAKCAQEA..."
        echo "-----END RSA PRIVATE KEY-----"
        echo ""
        JWT_RSA_PRIVATE_KEY=$(cat)
        echo ""
        
        echo "Enter RSA Public Key (multi-line, end with Ctrl+D on empty line):"
        echo "Example format:"
        echo "-----BEGIN PUBLIC KEY-----"
        echo "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8A..."
        echo "-----END PUBLIC KEY-----"
        echo ""
        JWT_RSA_PUBLIC_KEY=$(cat)
        echo ""
        ;;
    3)
        echo "Generating new RSA key pair (2048 bit)..."
        TEMP_KEY_DIR="/tmp/wealist-rsa-keys-$$"
        mkdir -p "$TEMP_KEY_DIR"
        
        # Generate private key
        openssl genrsa -out "$TEMP_KEY_DIR/private.pem" 2048 2>/dev/null
        # Extract public key
        openssl rsa -in "$TEMP_KEY_DIR/private.pem" -pubout -out "$TEMP_KEY_DIR/public.pem" 2>/dev/null
        
        JWT_RSA_PRIVATE_KEY=$(cat "$TEMP_KEY_DIR/private.pem")
        JWT_RSA_PUBLIC_KEY=$(cat "$TEMP_KEY_DIR/public.pem")
        
        echo -e "${GREEN}✅ RSA key pair generated!${NC}"
        echo ""
        echo "Private Key:"
        echo "$JWT_RSA_PRIVATE_KEY"
        echo ""
        echo "Public Key:"
        echo "$JWT_RSA_PUBLIC_KEY"
        echo ""
        
        # Backup keys
        BACKUP_KEY_FILE="rsa-keys-backup-$(date +%Y%m%d-%H%M%S).txt"
        cat > "$BACKUP_KEY_FILE" <<KEYEOF
# RSA Keys for Wealist ${ENVIRONMENT}
# Generated: $(date)
# KEEP THIS FILE SECURE!

=== PRIVATE KEY ===
$JWT_RSA_PRIVATE_KEY

=== PUBLIC KEY ===
$JWT_RSA_PUBLIC_KEY
KEYEOF
        
        echo -e "${YELLOW}⚠️  Keys backed up to: ${BACKUP_KEY_FILE}${NC}"
        echo -e "${RED}   IMPORTANT: Store this file securely and delete after saving!${NC}"
        echo ""
        
        # Clean up temp directory
        rm -rf "$TEMP_KEY_DIR"
        ;;
    *)
        echo "Using JWT_SECRET for both RSA keys (simple mode)"
        JWT_RSA_PRIVATE_KEY="${JWT_SECRET}"
        JWT_RSA_PUBLIC_KEY="${JWT_SECRET}"
        ;;
esac

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
  JWT_RSA_PRIVATE_KEY: |
$(echo "${JWT_RSA_PRIVATE_KEY}" | sed 's/^/    /')
  JWT_RSA_PUBLIC_KEY: |
$(echo "${JWT_RSA_PUBLIC_KEY}" | sed 's/^/    /')
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
echo "  - ${BACKUP_FILE} (Sealed Secrets private key - KEEP SECURE!)"
if [ -n "$BACKUP_KEY_FILE" ]; then
    echo "  - ${BACKUP_KEY_FILE} (RSA key pair backup - KEEP SECURE!)"
fi
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
echo "3. ${RED}CRITICAL: Backup the private keys securely!${NC}"
echo "   Files:"
echo "   - ${BACKUP_FILE} (Sealed Secrets controller key)"
if [ -n "$BACKUP_KEY_FILE" ]; then
    echo "   - ${BACKUP_KEY_FILE} (JWT RSA keys)"
fi
echo ""
echo "   Recommended secure storage options:"
echo "   a) Password Manager (1Password/LastPass/Bitwarden)"
echo "   b) AWS Secrets Manager / Google Secret Manager"
echo "   c) Encrypted backup:"
echo "      gpg --symmetric --cipher-algo AES256 ${BACKUP_FILE}"
if [ -n "$BACKUP_KEY_FILE" ]; then
    echo "      gpg --symmetric --cipher-algo AES256 ${BACKUP_KEY_FILE}"
fi
echo "      # Store .gpg files in secure private location"
echo "      # Delete unencrypted files"
echo ""
echo "4. Add key files to .gitignore:"
echo "   echo '*.key' >> .gitignore"
echo "   echo 'sealed-secrets-*.key' >> .gitignore"
echo "   echo 'rsa-keys-backup-*.txt' >> .gitignore"
echo "   git add .gitignore"
echo "   git commit -m 'ignore sealed-secrets key files'"
echo ""
echo -e "${YELLOW}⚠️  CRITICAL WARNINGS:${NC}"
echo "   - ${RED}NEVER commit ${BACKUP_FILE} to Git in plain text!${NC}"
if [ -n "$BACKUP_KEY_FILE" ]; then
    echo "   - ${RED}NEVER commit ${BACKUP_KEY_FILE} to Git in plain text!${NC}"
fi
echo "   - Without these keys, you cannot decrypt existing secrets"
echo "   - You'll need these keys when recreating the cluster"
echo "   - Store them in at least 2 secure locations"
echo ""
echo -e "${BLUE}🔍 Verify Everything:${NC}"
echo "   kubectl get secrets -n ${NAMESPACE}"
echo "   kubectl get sealedsecrets -n ${NAMESPACE}"
echo "   kubectl describe sealedsecret wealist-argocd-secret -n ${NAMESPACE}"
echo ""
echo "   # Verify RSA keys in secret:"
echo "   kubectl get secret wealist-argocd-secret -n ${NAMESPACE} -o jsonpath='{.data.JWT_RSA_PRIVATE_KEY}' | base64 -d | head -1"
echo "   kubectl get secret wealist-argocd-secret -n ${NAMESPACE} -o jsonpath='{.data.JWT_RSA_PUBLIC_KEY}' | base64 -d | head -1"
echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "${BLUE}💡 For future cluster migrations:${NC}"
echo "   Use this key file: ${BACKUP_FILE}"
echo "   Run: kubectl apply -f ${BACKUP_FILE}"
echo "   Then restart sealed-secrets controller"
echo ""