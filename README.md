# weAlist Project

weAlist 마이크로서비스 프로젝트입니다.

## 빠른 시작

### Docker Compose (로컬 개발)

```bash
./docker/scripts/dev.sh up
```

### Kubernetes (Kind 클러스터)

```bash
# 1. 클러스터 생성
make kind-setup

# 2. 이미지 빌드/로드
make kind-load-images

# 3. 배포
make kind-apply                                   # 기본: local.wealist.co.kr
make kind-apply LOCAL_DOMAIN=wonny.wealist.co.kr  # 커스텀 도메인
```

> 자세한 내용은 [NOTE.md](./NOTE.md) 참고

## 프로젝트 구조

```
.
├── services/           # 마이크로서비스
│   ├── auth-service/   # 인증 서비스 (Spring Boot)
│   ├── user-service/   # 사용자 서비스 (Go)
│   ├── board-service/  # 게시판 서비스 (Go)
│   ├── chat-service/   # 채팅 서비스 (Go)
│   ├── noti-service/   # 알림 서비스 (Go)
│   ├── storage-service/# 스토리지 서비스 (Go)
│   ├── video-service/  # 비디오 서비스 (Go)
│   └── frontend/       # 프론트엔드 (React)
├── k8s/                # Kubernetes 매니페스트
├── infrastructure/     # 인프라 (PostgreSQL, Redis, MinIO 등)
├── docker/             # Docker 관련 설정
└── Makefile            # 빌드/배포 명령어
```

## 주요 명령어

| 명령어 | 설명 |
|--------|------|
| `make help` | 사용 가능한 명령어 확인 |
| `make kind-setup` | Kind 클러스터 생성 |
| `make kind-load-images` | 이미지 빌드 및 로드 |
| `make kind-apply` | K8s 배포 (기본 도메인) |
| `make kind-apply LOCAL_DOMAIN=<도메인>` | 커스텀 도메인으로 배포 |
| `make kind-delete` | 클러스터 삭제 |
| `make status` | Pod 상태 확인 |
| `make <서비스>-all` | 개별 서비스 재배포 |

## 환경별 접속 정보

### Docker Compose

- Frontend: http://localhost:3000
- API Gateway: http://localhost:8080

### Kubernetes (Kind)

- Frontend: https://local.wealist.co.kr (또는 지정한 도메인)
- 자체 서명 인증서 사용 (브라우저 경고 무시)

## 문서

- [NOTE.md](./NOTE.md) - 빠른 시작 가이드
- [docker/README.md](./docker/README.md) - Docker 환경 가이드
- [docs/CONFIGURATION.md](./docs/CONFIGURATION.md) - 설정 가이드
