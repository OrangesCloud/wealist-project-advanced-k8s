#!/bin/bash
set -e

echo "🚀 Starting ArgoCD deployment..."

# GitHub 저장소 정보
REPO_URL="https://github.com/OrangesCloud/wealist-argo-helm.git"

# 1. ArgoCD 설치
echo "📦 Installing ArgoCD..."
kubectl create namespace argocd --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# 2. ArgoCD 서버 준비 대기
echo "⏳ Waiting for ArgoCD server..."
kubectl wait --for=condition=available --timeout=300s deployment/argocd-server -n argocd

# 3. 네임스페이스 생성
echo "📁 Creating application namespace..."
kubectl create namespace wealist-dev --dry-run=client -o yaml | kubectl apply -f -

# 4. GitHub 저장소 인증 설정
echo "🔑 Setting up GitHub repository access..."
echo "ℹ️  You need a GitHub Personal Access Token with 'repo' permissions"
echo "ℹ️  Create one at: https://github.com/settings/tokens"
echo

read -p "Enter your GitHub username: " GITHUB_USERNAME

# Personal Access Token 입력 (화면에 표시되지 않음)
echo -n "Enter your GitHub Personal Access Token: "
read -s GITHUB_TOKEN
echo

# 저장소 Secret 생성
echo "📝 Creating repository secret..."
kubectl create secret generic wealist-repo -n argocd \
  --from-literal=type=git \
  --from-literal=url=$REPO_URL \
  --from-literal=username=$GITHUB_USERNAME \
  --from-literal=password=$GITHUB_TOKEN \
  --dry-run=client -o yaml | kubectl apply -f -

# ArgoCD가 인식할 수 있도록 라벨 추가
kubectl label secret wealist-repo -n argocd \
  argocd.argoproj.io/secret-type=repository --overwrite

echo "✅ Repository access configured successfully!"

# 5. ArgoCD 서버가 완전히 준비될 때까지 추가 대기
echo "⏳ Waiting for ArgoCD to be fully ready..."
sleep 30

# 6. AppProject 생성
echo "🎯 Creating AppProject..."
kubectl apply -f k8s/argocd/apps/project.yaml

# 7. Root Application 생성
echo "🌟 Creating Root Application..."
kubectl apply -f k8s/argocd/apps/root-app.yaml

# 8. ArgoCD CLI 설정 (선택사항)
echo "🔧 Setting up ArgoCD CLI access..."
ARGOCD_PASSWORD=$(kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d)

# 9. 접속 정보 표시
echo ""
echo "✅ ArgoCD deployment completed!"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🌐 ArgoCD Access Information:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "URL:      https://localhost:8079"
echo "Username: admin"
echo "Password: $ARGOCD_PASSWORD"
echo ""
echo "📋 Next steps:"
echo "1. Access ArgoCD UI at the URL above"
echo "2. Login with admin credentials"
echo "3. Check Applications tab to see your services"
echo "4. Sync applications if needed"
echo ""
echo "🔍 Useful commands:"
echo "kubectl get applications -n argocd"
echo "kubectl get pods -n wealist-dev"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# 10. 포트포워딩 시작
echo "🌐 Starting port-forward (Ctrl+C to stop)..."
kubectl port-forward svc/argocd-server -n argocd 8079:443