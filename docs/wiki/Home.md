# weAlist Wiki

**weAlist**는 클라우드 네이티브 마이크로서비스 기반 협업 프로젝트 관리 플랫폼입니다.

---

## Quick Links

### Architecture
- [Architecture Overview](Architecture.md) - 전체 시스템 아키텍처
- [AWS Architecture](Architecture-AWS.md) - AWS 인프라 구성
- [CI/CD Pipeline](Architecture-CICD.md) - CI/CD 파이프라인
- [Security (VPC)](Architecture-VPC.md) - 네트워크 및 보안
- [Monitoring Stack](Architecture-Monitoring.md) - 모니터링 구성
- [Business Flow](Business-Flow.md) - 비즈니스 플로우

### Documentation
- [Requirements](Requirements.md) - 요구사항 정의서
- [Cloud Proposal](Cloud-Proposal.md) - 클라우드 제안서
- [ADR](ADR.md) - 아키텍처 결정 기록

### Guides
- [Getting Started](Getting-Started.md) - 시작 가이드
- [Development Guide](Development-Guide.md) - 개발 환경 설정

### Team
- [Team & Contributions](Team.md) - 팀 역할 및 진행 상황

---

## Project Overview

| 항목 | 내용 |
|------|------|
| **서비스 수** | 8개 (6 Go + 1 Spring Boot + 1 React) |
| **인프라** | Kubernetes + Helm + ArgoCD |
| **모니터링** | Prometheus + Loki + Grafana |
| **데이터베이스** | PostgreSQL 17, Redis 7.2 |
| **영상통화** | LiveKit + Coturn |

---

## External Documents

| 문서 | 링크 |
|------|------|
| 클라우드 제안서 | [Google Docs](https://docs.google.com/document/d/1DiVO6p0NjmxzoEXwG3hZ7KoZLpqU-iiSdLWmOvTuH_s) |
| 요구사항 정의서 | [Google Docs](https://docs.google.com/document/d/1Cmc4fSrtqnJRTxgARCCyQGNgOiVJ-vvIkmSqktE_hx8) |
| 클라우드 설계/아키텍처 | [Google Docs](https://docs.google.com/document/d/1K2L1s3t15OCGDkmCfuXjLbpeDbREeuoT1OP1ldCSGY8) |

---

## Repository

- [GitHub Repository](https://github.com/OrangesCloud/wealist-project-advanced-k8s)
