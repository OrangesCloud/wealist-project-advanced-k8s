# 🎨 AWS draw.io 다이어그램 스타일 가이드

> weAlist 프로젝트 아키텍처 다이어그램 작성용
> 작성일: 2025-12-16
> AWS 색상: 2024-2025 공식 팔레트 적용

---

## 📐 기본 설정

### 캔버스/이미지 크기
```yaml
권장 크기:
  - 너비: 1400px (최소 1000px)
  - 높이: 비율 유지 (약 900~1000px)

내보내기 설정:
  - Format: PNG 또는 SVG
  - Scale: 200% (고해상도)
  - Border: 20px
  - Background: 흰색 (#FFFFFF)

GitHub 위키용:
  - SVG 권장 (벡터라 확대해도 선명)
  - PNG는 1400px 이상
```

### 선 굵기
```yaml
권장:
  - 화살표/연결선: 1~1.5px
  - 그룹 테두리: 1~2px
  - 강조선: 2px
```

### 폰트 크기 (용도별)

#### GitHub Wiki / README (모니터 뷰)
```yaml
제목 (Title):
  - 크기: 18-24px
  - 스타일: Bold (#232F3E)
  - 예: "EKS Cluster - Workloads"

서비스명 (Service Name):
  - 크기: 12-14px
  - 스타일: Bold
  - 예: "auth-service", "user-service"

라벨/설명 (Labels):
  - 크기: 10-12px (절대 최소 10px)
  - 스타일: Regular
  - 예: "ClusterIP :8080", "deploy: 2 replicas"

캡션/주석:
  - 크기: 10px
  - 스타일: Italic (선택)
  - 예: "7 Databases", "JWT Tokens | Cache"
```

#### PPT 발표용 (프로젝터 뷰)
```yaml
제목 (Title):
  - 크기: 28-36px
  - 스타일: Bold

서비스명 (Service Name):
  - 크기: 18-24px
  - 스타일: Bold

라벨/설명 (Labels):
  - 크기: 14-16px
  - 스타일: Regular

캡션/주석:
  - 크기: 12px
  - 스타일: Regular
```

#### 폰트 크기 비교표
| 요소 | Wiki/README | PPT 발표 |
|------|-------------|----------|
| 제목 | 18-24px | 28-36px |
| 서비스명 | 12-14px | 18-24px |
| 라벨/설명 | 10-12px | 14-16px |
| 캡션 | 10px | 12px |

> **참고**: 프로젝터 발표 시 200% Scale PNG로 내보내면 더 선명

---

## 🏗️ AWS 그룹 스타일 (mxgraph.aws4.group)

### VPC
```
shape=mxgraph.aws4.group;
grIcon=mxgraph.aws4.group_vpc;
strokeColor=#248814;
fillColor=none;
verticalAlign=top;
align=left;
spacingLeft=30;
fontColor=#248814;
dashed=0;
```

### AWS Cloud
```
shape=mxgraph.aws4.group;
grIcon=mxgraph.aws4.group_aws_cloud;
strokeColor=#232F3E;
fillColor=none;
verticalAlign=top;
align=left;
spacingLeft=30;
dashed=0;
```

### Region
```
shape=mxgraph.aws4.group;
grIcon=mxgraph.aws4.group_region;
strokeColor=#147EBA;
fillColor=none;
verticalAlign=top;
align=left;
spacingLeft=30;
fontColor=#147EBA;
dashed=0;
```

### Security Group (⭐ 중요)
```
shape=mxgraph.aws4.group;
grIcon=mxgraph.aws4.group_security_group;
strokeColor=#DD3522;
fillColor=none;
verticalAlign=top;
align=left;
spacingLeft=30;
fontColor=#DD3522;
dashed=1;
```

### Public Subnet
```
shape=mxgraph.aws4.group;
grIcon=mxgraph.aws4.group_public_subnet;
strokeColor=#248814;
fillColor=none;
verticalAlign=top;
align=left;
spacingLeft=30;
fontColor=#248814;
dashed=0;
```

### Private Subnet
```
shape=mxgraph.aws4.group;
grIcon=mxgraph.aws4.group_private_subnet;
strokeColor=#147EBA;
fillColor=none;
verticalAlign=top;
align=left;
spacingLeft=30;
fontColor=#147EBA;
dashed=0;
```

