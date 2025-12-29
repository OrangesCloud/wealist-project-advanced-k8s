# Discord 배포 알림 설정 가이드

## 1. Discord Bot 생성

### Discord Developer Portal에서 Bot 생성
1. [Discord Developer Portal](https://discord.com/developers/applications) 접속
2. "New Application" 클릭
3. 애플리케이션 이름 입력 (예: "Wealist Deploy Bot")
4. "Bot" 탭으로 이동
5. "Add Bot" 클릭
6. Bot Token 복사 (나중에 사용)

### Bot 권한 설정
- `Send Messages` 권한 필요
- `Embed Links` 권한 필요
- `Use Slash Commands` 권한 (선택사항)

## 2. Discord 서버 설정

### 채널 생성
```
#deployment-alerts  # 배포 알림 전용 채널
```

### Bot 초대
1. Developer Portal에서 "OAuth2" > "URL Generator" 탭
2. Scopes: `bot` 선택
3. Bot Permissions: `Send Messages`, `Embed Links` 선택
4. 생성된 URL로 서버에 Bot 초대

### Webhook URL 생성 (대안)
```bash
# Discord 채널에서 우클릭 > 채널 편집 > 연동 > 웹후크
# 웹후크 URL 복사: https://discord.com/api/webhooks/CHANNEL_ID/TOKEN
```

## 3. ArgoCD 설정

### Notifications Controller 활성화
```bash
# ArgoCD에 notifications controller 설치
kubectl apply -n argocd -f k8s/argocd/notifications/discord-config.yaml
```

### Secret 업데이트
```bash
# Discord Bot Token 설정
kubectl create secret generic argocd-notifications-secret \
  --from-literal=discord-token="YOUR_BOT_TOKEN" \
  -n argocd --dry-run=client -o yaml | kubectl apply -f -
```

## 4. 애플리케이션별 알림 설정

각 ArgoCD Application에 알림 어노테이션 추가:

```yaml
# k8s/argocd/apps/prod/auth-service.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: auth-service-prod
  annotations:
    # Discord 알림 활성화
    notifications.argoproj.io/subscribe.on-deployed.discord: deployment-alerts
    notifications.argoproj.io/subscribe.on-sync-failed.discord: deployment-alerts
    notifications.argoproj.io/subscribe.on-sync-running.discord: deployment-alerts
  labels:
    environment: production
```

## 5. 테스트

### 수동 배포 테스트
```bash
# ArgoCD CLI로 테스트 배포
argocd app sync auth-service-prod
```

### 알림 확인
- Discord 채널에서 배포 시작/완료/실패 메시지 확인
- 메시지에 ArgoCD 링크, Grafana 링크 포함 확인

## 6. 고급 설정

### 환경별 채널 분리
```yaml
# dev 환경은 다른 채널로
subscriptions: |
  - recipients:
    - discord:dev-alerts
    triggers:
    - on-deployed
    selector: metadata.labels.environment == 'development'
  - recipients:
    - discord:prod-alerts
    triggers:
    - on-deployed
    - on-sync-failed
    selector: metadata.labels.environment == 'production'
```

### 서비스별 멘션
```yaml
template.app-deployed: |
  discord:
    title: "🚀 {{.app.metadata.name}} 배포 완료"
    description: |
      {{if eq .app.metadata.name "auth-service-prod"}}
      <@&BACKEND_TEAM_ROLE_ID> 인증 서비스가 배포되었습니다.
      {{else if eq .app.metadata.name "frontend-prod"}}
      <@&FRONTEND_TEAM_ROLE_ID> 프론트엔드가 배포되었습니다.
      {{end}}
```

## 7. 문제 해결

### 알림이 오지 않는 경우
```bash
# Notifications controller 로그 확인
kubectl logs -n argocd -l app.kubernetes.io/name=argocd-notifications-controller

# ConfigMap 확인
kubectl get cm argocd-notifications-cm -n argocd -o yaml

# Secret 확인
kubectl get secret argocd-notifications-secret -n argocd -o yaml
```

### Bot 권한 문제
- Discord 서버에서 Bot 역할 확인
- 채널 권한 확인 (메시지 보내기, 링크 임베드)

## 8. 보안 고려사항

- Bot Token은 절대 코드에 하드코딩하지 말 것
- GitHub Secrets 또는 Kubernetes Secret 사용
- 정기적으로 Token 갱신
- 최소 권한 원칙 적용