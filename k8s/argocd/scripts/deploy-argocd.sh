#!/bin/bash
set -e

echo "🚀 Starting ArgoCD deployment with Sealed Secrets..."

# 색상
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# GitHub 저장소 정보
REPO_URL="https://github.com/OrangesCloud/wealist-argo-helm.git"
SEALED_SECRETS_KEY="${1:-sealed-secrets-dev-20251218-121235.key}"

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Wealist Platform Deployment"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# ============================================
# 0. Sealed Secrets 키 확인
# ============================================
echo -e "${YELLOW}🔑 Step 0: Checking Sealed Secrets key...${NC}"

if [ -f "$SEALED_SECRETS_KEY" ]; then
    echo -e "${GREEN}✅ Found key backup: $SEALED_SECRETS_KEY${NC}"
    USE_EXISTING_KEY=true
else
    echo -e "${YELLOW}⚠️  Key file not found: $SEALED_SECRETS_KEY${NC}"
    echo ""
    echo "Options:"
    echo "  1) Provide key file path"
    echo "  2) Continue without key (new key will be generated)"
    echo ""
    read -p "Choose (1/2): " -n 1 -r
    echo ""
    
    if [[ $REPLY == "1" ]]; then
        read -p "Enter key file path: " SEALED_SECRETS_KEY
        if [ -f "$SEALED_SECRETS_KEY" ]; then
            USE_EXISTING_KEY=true
        else
            echo -e "${RED}❌ File not found: $SEALED_SECRETS_KEY${NC}"
            exit 1
        fi
    else
        echo -e "${YELLOW}⚠️  Proceeding without key backup${NC}"
        echo -e "${YELLOW}    New keys will be generated${NC}"
        echo -e "${YELLOW}    Existing SealedSecrets will NOT work!${NC}"
        USE_EXISTING_KEY=false
    fi
fi
echo ""

# ============================================
# 1. ArgoCD 설치
# ============================================
echo -e "${YELLOW}📦 Step 1: Installing ArgoCD...${NC}"
kubectl create namespace argocd --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
echo -e "${GREEN}✅ ArgoCD installed${NC}"
echo ""

# ============================================
# 2. Sealed Secrets 키 복원 (있으면)
# ============================================
if [ "$USE_EXISTING_KEY" = true ]; then
    echo -e "${YELLOW}🔑 Step 2: Restoring Sealed Secrets key...${NC}"
    
    # 기존 키 삭제 (있다면)
    kubectl delete secret -n kube-system -l sealedsecrets.bitnami.com/sealed-secrets-key 2>/dev/null || true
    
    # 키 복원
    kubectl create -f "$SEALED_SECRETS_KEY"
    echo -e "${GREEN}✅ Key restored from backup${NC}"
else
    echo -e "${YELLOW}⏭️  Step 2: Skipping key restoration${NC}"
fi
echo ""

# ============================================
# 3. Sealed Secrets Controller 설치
# ============================================
echo -e "${YELLOW}🔐 Step 3: Installing Sealed Secrets Controller...${NC}"
helm repo add sealed-secrets https://bitnami-labs.github.io/sealed-secrets 2>/dev/null || true
helm repo update

helm upgrade --install sealed-secrets sealed-secrets/sealed-secrets \
  -n kube-system \
  --set fullnameOverride=sealed-secrets \
  --wait --timeout=300s
echo -e "${GREEN}✅ Controller installed${NC}"
echo ""

# ============================================
# 4. Controller 재시작 (키 로드)
# ============================================
if [ "$USE_EXISTING_KEY" = true ]; then
    echo -e "${YELLOW}🔄 Step 4: Restarting controller to load key...${NC}"
    kubectl delete pod -n kube-system -l app.kubernetes.io/name=sealed-secrets 2>/dev/null || true
    sleep 5
    kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=sealed-secrets -n kube-system --timeout=300s
    echo -e "${GREEN}✅ Controller ready with restored key${NC}"
else
    echo -e "${YELLOW}⏭️  Step 4: Controller ready with new key${NC}"
fi
echo ""

# ============================================
# 5. ArgoCD 준비 대기
# ============================================
echo -e "${YELLOW}⏳ Step 5: Waiting for ArgoCD server...${NC}"
kubectl wait --for=condition=available --timeout=600s deployment/argocd-server -n argocd
echo -e "${GREEN}✅ ArgoCD ready${NC}"
echo ""