### Auto Scaling Group
```
shape=mxgraph.aws4.group;
grIcon=mxgraph.aws4.group_auto_scaling_group;
strokeColor=#ED7100;
fillColor=none;
dashed=1;
```

### Availability Zone
```
shape=mxgraph.aws4.group;
grIcon=mxgraph.aws4.group_availability_zone;
strokeColor=#147EBA;
fillColor=none;
dashed=1;
```

---

## 🎯 AWS 서비스 아이콘 (mxgraph.aws4.resourceIcon)

### 기본 스타일 템플릿
```
sketch=0;
outlineConnect=0;
fontColor=#232F3E;
gradientColor=[GRADIENT];
gradientDirection=north;
fillColor=[FILL];
strokeColor=#ffffff;
dashed=0;
verticalLabelPosition=bottom;
verticalAlign=top;
align=center;
html=1;
fontSize=12;
fontStyle=0;
aspect=fixed;
shape=mxgraph.aws4.resourceIcon;
resIcon=mxgraph.aws4.[SERVICE];
```

### 네트워킹 (보라색 #8C4FFF)
| 서비스 | resIcon | fillColor | gradientColor |
|--------|---------|-----------|---------------|
| CloudFront | `mxgraph.aws4.cloudfront` | #8C4FFF | #F34482 |
| ALB | `mxgraph.aws4.application_load_balancer` | #8C4FFF | #F34482 |
| NLB | `mxgraph.aws4.network_load_balancer` | #8C4FFF | #F34482 |
| VPC | `mxgraph.aws4.vpc` | #8C4FFF | #F34482 |
| Route53 | `mxgraph.aws4.route_53` | #8C4FFF | #F34482 |
| API Gateway | `mxgraph.aws4.api_gateway` | #8C4FFF | #F34482 |
| NAT Gateway | `mxgraph.aws4.nat_gateway` | #8C4FFF | #F34482 |
| Internet Gateway | `mxgraph.aws4.internet_gateway` | #8C4FFF | #F34482 |

### 컴퓨팅 (주황색 #ED7100)
| 서비스 | resIcon | fillColor | gradientColor |
|--------|---------|-----------|---------------|
| EKS | `mxgraph.aws4.elastic_kubernetes_service` | #ED7100 | #F78E04 |
| ECS | `mxgraph.aws4.elastic_container_service` | #ED7100 | #F78E04 |
| EC2 | `mxgraph.aws4.ec2` | #ED7100 | #F78E04 |
| Lambda | `mxgraph.aws4.lambda` | #ED7100 | #F78E04 |
| Fargate | `mxgraph.aws4.fargate` | #ED7100 | #F78E04 |
| ECR | `mxgraph.aws4.ecr` | #ED7100 | #F78E04 |

### 데이터베이스 (보라/핑크 #C925D1)
| 서비스 | resIcon | fillColor | gradientColor |
|--------|---------|-----------|---------------|
| RDS | `mxgraph.aws4.rds` | #C925D1 | #F34482 |
| Aurora | `mxgraph.aws4.aurora` | #C925D1 | #F34482 |
| DynamoDB | `mxgraph.aws4.dynamodb` | #C925D1 | #F34482 |
| ElastiCache | `mxgraph.aws4.elasticache` | #C925D1 | #F34482 |
| DocumentDB | `mxgraph.aws4.documentdb` | #C925D1 | #F34482 |

### 스토리지 (녹색 #7AA116)
| 서비스 | resIcon | fillColor | gradientColor |
|--------|---------|-----------|---------------|
| S3 | `mxgraph.aws4.s3` | #7AA116 | #60A337 |
| EBS | `mxgraph.aws4.elastic_block_store` | #7AA116 | #60A337 |
| EFS | `mxgraph.aws4.elastic_file_system` | #7AA116 | #60A337 |

### 보안 (빨간색 #DD344C)
| 서비스 | resIcon | fillColor | gradientColor |
|--------|---------|-----------|---------------|
| IAM | `mxgraph.aws4.identity_and_access_management` | #DD344C | #FF5252 |
| Secrets Manager | `mxgraph.aws4.secrets_manager` | #DD344C | #FF5252 |
| Certificate Manager | `mxgraph.aws4.certificate_manager` | #DD344C | #FF5252 |
| WAF | `mxgraph.aws4.waf` | #DD344C | #FF5252 |
| Cognito | `mxgraph.aws4.cognito` | #DD344C | #FF5252 |

