# Observability Stack

> **3대 신호**: Metrics (메트릭) + Logs (로그) + Traces (트레이스)

---

## 스택 개요

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      OBSERVABILITY STACK                                 │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │                         DATA SOURCES                               │ │
│  │  ┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────────┐        │ │
│  │  │ Services │   │   Pods   │   │  Istio   │   │   Nodes  │        │ │
│  │  │(metrics) │   │  (logs)  │   │(sidecar) │   │ (system) │        │ │
│  │  └────┬─────┘   └────┬─────┘   └────┬─────┘   └────┬─────┘        │ │
│  └───────┼──────────────┼──────────────┼──────────────┼──────────────┘ │
│          │              │              │              │                 │
│  ┌───────┼──────────────┼──────────────┼──────────────┼──────────────┐ │
│  │       ▼              ▼              ▼              ▼               │ │
│  │  ┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────────┐        │ │
│  │  │Prometheus│   │  Alloy   │   │   OTEL   │   │  Node    │        │ │
│  │  │ scrape   │   │(promtail)│   │Collector │   │ Exporter │        │ │
│  │  └────┬─────┘   └────┬─────┘   └────┬─────┘   └────┬─────┘        │ │
│  │       │              │              │              │               │ │
│  │       ▼              ▼              ▼              │               │ │
│  │  ┌──────────┐   ┌──────────┐   ┌──────────┐       │               │ │
│  │  │Prometheus│   │   Loki   │   │  Tempo   │◀──────┘               │ │
│  │  │  (TSDB)  │   │  (logs)  │   │ (traces) │                       │ │
│  │  └────┬─────┘   └────┬─────┘   └────┬─────┘                       │ │
│  │       │              │              │                              │ │
│  │       └──────────────┼──────────────┘                              │ │
│  │                      ▼                                             │ │
│  │              ┌──────────────┐                                      │ │
│  │              │   GRAFANA    │                                      │ │
│  │              │ (Dashboard)  │                                      │ │
│  │              └──────────────┘                                      │ │
│  │                   STORAGE & VISUALIZATION                          │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 컴포넌트 상세

### Prometheus (메트릭)

| 항목 | 값 |
|------|------|
| 버전 | v2.55.1 |
| 포트 | 9090 |
| 저장 | TSDB (로컬) |
| 보존 기간 | 15일 |

**수집 대상**:
- 서비스 `/metrics` 엔드포인트
- Istio sidecar 메트릭
- kube-state-metrics
- node-exporter

```yaml
# Prometheus 설정 (prod.yaml)
prometheus:
  enabled: true
  config:
    externalUrl: "https://api.wealist.co.kr/api/monitoring/prometheus"
  remoteWriteReceiver:
    enabled: true  # Tempo/OTEL에서 메트릭 수신
  exemplarStorage:
    enabled: true  # 메트릭 → 트레이스 연결
```

### Loki (로그)

| 항목 | 값 |
|------|------|
| 버전 | 3.6.3 |
| 포트 | 3100 |
| 저장 | S3 (prod), 로컬 (localhost) |
| 스키마 | TSDB v13 |

**수집 방식**:
- Alloy (구 Promtail)가 Pod 로그 수집
- 라벨: `namespace`, `pod`, `container`

```yaml
# Loki 설정 (prod.yaml)
loki:
  enabled: true
  config:
    s3:
      enabled: true
      bucket: "wealist-prod-loki-logs-{account}"
      region: "ap-northeast-2"
```

### Tempo (트레이스)

| 항목 | 값 |
|------|------|
| 버전 | 2.9.0 |
| 포트 | 3200 (HTTP), 4317 (OTLP) |
| 저장 | S3 (prod), 로컬 (localhost) |
| 보존 기간 | 7일 |

**수집 방식**:
- OTEL Collector에서 OTLP로 전송
- 서비스 → OTEL Collector → Tempo

```yaml
# Tempo 설정 (prod.yaml)
tempo:
  enabled: true
  metricsGenerator:
    enabled: true
    remoteWriteUrl: "http://prometheus:9090/api/monitoring/prometheus/api/v1/write"
  config:
    s3:
      enabled: true
      bucket: "wealist-prod-tempo-traces-{account}"
```

### OTEL Collector (텔레메트리 허브)

| 항목 | 값 |
|------|------|
| 버전 | 0.114.0 |
| OTLP gRPC | 4317 |
| OTLP HTTP | 4318 |
| 레플리카 | 2 (prod) |

**파이프라인**:
```
Services → OTEL Collector → Tempo (traces)
                         → Prometheus (metrics via remote write)
```

