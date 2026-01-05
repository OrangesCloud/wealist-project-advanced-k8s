# Terraform Infrastructure

weAlist 프로젝트의 AWS 인프라를 관리하는 Terraform 구성입니다.

## 디렉토리 구조

```
terraform/
├── modules/                    # 재사용 가능한 Terraform 모듈
│   ├── ecr/                    # ECR 저장소 모듈
│   ├── github-oidc/            # GitHub Actions OIDC 인증 모듈
│   ├── pod-identity/           # EKS Pod Identity 모듈
│   └── ssm-parameter/          # SSM Parameter Store 모듈
├── global/                     # 전역 리소스 (환경 간 공유)
│   └── oidc-iam/               # GitHub Actions OIDC IAM 역할
├── dev/                        # 개발 환경
│   └── foundation/             # ECR 저장소
├── prod/                       # 프로덕션 환경
│   ├── foundation/             # VPC, RDS, Redis, ECR, S3, KMS
│   └── compute/                # EKS, Node Groups, Pod Identity, Istio, ArgoCD
│
└── dev-environment/            # 현재 사용 중 (state: dev-enviorment/)
```

## 환경별 설명

| 환경 | 용도 | 주요 리소스 |
|------|------|-------------|
| global | 환경 간 공유 리소스 | GitHub OIDC IAM |
| dev | 개발 환경 | ECR 저장소, IAM User |
| prod/foundation | 프로덕션 인프라 | VPC, RDS, Redis, ECR, S3, KMS |
| prod/compute | 프로덕션 컴퓨팅 | EKS, Node Groups, Pod Identity, Istio, ArgoCD |

## 배포 순서

### 1. 전역 리소스 배포

```bash
cd terraform/global/oidc-iam
terraform init
terraform plan
terraform apply
```

### 2. 프로덕션 Foundation 배포 (약 15-20분)

```bash
cd terraform/prod/foundation
terraform init
terraform plan
terraform apply
```

### 3. 프로덕션 Compute 배포 (약 15-20분)

```bash
cd terraform/prod/compute
terraform init
terraform plan
terraform apply
```

### 4. Post-Terraform 설정

EKS 클러스터 생성 후 추가 설정:

```bash
# kubeconfig 설정
aws eks update-kubeconfig --name wealist-prod-eks --region ap-northeast-2

# Gateway API CRDs 설치
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.2.0/standard-install.yaml

# Note: Istio는 Terraform helm-releases.tf에서 자동 설치됨
# 수동 설치 필요시: istioctl install -y

# 네임스페이스 생성 및 Sidecar 주입 활성화
# Note: Terraform namespaces.tf에서 자동 생성됨
kubectl create namespace wealist-prod
kubectl label namespace wealist-prod istio-injection=enabled
```

## EKS 팀원 접근 관리

### 아키텍처

```
IAM Users (기존) → IAM Groups (Terraform) → IAM Roles → EKS Access
     ↓                    ↓                      ↓
  팀원 추가         권한 그룹 정의           클러스터 접근
(AWS Console)    (자동 생성)             (자동 설정)
```

### 권한 레벨

| IAM Group | Role | EKS Policy | Scope |
|-----------|------|------------|-------|
| `wealist-prod-eks-admin` | eks-admin | ClusterAdminPolicy | 전체 클러스터 |
| `wealist-prod-eks-developer` | eks-developer | EditPolicy | wealist-prod, argocd |
| `wealist-prod-eks-readonly` | eks-readonly | ViewPolicy | 전체 조회 |

### 팀원 추가 방법 (AWS Console)

1. AWS Console → IAM → User groups
2. 해당 그룹 선택 (예: `wealist-prod-eks-developer`)
3. "Add users" → 팀원 선택

### 팀원 PC 설정

```bash
# 1. AWS CLI 설정 (본인 Access Key)
aws configure

# 2. kubeconfig 설정 (권한에 맞는 Role ARN 사용)
# Admin:
aws eks update-kubeconfig --name wealist-prod-eks --region ap-northeast-2 \
  --role-arn arn:aws:iam::<ACCOUNT_ID>:role/wealist-prod-eks-admin

# Developer:
aws eks update-kubeconfig --name wealist-prod-eks --region ap-northeast-2 \
  --role-arn arn:aws:iam::<ACCOUNT_ID>:role/wealist-prod-eks-developer

# ReadOnly:
aws eks update-kubeconfig --name wealist-prod-eks --region ap-northeast-2 \
  --role-arn arn:aws:iam::<ACCOUNT_ID>:role/wealist-prod-eks-readonly

# 3. 테스트
kubectl get pods -n wealist-prod
```

### 관련 Terraform Outputs

```bash
cd terraform/prod/compute

# Group 이름 확인
terraform output eks_iam_groups

# 접근 명령어 확인
terraform output eks_access_commands

# 설정 가이드 확인
terraform output eks_access_setup_guide
```

## State 관리

모든 Terraform 상태는 S3에 저장됩니다:

| 디렉토리 | State 경로 | 상태 |
|----------|-----------|------|
| global/oidc-iam | `global/oidc-iam/terraform.tfstate` | 사용 중 |
| dev-environment | `dev-enviorment/terraform.tfstate` | 사용 중 |
| prod/foundation | `prod/foundation/terraform.tfstate` | 사용 중 |
| prod/compute | `prod/compute/terraform.tfstate` | 사용 중 |

### 초기 설정 (One-time Setup)

Terraform 상태 파일을 팀원들과 공유하기 위해 S3 버킷이 필요합니다.
**관리자 권한**으로 아래 명령어를 1회만 실행해주세요.

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

### State 백업