### 관리/모니터링 (분홍색 #BC1356)
| 서비스 | resIcon | fillColor | gradientColor |
|--------|---------|-----------|---------------|
| CloudWatch | `mxgraph.aws4.cloudwatch` | #BC1356 | #F34482 |
| CloudTrail | `mxgraph.aws4.cloudtrail` | #BC1356 | #F34482 |
| Systems Manager | `mxgraph.aws4.systems_manager` | #BC1356 | #F34482 |
| X-Ray | `mxgraph.aws4.xray` | #BC1356 | #F34482 |

### 개발자 도구 (파란색 #2E73B8)
| 서비스 | resIcon | fillColor | gradientColor |
|--------|---------|-----------|---------------|
| CodePipeline | `mxgraph.aws4.codepipeline` | #2E73B8 | #5294CF |
| CodeBuild | `mxgraph.aws4.codebuild` | #2E73B8 | #5294CF |
| CodeDeploy | `mxgraph.aws4.codedeploy` | #2E73B8 | #5294CF |
| CodeCommit | `mxgraph.aws4.codecommit` | #2E73B8 | #5294CF |

### 메시징/App Integration (핑크 #E7157B)
| 서비스 | resIcon | fillColor | gradientColor |
|--------|---------|-----------|---------------|
| SQS | `mxgraph.aws4.sqs` | #E7157B | #F34482 |
| SNS | `mxgraph.aws4.sns` | #E7157B | #F34482 |
| EventBridge | `mxgraph.aws4.eventbridge` | #E7157B | #F34482 |

---

## 🌐 일반 아이콘

### 인터넷/사용자
```
# Internet (구름)
shape=mxgraph.aws4.internet;
fillColor=#232F3E;
strokeColor=#232F3E;

# 사용자
shape=mxgraph.aws4.users;
fillColor=#232F3E;
strokeColor=#232F3E;

# 클라이언트 (데스크톱)
shape=mxgraph.aws4.client;
fillColor=#232F3E;
strokeColor=#232F3E;

# 모바일
shape=mxgraph.aws4.mobile_client;
fillColor=#232F3E;
strokeColor=#232F3E;
```

---

## 🎨 색상 코드 정리

### AWS 카테고리 색상 (2024-2025 최신)
| 카테고리 | AWS 이름 | Primary | 용도 |
|----------|----------|---------|------|
| Compute | Smile | #ED7100 | EC2, EKS, Lambda, Fargate |
| Storage | Endor | #7AA116 | S3, EBS, EFS |
| Database | Nebula | #C925D1 | RDS, ElastiCache, DynamoDB |
| Networking | Galaxy | #8C4FFF | VPC, ALB, CloudFront, Route53 |
| Security | Mars | #DD344C | IAM, WAF, Cognito |
| App Integration | Cosmos | #E7157B | SQS, SNS, EventBridge |
| Management | - | #BC1356 | CloudWatch, CloudTrail |
| Developer | - | #2E73B8 | CodePipeline, CodeBuild |
| 기본 텍스트 | Squid | #232F3E | 텍스트, 아이콘 기본색 |

### 그룹 테두리 색상
| 그룹 | 색상 | 스타일 |
|------|------|--------|
| VPC | #248814 (녹색) | 실선 |
| Region | #147EBA (파란색) | 실선 |
| Security Group | #DD3522 (빨간색) | 점선 |
| Public Subnet | #248814 (녹색) | 실선 |
| Private Subnet | #147EBA (파란색) | 실선 |
| AZ | #147EBA (파란색) | 점선 |

### 기본 색상
| 용도 | 색상 |
|------|------|
| 텍스트 (기본) | #232F3E |
| 아이콘 내부선 | #FFFFFF (흰색) |
| 배경 | #FFFFFF |

---

## 📋 weAlist 아키텍처용 체크리스트

### 필요한 서비스 아이콘
- [x] CloudFront (CDN)
- [x] ALB (Application Load Balancer)
- [x] EKS (Kubernetes)
- [x] RDS (PostgreSQL)
- [x] ElastiCache (Redis)
- [ ] ECR (Container Registry)
- [ ] S3 (Storage)
- [ ] Route53 (DNS)
- [ ] NAT Gateway
- [ ] Internet Gateway
- [ ] Secrets Manager
- [ ] CloudWatch
- [ ] CodePipeline
- [ ] CodeBuild

