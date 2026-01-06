# 알림 설정 가이드

> **목표**: 장애 조기 감지 및 신속한 대응

---

## 알림 아키텍처

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        ALERTING ARCHITECTURE                             │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌──────────┐   evaluate    ┌──────────────┐   notify    ┌──────────┐  │
│  │Prometheus│ ────────────▶ │ Alert Rules  │ ──────────▶│ Discord  │  │
│  │          │               │              │            │ Webhook  │  │
│  └──────────┘               └──────────────┘            └──────────┘  │
│                                    │                                    │
│                                    │                                    │
│                                    ▼                                    │
│                            ┌──────────────┐                            │
│                            │ ArgoCD       │                            │
│                            │ Notifications│                            │
│                            └──────────────┘                            │
│                                    │                                    │
│                                    ▼                                    │
│                            ┌──────────────┐                            │
│                            │  Deployment  │                            │
│                            │   Alerts     │                            │
│                            └──────────────┘                            │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 알림 규칙 카테고리

### 1. 서비스 헬스 알림

| 알림명 | 조건 | 심각도 |
|--------|------|--------|
| ServiceDown | 서비스 응답 없음 5분 | Critical |
| HighErrorRate | 5xx 에러율 > 5% | Warning |
| HighLatency | P99 지연 > 1초 | Warning |

**PromQL 예시**:
```yaml
# ServiceDown
- alert: ServiceDown
  expr: up{job=~".*-service"} == 0
  for: 5m
  labels:
    severity: critical
  annotations:
    summary: "Service {{ $labels.job }} is down"

# HighErrorRate
- alert: HighErrorRate
  expr: |
    sum(rate(istio_requests_total{response_code=~"5.."}[5m])) by (destination_service_name)
    / sum(rate(istio_requests_total[5m])) by (destination_service_name) > 0.05
  for: 5m
  labels:
    severity: warning
```

### 2. 리소스 알림

| 알림명 | 조건 | 심각도 |
|--------|------|--------|
| HighCPUUsage | CPU > 80% (15분) | Warning |
| HighMemoryUsage | Memory > 85% | Warning |
| PodOOMKilled | OOMKilled 발생 | Critical |
| PodCrashLooping | 재시작 > 5회/시간 | Critical |

**PromQL 예시**:
```yaml
# HighCPUUsage
- alert: HighCPUUsage
  expr: |
    sum(rate(container_cpu_usage_seconds_total[5m])) by (pod, namespace)
    / sum(kube_pod_container_resource_limits{resource="cpu"}) by (pod, namespace) > 0.8
  for: 15m
  labels:
    severity: warning

# PodCrashLooping
- alert: PodCrashLooping
  expr: rate(kube_pod_container_status_restarts_total[1h]) * 3600 > 5
  for: 5m
  labels:
    severity: critical
```

### 3. 데이터베이스 알림

| 알림명 | 조건 | 심각도 |
|--------|------|--------|
| PostgresHighConnections | 연결 > 80% | Warning |
| PostgresReplicationLag | 지연 > 30초 | Warning |
| RedisHighMemory | 메모리 > 80% | Warning |

**PromQL 예시**:
```yaml
# PostgresHighConnections
- alert: PostgresHighConnections
  expr: |
    pg_stat_activity_count / pg_settings_max_connections > 0.8
  for: 10m
  labels:
    severity: warning
```

### 4. SLO 알림

| 알림명 | 조건 | 심각도 |
|--------|------|--------|
| SLOViolation | 가용성 < 99.9% | Critical |
| ErrorBudgetBurn | 에러 예산 소진 > 50% | Warning |

**PromQL 예시**:
```yaml
# SLOViolation
- alert: SLOViolation
  expr: |
    (1 - sum(rate(istio_requests_total{response_code=~"5.."}[1h]))
    / sum(rate(istio_requests_total[1h]))) * 100 < 99.9
  for: 5m
  labels:
    severity: critical
  annotations:
    summary: "SLO violation: Availability below 99.9%"
```

