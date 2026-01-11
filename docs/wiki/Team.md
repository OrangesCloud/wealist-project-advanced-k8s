# Team & Contributions

weAlist 프로젝트 팀 역할 및 진행 상황입니다.

---

## Team Roles

| 역할 | 담당 | 주요 업무 |
|------|------|----------|
| **Service Mesh** | 혁준 | Istio + mTLS + Argo Rollouts |
| **Observability** | 원이 | Prometheus + Grafana + Loki + OTel + 부하 테스트 |
| **GitOps** | 명재 | ArgoCD + Sealed Secrets + Discord 알림 |
| **Security & IaC** | 재형 | Trivy + Kyverno + Terraform EKS |

---

## Project Roadmap

### Phase 1: 로컬 기반 구축 ✅
- [x] K8s manifest 정리
- [x] Kind 로컬 배포 테스트
- [x] 설계도 작성
- [x] Helm 차트 전환
- [x] ArgoCD 로컬 설치 + GitOps 테스트

### Phase 2: 모니터링/로깅 ✅
- [x] Prometheus + Grafana 설치
- [x] Loki 로그 수집
- [x] OpenTelemetry + Tempo 분산 추적
- [x] Pod 리소스 limits 튜닝

### Phase 3: 서비스 메시 + 고급 배포 ✅
- [x] Istio 1.24.0 Sidecar 모드 설치
- [x] mTLS STRICT 설정
- [x] AuthorizationPolicy 적용
- [x] Gateway API (HTTPRoute)
- [x] Argo Rollouts 카나리 배포

### Phase 4: AWS 인프라 ✅
- [x] Terraform으로 EKS 클러스터 생성 (3-layer: foundation, compute, argocd-apps)
- [x] Cluster Autoscaler 설정
- [x] AWS ALB Controller
- [x] External Secrets Operator (AWS Secrets Manager)
- [x] CloudFront + S3 Frontend 배포
- [x] k6 부하 테스트

---

## Key Achievements

### Kustomize → Helm 마이그레이션
- 110개 Kustomize 파일 → 9개 Helm 차트
- 18개 ConfigMap 패치 제거
- 156개 자동화 테스트 검증

### 서비스 표준화
- 5개 Go 서비스 공통 패키지 적용 (health, errors, middleware, ratelimit)
- Docker build 패턴 통일
- 환경별 설정 체계화 (base.yaml → env.yaml 오버라이드)

### Istio Service Mesh
- Ambient → Sidecar 모드 마이그레이션
- mTLS STRICT + AuthorizationPolicy
- Gateway API (HTTPRoute) 라우팅
- Argo Rollouts 카나리 배포

### Observability Stack
- Prometheus + Grafana 메트릭 시각화
- Loki 로그 집계
- OpenTelemetry + Tempo 분산 추적
- Span Metrics + Service Graph

### AWS Production 환경
- Terraform 3-layer 아키텍처 (foundation, compute, argocd-apps)
- EKS 1.34 + Managed Node Groups
- External Secrets Operator (AWS Secrets Manager)
- CloudFront + S3 정적 프론트엔드
- RDS PostgreSQL + ElastiCache Redis

### 문서화
- CLAUDE.md 개발자 가이드 (40KB+)
- ADR 9개 작성
- 트러블슈팅 가이드 12개
- draw.io 아키텍처 다이어그램 11개

---

## Collaboration

### 작업 시 주의사항
- Helm 변수 수정 시 팀 공유
- 환경 파일 변경 시 공지
- PR 리뷰 필수

### Communication
- Discord: 실시간 소통
- GitHub Issues: 작업 트래킹
- Weekly Meeting: 진행 상황 공유

---

## Related Pages

- [Getting Started](Getting-Started.md)
- [ADR](ADR.md)
