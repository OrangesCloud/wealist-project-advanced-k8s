
# Terraform Infrastructure

weAlist 프로젝트의 AWS 인프라를 관리하는 Terraform 설정입니다.
**협업을 위해 Terraform State를 S3 Backend로 관리**하며, 프론트엔드 및 백엔드 리소스를 계층별로 분리하여 구성합니다.

## 📂 디렉토리 구조


```

terraform/
├── modules/                    # [재사용 모듈]
│   ├── github-oidc/           # GitHub OIDC Provider + IAM Role
│   ├── ecr/                   # ECR 리포지토리
│   └── ssm-parameter/         # SSM Parameter Store (시크릿 저장)
│
├── oidc-iam/                  # [1단계: 인증] GitHub Actions용 OIDC/IAM
│   └── GitHub Actions가 AWS에 접근하기 위한 인증 설정 (S3 Backend)
│
├── dev-environment/           # [2단계: 개발환경] 로컬 PC Dev 환경
│   ├── 개발자용 ECR 접근 권한(IAM User) + 리포지토리 생성 (S3 Backend)
│   └── SSM Parameter Store (시크릿)
│
└── web-infra/                 # [3단계: 프론트엔드] 정적 웹 호스팅
    └── S3 + CloudFront (OAC) + Route53 (S3 Backend)
```

## ✅ 사전 요구사항

1.  **Terraform** >= 1.0
2.  **AWS CLI** (AdministratorAccess 권한이 있는 프로필 필수)
3.  **Terraform State 저장용 S3 버킷** (최초 1회 생성 필요)

---

## 🚀 초기 설정 (One-time Setup)

Terraform 상태 파일(`terraform.tfstate`)을 팀원들과 공유하기 위해 S3 버킷이 필요합니다.
추후 다른 AWS환경에서 최초 실행시 **관리자 권한**으로 아래 명령어를 1회만 실행해주세요.

```bash
# 1. 상태 저장용 S3 버킷 생성 (이름은 고유해야 함)
aws s3 mb s3://wealist-tf-state-advanced-k8s --region ap-northeast-2

# 2. 버킷 버전 관리 활성화 (실수 방지용)
aws s3api put-bucket-versioning --bucket wealist-tf-state-advanced-k8s --versioning-configuration Status=Enabled

# 3. 잠금(Lock)용 DynamoDB 테이블 생성
aws dynamodb create-table \
    --table-name terraform-lock \
    --attribute-definitions AttributeName=LockID,AttributeType=S \
    --key-schema AttributeName=LockID,KeyType=HASH \
    --provisioned-throughput ReadCapacityUnits=5,WriteCapacityUnits=5 \
    --region ap-northeast-2
    
```

---

## 🛠️ 사용 방법

> **⚠️ 중요:** 인프라 배포(`terraform apply`)는 권한이 있는 **관리자(Default) 프로필**로 실행해야 합니다. (`wealist-dev` 프로필은 개발용입니다.)

### 1. OIDC/IAM 설정 (GitHub Actions용)

GitHub Actions에서 AWS에 접근하기 위한 권한(OIDC Provider, IAM Role)을 생성합니다.

```bash
cd terraform/oidc-iam

# 1. 변수 파일 생성 및 편집 (aws_account_id 입력)
cp terraform.tfvars.example terraform.tfvars

# 2. Terraform 실행
terraform init  # S3 Backend 연결
terraform apply

# 3. [GitHub Secrets 등록] 출력된 값을 GitHub Repo Settings에 등록
# AWS_ROLE_ARN: terraform output github_actions_role_arn
# AWS_ACCOUNT_ID: 본인 AWS Account ID

```

### 2. Dev 환경 설정 (개발자 ECR 접근용)

개발자가 로컬 PC에서 ECR에 이미지를 푸시할 때 사용할 IAM 유저(`wealist-dev`)를 생성합니다.

```bash
cd terraform/dev-environment

# 1. 변수 파일 생성
cp terraform.tfvars.example terraform.tfvars
# terraform.tfvars에 시크릿 값 설정 (Google OAuth, JWT 등)

# 2. Terraform 실행
terraform init
terraform apply

# 3. [중요] 출력된 Access Key 확인
# terraform output dev_user_access_key_id
# terraform output -raw dev_user_secret_access_key

```

