# Security (VPC)

weAlist의 네트워크 및 보안 아키텍처입니다.

---

## Traffic Flow

![Traffic Flow](../images/wealist_vpc_security_aws-Traffic%20Flow.png)

### Traffic Flow Reference

| # | 구간 | Protocol | Port | Source |
|---|------|----------|------|--------|
| ① | Internet → CDN | HTTPS | 443 | 0.0.0.0/0 |
| ② | CDN → ALB | HTTP/S | 80, 443 | CloudFront IPs |
| ③ | ALB → EKS | HTTP | 8000-8081 | ALB-SG |
| ④ | EKS → RDS | TCP | 5432 | EKS-SG |
| ⑤ | EKS → Redis | TCP | 6379 | EKS-SG |

> **Note**: NAT Gateway는 EKS에서 외부 API 호출 시 outbound 전용 경로 (점선)

---

## Security Groups

![Security Groups](../images/wealist_vpc_security_aws-Security%20Groups.png)

### Security Group Rules

| SG | Inbound | Source |
|----|---------|--------|
| ALB-SG | 80, 443 | 0.0.0.0/0 |
| EKS-SG | 8000-8081 | ALB-SG |
| RDS-SG | 5432 | EKS-SG |
| Redis-SG | 6379 | EKS-SG |

### Service Ports (EKS-SG)

| Service | Port | Description |
|---------|------|-------------|
| auth-service | 8080 | JWT, OAuth2 |
| user-service | 8081 | Users, Workspaces |
| board-service | 8000 | Projects, Boards |
| chat-service | 8001 | Real-time messaging |
| noti-service | 8002 | Push notifications |
| storage-service | 8003 | File storage |
| frontend | 3000 | Web UI |

---

## Network Design

### VPC CIDR
```
VPC: 10.0.0.0/16

Public Subnets:
  - 10.0.1.0/24 (AZ-a)
  - 10.0.2.0/24 (AZ-b)

Private Subnets:
  - 10.0.11.0/24 (AZ-a) - Application
  - 10.0.12.0/24 (AZ-b) - Application
  - 10.0.21.0/24 (AZ-a) - Database
  - 10.0.22.0/24 (AZ-b) - Database
```

### Traffic Flow
```
Internet
    │
    ▼
Internet Gateway
    │
    ▼
ALB (Public Subnet)
    │
    ▼
EKS Pods (Private Subnet - App)
    │
    ▼
RDS/ElastiCache (Private Subnet - DB)
```

---

## Security Measures

### Network Security
- Private subnets for workloads
- NAT Gateway for outbound only
- Network ACLs

### Application Security
- mTLS with Istio (Phase 3)
- JWT token validation
- RBAC in Kubernetes

### Data Security
- Encryption at rest (RDS, S3)
- Encryption in transit (TLS)
- Secrets Manager for credentials

### Compliance
- **Trivy** - Container scanning
- **Kyverno** - Policy enforcement
- Audit logging

---

## Related Pages

- [Architecture Overview](Architecture.md)
- [AWS Architecture](Architecture-AWS.md)
