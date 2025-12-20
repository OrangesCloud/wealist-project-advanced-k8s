#!/bin/bash
set -e

echo "🚀 Starting ArgoCD deployment with Sealed Secrets..."

# 색상
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# 스크립트 실행 위치 저장
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ARGOCD_DIR="$(dirname "$SCRIPT_DIR")"  # k8s/argocd
PROJECT_ROOT="$(dirname "$(dirname "$ARGOCD_DIR")")"  # 프로젝트 루트

# GitHub 저장소 정보
REPO_URL="https://github.com/OrangesCloud/wealist-project-advanced-k8s.git"
SEALED_SECRETS_KEY="${1:-$SCRIPT_DIR/sealed-secrets-dev-20251218-152119.key}"

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Wealist Platform Deployment"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "📂 Paths:"
echo "   Script:  $SCRIPT_DIR"
echo "   ArgoCD:  $ARGOCD_DIR"
echo "   Root:    $PROJECT_ROOT"
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
# 8. GitHub 인증 정보 수집
# ============================================
echo -e "${YELLOW}🔑 Step 8: Collecting GitHub credentials...${NC}"
echo ""
echo "Enter GitHub credentials (for repository access AND container registry):"
echo ""
read -p "Enter your GitHub username: " GITHUB_USERNAME
echo -n "Enter your GitHub Personal Access Token (with repo and read:packages permissions): "
read -s GITHUB_TOKEN
echo ""
echo ""

# 입력값 검증
if [ -z "$GITHUB_USERNAME" ] || [ -z "$GITHUB_TOKEN" ]; then
    echo -e "${RED}❌ GitHub credentials are required${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Credentials collected${NC}"
echo ""

# ============================================
# 9. GHCR (GitHub Container Registry) 설정
# ============================================
echo -e "${YELLOW}🐳 Step 9: Setting up GitHub Container Registry access...${NC}"

# wealist-dev 네임스페이스에 GHCR secret 생성
kubectl create secret docker-registry ghcr-secret \
  --docker-server=ghcr.io \
  --docker-username="$GITHUB_USERNAME" \
  --docker-password="$GITHUB_TOKEN" \
  --docker-email="$GITHUB_USERNAME@users.noreply.github.com" \
  --namespace=wealist-dev \
  --dry-run=client -o yaml | kubectl apply -f -

# default ServiceAccount에 imagePullSecrets 추가
kubectl patch serviceaccount default \
  -p '{"imagePullSecrets": [{"name": "ghcr-secret"}]}' \
  -n wealist-dev

# 모든 서비스 ServiceAccount에 imagePullSecrets 추가
SERVICE_ACCOUNTS=("auth-service" "board-service" "chat-service" "noti-service" "storage-service" "user-service" "video-service")
for sa in "${SERVICE_ACCOUNTS[@]}"; do
  # ServiceAccount가 존재하는지 확인 후 패치
  if kubectl get serviceaccount "$sa" -n wealist-dev &>/dev/null; then
    echo "  Patching ServiceAccount: $sa"
    kubectl patch serviceaccount "$sa" \
      -p '{"imagePullSecrets": [{"name": "ghcr-secret"}]}' \
      -n wealist-dev
  else
    echo "  ServiceAccount not found (will be created by Helm): $sa"
  fi
done

echo -e "${GREEN}✅ GHCR access configured${NC}"
echo "   📦 Secret created: ghcr-secret"
echo "   🔗 Linked to default ServiceAccount"
echo ""

# GHCR 접근 테스트 (선택사항)
echo -e "${YELLOW}🧪 Testing GHCR access...${NC}"
TEST_POD=$(cat <<EOFTEST
apiVersion: v1
kind: Pod
metadata:
  name: ghcr-test
  namespace: wealist-dev
spec:
  restartPolicy: Never
  containers:
  - name: test
    image: ghcr.io/orangescloud/auth-service:latest
    command: ['echo', 'GHCR access test successful']
  imagePullSecrets:
  - name: ghcr-secret
EOFTEST
)

echo "$TEST_POD" | kubectl apply -f - 2>/dev/null || true
sleep 5

