# EKS 환경에서 ArgoCD Discord 알림 설정

## 1. 사전 준비사항

### EKS 클러스터 연결
```bash
# AWS CLI 설정 확인
aws sts get-caller-identity

# EKS 클러스터에 연결
aws eks update-kubeconfig --region ap-northeast-2 --name your-cluster-name

# 연결 확인
kubectl cluster-info
kubectl get nodes
```

### ArgoCD 설치 확인
```bash
# ArgoCD 네임스페이스 확인
kubectl get namespace argocd

# ArgoCD 컴포넌트 확인
kubectl get pods -n argocd

# Notifications Controller 확인
kubectl get deployment argocd-notifications-controller -n argocd
```

## 2. Discord Bot 설정

### Discord Developer Portal에서 Bot 생성
1. [Discord Developer Portal](https://discord.com/developers/applications) 접속
2. "New Application" 클릭 → "Wealist Production Bot"
3. "Bot" 탭 → "Add Bot" → Token 복사
4. Bot 권한: `Send Messages`, `Embed Links`

### Discord 서버 설정
```
채널 생성: #prod-deployment-alerts
Bot 초대: 위에서 생성한 OAuth2 URL 사용
```

## 3. AWS Secrets Manager 사용 (권장)

### Discord Bot Token을 AWS Secrets Manager에 저장
```bash
# Secrets Manager에 Discord Bot Token 저장
aws secretsmanager create-secret \
    --name "argocd/discord-bot-token" \
    --description "Discord Bot Token for ArgoCD notifications" \
    --secret-string "YOUR_DISCORD_BOT_TOKEN"

# 저장된 시크릿 확인
aws secretsmanager describe-secret --secret-id "argocd/discord-bot-token"
```

### EKS 서비스 계정에 Secrets Manager 권한 부여
```bash
# IAM 정책 생성
cat > discord-secrets-policy.json << EOF
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "secretsmanager:GetSecretValue"
            ],
            "Resource": "arn:aws:secretsmanager:ap-northeast-2:*:secret:argocd/discord-bot-token*"
        }
    ]
}
EOF

# IAM 정책 생성
aws iam create-policy \
    --policy-name ArgoCD-Discord-Secrets-Policy \
    --policy-document file://discord-secrets-policy.json

# EKS 노드 그룹 역할에 정책 연결 (또는 IRSA 사용)
aws iam attach-role-policy \
    --role-name your-eks-nodegroup-role \
    --policy-arn arn:aws:iam::YOUR_ACCOUNT:policy/ArgoCD-Discord-Secrets-Policy
```

## 4. ArgoCD Notifications Controller 설정

### Helm으로 ArgoCD 업그레이드 (notifications 활성화)
```bash
# ArgoCD Helm 차트에 notifications 활성화
helm upgrade argocd argo/argo-cd \
    --namespace argocd \
    --set notifications.enabled=true \
    --set notifications.argocdUrl="https://your-argocd-domain.com"
```

### 또는 별도 설치
```bash
# Notifications Controller만 별도 설치
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/notifications_install/install.yaml
```

## 5. Discord 알림 설정 적용

### 설정 파일 적용
```bash
# Discord 알림 설정 적용
kubectl apply -f k8s/argocd/notifications/discord-config.yaml

# AWS Secrets Manager 사용
./k8s/argocd/scripts/setup-discord-notifications.sh --use-aws-secrets

# 또는 환경변수 사용
export DISCORD_BOT_TOKEN="your-bot-token"
./k8s/argocd/scripts/setup-discord-notifications.sh
```

## 6. 설정 확인

### ArgoCD UI에서 확인
```bash
# ArgoCD UI 접속 (포트 포워딩)
kubectl port-forward svc/argocd-server -n argocd 8080:443

# 브라우저에서 https://localhost:8080 접속
# Settings → Notifications에서 설정 확인
```

### CLI로 확인
```bash
# ConfigMap 확인
kubectl get cm argocd-notifications-cm -n argocd -o yaml

# Secret 확인
kubectl get secret argocd-notifications-secret -n argocd -o yaml

# Notifications Controller 로그
kubectl logs -n argocd -l app.kubernetes.io/name=argocd-notifications-controller -f
```

## 7. 테스트

### 수동 배포로 테스트
```bash
# ArgoCD CLI 설치 (로컬)
curl -sSL -o argocd-linux-amd64 https://github.com/argoproj/argo-cd/releases/latest/download/argocd-linux-amd64
sudo install -m 555 argocd-linux-amd64 /usr/local/bin/argocd

# ArgoCD 로그인
argocd login localhost:8080

# 테스트 배포 (주의: Production 환경)
argocd app sync auth-service-prod
```

### GitHub Actions에서 테스트
```bash
# ci-prod-images.yaml 워크플로우 실행
# GitHub → Actions → "Update Production Image Tag" → Run workflow
```

## 8. 문제 해결

### 알림이 오지 않는 경우
```bash
# 1. Notifications Controller 상태 확인
kubectl get pods -n argocd -l app.kubernetes.io/name=argocd-notifications-controller

# 2. 로그 확인
kubectl logs -n argocd -l app.kubernetes.io/name=argocd-notifications-controller --tail=100

# 3. ConfigMap 설정 확인
kubectl describe cm argocd-notifications-cm -n argocd

# 4. Secret 확인
kubectl describe secret argocd-notifications-secret -n argocd

# 5. Application 라벨 확인
kubectl get app auth-service-prod -n argocd -o yaml | grep -A 10 labels
```

### Discord Bot 권한 문제
```bash
# Discord 서버에서 확인:
# 1. Bot이 서버에 있는지
# 2. #prod-deployment-alerts 채널에 메시지 보내기 권한
# 3. 링크 임베드 권한
```

### AWS Secrets Manager 권한 문제
```bash
# EKS 노드 그룹 역할 확인
aws iam list-attached-role-policies --role-name your-eks-nodegroup-role

# 또는 IRSA 사용
eksctl create iamserviceaccount \
    --name argocd-notifications-controller \
    --namespace argocd \
    --cluster your-cluster-name \
    --attach-policy-arn arn:aws:iam::YOUR_ACCOUNT:policy/ArgoCD-Discord-Secrets-Policy \
    --approve
```

## 9. 보안 고려사항

### 네트워크 정책
```yaml
# ArgoCD에서 Discord API 접근 허용
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: argocd-notifications-egress
  namespace: argocd
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/name: argocd-notifications-controller
  policyTypes:
  - Egress
  egress:
  - to: []
    ports:
    - protocol: TCP
      port: 443  # Discord API (HTTPS)
```

### IAM 최소 권한
- Secrets Manager 접근은 특정 시크릿만
- EKS 노드 그룹 또는 IRSA 사용
- 정기적인 Discord Bot Token 갱신

## 10. 모니터링

### CloudWatch 로그 (선택사항)
```bash
# ArgoCD 로그를 CloudWatch로 전송
kubectl apply -f https://raw.githubusercontent.com/aws/amazon-cloudwatch-agent/master/k8s-deploy-2.0.1/cloudwatch-namespace.yaml
```

### Prometheus 메트릭
```bash
# ArgoCD 메트릭 확인
kubectl port-forward svc/argocd-metrics -n argocd 8082:8082
curl http://localhost:8082/metrics | grep notification
```