```yaml
# OTEL Collector 설정 (prod.yaml)
otelCollector:
  enabled: true
  tailSampling:
    enabled: true
    policies:
      errorSampling: true           # 에러 100% 샘플링
      latencyThresholdMs: 1000      # 1초 이상 100% 샘플링
      defaultSamplingPercentage: 10 # 나머지 10%
  connectors:
    spanmetrics:
      enabled: true    # 트레이스에서 메트릭 생성
    servicegraph:
      enabled: true    # 서비스 의존성 그래프
```

---

## 데이터 흐름

### Metrics Pipeline

```
┌──────────┐   scrape    ┌────────────┐   store    ┌──────────┐
│ Services │ ──────────▶ │ Prometheus │ ──────────▶│   TSDB   │
│ /metrics │             │            │            │          │
└──────────┘             └────────────┘            └──────────┘
                               │
                               │ query
                               ▼
                         ┌──────────┐
                         │ Grafana  │
                         └──────────┘
```

### Logs Pipeline

```
┌──────────┐   stdout    ┌──────────┐   push    ┌──────────┐
│   Pods   │ ──────────▶ │  Alloy   │ ────────▶│   Loki   │
│          │             │(DaemonSet)│          │          │
└──────────┘             └──────────┘          └──────────┘
                                                    │
                                                    │ LogQL
                                                    ▼
                                              ┌──────────┐
                                              │ Grafana  │
                                              └──────────┘
```

### Traces Pipeline

```
┌──────────┐   OTLP     ┌──────────┐   OTLP    ┌──────────┐
│ Services │ ─────────▶ │   OTEL   │ ────────▶│  Tempo   │
│          │            │Collector │           │          │
└──────────┘            └──────────┘           └──────────┘
                              │                     │
                              │ remote write        │ TraceQL
                              ▼                     ▼
                        ┌──────────┐          ┌──────────┐
                        │Prometheus│◀─────────│ Grafana  │
                        └──────────┘          └──────────┘
```

---

## 서비스별 계측

### Go 서비스

```go
// OpenTelemetry 초기화
import "go.opentelemetry.io/otel"

tracer := otel.Tracer("board-service")
ctx, span := tracer.Start(ctx, "CreateBoard")
defer span.End()
```

**환경변수**:
```yaml
OTEL_EXPORTER_OTLP_ENDPOINT: "http://otel-collector:4318"
OTEL_EXPORTER_OTLP_PROTOCOL: "http/protobuf"
OTEL_TRACES_SAMPLER: "parentbased_traceidratio"
OTEL_TRACES_SAMPLER_ARG: "0.1"  # prod: 10%
```

### Spring Boot 서비스 (auth-service)

```yaml
# application.yml
management:
  tracing:
    sampling:
      probability: 0.1  # 10%
  otlp:
    tracing:
      endpoint: "http://otel-collector:4318/v1/traces"
```

### Istio Sidecar

Envoy sidecar가 자동으로 트레이스 헤더 전파:
- `x-request-id`
- `x-b3-traceid`
- `traceparent` (W3C)

---

## 메트릭 종류

### RED Metrics (서비스)

| 메트릭 | 설명 | PromQL 예시 |
|--------|------|-------------|
| Rate | 초당 요청 수 | `rate(http_requests_total[5m])` |
| Errors | 에러율 | `rate(http_requests_total{status=~"5.."}[5m])` |
| Duration | 응답 시간 | `histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))` |

### USE Metrics (리소스)

| 메트릭 | 설명 | PromQL 예시 |
|--------|------|-------------|
| Utilization | 사용률 | `container_cpu_usage_seconds_total` |
| Saturation | 포화도 | `container_memory_working_set_bytes` |
| Errors | 에러 수 | `container_oom_events_total` |

### Istio Metrics

| 메트릭 | 설명 |
|--------|------|
| `istio_requests_total` | 총 요청 수 |
| `istio_request_duration_milliseconds` | 요청 지연시간 |
| `istio_tcp_connections_opened_total` | TCP 연결 수 |

---

## 접속 URL

### Production

| 컴포넌트 | URL |
|----------|-----|
| Grafana | `https://api.wealist.co.kr/api/monitoring/grafana` |
| Prometheus | `https://api.wealist.co.kr/api/monitoring/prometheus` |
| Loki | `https://api.wealist.co.kr/api/monitoring/loki` |
| Kiali | `https://api.wealist.co.kr/api/monitoring/kiali` |
| Jaeger | `https://api.wealist.co.kr/api/monitoring/jaeger` |

### Localhost

| 컴포넌트 | URL |
|----------|-----|
| Grafana | `http://localhost:8080/api/monitoring/grafana` |
| Prometheus | `http://localhost:8080/api/monitoring/prometheus` |

---

## 관련 문서

- [대시보드 가이드](./dashboards.md)
- [알림 설정](./alerting.md)