```bash
# 전체 백업
aws s3 cp s3://wealist-tf-state-advanced-k8s/ ./terraform-state-backup/ --recursive
```

## 예상 비용 (월간)

### 프로덕션 환경 (~$210/월)

| 리소스 | 스펙 | 예상 비용 |
|--------|------|----------|
| EKS Control Plane | - | $73 |
| NAT Gateway | 1개 (단일) | ~$32 |
| RDS PostgreSQL | db.t4g.small (Single-AZ) | ~$23 |
| ElastiCache Redis | cache.t4g.small (1노드) | ~$20 |
| EC2 Spot | 3 x t3.medium | ~$30 |
| EBS 스토리지 | 50GB x 3 노드 | ~$15 |

### 비용 최적화 결정 사항

1. **NAT Gateway**: 단일 NAT Gateway 사용 (SPOF 주의)
2. **RDS**: Single-AZ 배포 (추후 Multi-AZ 전환 가능)
3. **Redis**: 단일 노드 (복제 없음)
4. **노드**: 전체 Spot 인스턴스 (다양한 타입으로 가용성 확보)

## Istio Sidecar 지원

### 필수 설정

prod/compute의 EKS 구성에 Istio Sidecar 모드를 위한 설정이 포함되어 있습니다:

1. **VPC CNI 설정**
   ```hcl
   POD_SECURITY_GROUP_ENFORCING_MODE = "standard"
   ```

2. **Security Group 포트**
   - TCP 15001-15006: Envoy Sidecar 트래픽 redirect
   - TCP 15012: XDS (istiod ↔ Sidecar 통신)
   - TCP 15020-15021: Metrics, readiness

### Istio 설치

Istio는 Terraform helm-releases.tf에서 자동 설치됩니다.

수동 설치 필요시:
```bash
# Istio Sidecar 모드 설치 (default profile)
istioctl install -y

# 상태 확인
istioctl proxy-status
kubectl get pods -n istio-system
```

## 모듈 문서

각 모듈과 환경의 상세 문서:

| 문서 | 설명 |
|------|------|
| [modules/pod-identity/README.md](modules/pod-identity/README.md) | Pod Identity 모듈 사용법 |
| [global/oidc-iam/README.md](global/oidc-iam/README.md) | GitHub OIDC 설정 |
| [dev/foundation/README.md](dev/foundation/README.md) | 개발 환경 리소스 |
| [prod/foundation/README.md](prod/foundation/README.md) | 프로덕션 인프라 |
| [prod/compute/README.md](prod/compute/README.md) | EKS 클러스터 설정 |

## 보안 가이드라인

1. **Git 업로드 절대 금지**:
   - `terraform.tfvars` (실제 비밀번호/키 값 포함)
   - `.terraform/` (임시 플러그인 폴더)
   - `*.tfstate*` (혹시 로컬에 생성된 백업 파일)

2. **권한 분리 원칙 (Least Privilege)**:
   - **인프라 관리자**: `default` 프로필 사용. VPC, IAM, CloudFront 등 리소스 생성/삭제 권한.
   - **서비스 개발자**: `wealist-dev` 프로필 사용. ECR Push, EKS 접근 등 개발 활동에 필요한 최소 권한.

3. **시크릿 관리**:
   - RDS, Redis 비밀번호: Secrets Manager 자동 관리
   - SSM Parameter Store: 애플리케이션 시크릿 저장
   - K8s: External Secrets Operator가 동기화

## 주의사항

1. **terraform apply 직접 실행**: 모든 apply는 사용자가 직접 실행해야 합니다
2. **순서 준수**: foundation → compute 순서로 배포 (의존성)
3. **State 백업**: 중요 변경 전 항상 State 백업

## 트러블슈팅

### EKS 클러스터 접근 불가

```bash
# kubeconfig 재설정
aws eks update-kubeconfig --name wealist-prod-eks --region ap-northeast-2

# IAM 권한 확인
aws sts get-caller-identity
```

### Terraform State 잠금

```bash
# DynamoDB 잠금 해제 (주의: 다른 작업 확인 필요)
aws dynamodb delete-item \
  --table-name terraform-lock \
  --key '{"LockID":{"S":"wealist-tf-state-advanced-k8s/prod/compute/terraform.tfstate"}}'
```

### Add-on 업데이트 실패

```bash
# Add-on 상태 확인
aws eks describe-addon --cluster-name wealist-prod-eks --addon-name vpc-cni

# 강제 업데이트
aws eks update-addon --cluster-name wealist-prod-eks --addon-name vpc-cni --resolve-conflicts OVERWRITE
```

## 리소스 삭제

과금이 걱정되거나 프로젝트를 종료할 때 사용합니다.
**순서 주의**: compute → foundation → global 순으로 삭제해야 합니다.

```bash
# 1. Compute 삭제 (EKS, Istio, ArgoCD)
cd terraform/prod/compute
terraform destroy

# 2. Foundation 삭제 (VPC, RDS, Redis)
cd terraform/prod/foundation
terraform destroy

# 3. Global 삭제 (OIDC IAM)
cd terraform/global/oidc-iam
terraform destroy
```

## 디렉토리 정리

### terraform/dev/foundation/ (새 구조, 미적용)

`dev-environment/`가 현재 사용 중. 마이그레이션하려면:

```bash
# State 복사 (경로에 오타 있음 주의: dev-enviorment)
aws s3 cp s3://wealist-tf-state-advanced-k8s/dev-enviorment/terraform.tfstate \
          s3://wealist-tf-state-advanced-k8s/dev/foundation/terraform.tfstate

cd terraform/dev/foundation
terraform init
terraform plan  # 변경사항 없어야 함

# 기존 디렉토리 삭제
rm -rf terraform/dev-environment/
```
