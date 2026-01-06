# API 명세서

## 개요

weAlist는 6개의 백엔드 서비스로 구성되어 있습니다. 모든 API는 RESTful 설계 원칙을 따릅니다.

---

## API Gateway 라우팅

Istio Gateway를 통해 경로 기반 라우팅이 적용됩니다.

| 경로 | 서비스 | 설명 |
|------|--------|------|
| `/api/svc/auth/*` | auth-service:8080 | 인증/토큰 관리 |
| `/api/svc/user/*` | user-service:8081 | 사용자/워크스페이스 |
| `/api/svc/board/*` | board-service:8000 | 프로젝트/보드/태스크 |
| `/api/svc/chat/*` | chat-service:8001 | 채팅/메시지 |
| `/api/svc/noti/*` | noti-service:8002 | 알림 |
| `/api/svc/storage/*` | storage-service:8003 | 파일 저장소 |
| `/api/svc/ops/*` | ops-service:8005 | 운영 API |

---

## 인증 방식

### JWT Bearer Token

```http
Authorization: Bearer <access_token>
```

- **발급**: auth-service `/api/auth/login` 또는 OAuth2
- **알고리즘**: RS256
- **만료**: Access Token 15분, Refresh Token 7일

### 인증 흐름

```
1. 로그인 → auth-service → JWT 토큰 반환
2. API 요청 시 Authorization 헤더에 JWT 포함
3. Istio Gateway에서 JWT 검증 (JWKS)
4. 검증 성공 시 클레임 헤더로 서비스에 전달
```

---

## 공통 응답 형식

### 성공 응답

```json
{
  "data": { ... },
  "message": "Success"
}
```

### 에러 응답

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "Resource not found",
    "details": "Board with ID xxx not found"
  }
}
```

### HTTP 상태 코드

| 코드 | 의미 | 사용 상황 |
|------|------|----------|
| 200 | OK | 성공 |
| 201 | Created | 리소스 생성 성공 |
| 400 | Bad Request | 유효성 검증 실패 |
| 401 | Unauthorized | 인증 필요 |
| 403 | Forbidden | 권한 부족 |
| 404 | Not Found | 리소스 없음 |
| 409 | Conflict | 중복/충돌 |
| 500 | Internal Server Error | 서버 오류 |

---

## 에러 코드

| 코드 | HTTP 상태 | 설명 |
|------|----------|------|
| `VALIDATION_ERROR` | 400 | 입력값 유효성 검증 실패 |
| `UNAUTHORIZED` | 401 | 인증 토큰 없음/만료 |
| `FORBIDDEN` | 403 | 접근 권한 없음 |
| `NOT_FOUND` | 404 | 리소스 찾을 수 없음 |
| `ALREADY_EXISTS` | 409 | 이미 존재하는 리소스 |
| `CONFLICT` | 409 | 상태 충돌 |
| `INTERNAL_ERROR` | 500 | 서버 내부 오류 |

---

## Rate Limiting

### 제한 설정

| 환경 | 분당 요청 | 버스트 |
|------|----------|--------|
| Production | 60 | 10 |
| Development | 1000 | 100 |

### 응답 헤더

```http
X-RateLimit-Limit: 60
X-RateLimit-Remaining: 45
X-RateLimit-Reset: 60
Retry-After: 60  (429 응답 시)
```

---

## 서비스별 API 문서

| 서비스 | 문서 | 주요 기능 |
|--------|------|----------|
| [auth-service](./auth-service.md) | 인증 API | 로그인, 토큰 갱신, OAuth2 |
| [user-service](./user-service.md) | 사용자 API | 프로필, 워크스페이스 |
| [board-service](./board-service.md) | 보드 API | 프로젝트, 태스크, 댓글 |
| [chat-service](./chat-service.md) | 채팅 API | 채팅방, 메시지, WebSocket |
| [noti-service](./noti-service.md) | 알림 API | 알림 조회, SSE 스트림 |
| [storage-service](./storage-service.md) | 저장소 API | 파일 업로드/다운로드 |
| [ops-service](./ops-service.md) | 운영 API | 서비스 상태, 메트릭 |

---

## Health Check Endpoints

모든 서비스는 표준 헬스체크 엔드포인트를 제공합니다.

| 엔드포인트 | 용도 | 응답 |
|-----------|------|------|
| `/health/live` | Liveness Probe | 항상 200 |
| `/health/ready` | Readiness Probe | DB/Redis 연결 확인 |

---

## Swagger 문서

개발 환경에서 각 서비스의 Swagger UI에 접근할 수 있습니다.

| 서비스 | Swagger URL |
|--------|-------------|
| auth-service | `http://localhost:8080/swagger-ui/index.html` |
| user-service | `http://localhost:8081/swagger/index.html` |
| board-service | `http://localhost:8000/swagger/index.html` |
| chat-service | `http://localhost:8001/swagger/index.html` |
| noti-service | `http://localhost:8002/swagger/index.html` |
| storage-service | `http://localhost:8003/swagger/index.html` |
