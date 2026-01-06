# Board Service API

> **기술**: Go + Gin
>
> **포트**: 8000
>
> **역할**: 프로젝트, 보드, 태스크, 댓글 관리

---

## Endpoints

### Projects

| Method | Endpoint | Auth | Description |
|--------|----------|:----:|-------------|
| GET | `/api/projects` | ✓ | 프로젝트 목록 |
| POST | `/api/projects` | ✓ | 프로젝트 생성 |
| GET | `/api/projects/{id}` | ✓ | 프로젝트 상세 |
| PUT | `/api/projects/{id}` | ✓ | 프로젝트 수정 |
| DELETE | `/api/projects/{id}` | ✓ | 프로젝트 삭제 |
| GET | `/api/projects/{id}/members` | ✓ | 프로젝트 멤버 |

### Boards

| Method | Endpoint | Auth | Description |
|--------|----------|:----:|-------------|
| GET | `/api/projects/{projectId}/boards` | ✓ | 보드 목록 |
| POST | `/api/projects/{projectId}/boards` | ✓ | 보드 생성 |
| GET | `/api/boards/{id}` | ✓ | 보드 상세 |
| PUT | `/api/boards/{id}` | ✓ | 보드 수정 |
| DELETE | `/api/boards/{id}` | ✓ | 보드 삭제 |
| PUT | `/api/boards/{id}/reorder` | ✓ | 보드 순서 변경 |

### Tasks

| Method | Endpoint | Auth | Description |
|--------|----------|:----:|-------------|
| GET | `/api/boards/{boardId}/tasks` | ✓ | 태스크 목록 |
| POST | `/api/boards/{boardId}/tasks` | ✓ | 태스크 생성 |
| GET | `/api/tasks/{id}` | ✓ | 태스크 상세 |
| PUT | `/api/tasks/{id}` | ✓ | 태스크 수정 |
| DELETE | `/api/tasks/{id}` | ✓ | 태스크 삭제 |
| PUT | `/api/tasks/{id}/move` | ✓ | 태스크 이동 |
| PUT | `/api/tasks/{id}/assign` | ✓ | 담당자 지정 |

### Comments

| Method | Endpoint | Auth | Description |
|--------|----------|:----:|-------------|
| GET | `/api/tasks/{taskId}/comments` | ✓ | 댓글 목록 |
| POST | `/api/tasks/{taskId}/comments` | ✓ | 댓글 작성 |
| PUT | `/api/comments/{id}` | ✓ | 댓글 수정 |
| DELETE | `/api/comments/{id}` | ✓ | 댓글 삭제 |

### WebSocket

| Endpoint | Auth | Description |
|----------|:----:|-------------|
| `/ws/project/{projectId}` | ✓ | 실시간 보드 업데이트 |

---

## Request/Response

### POST /api/projects

새 프로젝트를 생성합니다.

**Request**
```json
{
  "name": "New Project",
  "description": "Project description",
  "workspaceId": "workspace-uuid"
}
```

**Response** (201 Created)
```json
{
  "id": "uuid",
  "name": "New Project",
  "description": "Project description",
  "workspaceId": "workspace-uuid",
  "ownerId": "user-uuid",
  "createdAt": "2026-01-05T10:00:00Z"
}
```

---

### POST /api/projects/{projectId}/boards

새 보드를 생성합니다.

**Request**
```json
{
  "name": "To Do",
  "color": "#3B82F6",
  "position": 0
}
```

**Response** (201 Created)
```json
{
  "id": "uuid",
  "projectId": "project-uuid",
  "name": "To Do",
  "color": "#3B82F6",
  "position": 0,
  "createdAt": "2026-01-05T10:00:00Z"
}
```

---

### POST /api/boards/{boardId}/tasks

새 태스크를 생성합니다.

**Request**
```json
{
  "title": "Implement feature",
  "description": "Detailed description",
  "priority": "high",
  "dueDate": "2026-01-10T00:00:00Z",
  "assigneeId": "user-uuid"
}
```

**Response** (201 Created)
```json
{
  "id": "uuid",
  "boardId": "board-uuid",
  "title": "Implement feature",
  "description": "Detailed description",
  "priority": "high",
  "status": "todo",
  "position": 0,
  "dueDate": "2026-01-10T00:00:00Z",
  "assigneeId": "user-uuid",
  "createdBy": "user-uuid",
  "createdAt": "2026-01-05T10:00:00Z"
}
```

---

### PUT /api/tasks/{id}/move

태스크를 다른 보드로 이동합니다.

**Request**
```json
{
  "targetBoardId": "board-uuid",
  "position": 0
}
```

**Response** (200 OK)
```json
{
  "id": "uuid",
  "boardId": "board-uuid",
  "position": 0
}
```

---

## WebSocket Events

### 연결

```
ws://localhost:8080/api/svc/board/ws/project/{projectId}?token={jwt}
```

### 이벤트 타입

| Event | Direction | Description |
|-------|-----------|-------------|
| `task.created` | Server → Client | 태스크 생성됨 |
| `task.updated` | Server → Client | 태스크 수정됨 |
| `task.deleted` | Server → Client | 태스크 삭제됨 |
| `task.moved` | Server → Client | 태스크 이동됨 |
| `board.updated` | Server → Client | 보드 수정됨 |

### 메시지 형식

```json
{
  "type": "task.created",
  "payload": {
    "id": "uuid",
    "boardId": "board-uuid",
    "title": "New Task",
    ...
  },
  "timestamp": "2026-01-05T10:00:00Z"
}
```

---

## Data Models

### Project

| Field | Type | Description |
|-------|------|-------------|
| id | UUID | 프로젝트 ID |
| name | string | 이름 |
| description | string | 설명 |
| workspaceId | UUID | 워크스페이스 ID |
| ownerId | UUID | 소유자 ID |
| createdAt | datetime | 생성일 |

### Board

| Field | Type | Description |
|-------|------|-------------|
| id | UUID | 보드 ID |
| projectId | UUID | 프로젝트 ID |
| name | string | 이름 |
| color | string | 색상 (hex) |
| position | int | 순서 |

### Task

| Field | Type | Description |
|-------|------|-------------|
| id | UUID | 태스크 ID |
| boardId | UUID | 보드 ID |
| title | string | 제목 |
| description | string | 설명 |
| priority | enum | low, medium, high |
| status | enum | todo, in_progress, done |
| position | int | 순서 |
| dueDate | datetime | 마감일 |
| assigneeId | UUID | 담당자 ID |
| createdBy | UUID | 생성자 ID |

### Comment

| Field | Type | Description |
|-------|------|-------------|
| id | UUID | 댓글 ID |
| taskId | UUID | 태스크 ID |
| content | string | 내용 |
| authorId | UUID | 작성자 ID |
| createdAt | datetime | 작성일 |

---

## 관련 문서

- [API 개요](./README.md)
- [Chat Service API](./chat-service.md)
