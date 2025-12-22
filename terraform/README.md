# Terraform Infrastructure

weAlist 프로젝트의 AWS 인프라를 관리하는 Terraform 설정입니다.

## 디렉토리 구조

```
terraform/
├── modules/                    # 재사용 가능한 모듈
│   ├── github-oidc/           # GitHub OIDC Provider + IAM Role
│   └── ecr/                   # ECR 리포지토리
│
├── oidc-iam/                  # [독립 실행] GitHub Actions용 OIDC/IAM
│   └── GitHub Actions → AWS 인증 설정
│
└── dev-environment/           # [독립 실행] 로컬 PC Dev 환경
    └── ECR 리포지토리 + IAM User
```

## 사전 요구사항

- Terraform >= 1.0
- AWS CLI 설정 (관리자 권한)
- AWS Account ID

## 사용 방법

### 1. OIDC/IAM 설정 (GitHub Actions용)

GitHub Actions에서 AWS에 접근하기 위한 OIDC Provider와 IAM Role을 생성합니다.

```bash
cd terraform/oidc-iam

# 변수 파일 복사 및 편집
cp terraform.tfvars.example terraform.tfvars
# terraform.tfvars에 aws_account_id 설정

# Terraform 실행
terraform init
terraform plan
terraform apply

# 출력값을 GitHub Secrets에 등록:
# - AWS_ROLE_ARN: terraform output github_actions_role_arn
# - AWS_ACCOUNT_ID: 본인 AWS Account ID
```

### 2. Dev 환경 설정 (로컬 PC용)

로컬 PC에서 ECR에 접근하기 위한 설정입니다.

```bash
cd terraform/dev-environment

# 변수 파일 복사 및 편집
cp terraform.tfvars.example terraform.tfvars

# Terraform 실행
terraform init
terraform plan
terraform apply

# 출력된 Access Key를 로컬 PC에 설정
terraform output dev_user_access_key_id
terraform output -raw dev_user_secret_access_key

# AWS CLI 프로필 설정
aws configure --profile wealist-dev
# Access Key ID, Secret Access Key 입력
# Default region: ap-northeast-2
```

### ECR 로그인 (로컬 PC)

```bash
# AWS 프로필로 ECR 로그인
aws ecr get-login-password --region ap-northeast-2 --profile wealist-dev | \
  docker login --username AWS --password-stdin <ACCOUNT_ID>.dkr.ecr.ap-northeast-2.amazonaws.com
```

## 모듈 설명

### github-oidc

GitHub Actions에서 AWS에 OIDC로 인증하기 위한 설정:
- OIDC Provider 생성
- IAM Role 생성 (Trust: GitHub OIDC)
- IAM Policy 연결 (ECR, S3, CloudFront)

### ecr

ECR 리포지토리 생성:
- 서비스별 리포지토리
- 이미지 스캐닝 활성화
- 라이프사이클 정책 (선택)

## 보안 주의사항

1. **절대 git에 올리면 안 되는 파일**:
   - `*.tfstate` - Terraform 상태 파일
   - `terraform.tfvars` - 실제 변수값

2. **Access Key 관리**:
   - 로컬 PC에만 저장
   - 정기적으로 키 로테이션
   - 필요시 AWS SSO로 전환 가능

## 리소스 삭제

```bash
# 주의: 모든 리소스가 삭제됩니다
cd terraform/oidc-iam   # 또는 dev-environment
terraform destroy
```
