# Prod Foundation

프로덕션 환경의 기반 인프라 리소스입니다.

## 개요

이 레이어는 EKS 클러스터보다 변경이 적고 수명이 긴 리소스들을 관리합니다.

## 구성 요소

| 리소스 | 모듈/리소스 | 설명 |
|--------|------------|------|
| VPC | terraform-aws-modules/vpc | 네트워크 인프라 |
| RDS PostgreSQL | terraform-aws-modules/rds | 관계형 데이터베이스 |
| ElastiCache Redis | aws_elasticache_* | 캐시/세션 저장소 |
| ECR | ../../modules/ecr | 프로덕션 이미지 저장소 |
| S3 | aws_s3_bucket | 파일 스토리지 |
| KMS | terraform-aws-modules/kms | 암호화 키 |

## 아키텍처

```
┌─────────────────────────────────────────────────────────────────┐
│ VPC (10.0.0.0/16)                                               │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  │
│  │ Public Subnet   │  │ Public Subnet   │  │ Public Subnet   │  │
│  │ 10.0.101.0/24   │  │ 10.0.102.0/24   │  │ 10.0.103.0/24   │  │
│  │ (AZ-a)          │  │ (AZ-b)          │  │ (AZ-c)          │  │
│  └────────┬────────┘  └─────────────────┘  └─────────────────┘  │
│           │                                                     │
│       ┌───┴───┐                                                 │
│       │  NAT  │  ← 단일 NAT Gateway (비용 최적화)               │
│       └───┬───┘                                                 │
│           │                                                     │
│  ┌────────┴────────┐  ┌─────────────────┐  ┌─────────────────┐  │
│  │ Private Subnet  │  │ Private Subnet  │  │ Private Subnet  │  │
│  │ 10.0.1.0/24     │  │ 10.0.2.0/24     │  │ 10.0.3.0/24     │  │
│  │ (EKS Nodes)     │  │ (EKS Nodes)     │  │ (EKS Nodes)     │  │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘  │
│                                                                 │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐  │
│  │ Database Subnet │  │ Database Subnet │  │ Database Subnet │  │
│  │ 10.0.201.0/24   │  │ 10.0.202.0/24   │  │ 10.0.203.0/24   │  │
│  │ (RDS, Redis)    │  │                 │  │                 │  │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## 리소스 상세

### VPC

| 설정 | 값 | 비고 |
|------|-----|------|
| CIDR | 10.0.0.0/16 | 65,536 IP |
| AZs | 3개 (a, b, c) | 고가용성 |
| NAT Gateway | **1개** | 비용 최적화 (SPOF 주의) |
| VPC Flow Logs | 비활성화 | 비용 절감 |

### RDS PostgreSQL

| 설정 | 값 | 비고 |
|------|-----|------|
| 엔진 | PostgreSQL 17.7 | 최신 버전 |
| 인스턴스 | db.t4g.small | 2 vCPU, 2GB RAM |
| 스토리지 | 20GB (auto-scaling 100GB) | gp3 |
| Multi-AZ | **비활성화** | 비용 절감 (~$23/월) |
| 암호화 | KMS | at-rest |
| 백업 | 7일 보존 | 자동 백업 |
| 비밀번호 | Secrets Manager | 자동 관리 |

### ElastiCache Redis

| 설정 | 값 | 비고 |
|------|-----|------|
| 엔진 | Redis 7.1 | 최신 버전 |
| 노드 타입 | cache.t4g.small | 1.4GB RAM |
| 노드 수 | **1개** | 비용 절감 (~$20/월) |
| 클러스터 모드 | 비활성화 | 단일 샤드 |
| 암호화 | 활성화 | at-rest, in-transit |
| 인증 | AUTH 토큰 | Secrets Manager 저장 |

### ECR

| 설정 | 값 |
|------|-----|
| Prefix | prod/ |
| Tag Mutability | **Immutable** |
| 스캔 | 푸시 시 자동 |
| 정리 정책 | untagged 1일, 최근 50개 유지 |

### S3

| 설정 | 값 |
|------|-----|
| 버킷 이름 | wealist-prod-storage |
| 버전 관리 | 활성화 |
| 암호화 | SSE-S3 |
| CORS | wealist.co.kr 허용 |
| 수명 주기 | 불완전 업로드 7일 후 삭제 |

### KMS

| 설정 | 값 |
|------|-----|
| 별칭 | wealist-prod |
| 키 로테이션 | 활성화 (연간) |
| 삭제 대기 | 7일 |
| 용도 | RDS, Redis, S3, EKS Secrets |

## 배포

### 사전 요구사항

1. S3 버킷: `wealist-tf-state-advanced-k8s`
2. DynamoDB 테이블: `terraform-lock`
3. AWS CLI 설정

### 배포 명령

```bash
cd terraform/prod/foundation

