# AWS Architecture

weAlist의 AWS 인프라 아키텍처입니다.

---

## Architecture Diagram

![AWS Architecture](https://raw.githubusercontent.com/OrangesCloud/wealist-project-advanced-k8s/main/docs/images/wealist_aws_arch_v2.png)

---

## Components

### Compute
- **Amazon EKS** - Kubernetes 클러스터
- **EC2 Node Groups** - 워커 노드
- **Cluster Autoscaler** - 자동 스케일링

### Networking
- **VPC** - 격리된 네트워크
- **Public/Private Subnets** - 서브넷 분리
- **NAT Gateway** - 아웃바운드 트래픽
- **ALB** - Application Load Balancer

### Database
- **Amazon RDS (PostgreSQL)** - 관리형 DB
- **Amazon ElastiCache (Redis)** - 캐시

### Storage
- **Amazon S3** - 오브젝트 스토리지
- **EBS** - 블록 스토리지

### Security
- **IAM** - 권한 관리
- **Security Groups** - 네트워크 ACL
- **Secrets Manager** - 시크릿 관리

---

## Cost Optimization

| Resource | Strategy |
|----------|----------|
| EKS | Spot Instances for non-critical workloads |
| RDS | Reserved Instances for production |
| S3 | Lifecycle policies for old data |
| NAT | Single NAT for dev environments |

---

## Terraform Infrastructure (IaC)

2-layer 아키텍처로 인프라를 관리합니다.

```
terraform/
├── modules/                    # 재사용 가능한 Terraform 모듈
│   ├── ecr/                    # ECR 저장소 모듈
│   ├── github-oidc/            # GitHub Actions OIDC 인증 모듈
│   ├── pod-identity/           # EKS Pod Identity 모듈
│   └── ssm-parameter/          # SSM Parameter Store 모듈
├── global/                     # 전역 리소스 (환경 간 공유)
│   └── oidc-iam/               # GitHub Actions OIDC IAM 역할
├── dev-environment/            # 개발 환경 (ECR, IAM User)
├── web-infra/                  # CloudFront + S3 (Frontend)
└── prod/                       # 프로덕션 환경
    ├── foundation/             # Layer 1: VPC, RDS, Redis, ECR, S3, KMS
    └── compute/                # Layer 2: EKS, Istio, ArgoCD, Pod Identity
```

### Layer 1: Foundation (`prod/foundation/`)

| 리소스 | 파일 | 설명 |
|--------|------|------|
| VPC | `main.tf` | 10.0.0.0/16, Public/Private Subnets, NAT Gateway |
| RDS | `rds.tf` | PostgreSQL 17 (db.t4g.small, Single-AZ) |
| ElastiCache | `elasticache.tf` | Redis 7.2 (cache.t4g.small) |
| ECR | `ecr.tf` | 서비스별 컨테이너 레지스트리 |
| S3 | `s3.tf` | 파일 스토리지 버킷 |
| KMS | `kms.tf` | 암호화 키 |
| Secrets Manager | `secrets.tf` | DB/Redis 비밀번호 자동 관리 |

### Layer 2: Compute (`prod/compute/`)

| 리소스 | 파일 | 설명 |
|--------|------|------|
| EKS | `eks.tf` | Kubernetes 1.34, Managed Node Groups (Spot) |
| Helm Releases | `helm-releases.tf` | Istio, AWS LB Controller, ESO, Cluster Autoscaler |
| ArgoCD | `argocd-bootstrap.tf` | ArgoCD 설치 및 App of Apps 패턴 |
| Pod Identity | `pod-identity.tf` | S3, Secrets Manager 접근 권한 |
| IAM Access | `iam-eks-access.tf` | 팀원별 EKS 접근 권한 (admin/developer/readonly) |
| Scheduled Scaling | `scheduled-scaling.tf` | 야간/주말 노드 스케일 다운 |
| Namespaces | `namespaces.tf` | wealist-prod 네임스페이스 + Istio sidecar 주입 |

> **Note**: `terraform apply`는 layer 순서대로 실행: `foundation` → `compute`

---

## Detailed Design

> 상세 설계 문서: [클라우드 설계/아키텍처 (Google Docs)](https://docs.google.com/document/d/1K2L1s3t15OCGDkmCfuXjLbpeDbREeuoT1OP1ldCSGY8)

---

## Related Pages

- [Architecture Overview](Architecture)
- [Security (VPC)](Architecture-VPC)
- [CI/CD Pipeline](Architecture-CICD)