# ============================================
# 6. 네임스페이스 생성
# ============================================
echo -e "${YELLOW}📁 Step 6: Creating application namespace...${NC}"
kubectl create namespace wealist-dev --dry-run=client -o yaml | kubectl apply -f -
echo -e "${GREEN}✅ Namespace created${NC}"
echo ""

# ============================================
# 7. CRD 확인
# ============================================
echo -e "${YELLOW}🔍 Step 7: Verifying Sealed Secrets CRD...${NC}"
if kubectl get crd sealedsecrets.bitnami.com &> /dev/null; then
    echo -e "${GREEN}✅ CRD verified${NC}"
else
    echo -e "${RED}❌ CRD not found${NC}"
    exit 1
fi
echo ""

# ============================================
# 8. SealedSecret 적용
# ============================================
echo -e "${YELLOW}🔐 Step 8: Applying SealedSecrets...${NC}"

# 프로젝트 루트 기준 경로
SEALED_SECRET_FILE="k8s/argocd/scripts/secret/sealed-secret-dev.yaml"

if [ -f "$SEALED_SECRET_FILE" ]; then
    kubectl apply -f "$SEALED_SECRET_FILE"
    echo -e "${GREEN}✅ SealedSecret applied${NC}"
    
    # 복호화 확인
    echo "⏳ Waiting for decryption..."
    sleep 15
    
    if kubectl get secret wealist-argocd-secret -n wealist-dev &> /dev/null; then
        echo -e "${GREEN}✅ Secret successfully decrypted!${NC}"
    else
        echo -e "${RED}❌ Failed to decrypt secret: wealist-argocd-secret${NC}"
        echo ""
        echo "Checking SealedSecret status..."
        kubectl describe sealedsecret wealist-argocd-secret -n wealist-dev 2>/dev/null || true
        
        if [ "$USE_EXISTING_KEY" = false ]; then
            echo ""
            echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
            echo -e "${YELLOW}⚠️  This is EXPECTED with new keys!${NC}"
            echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
            echo ""
            echo "You need to re-seal the secrets with the new key:"
            echo "  cd k8s/helm/charts/wealist-infrastructure/templates"
            echo "  # Create plain secret, then:"
            echo "  kubeseal -f secret.yaml -w sealed-secret-dev.yaml \\"
            echo "    --controller-namespace=kube-system \\"
            echo "    --controller-name=sealed-secrets"
            echo ""
        else
            echo ""
            echo -e "${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
            echo -e "${RED}⚠️  DECRYPTION FAILED WITH RESTORED KEY!${NC}"
            echo -e "${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
            echo ""
            echo "Possible causes:"
            echo "  1. Wrong key file was provided"
            echo "  2. SealedSecret was encrypted with a different key"
            echo "  3. Controller not using the restored key"
            echo ""
            echo "Troubleshooting:"
            echo "  # Check controller logs:"
            echo "  kubectl logs -n kube-system -l app.kubernetes.io/name=sealed-secrets"
            echo ""
            echo "  # Verify key fingerprint:"
            echo "  kubeseal --fetch-cert --controller-namespace=kube-system"
            echo ""
            exit 1
        fi
    fi
else
    echo -e "${YELLOW}⚠️  SealedSecret file not found: $SEALED_SECRET_FILE${NC}"
fi
echo ""

# ============================================
# 8.5. ArgoCD SealedSecret 적용
# ============================================
echo -e "${YELLOW}🔐 Step 8.5: Applying ArgoCD SealedSecret...${NC}"
ARGOCD_SEALED_SECRET="k8s/argocd/sealed-secrets/wealist-argocd-secret.yaml"
if [ -f "$ARGOCD_SEALED_SECRET" ]; then
    kubectl apply -f "$ARGOCD_SEALED_SECRET"
    echo -e "${GREEN}✅ ArgoCD SealedSecret applied${NC}"
    
    # 복호화 확인
    echo "⏳ Waiting for ArgoCD secret decryption..."
    sleep 10
    
    if kubectl get secret wealist-argocd-secret -n wealist-dev &> /dev/null; then
        echo -e "${GREEN}✅ ArgoCD secret successfully decrypted!${NC}"
    else
        echo -e "${RED}❌ Failed to decrypt secret: wealist-argocd-secret${NC}"
        
        if [ "$USE_EXISTING_KEY" = false ]; then
            echo -e "${YELLOW}⚠️  This is expected with new keys - you need to re-seal this secret too${NC}"
        else
            echo -e "${RED}⚠️  Decryption failed with restored key${NC}"
            echo "This secret may have been encrypted with a different key"
        fi
    fi
