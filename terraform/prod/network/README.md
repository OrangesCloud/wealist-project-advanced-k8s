# Production Network Infrastructure

ALB, CloudFront, Route53 DNS 레코드를 관리하는 Terraform 구성입니다.

## 아키텍처

```
                    ┌─────────────────────────────────────────┐
                    │              Route53                     │
                    │  wealist.co.kr    api.wealist.co.kr     │
                    └──────┬────────────────────┬─────────────┘
                           │                    │
                           ▼                    ▼
                    ┌─────────────┐      ┌─────────────┐
                    │ CloudFront  │      │    ALB      │
                    │ (Frontend)  │      │   (API)     │
                    └──────┬──────┘      └──────┬──────┘
                           │                    │
                           ▼                    ▼
                    ┌─────────────┐      ┌─────────────┐
                    │  S3 Bucket  │      │ Istio GW    │
                    │  (React)    │      │ (EKS)       │
                    └─────────────┘      └──────┬──────┘
                                                │
                                         ┌──────┴──────┐
                                         │ Microservices│
                                         │ (EKS Pods)  │
                                         └─────────────┘
```

## 리소스

| 리소스 | 설명 |
|--------|------|
| ALB | API Gateway (api.wealist.co.kr) |
| Target Group | Istio Gateway Pod IP 대상 |
| CloudFront | 프론트엔드 CDN (wealist.co.kr) |
| S3 | 프론트엔드 정적 파일 저장소 |
| Route53 Records | DNS 레코드 (enable_dns 변수로 on/off) |

## 사전 요구사항

1. **foundation** 배포 완료 (VPC, Subnets)
2. **compute** 배포 완료 (EKS, Istio)
3. **ACM 인증서** 발급 완료:
   - `ap-northeast-2`: ALB용
   - `us-east-1`: CloudFront용

## 배포

### 1. web-infra에서 State 마이그레이션 (최초 1회)

기존 web-infra의 CloudFront/S3가 있다면 State를 마이그레이션합니다:

```bash
# 백업
aws s3 cp s3://wealist-tf-state-advanced-k8s/web-infra/terraform.tfstate \
          s3://wealist-tf-state-advanced-k8s/web-infra/terraform.tfstate.backup

# State 복사
aws s3 cp s3://wealist-tf-state-advanced-k8s/web-infra/terraform.tfstate \
          s3://wealist-tf-state-advanced-k8s/prod/network/terraform.tfstate
```

### 2. Terraform 초기화 및 적용

```bash
cd terraform/prod/network

# 초기화
terraform init

# 계획 확인
terraform plan

# 적용 (DNS 없이)
terraform apply

# 또는 DNS 포함
terraform apply -var="enable_dns=true"
```

> **주의**: `enable_dns=true`로 적용하기 전에 수동으로 생성된 Route53 레코드를 삭제하세요!

### 3. ALB → Istio Gateway 연결

Terraform apply 후, TargetGroupBinding을 Kubernetes에 적용합니다:

```bash
# TargetGroupBinding YAML 생성 및 적용
terraform output -raw target_group_binding_yaml | kubectl apply -f -

# 또는 수동으로
cat <<EOF | kubectl apply -f -
apiVersion: elbv2.k8s.aws/v1beta1
kind: TargetGroupBinding
metadata:
  name: istio-gateway-tgb
  namespace: istio-system
spec:
  serviceRef:
    name: istio-ingressgateway
    port: 80
  targetGroupARN: $(terraform output -raw istio_target_group_arn)
  targetType: ip
EOF
```

## 변수

| 변수 | 기본값 | 설명 |
|------|--------|------|
| `enable_dns` | `false` | Route53 레코드 생성 여부 |
| `domain_name` | `wealist.co.kr` | 도메인 이름 |
| `alb_deletion_protection` | `false` | ALB 삭제 보호 |
| `cloudfront_price_class` | `PriceClass_200` | US, EU, Asia |

## Outputs

| Output | 설명 |
|--------|------|
| `alb_dns_name` | ALB DNS 이름 |
| `cloudfront_domain_name` | CloudFront 도메인 |
| `istio_target_group_arn` | Istio Target Group ARN |
| `frontend_url` | 프론트엔드 URL |
| `api_url` | API URL |

```bash
# 모든 output 확인
terraform output

# 특정 output
terraform output istio_target_group_arn
terraform output -raw target_group_binding_yaml
```

## DNS 관리

### Route53 Hosted Zone
- Zone ID: `Z0954990337NMPX3FY1D6`
- Hosted Zone은 Terraform 외부에서 관리됩니다

### Terraform 관리 레코드 (enable_dns=true일 때)
- `wealist.co.kr` → CloudFront (A, AAAA)
- `api.wealist.co.kr` → ALB (A)

### Terraform 외부 관리 레코드
- `dev.wealist.co.kr` → Dev CloudFront
- `local.wealist.co.kr` → iptime (로컬 개발)
- ACM 검증 레코드

## 예상 비용

| 리소스 | 예상 비용 |
|--------|----------|
| ALB | ~$16/월 (기본) + LCU |
| CloudFront | 사용량 기반 |
| Route53 | ~$0.50/월 (레코드) |
| S3 | 사용량 기반 |

## 트러블슈팅

### ALB Target이 Unhealthy

```bash
# Target Group 상태 확인
aws elbv2 describe-target-health \
  --target-group-arn $(terraform output -raw istio_target_group_arn)

# Istio Gateway Pod 확인
kubectl get pods -n istio-system -l app=istio-ingressgateway

# TargetGroupBinding 확인
kubectl get targetgroupbinding -n istio-system
```

### CloudFront 403 에러

```bash
# S3 버킷 정책 확인
aws s3api get-bucket-policy --bucket wealist-frontend

# OAC 확인
aws cloudfront get-origin-access-control \
  --id $(aws cloudfront list-origin-access-controls --query 'OriginAccessControlList.Items[?Name==`wealist-prod-frontend-oac`].Id' --output text)
```

### DNS 레코드 충돌

`enable_dns=true` 적용 시 에러가 발생하면:

```bash
# 기존 레코드 확인
aws route53 list-resource-record-sets \
  --hosted-zone-id Z0954990337NMPX3FY1D6 \
  --query "ResourceRecordSets[?Name=='wealist.co.kr.' || Name=='api.wealist.co.kr.']"

# 수동 레코드 삭제 후 재시도
terraform apply -var="enable_dns=true"
```

## 관련 문서

- [ALB Target Group Binding](https://kubernetes-sigs.github.io/aws-load-balancer-controller/v2.4/guide/targetgroupbinding/targetgroupbinding/)
- [CloudFront OAC](https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/private-content-restricting-access-to-s3.html)
- [Istio Gateway](https://istio.io/latest/docs/tasks/traffic-management/ingress/gateway-api/)
