# weAlist - 협업 프로젝트 관리 플랫폼

> Docker Compose → Kubernetes 마이그레이션을 통한 클라우드 네이티브 전환 프로젝트

[![Kubernetes](https://img.shields.io/badge/Kubernetes-326CE5?logo=kubernetes&logoColor=white)](https://kubernetes.io/)
[![Helm](https://img.shields.io/badge/Helm-0F1689?logo=helm&logoColor=white)](https://helm.sh/)
[![ArgoCD](https://img.shields.io/badge/ArgoCD-EF7B4D?logo=argo&logoColor=white)](https://argoproj.github.io/cd/)
[![Go](https://img.shields.io/badge/Go-00ADD8?logo=go&logoColor=white)](https://golang.org/)
[![Spring Boot](https://img.shields.io/badge/Spring_Boot-6DB33F?logo=springboot&logoColor=white)](https://spring.io/projects/spring-boot)
[![React](https://img.shields.io/badge/React-61DAFB?logo=react&logoColor=black)](https://reactjs.org/)

---

## 📋 프로젝트 배경

| 단계 | 문서                                         | 설명                                  |
| ---- | -------------------------------------------- | ------------------------------------- |
| 1️⃣   | [요구사항 정의서](../../wiki/Requirements)   | 서비스 성장에 따른 확장성/유연성 요구 |
| 2️⃣   | [클라우드 제안서](../../wiki/Cloud-Proposal) | EKS 전환 제안 및 비용/효율 분석       |
| 3️⃣   | [아키텍처 설계](../../wiki/Architecture)     | K8s 기반 마이크로서비스 설계          |
| 🔧   | [트러블슈팅](../../wiki/Troubleshooting)     | 마이그레이션 과정 이슈 해결 기록      |

> **시나리오**: 성공적인 서비스 오픈 → 트래픽 증가로 기능 추가/확장 어려움 → 클라우드 네이티브 전환 결정

---

## Highlights

- **6개 백엔드 서비스** - 5 Go + 1 Spring Boot (auth)
- **Frontend** - React + Vite (S3 + CloudFront 별도 배포)
- **Kubernetes + Helm + ArgoCD** - GitOps 기반 배포 자동화
- **Prometheus + Loki + Grafana** - 통합 모니터링/로깅
- **LiveKit + Coturn** - WebRTC 기반 영상통화

---

## Architecture

![AWS Architecture](docs/images/wealist_aws_arch_v2.png)

> 상세 아키텍처: [Wiki - Architecture](../../wiki/Architecture)

---

## Services

| Service             | Tech          | Port | Description              |
| ------------------- | ------------- | ---- | ------------------------ |
| **auth-service**    | Spring Boot 3 | 8080 | JWT/OAuth2 인증          |
| **user-service**    | Go + Gin      | 8081 | 사용자/워크스페이스      |
| **board-service**   | Go + Gin      | 8000 | 프로젝트/보드/댓글       |
| **chat-service**    | Go + Gin      | 8001 | 실시간 채팅 (WebSocket)  |
| **noti-service**    | Go + Gin      | 8002 | 알림 (SSE)               |
| **storage-service** | Go + Gin      | 8003 | 파일 스토리지 (S3)       |
| **frontend**        | React + Vite  | -    | Web UI (S3 + CloudFront) |

---

## Tech Stack

| Category           | Technologies                             |
| ------------------ | ---------------------------------------- |
| **Backend**        | Go 1.24, Spring Boot 3, Gin, GORM        |
| **Frontend**       | React 18, TypeScript, Vite, TailwindCSS  |
| **Database**       | PostgreSQL 17, Redis 7.2                 |
| **Infrastructure** | Kubernetes, Helm, ArgoCD, NGINX Ingress  |
| **Monitoring**     | Prometheus, Loki, Grafana                |
| **Media**          | LiveKit (WebRTC SFU), Coturn (TURN/STUN) |
| **Storage**        | MinIO (S3 Compatible)                    |

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

## Documentation

| 문서                                             | 설명                 |
| ------------------------------------------------ | -------------------- |
| [Architecture](../../wiki/Architecture)          | 시스템 아키텍처 상세 |
| [AWS Architecture](../../wiki/Architecture-AWS)  | AWS 인프라 구성      |
| [CI/CD Pipeline](../../wiki/Architecture-CICD)   | CI/CD 파이프라인     |
| [Security (VPC)](../../wiki/Architecture-VPC)    | 네트워크 및 보안     |
| [Monitoring](../../wiki/Architecture-Monitoring) | 모니터링 스택        |
| [Requirements](../../wiki/Requirements)          | 요구사항 정의서      |
| [Cloud Proposal](../../wiki/Cloud-Proposal)      | 클라우드 제안서      |
| [ADR](../../wiki/ADR)                            | 아키텍처 결정 기록   |
| [Getting Started](../../wiki/Getting-Started)    | 시작 가이드          |

---

## Project Status

### Phase 1: 로컬 기반 구축

- [x] K8s manifest 정리
- [x] Kind 로컬 배포 테스트
- [x] Helm 차트 전환
- [ ] ArgoCD 로컬 설치 + GitOps 테스트

### Phase 2: 모니터링/로깅

- [ ] Prometheus + Grafana 설치
- [ ] Loki 로그 수집
- [ ] Pod 리소스 튜닝

### Phase 3: 서비스 메시 + 고급 배포

- [ ] Istio 설치
- [ ] mTLS 설정
- [ ] Argo Rollouts 카나리 배포

### Phase 4: AWS 인프라

- [ ] Terraform EKS 클러스터
- [ ] Cluster Autoscaler
- [ ] ALB Ingress Controller
- [ ] 부하 테스트 (k6)

---

## Team

| 역할               | 담당 | 주요 업무                              |
| ------------------ | ---- | -------------------------------------- |
| **Service Mesh**   | 혁준 | Istio + mTLS + Argo Rollouts           |
| **Observability**  | 원이 | Prometheus + Grafana + Loki + OTel     |
| **GitOps**         | 명재 | ArgoCD + Sealed Secrets + Discord 알림 |
| **Security & IaC** | 재형 | Trivy + Kyverno + Terraform EKS        |

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

## License

This project is licensed under the MIT License.
