# Production Network Infrastructure

ALB, Route53 API DNS 레코드를 관리하는 Terraform 구성입니다.

> **Note**: CloudFront + S3 (Frontend)는 AWS Console에서 수동 관리 (Flat-Rate Free Plan, 2025년 11월 출시)

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
                    │ (Console)   │      │   (API)     │
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

| 리소스 | 관리 방식 | 설명 |
|--------|----------|------|
| ALB | Terraform | API Gateway (api.wealist.co.kr) |
| Target Group | Terraform | Istio Gateway Pod IP 대상 |
| Route53 (API) | Terraform | API DNS 레코드 (enable_dns 변수) |
| **CloudFront** | **AWS Console** | Flat-Rate Free Plan 사용 |
| **S3** | **AWS Console** | 프론트엔드 정적 파일 (CloudFront와 함께) |
| **Route53 (Frontend)** | **AWS Console** | CloudFront와 함께 관리 |

## CloudFront Flat-Rate Free Plan

2025년 11월 AWS에서 출시한 새로운 가격 정책입니다.

| 항목 | Free Plan |
|------|-----------|
| 가격 | $0/월 |
| 요청 | 1M/월 |
| 데이터 전송 | 100 GB/월 |
| WAF 규칙 | 5개 포함 |
| S3 스토리지 | 5 GB 포함 |
| 초과 시 | 추가 비용 없음 (성능 저하만) |

> **Terraform 미지원**: 현재 Terraform AWS Provider는 Flat-Rate Plan을 지원하지 않습니다.
> AWS Console에서 CloudFront + S3를 직접 생성해야 합니다.

## 사전 요구사항

1. **foundation** 배포 완료 (VPC, Subnets)
2. **compute** 배포 완료 (EKS, Istio)
3. **ACM 인증서** 발급 완료:
   - `ap-northeast-2`: ALB용

## 배포

```bash
cd terraform/prod/network

# 초기화
terraform init

# 계획 확인
terraform plan

# 적용
terraform apply

# API DNS 포함 (enable_dns=true)
terraform apply -var="enable_dns=true"
```

> **주의**: `enable_dns=true`로 적용하기 전에 수동으로 생성된 Route53 API 레코드를 삭제하세요!

### ALB → Istio Gateway 연결

Terraform apply 후, TargetGroupBinding을 Kubernetes에 적용합니다:

```bash
# TargetGroupBinding YAML 생성 및 적용
terraform output -raw target_group_binding_yaml | kubectl apply -f -
```

## 변수

| 변수 | 기본값 | 설명 |
|------|--------|------|
| `enable_dns` | `false` | Route53 API 레코드 생성 여부 |
| `domain_name` | `wealist.co.kr` | 도메인 이름 |
| `alb_deletion_protection` | `false` | ALB 삭제 보호 |

## Outputs

| Output | 설명 |
|--------|------|
| `alb_dns_name` | ALB DNS 이름 |
| `istio_target_group_arn` | Istio Target Group ARN |
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
- Hosted Zone은 Terraform 외부에서 관리

### Terraform 관리 레코드 (enable_dns=true)
- `api.wealist.co.kr` → ALB (A)

### AWS Console 관리 레코드
- `wealist.co.kr` → CloudFront (A, AAAA)
- `dev.wealist.co.kr` → Dev CloudFront
- `local.wealist.co.kr` → iptime (로컬 개발)

## 예상 비용

| 리소스 | 예상 비용 |
|--------|----------|
| ALB | ~$16/월 (기본) + LCU |
| CloudFront | **$0/월** (Free Plan) |
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

## 관련 문서

- [ALB Target Group Binding](https://kubernetes-sigs.github.io/aws-load-balancer-controller/v2.4/guide/targetgroupbinding/targetgroupbinding/)
- [CloudFront Flat-Rate Plans](https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/flat-rate-pricing-plan.html)
- [Istio Gateway](https://istio.io/latest/docs/tasks/traffic-management/ingress/gateway-api/)
