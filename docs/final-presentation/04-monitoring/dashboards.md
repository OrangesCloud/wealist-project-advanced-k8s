# Grafana 대시보드 가이드

> **역할 기반 대시보드**: 각 이해관계자에 최적화된 시각화

---

## 대시보드 구성

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      ROLE-BASED DASHBOARDS (V4)                          │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐         │
│  │    BUSINESS     │  │    DEVELOPER    │  │      INFRA      │         │
│  │   Dashboard     │  │   Dashboard     │  │   Dashboard     │         │
│  │                 │  │                 │  │                 │         │
│  │ - SLO/SLA      │  │ - Service Detail│  │ - Node Status   │         │
│  │ - User Metrics │  │ - API Latency   │  │ - Resource Usage│         │
│  │ - Traffic      │  │ - Error Traces  │  │ - Pod Health    │         │
│  │ - Conversion   │  │ - Log Search    │  │ - Network I/O   │         │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘         │
│                                                                          │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐         │
│  │   SERVICE       │  │    DATABASE     │  │   LOG ANALYSIS  │         │
│  │   Overview      │  │   Dashboard     │  │   Dashboard     │         │
│  │                 │  │                 │  │                 │         │
│  │ - All Services │  │ - PostgreSQL    │  │ - Error Logs    │         │
│  │ - Health Map   │  │ - Redis         │  │ - Request Logs  │         │
│  │ - Dependencies │  │ - Connections   │  │ - Trace Links   │         │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘         │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 대시보드 목록

| 대시보드 | 대상 | 주요 메트릭 |
|----------|------|------------|
| Business Dashboard | 비즈니스 | SLO, 사용자 활동, 트래픽 |
| Developer Dashboard | 개발자 | API 성능, 에러 분석, 트레이스 |
| Infra Dashboard | 인프라 | 노드, Pod, 리소스 사용량 |
| Services Overview | 전체 | 서비스 헬스맵, 의존성 |
| Service Detail | 개발자 | 개별 서비스 상세 메트릭 |
| Database Dashboard | DBA | PostgreSQL, Redis 메트릭 |
| Log Analysis | 운영 | 로그 검색, 에러 추적 |

---

## Business Dashboard

### SLO/SLA 패널

```
┌────────────────────────────────────────────────────────────┐
│                   SLO COMPLIANCE (99.9%)                   │
├────────────────────────────────────────────────────────────┤
│  Availability: 99.95%  │  Latency P99: 245ms  │  Error: 0.02% │
└────────────────────────────────────────────────────────────┘
```

**PromQL 예시**:
```promql
# Availability SLO
(1 - (
  sum(rate(istio_requests_total{response_code=~"5.."}[1h]))
  / sum(rate(istio_requests_total[1h]))
)) * 100

# Latency SLO (P99 < 500ms)
histogram_quantile(0.99,
  sum(rate(istio_request_duration_milliseconds_bucket[5m])) by (le)
)
```

### 트래픽 분석

| 패널 | 메트릭 |
|------|--------|
| RPS by Service | `rate(istio_requests_total[5m])` |
| Active Users | 커스텀 비즈니스 메트릭 |
| Geographic Distribution | CloudFront 메트릭 |

---

## Developer Dashboard

### API 성능 패널

```
┌────────────────────────────────────────────────────────────┐
│                   API LATENCY BY ENDPOINT                  │
├────────────────────────────────────────────────────────────┤
│  GET /api/boards        P50: 45ms   P99: 120ms            │
│  POST /api/tasks        P50: 80ms   P99: 250ms            │
│  GET /api/users/me      P50: 30ms   P99: 85ms             │
└────────────────────────────────────────────────────────────┘
```

### 에러 분석

| 패널 | 내용 |
|------|------|
| Error Rate Timeline | 시간대별 에러율 |
| Error by Status Code | 4xx vs 5xx 분포 |
| Error Traces | Tempo 링크 (클릭 시 트레이스 상세) |

### Trace-Log 연결

```
[Error Log] → [Trace ID: abc123] → [Tempo Trace View]
     ↓
[span: board-service/CreateBoard]
  └── [span: user-service/GetUser]
       └── [span: postgres/query]
```

---