# 초기화
terraform init

# 계획 확인 (약 2분)
terraform plan

# 적용 (약 15-20분)
# 주의: RDS, Redis 생성에 시간 소요
terraform apply
```

### 배포 소요 시간

| 리소스 | 예상 시간 |
|--------|----------|
| VPC | 2-3분 |
| RDS | 8-12분 |
| Redis | 5-8분 |
| ECR | 1분 |
| S3, KMS | 1분 |

## 출력 값

```hcl
# VPC
vpc_id              = "vpc-xxx"
private_subnet_ids  = ["subnet-a", "subnet-b", "subnet-c"]

# RDS
rds_endpoint        = "wealist-prod-postgres.xxx.ap-northeast-2.rds.amazonaws.com:5432"
rds_master_user_secret_arn = "arn:aws:secretsmanager:..."

# Redis
redis_endpoint      = "wealist-prod-redis.xxx.cache.amazonaws.com"
redis_auth_token_secret_arn = "arn:aws:secretsmanager:..."

# S3
s3_bucket_name      = "wealist-prod-storage"

# KMS
kms_key_arn         = "arn:aws:kms:..."
```

## prod/compute 연동

이 레이어의 출력은 `prod/compute`에서 `terraform_remote_state`로 참조합니다:

```hcl
# prod/compute/main.tf
data "terraform_remote_state" "foundation" {
  backend = "s3"
  config = {
    bucket = "wealist-tf-state-advanced-k8s"
    key    = "prod/foundation/terraform.tfstate"
    region = "ap-northeast-2"
  }
}

# 사용 예시
vpc_id = data.terraform_remote_state.foundation.outputs.vpc_id
```

## Secret 접근

### RDS 비밀번호

```bash
# Secrets Manager에서 조회
aws secretsmanager get-secret-value \
  --secret-id $(terraform output -raw rds_master_user_secret_arn) \
  --query SecretString --output text | jq -r .password
```

### Redis AUTH 토큰

```bash
aws secretsmanager get-secret-value \
  --secret-id $(terraform output -raw redis_auth_token_secret_arn) \
  --query SecretString --output text
```

## 비용 최적화 결정사항

### 현재 설정 (최소 비용)

| 항목 | 선택 | 비용/월 | 트레이드오프 |
|------|------|---------|-------------|
| NAT Gateway | 1개 | ~$32 | AZ 장애 시 outbound 불가 |
| RDS | Single-AZ | ~$23 | DB 장애 시 복구 시간 필요 |
| Redis | 1노드 | ~$20 | 캐시 장애 시 복구 필요 |

### 향후 확장 시

```hcl
# Multi-AZ RDS (+$23/월)
variable "rds_multi_az" {
  default = true
}

# NAT Gateway 3개 (+$64/월)
variable "single_nat_gateway" {
  default = false
}

# Redis 복제 (+$20/월)
variable "redis_num_cache_clusters" {
  default = 2
}
```

## State 위치

```
s3://wealist-tf-state-advanced-k8s/prod/foundation/terraform.tfstate
```

## 주의사항

1. **삭제 보호**: RDS에 deletion protection 활성화 (기본값)
2. **암호화**: 모든 데이터 암호화 (KMS)
3. **네트워크 격리**: Database subnet은 인터넷 접근 불가
4. **순서**: 이 레이어가 prod/compute보다 먼저 배포되어야 함

## 트러블슈팅

### RDS 연결 실패

```bash
# Security Group 확인
aws ec2 describe-security-groups --group-ids <rds-sg-id>

# VPC 엔드포인트 확인 (필요 시)
aws ec2 describe-vpc-endpoints --filters Name=vpc-id,Values=<vpc-id>
```

### Redis 연결 실패

```bash
# ElastiCache 상태 확인
aws elasticache describe-replication-groups --replication-group-id wealist-prod-redis

# AUTH 토큰 확인
aws secretsmanager get-secret-value --secret-id <secret-arn>
```
