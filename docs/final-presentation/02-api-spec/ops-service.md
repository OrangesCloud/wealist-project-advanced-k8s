# Ops Service API

> **기술**: Spring Boot 3.4
>
> **포트**: 8005
>
> **역할**: 운영 API, 서비스 모니터링

---

## Endpoints

### Health & Status

| Method | Endpoint | Auth | Description |
|--------|----------|:----:|-------------|
| GET | `/api/health` | - | 서비스 상태 |
| GET | `/api/status` | ✓ | 전체 시스템 상태 |
| GET | `/api/services` | ✓ | 서비스 목록 |
| GET | `/api/services/{name}/health` | ✓ | 특정 서비스 상태 |

### Metrics

| Method | Endpoint | Auth | Description |
|--------|----------|:----:|-------------|
| GET | `/api/metrics/overview` | ✓ | 메트릭 개요 |
| GET | `/api/metrics/services` | ✓ | 서비스별 메트릭 |
| GET | `/actuator/prometheus` | - | Prometheus 메트릭 |

### Users (Admin)

| Method | Endpoint | Auth | Description |
|--------|----------|:----:|-------------|
| GET | `/api/admin/users` | ✓ (Admin) | 사용자 목록 |
| GET | `/api/admin/users/{id}` | ✓ (Admin) | 사용자 상세 |
| PUT | `/api/admin/users/{id}/status` | ✓ (Admin) | 사용자 상태 변경 |

---

## Request/Response

### GET /api/status

전체 시스템 상태를 조회합니다.

**Response** (200 OK)
```json
{
  "status": "healthy",
  "timestamp": "2026-01-05T10:00:00Z",
  "services": {
    "auth-service": {
      "status": "up",
      "responseTime": 45
    },
    "user-service": {
      "status": "up",
      "responseTime": 32
    },
    "board-service": {
      "status": "up",
      "responseTime": 28
    },
    "chat-service": {
      "status": "up",
      "responseTime": 35
    },
    "noti-service": {
      "status": "up",
      "responseTime": 30
    },
    "storage-service": {
      "status": "up",
      "responseTime": 40
    }
  },
  "infrastructure": {
    "postgres": {
      "status": "up",
      "connections": 25
    },
    "redis": {
      "status": "up",
      "connectedClients": 12
    }
  }
}
```

---

### GET /api/services

등록된 서비스 목록을 조회합니다.

**Response** (200 OK)
```json
{
  "services": [
    {
      "name": "auth-service",
      "version": "1.0.0",
      "port": 8080,
      "technology": "Spring Boot",
      "healthEndpoint": "/actuator/health",
      "replicas": 2
    },
    {
      "name": "user-service",
      "version": "1.0.0",
      "port": 8081,
      "technology": "Go",
      "healthEndpoint": "/health/ready",
      "replicas": 2
    }
  ]
}
```

---

### GET /api/metrics/overview

시스템 메트릭 개요를 조회합니다.

**Response** (200 OK)
```json
{
  "timestamp": "2026-01-05T10:00:00Z",
  "period": "1h",
  "summary": {
    "totalRequests": 15420,
    "avgResponseTime": 85,
    "errorRate": 0.02,
    "p99Latency": 250
  },
  "topEndpoints": [
    {
      "path": "/api/boards",
      "method": "GET",
      "count": 5230,
      "avgLatency": 45
    }
  ]
}
```

---

### GET /api/admin/users

관리자용 사용자 목록을 조회합니다.

**Query Parameters**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| page | int | 1 | 페이지 번호 |
| limit | int | 20 | 페이지 크기 |
| status | string | - | 상태 필터 (active, inactive, suspended) |
| search | string | - | 검색어 (이름, 이메일) |

**Response** (200 OK)
```json
{
  "users": [
    {
      "id": "uuid",
      "email": "user@example.com",
      "name": "John Doe",
      "status": "active",
      "createdAt": "2026-01-01T00:00:00Z",
      "lastLoginAt": "2026-01-05T09:00:00Z",
      "workspacesCount": 3
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 150
  }
}
```

---

### PUT /api/admin/users/{id}/status

사용자 상태를 변경합니다.

**Request**
```json
{
  "status": "suspended",
  "reason": "Terms of service violation"
}
```

**Response** (200 OK)
```json
{
  "id": "uuid",
  "status": "suspended",
  "updatedAt": "2026-01-05T10:00:00Z"
}
```

---

## 인증

ops-service API는 JWT 인증을 사용하며, 관리자 권한이 필요한 엔드포인트가 있습니다.

### 역할

| Role | Description |
|------|-------------|
| `viewer` | 읽기 전용 |
| `operator` | 서비스 상태 조회 |
| `admin` | 전체 관리 권한 |

### Admin 엔드포인트 접근

```
Authorization: Bearer <access_token>
```

JWT 클레임에 `role: admin`이 포함되어야 합니다.

---

## Ops Portal

ops-service의 프론트엔드인 ops-portal은 별도 포트 (3001)에서 서빙됩니다.

| 환경 | URL |
|------|-----|
| Production | `https://api.wealist.co.kr/api/svc/ops-portal` |
| Development | `http://localhost:3001` |

---

## 관련 문서

- [API 개요](./README.md)
- [모니터링 가이드](../04-monitoring/observability-stack.md)
