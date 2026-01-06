# Chat Service API

> **기술**: Go + Gin
>
> **포트**: 8001
>
> **역할**: 실시간 채팅, 메시지 관리

---

## Endpoints

### Chats

| Method | Endpoint | Auth | Description |
|--------|----------|:----:|-------------|
| GET | `/api/chats` | ✓ | 내 채팅방 목록 |
| POST | `/api/chats` | ✓ | 채팅방 생성 |
| GET | `/api/chats/{id}` | ✓ | 채팅방 상세 |
| PUT | `/api/chats/{id}` | ✓ | 채팅방 수정 |
| DELETE | `/api/chats/{id}` | ✓ | 채팅방 삭제 |
| GET | `/api/chats/{id}/participants` | ✓ | 참가자 목록 |
| POST | `/api/chats/{id}/participants` | ✓ | 참가자 추가 |
| DELETE | `/api/chats/{id}/participants/{userId}` | ✓ | 참가자 제거 |

### Messages

| Method | Endpoint | Auth | Description |
|--------|----------|:----:|-------------|
| GET | `/api/chats/{chatId}/messages` | ✓ | 메시지 목록 |
| POST | `/api/chats/{chatId}/messages` | ✓ | 메시지 전송 |
| PUT | `/api/messages/{id}` | ✓ | 메시지 수정 |
| DELETE | `/api/messages/{id}` | ✓ | 메시지 삭제 |

### WebSocket

| Endpoint | Auth | Description |
|----------|:----:|-------------|
| `/api/chats/ws/{chatId}` | ✓ | 실시간 채팅 |
| `/api/chats/ws/presence` | ✓ | 온라인 상태 |

---

## Request/Response

### POST /api/chats

새 채팅방을 생성합니다.

**Request**
```json
{
  "name": "Team Chat",
  "type": "group",
  "participantIds": ["user-uuid-1", "user-uuid-2"]
}
```

**Response** (201 Created)
```json
{
  "id": "uuid",
  "name": "Team Chat",
  "type": "group",
  "creatorId": "user-uuid",
  "participants": [
    {
      "userId": "user-uuid-1",
      "joinedAt": "2026-01-05T10:00:00Z"
    }
  ],
  "createdAt": "2026-01-05T10:00:00Z"
}
```

---

### GET /api/chats/{chatId}/messages

채팅방의 메시지 목록을 조회합니다 (페이지네이션).

**Query Parameters**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| limit | int | 50 | 조회 개수 |
| before | string | - | 이전 메시지 커서 |
| after | string | - | 이후 메시지 커서 |

**Response** (200 OK)
```json
{
  "messages": [
    {
      "id": "uuid",
      "chatId": "chat-uuid",
      "senderId": "user-uuid",
      "senderName": "John Doe",
      "content": "Hello!",
      "type": "text",
      "createdAt": "2026-01-05T10:00:00Z"
    }
  ],
  "hasMore": true,
  "nextCursor": "cursor-string"
}
```

---

### POST /api/chats/{chatId}/messages

메시지를 전송합니다.

**Request**
```json
{
  "content": "Hello, team!",
  "type": "text"
}
```

**Response** (201 Created)
```json
{
  "id": "uuid",
  "chatId": "chat-uuid",
  "senderId": "user-uuid",
  "content": "Hello, team!",
  "type": "text",
  "createdAt": "2026-01-05T10:00:00Z"
}
```

---

## WebSocket

### 채팅 연결

```
ws://localhost:8080/api/svc/chat/api/chats/ws/{chatId}?token={jwt}
```

### 이벤트 타입

| Event | Direction | Description |
|-------|-----------|-------------|
| `message` | Bidirectional | 메시지 전송/수신 |
| `typing` | Client → Server | 타이핑 중 |
| `read` | Client → Server | 메시지 읽음 |

### 메시지 전송

```json
{
  "type": "message",
  "payload": {
    "content": "Hello!",
    "messageType": "text"
  }
}
```

### 메시지 수신

```json
{
  "type": "message",
  "payload": {
    "id": "uuid",
    "senderId": "user-uuid",
    "senderName": "John",
    "content": "Hello!",
    "createdAt": "2026-01-05T10:00:00Z"
  }
}
```

---

### 온라인 상태 연결

```
ws://localhost:8080/api/svc/chat/api/chats/ws/presence?token={jwt}
```

### 상태 이벤트

```json
{
  "type": "presence",
  "payload": {
    "userId": "user-uuid",
    "status": "online",
    "lastSeen": "2026-01-05T10:00:00Z"
  }
}
```

---

## Data Models

### Chat

| Field | Type | Description |
|-------|------|-------------|
| id | UUID | 채팅방 ID |
| name | string | 채팅방 이름 |
| type | enum | direct, group |
| creatorId | UUID | 생성자 ID |
| createdAt | datetime | 생성일 |

### ChatParticipant

| Field | Type | Description |
|-------|------|-------------|
| chatId | UUID | 채팅방 ID |
| userId | UUID | 사용자 ID |
| joinedAt | datetime | 참가일 |
| lastReadAt | datetime | 마지막 읽은 시간 |

### Message

| Field | Type | Description |
|-------|------|-------------|
| id | UUID | 메시지 ID |
| chatId | UUID | 채팅방 ID |
| senderId | UUID | 발신자 ID |
| content | string | 내용 |
| type | enum | text, image, file |
| createdAt | datetime | 작성일 |
| updatedAt | datetime | 수정일 |

---

## 관련 문서

- [API 개요](./README.md)
- [Noti Service API](./noti-service.md)
