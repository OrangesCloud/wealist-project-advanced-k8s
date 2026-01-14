# weAlist - Cloud Native Project Management Platform

> Production-ready 마이크로서비스 아키텍처로 구현된 협업 프로젝트 관리 플랫폼

[![Kubernetes](https://img.shields.io/badge/Kubernetes-326CE5?logo=kubernetes&logoColor=white)](https://kubernetes.io/)
[![Helm](https://img.shields.io/badge/Helm-0F1689?logo=helm&logoColor=white)](https://helm.sh/)
[![ArgoCD](https://img.shields.io/badge/ArgoCD-EF7B4D?logo=argo&logoColor=white)](https://argoproj.github.io/cd/)
[![Istio](https://img.shields.io/badge/Istio-466BB0?logo=istio&logoColor=white)](https://istio.io/)
[![Terraform](https://img.shields.io/badge/Terraform-7B42BC?logo=terraform&logoColor=white)](https://www.terraform.io/)
[![Go](https://img.shields.io/badge/Go-00ADD8?logo=go&logoColor=white)](https://golang.org/)
[![Spring Boot](https://img.shields.io/badge/Spring_Boot-6DB33F?logo=springboot&logoColor=white)](https://spring.io/projects/spring-boot)

---

## Demo

[![Demo Video](https://img.youtube.com/vi/UTe8f_IYyWs/maxresdefault.jpg)](https://youtu.be/UTe8f_IYyWs)

> 클릭하면 YouTube에서 시연 영상을 볼 수 있습니다.


---

## Documentation

| 문서                                               | 설명                                    |
|--------------------------------------------------|---------------------------------------|
| [wiki 전체보기](../../wiki/home)                     | wiki 전체보기                             |
| [Architecture](../../wiki/Architecture)          | 전체 시스템 아키텍처, AWS 인프라, Terraform IaC   |
| [Kubernetes](../../wiki/Architecture-K8s)        | EKS 클러스터, Istio, ArgoCD, Helm 구성      |
| [CI/CD Pipeline](../../wiki/Architecture-CICD)   | GitHub Actions, ArgoCD GitOps 플로우     |
| [Security (VPC)](../../wiki/Architecture-VPC)    | 네트워크 보안, Private Subnet 구성            |
| [Monitoring](../../wiki/Architecture-Monitoring) | LGTM Stack, OTEL, Distributed Tracing |
| [Requirements](../../wiki/Requirements)          | 요구사항 정의서                              |
| [Cloud Proposal](../../wiki/Cloud-Proposal)      | 클라우드 제안서                              |
| [ADR](../../wiki/ADR)                            | 아키텍처 결정 기록                            |


---

## Key Highlights

### Istio Service Mesh
- **mTLS** 전 서비스 암호화 통신
- **AuthorizationPolicy** 서비스 간 접근 제어
- **Argo Rollouts** 카나리 배포 (10% → 30% → 50% → 100%)

### GitOps & CI/CD
- **ArgoCD** App-of-Apps 패턴 선언적 배포
- **GitHub Actions** 빌드 → ECR 푸시 → ArgoCD 동기화
- **ExternalSecrets** AWS Secrets Manager 연동

### Full Observability (LGTM Stack)
- **Metrics**: Prometheus + Istio sidecar 메트릭
- **Traces**: OpenTelemetry SDK → Tempo, Span Metrics
- **Logs**: Alloy → Loki, trace_id 상관분석

### AWS Infrastructure (Terraform)
- **2-Layer IaC**: Foundation (VPC, RDS, Redis) → Compute (EKS, Istio)
- **Cost Optimization**: 100% Spot Instances, Scheduled Scaling
- **Security**: Private Subnet, Pod Identity, Secrets Manager


---

---

## Overview

| 단계 | 문서 | 설명 |
|------|------|------|
| 1️⃣ | [요구사항 정의서](../../wiki/Requirements) | 서비스 성장에 따른 확장성/유연성 요구 |
| 2️⃣ | [클라우드 제안서](../../wiki/Cloud-Proposal) | EKS 전환 제안 및 비용/효율 분석 |
| 3️⃣ | [아키텍처 설계](../../wiki/Architecture) | K8s 기반 마이크로서비스 설계 |
| 🔧 | [트러블슈팅](../../wiki/Troubleshooting) | 마이그레이션 과정 이슈 해결 기록 |

> **시나리오**: 성공적인 서비스 오픈 → 트래픽 증가로 기능 추가/확장 어려움 → 클라우드 네이티브 전환 결정

---

## Architecture

![AWS Architecture](docs/images/wealist_aws_arch_v2.png)

> 상세 아키텍처: [Wiki - Architecture](../../wiki/Architecture)

---

## Services

| Service | Tech | Port | Description |
|---------|------|------|-------------|
| **auth-service** | Spring Boot 3.4 | 8080 | JWT/OAuth2 인증 |
| **user-service** | Go + Gin | 8081 | 사용자/워크스페이스 |
| **board-service** | Go + Gin | 8000 | 프로젝트/보드/댓글 |
| **chat-service** | Go + Gin | 8001 | 실시간 채팅 (WebSocket) |
| **noti-service** | Go + Gin | 8002 | 알림 (SSE) |
| **storage-service** | Go + Gin | 8003 | 파일 스토리지 (S3) |
| **ops-service** | Go + Gin | 8004 | 운영 대시보드 |

---

## Tech Stack

| Category | Technologies |
|----------|-------------|
| **Backend** | Go 1.24, Spring Boot 3.4 (Java 21), Gin, GORM |
| **Frontend** | React 19, TypeScript 5, Vite 5, TailwindCSS |
| **Service Mesh** | Istio 1.28 (Sidecar mTLS) |
| **Database** | PostgreSQL 17, Redis 7.2 |
| **Infrastructure** | AWS EKS, Terraform, Helm, ArgoCD |
| **Observability** | Prometheus, Grafana, Loki, Tempo, OpenTelemetry |
| **Storage** | AWS S3 (prod), MinIO (local) |

---

---

## Project Status

### Phase 1: 로컬 기반 구축

- [x] K8s manifest 정리
- [x] Kind 로컬 배포 테스트
- [x] Helm 차트 전환
- [x] ArgoCD 로컬 설치 + GitOps 테스트

### Phase 2: 모니터링/로깅

- [x] Prometheus + Grafana 설치
- [x] Loki 로그 수집
- [x] Pod 리소스 튜닝

### Phase 3: 서비스 메시 + 고급 배포

- [x] Istio 설치
- [x] mTLS 설정
- [x] Argo Rollouts 카나리 배포

### Phase 4: AWS 인프라

- [x] Terraform EKS 클러스터
- [x] Cluster Autoscaler
- [x] ALB Ingress Controller
- [x] 부하 테스트 (k6)

---

## Team

| 역할 | 담당 | 주요 업무 |
|------|------|----------|
| **Service Mesh** | 혁준 | Istio + mTLS + Argo Rollouts |
| **Observability** | 원이 | Prometheus + Grafana + Loki + OTel |
| **GitOps** | 명재 | ArgoCD + Sealed Secrets + Discord 알림 |
| **Security & IaC** | 재형 | Trivy + Kyverno + Terraform EKS |

---

## Commands Reference

```bash
# Development
make dev-up              # Docker Compose 시작
make dev-down            # 종료
make dev-logs            # 로그

# Kubernetes (Helm)
make helm-install-all    # 전체 설치
make helm-upgrade-all    # 업그레이드
make helm-uninstall-all  # 삭제
make helm-validate       # 검증 (156 테스트)

# Per-Service
make {service}-build     # 이미지 빌드
make {service}-load      # 레지스트리 푸시
make {service}-redeploy  # 재배포
make {service}-all       # 빌드 + 로드 + 재배포

# Utilities
make status              # Pod 상태
make redeploy-all        # 전체 재시작
```
---

## Quick Start

### Prerequisites

- Docker & Docker Compose
- Kind (Kubernetes in Docker)
- Helm 3.x
- kubectl

### Local Development (Kind + Helm)

```bash
# 1. 클러스터 생성
make kind-setup

# 2. 이미지 빌드 및 로드
make kind-load-images

# 3. Helm으로 전체 배포
make helm-install-all ENV=localhost

# 4. 상태 확인
make status

# 접속: http://localhost
```

### Docker Compose (간단 테스트)

```bash
# 환경 변수 설정
cp docker/env/.env.dev.example docker/env/.env.dev

# 전체 서비스 시작
make dev-up

# 접속: http://localhost:3000
```


---

## License

This project is licensed under the MIT License.
