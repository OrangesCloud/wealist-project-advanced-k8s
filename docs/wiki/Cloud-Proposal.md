# Cloud Proposal

weAlist 클라우드 제안서입니다.

---

## Full Document

> 상세 클라우드 제안서: [Google Docs](https://docs.google.com/document/d/1DiVO6p0NjmxzoEXwG3hZ7KoZLpqU-iiSdLWmOvTuH_s)

> 클라우드 설계/아키텍처: [Google Docs](https://docs.google.com/document/d/1K2L1s3t15OCGDkmCfuXjLbpeDbREeuoT1OP1ldCSGY8)

---

## Executive Summary

weAlist는 AWS 기반 클라우드 네이티브 아키텍처를 채택하여 높은 가용성, 확장성, 보안을 제공합니다.

---

## Proposed Architecture

### Compute
| Service | AWS Resource | Spec |
|---------|--------------|------|
| Kubernetes | EKS | 1.28+ |
| Worker Nodes | EC2 (m5.large) | 3-6 nodes |
| Auto Scaling | Cluster Autoscaler | Min 3, Max 10 |

### Database
| Service | AWS Resource | Spec |
|---------|--------------|------|
| PostgreSQL | RDS | db.t3.medium |
| Redis | ElastiCache | cache.t3.small |

### Storage
| Service | AWS Resource | Spec |
|---------|--------------|------|
| Object Storage | S3 | Standard |
| Block Storage | EBS | gp3 |

### Networking
| Service | AWS Resource | Spec |
|---------|--------------|------|
| Load Balancer | ALB | Public |
| DNS | Route53 | wealist.co.kr |
| CDN | CloudFront | Optional |

---

## Cost Estimation (Monthly)

| Resource | Spec | Cost (USD) |
|----------|------|------------|
| EKS Cluster | 1 cluster | $73 |
| EC2 (m5.large) | 3 nodes | $210 |
| RDS (PostgreSQL) | db.t3.medium | $50 |
| ElastiCache | cache.t3.small | $25 |
| ALB | 1 LB | $20 |
| S3 | 100GB | $3 |
| Data Transfer | 100GB | $10 |
| **Total** | | **~$400** |

> 실제 비용은 사용량에 따라 변동됩니다.

---

## Migration Plan

### Phase 1: Infrastructure
1. VPC/Subnet 구성
2. EKS 클러스터 생성
3. RDS/ElastiCache 프로비저닝

### Phase 2: Application
1. Container Registry 설정
2. Helm 차트 배포
3. DNS 설정

### Phase 3: Operations
1. 모니터링 구성
2. 백업 정책 설정
3. CI/CD 파이프라인 연결

---

## Related Pages

- [Requirements](Requirements.md)
- [AWS Architecture](Architecture-AWS.md)
