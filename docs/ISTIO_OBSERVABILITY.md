# Istio Telemetry Guide

wealist 프로젝트의 Istio 서비스 메시 텔레메트리(Telemetry) 가이드입니다.

## 목차

1. [개요](#개요)
2. [메트릭 수집 (Metrics)](#메트릭-수집-metrics)
3. [로그 수집 (Access Logs)](#로그-수집-access-logs)
4. [분산 추적 헤더 (Tracing Headers)](#분산-추적-헤더-tracing-headers)

---

## 개요

Istio는 Envoy 프록시(사이드카 또는 ztunnel)를 통해 모든 트래픽을 가로채고, 자동으로 텔레메트리 데이터를 수집합니다.

```
┌─────────────────────────────────────────────────────────────┐
│                        Istio Mesh                           │
│  ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐  │
│  │ Service │    │ Service │    │ Service │    │ Service │  │
│  │    A    │───▶│    B    │───▶│    C    │───▶│    D    │  │
│  └────┬────┘    └────┬────┘    └────┬────┘    └────┬────┘  │
│       │              │              │              │        │
│       ▼              ▼              ▼              ▼        │
│  ┌──────────────────────────────────────────────────────┐  │
│  │              Telemetry Data (자동 수집)              │  │
│  │  • Metrics (Prometheus 형식)                         │  │
│  │  • Access Logs (JSON/Text)                           │  │
│  │  • Trace Headers (B3/W3C)                            │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

---

## 메트릭 수집 (Metrics)

### Istio 표준 메트릭

Envoy 프록시가 자동으로 수집하는 Prometheus 형식 메트릭:

#### 요청 메트릭 (Request Metrics)

| 메트릭 | 설명 | 타입 |
|--------|------|------|
| `istio_requests_total` | 총 요청 수 | Counter |
| `istio_request_duration_milliseconds` | 요청 처리 시간 | Histogram |
| `istio_request_bytes` | 요청 바이트 크기 | Histogram |
| `istio_response_bytes` | 응답 바이트 크기 | Histogram |

#### TCP 메트릭 (TCP Metrics)

| 메트릭 | 설명 | 타입 |
|--------|------|------|
| `istio_tcp_connections_opened_total` | 열린 TCP 연결 수 | Counter |
| `istio_tcp_connections_closed_total` | 닫힌 TCP 연결 수 | Counter |
| `istio_tcp_sent_bytes_total` | 전송된 바이트 | Counter |
| `istio_tcp_received_bytes_total` | 수신된 바이트 | Counter |

### 메트릭 레이블 (Labels)

모든 메트릭에 자동으로 추가되는 레이블:

```
istio_requests_total{
  source_workload="frontend",
  source_workload_namespace="wealist",
  destination_workload="board-service",
  destination_workload_namespace="wealist",
  destination_service="board-service.wealist.svc.cluster.local",
  request_protocol="http",
  response_code="200",
  response_flags="-",
  connection_security_policy="mutual_tls"
}
```

### 메트릭 스크래핑 설정

Prometheus가 Istio 메트릭을 수집하도록 설정:

```yaml
# Prometheus scrape config
scrape_configs:
  # Istio control plane
  - job_name: 'istiod'
    kubernetes_sd_configs:
      - role: pod
        namespaces:
          names: ['istio-system']
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_label_app]
        regex: istiod
        action: keep

  # Ambient mode: ztunnel metrics
  - job_name: 'ztunnel'
    kubernetes_sd_configs:
      - role: pod
        namespaces:
          names: ['istio-system']
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_label_app]
        regex: ztunnel
        action: keep

  # Ambient mode: Waypoint proxy metrics (if deployed)
  - job_name: 'waypoint'
    kubernetes_sd_configs:
      - role: pod
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_label_gateway_networking_k8s_io/gateway-name]
        regex: .+
        action: keep
```

### 유용한 PromQL 쿼리

```promql
# 서비스별 초당 요청 수 (RPS)
sum(rate(istio_requests_total{destination_service=~".*wealist.*"}[5m])) by (destination_service)

# 서비스별 에러율 (5xx)
sum(rate(istio_requests_total{response_code=~"5.*"}[5m])) by (destination_service)
/
sum(rate(istio_requests_total[5m])) by (destination_service) * 100

# P99 레이턴시
histogram_quantile(0.99,
  sum(rate(istio_request_duration_milliseconds_bucket[5m])) by (destination_service, le)
)

# 서비스 간 트래픽 (source → destination)
sum(rate(istio_requests_total[5m])) by (source_workload, destination_workload)

# mTLS 적용 여부 확인
sum(istio_requests_total{connection_security_policy="mutual_tls"}) by (destination_service)
```

---

## 로그 수집 (Access Logs)

### Envoy Access Log 활성화

Istio Telemetry API로 액세스 로그 설정:

```yaml
apiVersion: telemetry.istio.io/v1alpha1
kind: Telemetry
metadata:
  name: access-logging
  namespace: istio-system  # 전체 메시에 적용
spec:
  accessLogging:
    - providers:
        - name: envoy
```

### Access Log 형식

기본 JSON 형식:

```json
{
  "start_time": "2025-12-18T10:00:00.000Z",
  "method": "GET",
  "path": "/api/boards",
  "protocol": "HTTP/1.1",
  "response_code": 200,
  "response_flags": "-",
  "bytes_received": 0,
  "bytes_sent": 1234,
  "duration": 45,
  "upstream_service_time": 42,
  "x_forwarded_for": "10.0.0.1",
  "user_agent": "Mozilla/5.0",
  "request_id": "abc-123-def",
  "authority": "board-service:8000",
  "upstream_host": "10.1.2.3:8000",
  "upstream_cluster": "outbound|8000||board-service.wealist.svc.cluster.local",
  "upstream_local_address": "10.1.2.4:54321",
  "downstream_local_address": "10.1.2.5:8000",
  "downstream_remote_address": "10.1.2.6:12345",
  "requested_server_name": "-",
  "route_name": "default"
}
```

### Response Flags 의미

| Flag | 설명 |
|------|------|
| `-` | 정상 |
| `UH` | Upstream host 없음 (503) |
| `UF` | Upstream 연결 실패 |
| `UO` | Upstream overflow (circuit breaker) |
| `NR` | No route configured |
| `UC` | Upstream 연결 종료 |
| `UT` | Upstream 타임아웃 |
| `DC` | Downstream 연결 종료 |
| `RL` | Rate limited |

### 네임스페이스별 로그 설정

특정 네임스페이스에만 적용:

```yaml
apiVersion: telemetry.istio.io/v1alpha1
kind: Telemetry
metadata:
  name: access-logging
  namespace: wealist  # 특정 네임스페이스
spec:
  accessLogging:
    - providers:
        - name: envoy
      # 선택: 필터 조건
      filter:
        expression: response.code >= 400  # 에러만 로깅
```

### kubectl로 로그 확인

**Ambient 모드** (ztunnel + waypoint):

```bash
# ztunnel 로그 확인 (L4 트래픽)
kubectl logs -l app=ztunnel -n istio-system --tail=100

# Waypoint 프록시 로그 확인 (L7 트래픽)
kubectl logs -l gateway.networking.k8s.io/gateway-name=wealist-waypoint -n wealist-kind-local --tail=100

# 실시간 ztunnel 로그 스트리밍
kubectl logs -f -l app=ztunnel -n istio-system
```

**Sidecar 모드** (레거시):

```bash
# Envoy 프록시 로그 확인
kubectl logs deploy/board-service -c istio-proxy -n wealist --tail=100
```

---

## 분산 추적 헤더 (Tracing Headers)

### 헤더 전파

Istio는 다음 헤더를 자동으로 생성하지만, **서비스 간 전파는 애플리케이션이 담당**:

```
x-request-id          # 요청 고유 ID
x-b3-traceid          # 트레이스 ID (B3 형식)
x-b3-spanid           # 스팬 ID
x-b3-parentspanid     # 부모 스팬 ID
x-b3-sampled          # 샘플링 여부
traceparent           # W3C Trace Context
tracestate            # W3C Trace State
```

### Go 서비스에서 헤더 전파

```go
// HTTP 클라이언트에서 헤더 전파
func propagateTracingHeaders(ctx context.Context, req *http.Request) {
    headers := []string{
        "x-request-id",
        "x-b3-traceid",
        "x-b3-spanid",
        "x-b3-parentspanid",
        "x-b3-sampled",
        "x-b3-flags",
        "traceparent",
        "tracestate",
    }

    // Gin context에서 헤더 추출
    if ginCtx, ok := ctx.(*gin.Context); ok {
        for _, h := range headers {
            if v := ginCtx.GetHeader(h); v != "" {
                req.Header.Set(h, v)
            }
        }
    }
}
```

### 샘플링 설정

```yaml
apiVersion: telemetry.istio.io/v1alpha1
kind: Telemetry
metadata:
  name: tracing-config
  namespace: istio-system
spec:
  tracing:
    - randomSamplingPercentage: 100  # 개발: 100%, 프로덕션: 1-10%
```

---

## 참고 자료

- [Istio Telemetry API](https://istio.io/latest/docs/tasks/observability/telemetry/)
- [Envoy Access Logging](https://www.envoyproxy.io/docs/envoy/latest/configuration/observability/access_log/usage)
- [Istio Standard Metrics](https://istio.io/latest/docs/reference/config/metrics/)
