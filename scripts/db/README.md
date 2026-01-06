# Local Database Setup Scripts

K8s 클러스터 재생성 시 데이터 유실을 방지하기 위한 로컬 PostgreSQL/Redis 설치 스크립트.

## Quick Start

```bash
# Ubuntu에서 실행 (PostgreSQL 17 + Redis 7 설치 + DB 생성)
cd scripts/db
sudo ./setup-local-db.sh
```

## Scripts

| Script | Description |
|--------|-------------|
| `setup-local-db.sh` | 전체 설치 (PostgreSQL 17, Redis 7, DB 생성, 원격 접속 설정) |
| `init-databases-only.sh` | DB/User 생성만 (PostgreSQL 이미 설치된 경우) |
| `01-create-databases.sql` | 6개 서비스 DB 및 User 생성 SQL |
| `drop-all-databases.sql` | 전체 DB 삭제 (초기화용) |

## Databases Created

| Database | User | Password | Service |
|----------|------|----------|---------|
| user_db | user_service | user_service_password | user-service |
| board_db | board_service | board_service_password | board-service |
| chat_db | chat_service | chat_service_password | chat-service |
| noti_db | noti_service | noti_service_password | noti-service |
| storage_db | storage_service | storage_service_password | storage-service |

> Note: auth-service는 Redis만 사용 (DB 없음)

## K8s 연결 설정

설치 후 Helm values 파일 수정 필요:

```yaml
# helm/environments/local-ubuntu.yaml
shared:
  config:
    POSTGRES_HOST: "YOUR_LOCAL_IP"  # 예: 192.168.1.100
    DB_HOST: "YOUR_LOCAL_IP"
    REDIS_HOST: "YOUR_LOCAL_IP"
```

또는 `local-ubuntu-secrets.yaml`에 추가.

## 수동 테스트

```bash
# PostgreSQL 연결 테스트
psql -h <LOCAL_IP> -U user_service -d user_db

# Redis 연결 테스트
redis-cli -h <LOCAL_IP> ping
```

## 주의사항

- `setup-local-db.sh`는 PostgreSQL/Redis를 모든 IP에서 접속 가능하도록 설정합니다.
- 프로덕션 환경에서는 절대 사용하지 마세요!
- 방화벽 설정 확인 필요 (5432, 6379 포트 개방)