else
    echo -e "${YELLOW}⚠️  ArgoCD SealedSecret file not found: $ARGOCD_SEALED_SECRET${NC}"
fi
echo ""

# ============================================
# 9. GitHub 저장소 인증
# ============================================
echo -e "${YELLOW}🔑 Step 9: Setting up GitHub repository access...${NC}"
echo ""
read -p "Enter your GitHub username: " GITHUB_USERNAME
echo -n "Enter your GitHub Personal Access Token: "
read -s GITHUB_TOKEN
echo ""

kubectl create secret generic wealist-repo -n argocd \
  --from-literal=type=git \
  --from-literal=url=$REPO_URL \
  --from-literal=username=$GITHUB_USERNAME \
  --from-literal=password=$GITHUB_TOKEN \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl label secret wealist-repo -n argocd \
  argocd.argoproj.io/secret-type=repository --overwrite

echo -e "${GREEN}✅ Repository configured${NC}"
echo ""

# ============================================
# 10. ArgoCD 추가 대기
# ============================================
echo -e "${YELLOW}⏳ Step 10: Final preparations...${NC}"
sleep 10
echo -e "${GREEN}✅ Ready${NC}"
echo ""

# ============================================
# 11. AppProject 생성
# ============================================
echo -e "${YELLOW}🎯 Step 11: Creating AppProject...${NC}"
PROJECT_FILE="k8s/argocd/apps/project.yaml"
if [ -f "$PROJECT_FILE" ]; then
    kubectl apply -f "$PROJECT_FILE"
    echo -e "${GREEN}✅ AppProject created${NC}"
else
    echo -e "${YELLOW}⚠️  Project file not found: $PROJECT_FILE${NC}"
fi
echo ""

# ============================================
# 12. Root Application 생성
# ============================================
echo -e "${YELLOW}🌟 Step 12: Creating Root Application...${NC}"
ROOT_APP_FILE="k8s/argocd/apps/root-app.yaml"
if [ -f "$ROOT_APP_FILE" ]; then
    kubectl apply -f "$ROOT_APP_FILE"
    echo -e "${GREEN}✅ Root Application created${NC}"
else
    echo -e "${YELLOW}⚠️  Root app file not found: $ROOT_APP_FILE${NC}"
fi
echo ""

# ============================================
# 13. 새 키 백업 (새로 생성된 경우)
# ============================================
if [ "$USE_EXISTING_KEY" = false ]; then
    echo -e "${YELLOW}💾 Step 13: Backing up new keys...${NC}"
    NEW_KEY_FILE="sealed-secrets-new-$(date +%Y%m%d-%H%M%S).key"
    kubectl get secret -n kube-system -l sealedsecrets.bitnami.com/sealed-secrets-key -o yaml > "$NEW_KEY_FILE"
    echo -e "${GREEN}✅ New key backed up: $NEW_KEY_FILE${NC}"
    echo -e "${RED}⚠️  IMPORTANT: Store this file securely!${NC}"
else
    echo -e "${YELLOW}⏭️  Step 13: Using existing key (no backup needed)${NC}"
fi
echo ""
# kubectl patch secret wealist-argocd-secret -n wealist-dev --type='merge' -p='{"data":{"S3_ACCESS_KEY":"bWluaW9hZG1pbg==","S3_SECRET_KEY":"bWluaW9hZG1pbg=="}}'

# ============================================
# 14. ArgoCD 비밀번호
# ============================================
ARGOCD_PASSWORD=$(kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d 2>/dev/null || echo "Password not found")

# ============================================
# 최종 정보
# ============================================
echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✅ Deployment Complete!${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo "🌐 ArgoCD Access:"
echo "   URL:      https://localhost:8079"
echo "   Username: admin"
echo "   Password: $ARGOCD_PASSWORD"
echo ""
echo "🔐 Sealed Secrets:"
echo "   Controller: sealed-secrets (kube-system)"
if [ "$USE_EXISTING_KEY" = true ]; then
    echo "   Key:        Restored from backup ✅"
else
    echo "   Key:        Newly generated ⚠️"
    echo "   Backup:     $NEW_KEY_FILE"
fi
echo ""
echo "🔍 Verification:"
echo "   kubectl get applications -n argocd"
echo "   kubectl get pods -n wealist-dev"
echo "   kubectl get sealedsecrets -n wealist-dev"
echo "   kubectl get secrets -n wealist-dev"
echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo "🌐 Starting port-forward..."
kubectl port-forward svc/argocd-server -n argocd 8079:443