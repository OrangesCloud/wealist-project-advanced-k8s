# docker-compose

```
# 1) 환경 파일 확인 (이미 생성됨)
ls docker/env/.env.dev

# 2) 전체 서비스 + 모니터링 실행
make dev-up
# 또는 모노레포 빌드 (Go 서비스 빌드가 더 빠름)
make dev-mono-up

# 3) 로그 확인
make dev-logs

# 4) 접속 테스트
# Grafana: http://localhost:3001 (admin/admin)
# Prometheus: http://localhost:9090
# Loki: http://localhost:3100

# Prometheus 살아있나?
curl -s http://localhost:9090/-/ready
# Loki 살아있나?
curl -s http://localhost:3100/ready

# Promtail이 Loki로 보내는지 확인 (라벨 있으면 OK)
curl -s 'http://localhost:3100/loki/api/v1/labels' | jq .

# 5) 메트릭 수집 확인
# Prometheus UI → Status → Targets 에서 서비스들이 UP인지 확인
# 쿼리 테스트: {__name__=~".+_http_requests_total"}

# 6) 로그 수집 확인
# Grafana → Explore → Loki 선택
# 쿼리: {compose_service=~".+"}

# 7) 종료
make dev-down
```

# prometheus curl query

```
# 서비스 상태 확인 (up)
curl -s 'http://localhost:9090/api/v1/query?query=up' | jq .

# HTTP 요청 총계 (go-pkg 메트릭)
curl -s 'http://localhost:9090/api/v1/query?query={__name__=~".%2B_http_requests_total"}' | jq .

# 특정 서비스 메트릭 확인 (예: board-service)
curl -s 'http://localhost:9090/api/v1/query?query=board_service_http_requests_total' | jq .

# 전체 메트릭 이름 목록
curl -s 'http://localhost:9090/api/v1/label/__name__/values' | jq .

# Targets 상태 확인
curl -s 'http://localhost:9090/api/v1/targets' | jq '.data.activeTargets[] | {job: .labels.job, health: .health}'
```

# prometheus curl query

```
# 최근 로그 10개 (compose_service 라벨로)

curl -s 'http://localhost:3100/loki/api/v1/query_range' \
 --data-urlencode 'query={compose_service=~".+"}' \
 --data-urlencode 'limit=10' | jq .

# 특정 서비스 로그 (예: user-service)

curl -s 'http://localhost:3100/loki/api/v1/query_range' \
 --data-urlencode 'query={compose_service="user-service"}' \
 --data-urlencode 'limit=5' | jq .

# 사용 가능한 라벨 목록

curl -s 'http://localhost:3100/loki/api/v1/labels' | jq .

# compose_service 라벨 값들

curl -s 'http://localhost:3100/loki/api/v1/label/compose_service/values' | jq .

# Loki 준비 상태 확인

curl -s 'http://localhost:3100/ready'

```

# helm(kind) 테스트

```
# 1) Kind 클러스터 생성 (이미 있으면 스킵)
make kind-setup

# 2) 이미지 빌드 및 로드
make kind-load-images-mono

# 3) dev.yaml 생성
cp helm/environments/local-kind.yaml helm/environments/dev.yaml

# 4) 서비스 설치
make helm-install-all ENV=dev

# 5) 모니터링 스택 별도 설치 (wealist-monitoring 차트)
kubectl create namespace wealist-monitoring 2>/dev/null || true
helm install wealist-monitoring helm/charts/wealist-monitoring \
  -n wealist-monitoring \
  -f helm/charts/wealist-monitoring/values.yaml

# 6) Grafana 포트포워딩
kubectl port-forward svc/grafana -n wealist-monitoring 3001:3000

# 7) 접속
# Grafana: http://localhost:3001 (admin/admin)

# 8) 정리
helm uninstall wealist-monitoring -n wealist-monitoring
make helm-uninstall-all ENV=dev

```

```
미니 PC 구성:

PostgreSQL 설치 (sudo apt install postgresql)
Redis 설치 (sudo apt install redis-server)
PostgreSQL/Redis가 외부 접속 허용하도록 설정 (dev.yaml 주석 참고)
helm-install-all 하면:

Go 서비스들 → 호스트 PC의 PostgreSQL/Redis에 연결
postgres-exporter → 호스트 PC의 PostgreSQL 메트릭 수집
redis-exporter → 호스트 PC의 Redis 메트릭 수집
Prometheus → 이 Exporter들 + 서비스들 메트릭 수집
```