## Infra Dashboard

### 클러스터 상태

```
┌────────────────────────────────────────────────────────────┐
│                    CLUSTER HEALTH                          │
├────────────────────────────────────────────────────────────┤
│  Nodes: 3/3 Ready  │  Pods: 45/50  │  PVC: 5/5 Bound      │
└────────────────────────────────────────────────────────────┘
```

### 리소스 사용량

| 패널 | PromQL |
|------|--------|
| CPU Usage | `sum(rate(container_cpu_usage_seconds_total[5m])) by (pod)` |
| Memory Usage | `sum(container_memory_working_set_bytes) by (pod)` |
| Network I/O | `rate(container_network_receive_bytes_total[5m])` |

### Pod 상태

| 상태 | 색상 |
|------|------|
| Running | 초록 |
| Pending | 노랑 |
| Failed | 빨강 |
| Unknown | 회색 |

---

## Services Overview

### 서비스 헬스맵

```
┌─────────────────────────────────────────────────────────────┐
│                    SERVICE HEALTH MAP                        │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│   [auth] ✓    [user] ✓    [board] ✓                        │
│                                                              │
│   [chat] ✓    [noti] ✓    [storage] ✓                      │
│                                                              │
│   ✓ = Healthy    ⚠ = Warning    ✗ = Critical               │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 서비스 의존성 그래프

```
Service Graph (from OTEL Connector):

     ┌──────────┐
     │  board   │
     └────┬─────┘
          │
    ┌─────┴─────┐
    │           │
    ▼           ▼
┌──────┐   ┌──────┐
│ user │   │ noti │
└──────┘   └──────┘
```

---

## Database Dashboard

### PostgreSQL 메트릭

| 패널 | 메트릭 |
|------|--------|
| Active Connections | `pg_stat_activity_count` |
| Query Duration | `pg_stat_statements_seconds_total` |
| Cache Hit Rate | `pg_stat_database_blks_hit / (hit + read)` |
| Replication Lag | `pg_replication_lag` |

### Redis 메트릭

| 패널 | 메트릭 |
|------|--------|
| Connected Clients | `redis_connected_clients` |
| Memory Usage | `redis_memory_used_bytes` |
| Hit Rate | `redis_keyspace_hits / (hits + misses)` |
| Commands/sec | `rate(redis_commands_total[5m])` |

---

## Log Analysis Dashboard

### 로그 검색

**LogQL 예시**:
```logql
# 에러 로그 검색
{namespace="wealist-prod"} |= "error" | json | level="error"

# 특정 서비스 로그
{namespace="wealist-prod", container="board-service"} | json

# 특정 Trace ID 검색
{namespace="wealist-prod"} |= "trace_id=abc123"
```

### 에러 히스토그램

```
┌────────────────────────────────────────────────────────────┐
│                    ERROR COUNT BY SERVICE                   │
├────────────────────────────────────────────────────────────┤
│  board-service   ████████████ 45                           │
│  auth-service    ████ 12                                   │
│  user-service    ██ 5                                      │
└────────────────────────────────────────────────────────────┘
```

---

## 대시보드 변수

### 공통 변수

| 변수 | 타입 | 예시 |
|------|------|------|
| `$namespace` | Query | `wealist-prod` |
| `$service` | Query | `board-service` |
| `$interval` | Interval | `5m`, `1h` |

### 변수 쿼리

```promql
# 네임스페이스 목록
label_values(kube_namespace_labels, namespace)

# 서비스 목록
label_values(kube_deployment_labels{namespace="$namespace"}, deployment)
```

---

## Grafana 설정

### 데이터소스

| 이름 | 타입 | URL |
|------|------|-----|
| Prometheus | prometheus | `http://prometheus:9090` |
| Loki | loki | `http://loki:3100` |
| Tempo | tempo | `http://tempo:3200` |

### 프로비저닝

```yaml
# grafana/provisioning/dashboards/dashboards.yaml
apiVersion: 1
providers:
  - name: 'wealist'
    folder: 'Wealist'
    type: file
    options:
      path: /var/lib/grafana/dashboards/json
```

---

## 관련 문서

- [Observability Stack](./observability-stack.md)
- [알림 설정](./alerting.md)

