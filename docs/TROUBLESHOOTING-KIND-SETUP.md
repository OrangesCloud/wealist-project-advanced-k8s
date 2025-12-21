# Kind 개발환경 트러블슈팅 가이드

이 문서는 다양한 OS 환경(macOS, WSL, Linux)에서 Kind 클러스터 설정 시 발생할 수 있는 문제와 해결책을 정리합니다.

## 목차
1. [OS별 DB 호스트 설정](#1-os별-db-호스트-설정)
2. [Docker 이미지 로드 실패](#2-docker-이미지-로드-실패)
3. [GHCR 인증 문제](#3-ghcr-인증-문제)
4. [PostgreSQL 외부 연결](#4-postgresql-외부-연결)
5. [Redis 외부 연결](#5-redis-외부-연결)
6. [멀티 아키텍처 이미지](#6-멀티-아키텍처-이미지)
7. [Shell 호환성](#7-shell-호환성)
8. [Port Forwarding (외부 접근)](#8-port-forwarding-외부-접근)

---

## 1. OS별 DB 호스트 설정

### 문제
Kind 클러스터에서 호스트 PC의 PostgreSQL/Redis에 연결할 때, OS마다 사용해야 하는 호스트 주소가 다릅니다.

### 원인
- **macOS (Docker Desktop)**: Docker Desktop이 `host.docker.internal` DNS를 자동 제공
- **WSL**: WSL의 IP 주소가 Windows 호스트와 별도 (172.29.x.x 대역)
- **Linux (native)**: Docker 브릿지 네트워크 게이트웨이 사용 (172.18.0.1)

### 해결책
```bash
# OS 자동 감지 스크립트
if [ "$(uname)" = "Darwin" ]; then
    DB_HOST="host.docker.internal"
elif grep -qi microsoft /proc/version 2>/dev/null; then
    DB_HOST=$(hostname -I | awk '{print $1}')  # WSL IP
else
    DB_HOST="172.18.0.1"  # Linux Docker bridge
fi
```

### 적용 위치
- `makefiles/kind.mk` - `kind-dev-setup` 타겟

---

## 2. Docker 이미지 로드 실패

### 문제 A: Docker Desktop containerd 모드

**증상:**
```
ERROR: failed to load image: command "docker exec ... ctr ... images import" failed
ctr: content digest sha256:xxx: not found
```

**원인:**
Docker Desktop의 "Use containerd for pulling and storing images" 옵션이 활성화된 경우 `kind load` 명령어와 호환되지 않음.

**해결책 (GUI):**
1. Docker Desktop → Settings → General
2. "Use containerd for pulling and storing images" **비활성화**
3. Apply & Restart

### 문제 B: WSL native Docker storage driver

**증상:**
동일한 `ctr: content digest not found` 에러

**원인:**
WSL에 설치된 Docker가 `stargz` 또는 `containerd` storage driver 사용

**해결책 (코드):**
```bash
# Storage driver 확인
docker info | grep "Storage Driver"

# overlay2로 변경
sudo tee /etc/docker/daemon.json << 'EOF'
{
  "storage-driver": "overlay2"
}
EOF

sudo systemctl restart docker
```

### 문제 C: 모든 로드 방법 실패 시

**해결책 - 3단계 Fallback:**
```bash
# 방법 1: docker-image (가장 빠름)
kind load docker-image $IMAGE --name wealist

# 방법 2: image-archive (tar 저장)
docker save $IMAGE -o /tmp/image.tar
kind load image-archive /tmp/image.tar --name wealist

# 방법 3: ctr 직접 import (최후의 수단)
docker save $IMAGE -o /tmp/image.tar
docker exec -i wealist-control-plane ctr -n k8s.io images import - < /tmp/image.tar
```

### 적용 위치
- `k8s/helm/scripts/dev/1.load_infra_images.sh` - `load_to_kind()` 함수

---

## 3. GHCR 인증 문제

### 문제
`ImagePullBackOff` 에러 - Kind 클러스터에서 GHCR 이미지를 pull 못함

**증상:**
```
Failed to pull image "ghcr.io/orangescloud/auth-service:latest":
unauthorized: authentication required
```

### 원인
Docker Desktop의 credential helper가 토큰을 OS keychain에 저장하여 `~/.docker/config.json`에 실제 토큰이 없음.

**잘못된 방법:**
```bash
# Docker config 파일 기반 - 토큰이 없어서 실패
kubectl create secret generic ghcr-secret \
    --from-file=.dockerconfigjson=~/.docker/config.json
```

### 해결책
```bash
# 명시적 credentials로 secret 생성
kubectl create secret docker-registry ghcr-secret \
    --docker-server=ghcr.io \
    --docker-username="YOUR_GITHUB_USERNAME" \
    --docker-password="YOUR_GITHUB_PAT" \
    -n wealist-dev
```

### 적용 위치
- `makefiles/kind.mk` - `kind-dev-setup` 6단계

---

## 4. PostgreSQL 외부 연결

### 문제
Kind Pod에서 호스트 PostgreSQL 연결 실패

**증상:**
```
pg_isready -h 172.29.102.162 -p 5432
# connection refused
```

### 원인
1. PostgreSQL이 localhost만 listen
2. pg_hba.conf에 Docker 네트워크 대역 미허용

### 해결책

**postgresql.conf:**
```conf
listen_addresses = '*'
```

**pg_hba.conf:**
```conf
# WSL의 경우 (IP 대역 자동 계산)
host    all    all    172.29.0.0/16    md5

# Linux Docker의 경우
host    all    all    172.18.0.0/16    md5
```

**자동 설정 스크립트:**
```bash
# PostgreSQL 설정 파일 찾기
PG_CONF=$(sudo -u postgres psql -t -c "SHOW config_file" | tr -d ' ')
PG_HBA=$(sudo -u postgres psql -t -c "SHOW hba_file" | tr -d ' ')

# listen_addresses 설정
sudo sed -i "s/^#*listen_addresses.*/listen_addresses = '*'/" "$PG_CONF"

# DB 서브넷 허용 (동적 계산)
DB_SUBNET=$(echo "$DB_HOST" | sed 's/\.[0-9]*\.[0-9]*$/.0.0\/16/')
echo "host    all    all    $DB_SUBNET    md5" | sudo tee -a "$PG_HBA"

# 재시작
sudo systemctl restart postgresql
```

### 적용 위치
- `makefiles/kind.mk` - `kind-dev-setup` 5단계

---

## 5. Redis 외부 연결

### 문제
Kind Pod에서 호스트 Redis 연결 실패

**증상:**
```
redis-cli -h 172.29.102.162 ping
# Connection refused
```

### 원인
1. Redis가 127.0.0.1만 bind
2. protected-mode 활성화

### 해결책

**redis.conf:**
```conf
bind 0.0.0.0
protected-mode no
```

**자동 설정 스크립트:**
```bash
# Redis 설정 파일 찾기 (sudo 필요)
for path in /etc/redis/redis.conf /etc/redis.conf /usr/local/etc/redis.conf; do
    if sudo test -f "$path" 2>/dev/null; then
        REDIS_CONF="$path"
        break
    fi
done

# 설정 변경 (sudo로 grep/sed)
sudo sed -i 's/^bind .*/bind 0.0.0.0/' "$REDIS_CONF"
sudo sed -i 's/^protected-mode yes/protected-mode no/' "$REDIS_CONF"

# 설정이 없으면 추가
if ! sudo grep -q "^bind 0.0.0.0" "$REDIS_CONF"; then
    echo "bind 0.0.0.0" | sudo tee -a "$REDIS_CONF"
fi

# 재시작
sudo systemctl restart redis
```

### 주의사항
- Redis 설정 파일 읽기에 `sudo` 필요 (권한 문제)
- `grep: Permission denied` 에러 시 `sudo grep` 사용

### 적용 위치
- `makefiles/kind.mk` - `kind-dev-setup` 5단계

---

## 6. 멀티 아키텍처 이미지

### 문제
Mac (arm64)에서 빌드한 이미지가 WSL (amd64)에서 실행 안됨

**증상:**
```
Failed to pull image: no match for platform in manifest: not found
```

**확인 방법:**
```bash
docker manifest inspect ghcr.io/orangescloud/auth-service:latest
# "architecture": "arm64" 만 있으면 문제
```

### 해결책
`docker buildx`로 멀티 아키텍처 빌드:

```bash
# buildx 빌더 생성 (최초 1회)
docker buildx create --name multiarch-builder --use --bootstrap

# 멀티 플랫폼 빌드 및 푸시
docker buildx build --platform linux/amd64,linux/arm64 \
    -t ghcr.io/orangescloud/auth-service:latest \
    --push .
```

### 적용 위치
- `makefiles/services.mk` - `ghcr-push-all` 타겟

**사용법:**
```bash
make ghcr-push-all ENV=dev
```

---

## 7. Shell 호환성

### 문제
WSL에서 Makefile 스크립트 실행 시 `read -s` 옵션 에러

**증상:**
```
read: Illegal option -s
```

### 원인
WSL의 기본 `/bin/sh`가 dash이며, `read -s`는 bash 전용 옵션

### 해결책
POSIX 호환 방식으로 변경:

```bash
# 잘못된 방법 (bash 전용)
read -s PASSWORD

# 올바른 방법 (POSIX 호환)
stty -echo 2>/dev/null || true
read PASSWORD
stty echo 2>/dev/null || true
```

### 적용 위치
- `makefiles/kind.mk` - GHCR 로그인 토큰 입력 부분

---

## 8. Port Forwarding (외부 접근)

### 구성
```
CloudFront → DDNS (wonnyhome.myddns.me:80)
    → 공유기 포트포워딩
    → WSL (172.29.102.162)
    → Kind NodePort (30080)
    → Istio Gateway
    → Services
```

### Kind NodePort 확인
```bash
kubectl get svc -n istio-system | grep ingressgateway
# 80:30080/TCP 확인
```

### 방법 A: 공유기에서 직접 포워딩
```
외부 80 → WSL_IP:30080
```

### 방법 B: WSL에서 iptables 포워딩
```bash
sudo iptables -t nat -A PREROUTING -p tcp --dport 80 -j REDIRECT --to-port 30080
sudo iptables -t nat -A OUTPUT -p tcp --dport 80 -j REDIRECT --to-port 30080
```

### 연결 테스트
```bash
# 로컬 테스트
curl -v http://localhost:30080/

# Kind 노드 직접 테스트
curl -v http://172.18.0.3:30080/
```

---

## 빠른 참조: 환경별 설정 요약

| 항목 | macOS (Docker Desktop) | WSL (native Docker) | Linux |
|------|------------------------|---------------------|-------|
| DB Host | `host.docker.internal` | WSL IP (동적) | `172.18.0.1` |
| containerd 이슈 | Docker Desktop 설정 비활성화 | daemon.json 수정 | 보통 없음 |
| Storage Driver | - | overlay2 권장 | overlay2 |
| Image Load | tar 방식 권장 | 3단계 fallback | docker-image |

---

## 관련 파일
- `makefiles/kind.mk` - Kind 클러스터 설정
- `makefiles/services.mk` - 서비스 빌드/푸시
- `k8s/helm/scripts/dev/1.load_infra_images.sh` - 인프라 이미지 로드
- `k8s/helm/environments/dev.yaml` - dev 환경 설정
