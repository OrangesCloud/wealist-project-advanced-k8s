# Noti Service API

> **기술**: Go + Gin
>
> **포트**: 8002
>
> **역할**: 알림 관리, 실시간 알림 스트림

---

## Endpoints

### Notifications

| Method | Endpoint | Auth | Description |
|--------|----------|:----:|-------------|
| GET | `/api/notifications` | ✓ | 알림 목록 |
| GET | `/api/notifications/{id}` | ✓ | 알림 상세 |
| PUT | `/api/notifications/{id}/read` | ✓ | 읽음 처리 |
| PUT | `/api/notifications/read-all` | ✓ | 전체 읽음 |
| DELETE | `/api/notifications/{id}` | ✓ | 알림 삭제 |
| GET | `/api/notifications/unread-count` | ✓ | 읽지 않은 개수 |

### SSE Stream

| Endpoint | Auth | Description |
|----------|:----:|-------------|
| `/api/notifications/stream` | ✓ | 실시간 알림 스트림 |

---

## Request/Response

### GET /api/notifications

알림 목록을 조회합니다 (페이지네이션).

**Query Parameters**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| page | int | 1 | 페이지 번호 |
| limit | int | 20 | 페이지 크기 |
| unreadOnly | bool | false | 읽지 않은 것만 |

**Response** (200 OK)
```json
{
  "notifications": [
    {
      "id": "uuid",
      "type": "task_assigned",
      "title": "New task assigned",
      "message": "You have been assigned to 'Implement feature'",
      "data": {
        "taskId": "task-uuid",
        "projectId": "project-uuid"
      },
      "isRead": false,
      "createdAt": "2026-01-05T10:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 45,
    "totalPages": 3
  }
}
```

---

### PUT /api/notifications/{id}/read

알림을 읽음 처리합니다.

**Response** (200 OK)
```json
{
  "id": "uuid",
  "isRead": true,
  "readAt": "2026-01-05T10:00:00Z"
}
```

---

### PUT /api/notifications/read-all

모든 알림을 읽음 처리합니다.

**Response** (200 OK)
```json
{
  "message": "All notifications marked as read",
  "count": 15
}
```

---

### GET /api/notifications/unread-count

읽지 않은 알림 개수를 조회합니다.

**Response** (200 OK)
```json
{
  "count": 5
}
```

---

## SSE Stream

### 연결

```
GET /api/notifications/stream?token={jwt}
```

또는

```
GET /api/notifications/stream
Authorization: Bearer {jwt}
```

### 이벤트 형식

```
event: notification
data: {"id":"uuid","type":"task_assigned","title":"New task","message":"..."}

event: heartbeat
data: {"timestamp":"2026-01-05T10:00:00Z"}
```

### 이벤트 타입

| Event | Description |
|-------|-------------|
| `notification` | 새 알림 |
| `heartbeat` | 연결 유지 (30초마다) |

---

## Notification Types

| Type | Trigger | Description |
|------|---------|-------------|
| `task_assigned` | 태스크 할당 | 태스크가 나에게 할당됨 |
| `task_due_soon` | 마감 임박 | 태스크 마감일 임박 |
| `task_completed` | 태스크 완료 | 내 태스크가 완료됨 |
| `comment_added` | 댓글 추가 | 내 태스크에 댓글 |
| `mention` | 멘션 | 댓글에서 멘션됨 |
| `chat_message` | 새 메시지 | 채팅 메시지 도착 |
| `workspace_invited` | 워크스페이스 초대 | 워크스페이스에 초대됨 |
| `project_invited` | 프로젝트 초대 | 프로젝트에 초대됨 |

---

## Data Models

### Notification

| Field | Type | Description |
|-------|------|-------------|
| id | UUID | 알림 ID |
| userId | UUID | 수신자 ID |
| type | enum | 알림 타입 |
| title | string | 제목 |
| message | string | 내용 |
| data | object | 추가 데이터 |
| isRead | bool | 읽음 여부 |
| readAt | datetime | 읽은 시간 |
| createdAt | datetime | 생성일 |

---

## Internal API

다른 서비스에서 알림을 생성할 때 사용합니다.

### POST /internal/notifications

**Headers**
```
X-Internal-API-Key: {internal_api_key}
```

**Request**
```json
{
  "userId": "user-uuid",
  "type": "task_assigned",
  "title": "New task assigned",
  "message": "You have been assigned to 'Implement feature'",
  "data": {
    "taskId": "task-uuid",
    "projectId": "project-uuid"
  }
}
```

---

## 관련 문서

- [API 개요](./README.md)
- [Storage Service API](./storage-service.md)
