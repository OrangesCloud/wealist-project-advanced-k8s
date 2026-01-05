#!/bin/bash

# ArgoCD Dev Discord 알림 설정 스크립트

set -e

echo "🔔 Setting up ArgoCD Discord notifications for DEV..."

# Discord Webhook URL 확인
if [ -z "$DISCORD_WEBHOOK_URL" ]; then
    echo "❌ DISCORD_WEBHOOK_URL 환경변수가 설정되지 않았습니다."
    echo ""
    echo "다음과 같이 설정하세요:"
    echo 'export DISCORD_WEBHOOK_URL="https://discord.com/api/webhooks/..."'
    exit 1
fi

# ArgoCD 네임스페이스 확인
if ! kubectl get namespace argocd >/dev/null 2>&1; then
    echo "❌ ArgoCD 네임스페이스가 존재하지 않습니다."
    echo "먼저 ArgoCD를 설치하세요."
    exit 1
fi

echo "📝 Creating Discord notifications configuration for Dev..."

# ConfigMap 적용 (discord-config.yaml에서 Secret 부분 제외)
kubectl apply -f - <<'EOF'
apiVersion: v1
kind: ConfigMap
metadata:
  name: argocd-notifications-cm
  namespace: argocd
data:
  context: |
    argocdUrl: https://dev.wealist.co.kr/api/argo

  service.webhook.discord: |
    url: $discord-webhook-url
    headers:
    - name: Content-Type
      value: application/json

  template.app-deployed: |
    webhook:
      discord:
        method: POST
        body: |
          {
            "embeds": [{
              "title": ":rocket: Dev 배포 완료",
              "color": 3066993,
              "fields": [
                {"name": "Application", "value": "{{.app.metadata.name}}", "inline": true},
                {"name": "Status", "value": "{{.app.status.health.status}}", "inline": true},
                {"name": "Sync", "value": "{{.app.status.sync.status}}", "inline": true},
                {"name": "Revision", "value": "{{.app.status.sync.revision | trunc 7}}", "inline": true}
              ],
              "timestamp": "{{.app.status.operationState.finishedAt}}"
            }]
          }

  template.app-sync-failed: |
    webhook:
      discord:
        method: POST
        body: |
          {
            "embeds": [{
              "title": ":x: Dev 배포 실패",
              "color": 15158332,
              "fields": [
                {"name": "Application", "value": "{{.app.metadata.name}}", "inline": true},
                {"name": "Error", "value": "{{.app.status.operationState.message | trunc 200}}", "inline": false}
              ],
              "timestamp": "{{.app.status.operationState.finishedAt}}"
            }]
          }

  template.app-sync-running: |
    webhook:
      discord:
        method: POST
        body: |
          {
            "embeds": [{
              "title": ":hourglass: Dev 배포 시작",
              "color": 16776960,
              "fields": [
                {"name": "Application", "value": "{{.app.metadata.name}}", "inline": true},
                {"name": "Started", "value": "{{.app.status.operationState.startedAt}}", "inline": true}
              ]
            }]
          }

  template.app-health-degraded: |
    webhook:
      discord:
        method: POST
        body: |
          {
            "embeds": [{
              "title": ":warning: Dev 서비스 상태 이상",
              "color": 16744448,
              "fields": [
                {"name": "Application", "value": "{{.app.metadata.name}}", "inline": true},
                {"name": "Health", "value": "{{.app.status.health.status}}", "inline": true},
                {"name": "Message", "value": "{{.app.status.health.message | default \"No message\" | trunc 200}}", "inline": false}
              ]
            }]
          }

  trigger.on-deployed: |
    - description: 배포 완료 시 알림
      send:
      - app-deployed
      when: app.status.operationState.phase in ['Succeeded'] and app.status.health.status == 'Healthy'

  trigger.on-sync-failed: |
    - description: 배포 실패 시 알림
      send:
      - app-sync-failed
      when: app.status.operationState.phase in ['Error', 'Failed']

  trigger.on-sync-running: |
    - description: 배포 시작 시 알림
      send:
      - app-sync-running
      when: app.status.operationState.phase in ['Running']

  trigger.on-health-degraded: |
    - description: 서비스 상태 이상 시 알림
      send:
      - app-health-degraded
      when: app.status.health.status == 'Degraded'

  subscriptions: |
    - recipients:
      - webhook:discord
      triggers:
      - on-deployed
      - on-sync-failed
      - on-health-degraded
EOF

echo "🔐 Creating Discord webhook secret..."

# Secret 생성 (webhook URL)
kubectl create secret generic argocd-notifications-secret \
    --from-literal=discord-webhook-url="$DISCORD_WEBHOOK_URL" \
    -n argocd --dry-run=client -o yaml | kubectl apply -f -

echo "✅ Dev Discord notifications configuration applied!"

# Notifications Controller 재시작
echo "🔄 Restarting notifications controller..."
kubectl rollout restart deployment/argocd-notifications-controller -n argocd

# 상태 확인
echo "⏳ Waiting for notifications controller to be ready..."
kubectl wait --for=condition=available --timeout=60s deployment/argocd-notifications-controller -n argocd

echo "🎉 Dev Discord notifications setup completed!"
echo ""
echo "📋 Dev 서비스 알림이 활성화됩니다:"
echo "  - auth-service-dev"
echo "  - board-service-dev"
echo "  - chat-service-dev"
echo "  - noti-service-dev"
echo "  - storage-service-dev"
echo "  - user-service-dev"
echo ""
echo "🔗 유용한 명령어:"
echo "  # 알림 설정 확인"
echo "  kubectl get cm argocd-notifications-cm -n argocd -o yaml"
echo ""
echo "  # Notifications controller 로그 확인"
echo "  kubectl logs -n argocd -l app.kubernetes.io/name=argocd-notifications-controller -f"
echo ""
echo "  # 테스트 배포"
echo "  argocd app sync auth-service-dev"
