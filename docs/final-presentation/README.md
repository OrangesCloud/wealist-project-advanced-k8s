# weAlist 최종 발표 문서

> **프로젝트**: weAlist - 협업 프로젝트 관리 플랫폼
>
> **버전**: 1.0.0
>
> **작성일**: 2026-01-05

---

## 문서 구조

```
docs/final-presentation/
├── README.md                    # 이 문서
├── 01-architecture/             # 아키텍처 문서
│   ├── overview.md              # 아키텍처 개요
│   ├── service-diagram.drawio   # 서비스 관계도 (Draw.io)
│   └── tech-stack.md            # 기술 스택 설명
├── 02-api-spec/                 # API 명세서
│   ├── README.md                # API 개요
│   ├── auth-service.md          # 인증 서비스 API
│   ├── user-service.md          # 사용자 서비스 API
│   ├── board-service.md         # 보드 서비스 API
│   ├── chat-service.md          # 채팅 서비스 API
│   ├── noti-service.md          # 알림 서비스 API
│   ├── storage-service.md       # 스토리지 서비스 API
│   └── ops-service.md           # 운영 서비스 API
├── 03-deployment/               # 배포 가이드
│   ├── ci-cd-pipeline.md        # CI/CD 파이프라인
│   ├── argocd-setup.md          # ArgoCD GitOps 설정
│   └── environment-config.md    # 환경별 설정
└── 04-monitoring/               # 모니터링 가이드
    ├── observability-stack.md   # Prometheus/Grafana/Loki/Tempo
    ├── dashboards.md            # Grafana 대시보드
    └── alerting.md              # 알림 설정
```

---

## 프로젝트 개요

### weAlist란?

weAlist는 팀 협업을 위한 프로젝트 관리 플랫폼입니다. Trello/Notion 스타일의 칸반 보드, 실시간 채팅, 파일 공유 기능을 제공합니다.

### 주요 기능

| 기능 | 설명 | 서비스 |
|------|------|--------|
| **인증** | Google OAuth2, JWT 토큰 관리 | auth-service |
| **사용자 관리** | 프로필, 워크스페이스 | user-service |
| **프로젝트 관리** | 칸반 보드, 태스크, 댓글 | board-service |
| **실시간 채팅** | WebSocket 기반 메시징 | chat-service |
| **알림** | SSE 기반 실시간 알림 | noti-service |
| **파일 저장소** | S3 기반 파일 업로드/다운로드 | storage-service |
| **운영 대시보드** | 서비스 상태 모니터링 | ops-service |

---

## 기술 스택 요약

### Backend

| 기술 | 버전 | 용도 |
|------|------|------|
| Go + Gin | 1.24 | 5개 마이크로서비스 (user, board, chat, noti, storage) |
| Spring Boot | 3.4 | 인증 서비스 (auth) |
| PostgreSQL | 17 | 관계형 데이터베이스 |
| Redis | 7.2 | 캐시, 세션, Rate Limiting |

### Frontend

| 기술 | 버전 | 용도 |
|------|------|------|
| React | 18 | UI 프레임워크 |
| Vite | 6 | 빌드 도구 |
| TypeScript | 5 | 타입 안전성 |
| TailwindCSS | 3 | 스타일링 |

### Infrastructure

| 기술 | 버전 | 용도 |
|------|------|------|
| Kubernetes (EKS) | 1.34 | 컨테이너 오케스트레이션 |
| Istio | 1.24 | 서비스 메시 (mTLS, 트래픽 관리) |
| ArgoCD | 2.x | GitOps 배포 |
| Terraform | 1.10 | 인프라 코드 관리 |

### Observability

| 기술 | 버전 | 용도 |
|------|------|------|
| Prometheus | 2.55 | 메트릭 수집 |
| Grafana | 10.4 | 시각화 대시보드 |
| Loki | 3.6 | 로그 집계 |
| Tempo | 2.9 | 분산 트레이싱 |
| OTEL Collector | 0.114 | 텔레메트리 수집 |

---

## 아키텍처 하이라이트

### 마이크로서비스 아키텍처

- **6개 백엔드 서비스** (5 Go + 1 Spring Boot)
- **Clean Architecture** 패턴 적용
- **도메인 기반 분리** (인증, 사용자, 보드, 채팅, 알림, 저장소)

### 서비스 메시 (Istio)

- **mTLS**: 서비스 간 암호화 통신
- **Traffic Management**: 카나리 배포, 트래픽 분할
- **Observability**: 분산 트레이싱, 메트릭 자동 수집

### GitOps (ArgoCD)

- **선언적 배포**: Helm 차트 기반
- **자동 동기화**: Git 변경 감지 → 자동 배포
- **롤백**: 버전 관리된 배포 히스토리

---

## 빠른 링크

- [아키텍처 개요](./01-architecture/overview.md)
- [기술 스택 상세](./01-architecture/tech-stack.md)
- [API 명세서](./02-api-spec/README.md)
- [배포 가이드](./03-deployment/ci-cd-pipeline.md)
- [모니터링 가이드](./04-monitoring/observability-stack.md)

---

## 환경별 접속 정보

| 환경 | 도메인 | 용도 |
|------|--------|------|
| localhost | http://localhost:8080 | Kind 클러스터 개발 |
| dev | https://dev.wealist.co.kr | 개발 환경 |
| prod | https://wealist.co.kr | 운영 환경 |

---

## 관련 자료

- [GitHub Repository](https://github.com/OrangesCloud/wealist-project-advanced-k8s)
- [프로젝트 Wiki](../wiki/)
- [Helm Charts](../../k8s/helm/charts/)
- [Terraform Modules](../../terraform/)