# 테스트 결과 확인
if kubectl get pod ghcr-test -n wealist-dev &>/dev/null; then
    POD_STATUS=$(kubectl get pod ghcr-test -n wealist-dev -o jsonpath='{.status.phase}')
    if [ "$POD_STATUS" = "Succeeded" ] || [ "$POD_STATUS" = "Running" ]; then
        echo -e "${GREEN}✅ GHCR access test successful${NC}"
    else
        echo -e "${YELLOW}⚠️  GHCR test inconclusive (Status: $POD_STATUS)${NC}"
        echo "   Check with: kubectl describe pod ghcr-test -n wealist-dev"
    fi
    kubectl delete pod ghcr-test -n wealist-dev 2>/dev/null || true
else
    echo -e "${YELLOW}⚠️  GHCR test pod not found${NC}"
fi
echo ""

# ============================================
# 10. SealedSecret 적용
# ============================================
echo -e "${YELLOW}🔐 Step 10: Applying SealedSecret...${NC}"
SEALED_SECRET_FILE="$ARGOCD_DIR/sealed-secrets/wealist-argocd-secret.yaml"

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
            echo "  kubeseal --fetch-cert \\"
            echo "    --controller-namespace=kube-system > pub-cert.pem"
            echo "  kubeseal -f secret.yaml -w sealed-secret.yaml --cert=pub-cert.pem"
            echo ""
        else
            echo -e "${RED}⚠️  DECRYPTION FAILED WITH RESTORED KEY!${NC}"
        fi
    fi
else
    echo -e "${YELLOW}⚠️  SealedSecret file not found: $SEALED_SECRET_FILE${NC}"
fi
echo ""

# ============================================
# 11. GitHub 저장소 인증
# ============================================
echo -e "${YELLOW}🔗 Step 11: Setting up GitHub repository access...${NC}"

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
# 12. ArgoCD 추가 대기
# ============================================
echo -e "${YELLOW}⏳ Step 12: Final preparations...${NC}"
sleep 10
echo -e "${GREEN}✅ Ready${NC}"
echo ""

# ============================================
# 13. AppProject 생성
# ============================================
echo -e "${YELLOW}🎯 Step 13: Creating AppProject...${NC}"
PROJECT_FILE="$ARGOCD_DIR/apps/project.yaml"

if [ -f "$PROJECT_FILE" ]; then
    kubectl apply -f "$PROJECT_FILE"
    echo -e "${GREEN}✅ AppProject created${NC}"
else
    echo -e "${YELLOW}⚠️  Project file not found: $PROJECT_FILE${NC}"
    echo -e "${YELLOW}   Creating default project...${NC}"
    
    cat <<EOFPROJECT | kubectl apply -f -
apiVersion: argoproj.io/v1alpha1
kind: AppProject
metadata:
  name: wealist
  namespace: argocd
spec:
  description: Wealist Platform
  sourceRepos:
    - 'https://github.com/OrangesCloud/wealist-project-advanced-k8s.git'
  destinations:
    - namespace: 'wealist-*'
      server: https://kubernetes.default.svc
    - namespace: argocd
      server: https://kubernetes.default.svc
  clusterResourceWhitelist:
    - group: '*'
      kind: '*'
EOFPROJECT
    
    echo -e "${GREEN}✅ Default AppProject created${NC}"
fi
echo ""

# ============================================
# 14. Root Application 생성 (App of Apps)
# ============================================
echo -e "${YELLOW}🌟 Step 14: Creating Root Application (App of Apps)...${NC}"
ROOT_APP_FILE="$ARGOCD_DIR/apps/root-app.yaml"

if [ -f "$ROOT_APP_FILE" ]; then
    kubectl apply -f "$ROOT_APP_FILE"
    echo -e "${GREEN}✅ Root Application created${NC}"
    echo -e "${YELLOW}   ⏳ Root app will auto-create all child applications...${NC}"
    sleep 5
