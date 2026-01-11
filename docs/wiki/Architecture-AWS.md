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

3-layer 아키텍처로 인프라를 관리합니다.

```
terraform/prod/
├── foundation/      # Layer 1: VPC, Subnets, Security Groups, IAM
├── compute/         # Layer 2: EKS, Node Groups, ALB Controller, ESO
└── argocd-apps/     # Layer 3: ArgoCD Applications (App of Apps)
```

### Layer 1: Foundation
- VPC (10.0.0.0/16)
- Public/Private Subnets (Multi-AZ)
- Security Groups
- IAM Roles (IRSA)
- KMS Keys

### Layer 2: Compute
- EKS 1.34 Cluster
- Managed Node Groups (t3.medium)
- AWS Load Balancer Controller
- External Secrets Operator (ESO)
- Cluster Autoscaler

### Layer 3: ArgoCD Apps
- ArgoCD Application 정의 (App of Apps 패턴)
- 서비스별 Application CRD
- 환경별 분리 (dev, prod)

> **Note**: `terraform apply`는 layer 순서대로 실행: foundation → compute → argocd-apps

---

## Detailed Design

> 상세 설계 문서: [클라우드 설계/아키텍처 (Google Docs)](https://docs.google.com/document/d/1K2L1s3t15OCGDkmCfuXjLbpeDbREeuoT1OP1ldCSGY8)

---

## Related Pages

- [Architecture Overview](Architecture)
- [Security (VPC)](Architecture-VPC)
- [CI/CD Pipeline](Architecture-CICD)