---

## ArgoCD 배포 알림

### Discord 알림 설정

```yaml
# ArgoCD Application 어노테이션
metadata:
  annotations:
    # 배포 완료
    notifications.argoproj.io/subscribe.on-deployed.discord: prod-deployment-alerts
    # 동기화 실패
    notifications.argoproj.io/subscribe.on-sync-failed.discord: prod-deployment-alerts
    # 헬스 저하
    notifications.argoproj.io/subscribe.on-health-degraded.discord: prod-deployment-alerts
```

### 알림 트리거

| 이벤트 | 트리거 | 채널 |
|--------|--------|------|
| 배포 성공 | `on-deployed` | #prod-deployment-alerts |
| 동기화 실패 | `on-sync-failed` | #prod-deployment-alerts |
| 헬스체크 실패 | `on-health-degraded` | #prod-deployment-alerts |
| Rollout 진행 | `on-sync-running` | #prod-deployment-alerts |

---

## 알림 심각도

| 심각도 | 설명 | 대응 시간 |
|--------|------|-----------|
| **Critical** | 서비스 중단, 즉시 대응 필요 | < 15분 |
| **Warning** | 잠재적 문제, 모니터링 필요 | < 1시간 |
| **Info** | 정보성, 참고용 | - |

---

## 알림 규칙 파일

### Prometheus 알림 규칙

```yaml
# prometheus/rules/alerts.yaml
groups:
  - name: service-alerts
    rules:
      - alert: ServiceDown
        expr: up{job=~".*-service"} == 0
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Service {{ $labels.job }} is down"
          description: "{{ $labels.job }} has been down for 5 minutes"

  - name: resource-alerts
    rules:
      - alert: HighMemoryUsage
        expr: |
          container_memory_working_set_bytes
          / container_spec_memory_limit_bytes > 0.85
        for: 10m
        labels:
          severity: warning

  - name: slo-alerts
    rules:
      - alert: AvailabilitySLO
        expr: |
          (1 - sum(rate(istio_requests_total{response_code=~"5.."}[1h]))
          / sum(rate(istio_requests_total[1h]))) < 0.999
        for: 5m
        labels:
          severity: critical
```

---

## On-Call 대응 가이드

### Critical 알림 대응

```
1. ServiceDown 알림 수신
   ↓
2. Grafana 대시보드 확인
   - Services Overview → 헬스맵
   - Service Detail → 상세 메트릭
   ↓
3. 로그 확인
   - Loki: {namespace="wealist-prod", container="xxx"}
   ↓
4. 트레이스 확인 (에러 있는 경우)
   - Tempo: 에러 트레이스 분석
   ↓
5. 조치
   - kubectl rollout restart (재시작)
   - kubectl scale (스케일 조정)
   - ArgoCD rollback (롤백)
```

### 에스컬레이션 경로

```
L1: 자동 알림 → Discord
 ↓ (15분 무응답)
L2: 담당 개발자 직접 연락
 ↓ (30분 미해결)
L3: 팀 리드 에스컬레이션
```

---

## 알림 음소거

### 유지보수 시간 설정

```yaml
# Prometheus Alertmanager silence
inhibit_rules:
  - source_match:
      alertname: MaintenanceMode
    target_match_re:
      severity: warning|info
    equal: ['namespace']
```

### 수동 음소거

```bash
# Alertmanager API로 silence 생성
curl -X POST http://alertmanager:9093/api/v1/silences \
  -d '{
    "matchers": [{"name": "alertname", "value": "HighCPUUsage"}],
    "startsAt": "2026-01-05T10:00:00Z",
    "endsAt": "2026-01-05T12:00:00Z",
    "comment": "Maintenance window"
  }'
```

---

## 관련 문서

- [Observability Stack](./observability-stack.md)
- [대시보드 가이드](./dashboards.md)

