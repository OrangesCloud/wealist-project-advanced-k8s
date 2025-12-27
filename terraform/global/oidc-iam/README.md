# Global OIDC IAM

GitHub Actions에서 AWS 리소스에 접근하기 위한 OIDC 기반 IAM 역할 구성입니다.

## 개요

AWS IAM OIDC(OpenID Connect)를 사용하여 GitHub Actions 워크플로우가 AWS 자격 증명 없이 AWS 리소스에 접근할 수 있습니다.

### 장점

- AWS Access Key 저장 불필요
- 단기 자격 증명 사용 (보안 강화)
- 저장소/브랜치별 세밀한 권한 제어
- 자격 증명 로테이션 불필요

## 구성 요소

| 리소스 | 설명 |
|--------|------|
| IAM OIDC Provider | GitHub Actions용 OIDC 공급자 |
| IAM Role (backend) | 백엔드 배포용 역할 |
| IAM Role (frontend) | 프론트엔드 배포용 역할 |

## 권한 범위

### Backend Role

| 권한 | 용도 |
|------|------|
| ECR | 이미지 Push/Pull |
| S3 | Terraform State 접근 |
| DynamoDB | State Lock |
| EKS | 클러스터 접근 (kubectl/Helm) |
| Secrets Manager | 배포 시크릿 조회 |

### Frontend Role

| 권한 | 용도 |
|------|------|
| S3 | 정적 파일 업로드 |
| CloudFront | 캐시 무효화 |

## 사용법

### GitHub Actions 워크플로우

```yaml
name: Deploy to EKS

on:
  push:
    branches: [main]

permissions:
  id-token: write   # OIDC 토큰 발급 필수
  contents: read

jobs:
  deploy:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v4

      - name: Configure AWS credentials
        uses: aws-actions/configure-aws-credentials@v4
        with:
          role-to-assume: arn:aws:iam::123456789:role/wealist-github-backend
          aws-region: ap-northeast-2

      - name: Login to ECR
        uses: aws-actions/amazon-ecr-login@v2

      - name: Deploy to EKS
        run: |
          aws eks update-kubeconfig --name wealist-prod-eks --region ap-northeast-2
          kubectl apply -f k8s/manifests/
```

## 변수 설정

### terraform.tfvars 예시

```hcl
github_org = "OrangesCloud"

github_repositories = [
  "wealist-project-advanced-k8s"
]

# EKS 접근 활성화
enable_eks_access = true
eks_cluster_arns  = ["arn:aws:eks:ap-northeast-2:123456789:cluster/wealist-prod-eks"]

# ECR 저장소 (backend role)
ecr_repository_arns = [
  "arn:aws:ecr:ap-northeast-2:123456789:repository/auth-service",
  "arn:aws:ecr:ap-northeast-2:123456789:repository/user-service",
  # ... 기타 서비스
]

# S3 버킷 (Terraform state)
s3_bucket_arns = [
  "arn:aws:s3:::wealist-tf-state-advanced-k8s",
  "arn:aws:s3:::wealist-tf-state-advanced-k8s/*"
]

# Frontend 배포용 S3
frontend_s3_bucket_arns = [
  "arn:aws:s3:::wealist-frontend-prod",
  "arn:aws:s3:::wealist-frontend-prod/*"
]

# CloudFront
cloudfront_distribution_arns = [
  "arn:aws:cloudfront::123456789:distribution/XXXXX"
]
```

## 배포

```bash
cd terraform/global/oidc-iam

# 초기화
terraform init

# 계획 확인
terraform plan -var-file="terraform.tfvars"

# 적용 (사용자 직접 실행)
terraform apply -var-file="terraform.tfvars"
```

## 출력 값

```bash
# 배포 후 확인
terraform output

# backend_role_arn = "arn:aws:iam::123456789:role/wealist-github-backend"
# frontend_role_arn = "arn:aws:iam::123456789:role/wealist-github-frontend"
```

## State 위치

```
s3://wealist-tf-state-advanced-k8s/global/oidc-iam/terraform.tfstate
```

## 보안 고려사항

### 저장소 제한

특정 저장소만 역할을 사용할 수 있도록 제한됩니다:

```hcl
Condition = {
  StringLike = {
    "token.actions.githubusercontent.com:sub" = [
      "repo:OrangesCloud/wealist-project-advanced-k8s:*"
    ]
  }
}
```

### 브랜치 제한 (선택)

프로덕션 배포를 특정 브랜치로 제한하려면:

```hcl
Condition = {
  StringEquals = {
    "token.actions.githubusercontent.com:sub" = "repo:OrangesCloud/wealist-project-advanced-k8s:ref:refs/heads/main"
  }
}
```

## 트러블슈팅

### OIDC 토큰 발급 실패

GitHub Actions 워크플로우에 `permissions` 설정 확인:

```yaml
permissions:
  id-token: write   # 필수!
  contents: read
```

### AssumeRole 실패

1. OIDC Provider 썸프린트 확인
2. Trust Policy의 저장소 이름 확인
3. AWS 계정 ID 확인

### EKS 접근 불가

1. `enable_eks_access = true` 설정 확인
2. `eks_cluster_arns`에 클러스터 ARN 포함 확인
3. EKS 클러스터의 aws-auth ConfigMap 또는 Access Entries 확인
