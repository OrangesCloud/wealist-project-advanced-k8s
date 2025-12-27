# Dev Foundation

개발 환경을 위한 기본 인프라 리소스입니다.

## 개요

개발 환경에서는 로컬 Kind 클러스터 또는 개발 서버에서 사용할 ECR 저장소만 관리합니다.

> **Note**: 개발 환경의 데이터베이스(PostgreSQL, Redis)는 Docker Compose 또는 Kind 클러스터 내부에서 실행됩니다.

## 구성 요소

| 리소스 | 설명 |
|--------|------|
| ECR Repositories | 개발용 컨테이너 이미지 저장소 |

## ECR 저장소 목록

| 저장소 | 서비스 |
|--------|--------|
| `auth-service` | 인증 서비스 (Spring Boot) |
| `user-service` | 사용자 서비스 (Go) |
| `board-service` | 보드 서비스 (Go) |
| `chat-service` | 채팅 서비스 (Go) |
| `noti-service` | 알림 서비스 (Go) |
| `storage-service` | 스토리지 서비스 (Go) |
| `video-service` | 비디오 서비스 (Go) |
| `frontend` | 프론트엔드 (React) |

## 사용법

### 배포

```bash
cd terraform/dev/foundation

# 초기화
terraform init

# 계획 확인
terraform plan

# 적용 (사용자 직접 실행)
terraform apply
```

### ECR 로그인

```bash
# AWS CLI로 ECR 로그인
aws ecr get-login-password --region ap-northeast-2 | \
  docker login --username AWS --password-stdin <account-id>.dkr.ecr.ap-northeast-2.amazonaws.com
```

### 이미지 Push

```bash
# 예: user-service
docker build -t user-service -f services/user-service/docker/Dockerfile .
docker tag user-service:latest <account-id>.dkr.ecr.ap-northeast-2.amazonaws.com/user-service:latest
docker push <account-id>.dkr.ecr.ap-northeast-2.amazonaws.com/user-service:latest
```

## State 위치

```
s3://wealist-tf-state-advanced-k8s/dev/foundation/terraform.tfstate
```

## 개발 환경 vs 프로덕션 환경

| 항목 | Dev | Prod |
|------|-----|------|
| ECR | Mutable tags | Immutable tags |
| VPC | 없음 (로컬) | 전용 VPC |
| RDS | Docker/Kind 내부 | 관리형 RDS |
| Redis | Docker/Kind 내부 | ElastiCache |
| EKS | Kind 클러스터 | 관리형 EKS |

## ECR 정리 정책

개발 ECR은 비용 절감을 위해 오래된 이미지를 자동 삭제합니다:

- Untagged 이미지: 1일 후 삭제
- 태그된 이미지: 최근 30개만 유지

## 관련 문서

- 프로덕션 인프라: [../prod/foundation/README.md](../../prod/foundation/README.md)
- 프로덕션 EKS: [../prod/compute/README.md](../../prod/compute/README.md)