#### 👨‍💻 개발자 로컬 PC 설정 (wealist-dev 프로필)

위에서 얻은 키를 사용하여 개발자 PC에 프로필을 등록합니다.

```bash
aws configure --profile wealist-dev
# Access Key ID: (위에서 출력된 값)
# Secret Access Key: (위에서 출력된 값)
# Region: ap-northeast-2

```

### 3. SSM Parameter Store (시크릿 관리)

dev-environment에 SSM Parameter Store로 시크릿을 저장합니다.
External Secrets Operator가 Kind 클러스터에서 이 값들을 K8s Secret으로 동기화합니다.

```bash
cd terraform/dev-environment

# 시크릿만 생성/업데이트
terraform apply -target=module.parameters

# SSM 파라미터 확인
aws ssm get-parameters-by-path --path "/wealist/dev" --recursive --with-decryption
```

**생성되는 SSM 파라미터:**
```
/wealist/dev/google-oauth/client-id
/wealist/dev/google-oauth/client-secret
/wealist/dev/jwt/secret
/wealist/dev/database/superuser-password
/wealist/dev/database/user-password
/wealist/dev/redis/password
/wealist/dev/minio/root-password
/wealist/dev/minio/access-key
/wealist/dev/minio/secret-key
/wealist/dev/livekit/api-key
/wealist/dev/livekit/api-secret
/wealist/dev/internal/api-key
```

### 4. Web Infra 설정 (프론트엔드 배포)

정적 웹사이트를 배포하기 위한 S3와 CloudFront를 구축합니다.

```bash
cd terraform/web-infra

# 1. 변수 파일 생성 (기존 버킷 이름 등 입력)
cp terraform.tfvars.example terraform.tfvars

# 2. Terraform 실행
terraform init
terraform apply

# 3. 배포된 도메인 확인
# terraform output cloudfront_domain_name

```

---

## 🏗️ 아키텍처 및 모듈 설명

### Backend Strategy (S3 Remote State)

* S3: 모든 인프라 상태(terraform.tfstate)를 중앙 저장소에 저장해 팀원 간 상태를 공유합니다.
* DynamoDB: `terraform apply` 실행시 state에 Lock을 걸어 동시에 여러 명이 배포해 상태가 꺠지는것을 방지합니다. 

### 주요 컴포넌트

1. **github-oidc (Module)**: Key가 없는 안전한 인증 방식(OIDC)을 사용하여 GitHub Actions에 임시 자격 증명을 부여합니다.
2. **ecr (Module)**: 마이크로서비스용 컨테이너 리포지토리를 생성하고 수명 주기 정책을 관리합니다.
3. **ssm-parameter (Module)**: SSM Parameter Store 시크릿 관리 - SecureString 타입으로 암호화 저장, External Secrets Operator와 연동
4. **web-infra**:
   * **S3**: 정적 파일 호스팅 (직접 접근 차단)
   * **CloudFront**: 전역 캐싱 및 HTTPS 제공, OAC(Origin Access Control)를 통한 보안 접근
   * **Route53**: 커스텀 도메인 연결 (선택 사항)


---

## 🔒 보안 가이드라인

1. **Git 업로드 절대 금지**:
   * `terraform.tfvars` (실제 비밀번호/키 값 포함)
   * `.terraform/` (임시 플러그인 폴더)
   * `*.tfstate*` (혹시 로컬에 생성된 백업 파일)

2. **권한 분리 원칙 (Least Privilege)**:
   * **인프라 관리자**: `default` 프로필 사용. VPC, IAM, CloudFront 등 리소스 생성/삭제 권한.
   * **서비스 개발자**: `wealist-dev` 프로필 사용. ECR Push, EKS 접근 등 개발 활동에 필요한 최소 권한.

3. **시크릿 관리**:
   * `terraform.tfvars`에 시크릿 저장 (gitignore됨)
   * SSM Parameter Store에 암호화 저장
   * K8s에서는 External Secrets Operator가 동기화

## 🗑️ 리소스 삭제

과금이 걱정되거나 프로젝트를 종료할 때 사용합니다.

```bash
# 각 디렉토리(web-infra, dev-environment 등)로 이동하여 수행
terraform destroy

```

```
