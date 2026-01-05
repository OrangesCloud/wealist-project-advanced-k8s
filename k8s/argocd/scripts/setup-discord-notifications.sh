#!/bin/bash

# ArgoCD Production Discord 알림 설정 스크립트

set -e

echo "🔔 Setting up ArgoCD Discord notifications for PRODUCTION..."

# Discord Bot Token 확인
if [ -z "$DISCORD_BOT_TOKEN" ]; then
    echo "❌ DISCORD_BOT_TOKEN 환경변수가 설정되지 않았습니다."
    echo "Discord Developer Portal에서 Bot Token을 생성하고 설정하세요:"
    echo "export DISCORD_BOT_TOKEN='your-bot-token-here'"
    exit 1
fi

# ArgoCD 네임스페이스 확인
if ! kubectl get namespace argocd >/dev/null 2>&1; then
    echo "❌ ArgoCD 네임스페이스가 존재하지 않습니다."
    echo "먼저 ArgoCD를 설치하세요."
    exit 1
fi

echo "📝 Creating Discord notifications configuration for Production..."

# Discord 알림 설정 적용
kubectl apply -f k8s/argocd/notifications/discord-config.yaml

# Secret 업데이트 (Bot Token)
kubectl create secret generic argocd-notifications-secret \
    --from-literal=discord-token="$DISCORD_BOT_TOKEN" \
    -n argocd --dry-run=client -o yaml | kubectl apply -f -

echo "✅ Production Discord notifications configuration applied!"

# Notifications Controller 재시작
echo "🔄 Restarting notifications controller..."
kubectl rollout restart deployment/argocd-notifications-controller -n argocd

# 상태 확인
echo "⏳ Waiting for notifications controller to be ready..."
kubectl wait --for=condition=available --timeout=60s deployment/argocd-notifications-controller -n argocd

echo "🎉 Production Discord notifications setup completed!"
echo ""
echo "📋 Next steps:"
echo "1. Discord 서버에 Bot을 초대하세요"
echo "2. #prod-deployment-alerts 채널을 생성하세요"
echo "3. Production 배포를 실행하여 알림을 확인하세요"
echo ""
echo "🚀 Production services with Discord notifications:"
echo "  - auth-service-prod"
echo "  - board-service-prod"
echo "  - chat-service-prod"
echo "  - noti-service-prod"
echo "  - storage-service-prod"
echo "  - user-service-prod"
echo ""
echo "🔗 Useful commands:"
echo "  # 알림 설정 확인"
echo "  kubectl get cm argocd-notifications-cm -n argocd -o yaml"
echo ""
echo "  # Notifications controller 로그 확인"
echo "  kubectl logs -n argocd -l app.kubernetes.io/name=argocd-notifications-controller"
echo ""
echo "  # 테스트 배포 (주의: Production 환경)"
echo "  argocd app sync auth-service-prod"