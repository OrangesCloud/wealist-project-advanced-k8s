# weAlist Performance Testing

k6를 사용한 weAlist 마이크로서비스 성능 테스트 가이드입니다.

## 설치

### k6 설치

```bash
# macOS
brew install k6

# Linux (Debian/Ubuntu)
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg \
  --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | \
  sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update && sudo apt-get install k6

# Docker
docker pull grafana/k6
```

## 테스트 종류

| 테스트 | 설명 | VUs | 기간 |
|--------|------|-----|------|
| **Load Test** | 정상 부하 상황 측정 | 1→20 | 5분 |
| **Stress Test** | 시스템 한계점 확인 | 50→300 | 30분 |
| **Spike Test** | 급작스러운 트래픽 급증 시뮬레이션 | 20→500 | 10분 |

## 사용법

### 기본 실행

```bash
cd tests/performance

# Load Test
make k6-load

# Stress Test
make k6-stress

# Spike Test
make k6-spike

# 전체 테스트
make k6-all
```

### 환경변수 설정

```bash
# 프로덕션 환경 테스트
BASE_URL=https://api.wealist.co.kr make k6-load

# 인증 토큰 사용
TEST_TOKEN=your-jwt-token make k6-load

# Prometheus로 메트릭 전송
K6_PROMETHEUS_RW_URL=http://prometheus:9090/api/v1/write make k6-load
```

### 시나리오 테스트

```bash
# 인증 플로우 테스트
make k6-auth

# Board CRUD 테스트
make k6-board-crud

# 전체 시나리오
make k6-scenarios
```

## 디렉토리 구조

```
tests/performance/
├── k6/
│   ├── scripts/
│   │   ├── load-test.js           # Load Test
│   │   ├── stress-test.js         # Stress Test
│   │   ├── spike-test.js          # Spike Test
│   │   └── scenarios/
│   │       ├── auth-flow.js       # 인증 플로우
│   │       └── board-crud.js      # Board CRUD
│   └── config/
│       └── thresholds.json        # 임계값 설정
├── Makefile
└── README.md
```

## 메트릭 설명

### 기본 메트릭

| 메트릭 | 설명 |
|--------|------|
| `http_req_duration` | HTTP 요청 응답 시간 |
| `http_req_failed` | HTTP 요청 실패율 |
| `http_reqs` | 총 HTTP 요청 수 |
| `vus` | 현재 활성 Virtual Users |

### 커스텀 메트릭

| 메트릭 | 설명 |
|--------|------|
| `errors` | 커스텀 에러율 |
| `board_latency` | Board 서비스 응답 시간 |
| `user_latency` | User 서비스 응답 시간 |
| `spike_latency` | Spike 기간 응답 시간 |
| `recovery_latency` | 복구 기간 응답 시간 |

## 임계값 (Thresholds)

### Load Test
- p95 응답시간 < 500ms
- p99 응답시간 < 1000ms
- 에러율 < 1%

### Stress Test
- p95 응답시간 < 2000ms
- 에러율 < 10%

### Spike Test
- p95 응답시간 < 3000ms
- 에러율 < 20%

## Grafana 연동

k6 메트릭을 Grafana에서 시각화하려면 Prometheus Remote Write를 사용합니다.

### 실행 방법

```bash
# Prometheus Remote Write 활성화
K6_PROMETHEUS_RW_SERVER_URL=http://prometheus:9090/api/v1/write \
k6 run --out experimental-prometheus-rw k6/scripts/load-test.js
```

### Grafana 대시보드

`k8s/helm/charts/wealist-monitoring/dashboards/k6-performance-dashboard.json`에서
성능 테스트 결과를 시각화할 수 있습니다.

## K8s에서 실행

### ConfigMap으로 스크립트 배포

```bash
# Helm으로 k6 Job 배포
helm upgrade wealist-monitoring ./k8s/helm/charts/wealist-monitoring \
  --set k6.enabled=true \
  -n wealist-prod
```

### Job 실행

```bash
kubectl create job k6-load-test \
  --from=cronjob/k6-load-test \
  -n wealist-prod
```

## 트러블슈팅

### k6 설치 확인

```bash
make check-k6
```

### 서비스 헬스 체크

```bash
curl http://localhost:8080/svc/board/health/live
curl http://localhost:8080/svc/user/health/live
```

### 결과 파일 확인

```bash
cat load-test-summary.json
cat stress-test-summary.json
cat spike-test-summary.json
```

## 참고 자료

- [k6 Documentation](https://k6.io/docs/)
- [k6 Prometheus Integration](https://grafana.com/docs/k6/latest/results-output/grafana-dashboards/)
- [Load Testing Best Practices](https://grafana.com/blog/2025/11/12/performance-testing-best-practices-how-to-prepare-for-peak-demand-with-grafana-cloud-k6/)
