# Architecture Decision Records (ADR)

weAlist 프로젝트의 주요 아키텍처 결정 기록입니다.

---

## ADR-001: 마이크로서비스 아키텍처 선택

**상황**: 협업 플랫폼의 여러 도메인을 어떻게 구성할 것인가?

**결정**: 도메인별 마이크로서비스 분리
- auth-service, user-service, board-service, chat-service, noti-service, storage-service, video-service

**이유**:
- 각 도메인의 독립적인 배포 및 확장 가능
- 팀별 독립적인 개발 가능
- 장애 격리 (한 서비스 장애가 전체에 영향 X)

**결과**: 8개 독립 서비스로 구성

---

## ADR-002: 서비스별 데이터베이스 분리

**상황**: 마이크로서비스들이 데이터베이스를 공유할 것인가?

**결정**: 서비스별 독립 데이터베이스 사용 (6개 DB)

**이유**:
- 서비스 간 데이터 결합도 최소화
- 독립적인 스키마 변경 가능
- 서비스별 최적화된 DB 선택 가능

**결과**: PostgreSQL 7개 DB (서비스 6개 + SonarQube 1개)

---

## ADR-003: LiveKit 기반 영상통화

**상황**: 영상/음성 통화 기능을 어떻게 구현할 것인가?

**결정**: LiveKit SFU + Coturn TURN 서버

**이유**:
- 오픈소스 SFU 솔루션 (Selective Forwarding Unit)
- 클라이언트 SDK 제공 (React)
- Coturn으로 NAT/방화벽 환경 지원
- 녹화, 스트리밍 기능 확장 가능

**대안 검토**:
- Jitsi: 무거움, 커스터마이징 어려움
- Twilio: 비용 부담

---

## ADR-004: Health Check 분리 (Liveness vs Readiness)

**상황**: Kubernetes에서 Pod 상태를 어떻게 체크할 것인가?

**결정**: Liveness와 Readiness 프로브 분리

```yaml
livenessProbe:   # 서비스 자체가 살아있는지 (DB 무관)
  path: /health/live

readinessProbe:  # 트래픽 수신 가능한지 (DB 연결 포함)
  path: /health/ready
```

**이유**:
- DB 일시적 장애 시 Pod 재시작 방지 (Liveness)
- DB 연결 안 되면 트래픽 차단 (Readiness)
- 안정적인 롤링 업데이트 지원

**결과**: 공통 health package로 표준화 (`packages/wealist-advanced-go-pkg/health/`)

---

## ADR-005: Prometheus + Loki 모니터링

**상황**: 분산 시스템의 모니터링을 어떻게 구성할 것인가?

**결정**: Prometheus (메트릭) + Loki (로그) + Grafana (시각화)

**이유**:
- Prometheus: 시계열 메트릭 수집의 표준
- Loki: Prometheus와 유사한 라벨 기반 로그 시스템
- Grafana: 통합 대시보드 제공
- 오픈소스 + 커뮤니티 지원 활발

**대안 검토**:
- ELK Stack: 리소스 부담, 복잡한 설정
- Datadog: 비용 부담

---

## ADR-006: Kustomize → Helm 마이그레이션

**상황**: Kubernetes 매니페스트 관리 방식 개선 필요

**결정**: Kustomize에서 Helm으로 전환

**이유**:
- 환경별 설정 관리 용이 (values.yaml)
- 재사용 가능한 차트
- ArgoCD와 통합 편리
- 커뮤니티 차트 활용 가능

**결과**:
- 110개 Kustomize 파일 → 9개 Helm 차트
- 18개 ConfigMap 패치 제거
- 156개 자동화 테스트 통과

> 상세: [MIGRATION_COMPLETE.md](../MIGRATION_COMPLETE.md)

---

## ADR-007: Secrets 관리 분리

**상황**: 민감 정보를 어떻게 관리할 것인가?

**결정**: ConfigMap과 Secret 분리

```yaml
# 비민감 정보 → shared.config (ConfigMap)
DB_HOST, DB_PORT, REDIS_HOST

# 민감 정보 → shared.secrets (Secret)
DB_PASSWORD, JWT_SECRET, GOOGLE_CLIENT_SECRET
```

**이유**:
- GitOps에서 민감 정보 노출 방지
- 환경별 secrets 파일 분리 (gitignored)
- Sealed Secrets 도입 준비 (Phase 3)

---

## Related Pages

- [Architecture Overview](Architecture.md)
- [Requirements](Requirements.md)