### 필요한 그룹
- [x] VPC
- [x] Security Group (ALB-SG, EKS-SG, RDS-SG, Redis-SG)
- [ ] Public Subnet
- [ ] Private Subnet (App)
- [ ] Private Subnet (DB)
- [ ] Region
- [ ] Availability Zone

### weAlist 서비스 (8개)
| 서비스 | 포트 | 언어 |
|--------|------|------|
| auth-service | 8080 | Java/Spring |
| user-service | 8081 | Go |
| board-service | 8000 | Go |
| chat-service | 8001 | Go |
| noti-service | 8002 | Go |
| storage-service | 8003 | Go |
| video-service | 8004 | Go |
| frontend | 3000 | React |

---

## 📝 Claude Code 프롬프트 템플릿

### 기본 요청 템플릿
```markdown
draw.io MCP를 사용해서 AWS 아키텍처 다이어그램 생성해줘.

### 스타일 요구사항
- AWS 공식 그룹 스타일 사용 (mxgraph.aws4.group)
- AWS 공식 서비스 아이콘 사용 (mxgraph.aws4.resourceIcon)
- 아이콘 내부선: 흰색 (#FFFFFF)
- 선 굵기: 1~1.5px
- 캔버스 크기: 1400x900px 이상
- 배경: 흰색

### 색상 적용 (2024-2025 최신)
- Compute (EKS, EC2): 주황색 #ED7100
- Storage (S3): 녹색 #7AA116
- Database (RDS, ElastiCache): 보라/핑크 #C925D1
- Networking (ALB, VPC, CloudFront): 보라색 #8C4FFF
- Security Group: 빨간 점선 #DD3522

### 파일
- 파일명: [파일명].drawio.svg
- 위치: docs/images/
```

### VPC Security Groups 다이어그램 요청
```markdown
weAlist VPC Security Groups 다이어그램 생성

### 구조
1. VPC (10.0.0.0/16) - 녹색 테두리
   - grIcon=mxgraph.aws4.group_vpc

2. ALB-SG (빨간 점선)
   - ← 80, 443 from 0.0.0.0/0
   - ALB 아이콘 (보라색)

3. EKS-SG (빨간 점선)
   - ← 8000-8081 from ALB-SG
   - EKS 아이콘 (주황색)
   - 내부 서비스 8개 박스

4. RDS-SG (빨간 점선)
   - ← 5432 from EKS-SG
   - RDS 아이콘 (파란색)

5. Redis-SG (빨간 점선)
   - ← 6379 from EKS-SG
   - ElastiCache 아이콘 (파란색)

### 파일
docs/images/wealist_vpc_security.drawio.svg
```

### Traffic Flow 다이어그램 요청
```markdown
weAlist Traffic Flow 다이어그램 생성

### 흐름 (번호 표시)
① Internet → CloudFront (HTTPS 443)
② CloudFront → ALB (HTTP/S 80, 443)
③ ALB → EKS Services (8000-8081)
④ EKS → RDS (5432)
⑤ EKS → Redis (6379)

### 영역
- Public Subnet: CloudFront, ALB, NAT Gateway
- Private Subnet (App): EKS Services
- Private Subnet (DB): RDS, ElastiCache

### 파일
docs/images/wealist_vpc_traffic.drawio.svg
```

---

## 🔗 참고 링크

- [AWS Architecture Icons 공식](https://aws.amazon.com/architecture/icons/)
- [draw.io AWS 다이어그램 가이드](https://www.drawio.com/blog/aws-diagrams)
- [draw.io AWS18 라이브러리](https://www.draw.io/?splash=0&libs=aws4)

---

## 📂 파일 구조 (예정)

```
docs/images/
├── wealist_vpc_traffic.drawio.svg      # 트래픽 흐름
├── wealist_vpc_security.drawio.svg     # Security Groups
├── wealist_aws_arch.drawio.svg         # 전체 AWS 아키텍처
├── wealist_cicd.drawio.svg             # CI/CD 파이프라인
├── wealist_microservices.drawio.svg    # 마이크로서비스 구조
└── wealist_monitoring.drawio.svg       # 모니터링 스택
```