else
    echo -e "${YELLOW}⚠️  Root app not found: $ROOT_APP_FILE${NC}"
    echo -e "${YELLOW}   Creating individual applications...${NC}"
    
    # Root app이 없으면 개별 application 적용
    APPS_DIR="$ARGOCD_DIR/apps"
    if [ -d "$APPS_DIR" ]; then
        APPLICATION_COUNT=0
        for app_file in "$APPS_DIR"/*.yaml; do
            if [ -f "$app_file" ]; then
                filename=$(basename "$app_file")
                # project.yaml과 root-app.yaml 제외
                if [[ "$filename" != "project.yaml" ]] && [[ "$filename" != "root-app.yaml" ]]; then
                    echo "  Applying: $filename"
                    kubectl apply -f "$app_file"
                    APPLICATION_COUNT=$((APPLICATION_COUNT + 1))
                fi
            fi
        done
        
        if [ $APPLICATION_COUNT -gt 0 ]; then
            echo -e "${GREEN}✅ Created $APPLICATION_COUNT Application(s)${NC}"
        else
            echo -e "${YELLOW}⚠️  No application files found${NC}"
        fi
    else
        echo -e "${YELLOW}⚠️  Applications directory not found: $APPS_DIR${NC}"
    fi
fi
echo ""

# ============================================
# 15. 새 키 백업
# ============================================
if [ "$USE_EXISTING_KEY" = false ]; then
    echo -e "${YELLOW}💾 Step 15: Backing up new keys...${NC}"
    NEW_KEY_FILE="$SCRIPT_DIR/sealed-secrets-new-$(date +%Y%m%d-%H%M%S).key"
    kubectl get secret -n kube-system -l sealedsecrets.bitnami.com/sealed-secrets-key -o yaml > "$NEW_KEY_FILE"
    echo -e "${GREEN}✅ New key backed up: $NEW_KEY_FILE${NC}"
    echo -e "${RED}⚠️  IMPORTANT: Store this file securely!${NC}"
else
    echo -e "${YELLOW}⏭️  Step 15: Using existing key${NC}"
fi
echo ""

# ============================================
# 16. 추가 네임스페이스에 GHCR Secret 복사 (선택사항)
# ============================================
echo -e "${YELLOW}📋 Step 16: Setting up GHCR access for additional namespaces...${NC}"

# 추가 네임스페이스 목록 (필요에 따라 수정)
ADDITIONAL_NAMESPACES=("wealist-prod" "wealist-staging")

for namespace in "${ADDITIONAL_NAMESPACES[@]}"; do
    if kubectl get namespace "$namespace" &>/dev/null; then
        echo "  Setting up GHCR for namespace: $namespace"
        
        # GHCR secret 생성
        kubectl create secret docker-registry ghcr-secret \
          --docker-server=ghcr.io \
          --docker-username="$GITHUB_USERNAME" \
          --docker-password="$GITHUB_TOKEN" \
          --docker-email="$GITHUB_USERNAME@users.noreply.github.com" \
          --namespace="$namespace" \
          --dry-run=client -o yaml | kubectl apply -f -
        
        # default ServiceAccount에 연결
        kubectl patch serviceaccount default \
          -p '{"imagePullSecrets": [{"name": "ghcr-secret"}]}' \
          -n "$namespace"
        
        # 모든 서비스 ServiceAccount에 연결
        for sa in "${SERVICE_ACCOUNTS[@]}"; do
          if kubectl get serviceaccount "$sa" -n "$namespace" &>/dev/null; then
            echo "    Patching ServiceAccount: $sa"
            kubectl patch serviceaccount "$sa" \
              -p '{"imagePullSecrets": [{"name": "ghcr-secret"}]}' \
              -n "$namespace"
          fi
        done
        
        echo -e "${GREEN}  ✅ $namespace configured${NC}"
    fi
done
echo ""

# ============================================
# 17. 최종 정보
# ============================================
ARGOCD_PASSWORD=$(kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d 2>/dev/null || echo "Password not found")

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
echo "🐳 Container Registry:"
echo "   Registry:   ghcr.io (GitHub Container Registry)"
echo "   Username:   $GITHUB_USERNAME"
echo "   Secret:     ghcr-secret (wealist-dev)"
echo "   Status:     ✅ Configured"
echo ""
echo "🔍 Verification Commands:"
echo "   kubectl get applications -n argocd"
echo "   kubectl get pods -n wealist-dev"
echo "   kubectl get secret ghcr-secret -n wealist-dev"
echo "   kubectl describe sa default -n wealist-dev"
echo ""
echo "🧪 Test Container Registry:"
echo "   kubectl run test-ghcr --image=ghcr.io/orangescloud/auth-service:latest -n wealist-dev"
echo ""
echo "📊 Application Status:"
kubectl get applications -n argocd 2>/dev/null || echo "   No applications found"
echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo "🌐 Starting port-forward..."
kubectl port-forward svc/argocd-server -n argocd 8079:443