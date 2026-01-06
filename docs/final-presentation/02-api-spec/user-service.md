# User Service API

> **기술**: Go + Gin
>
> **포트**: 8081
>
> **역할**: 사용자 관리, 워크스페이스 관리

---

## Endpoints

### Users

| Method | Endpoint | Auth | Description |
|--------|----------|:----:|-------------|
| POST | `/api/users` | - | 사용자 생성 |
| GET | `/api/users/me` | ✓ | 현재 사용자 조회 |
| GET | `/api/users/{userId}` | ✓ | ID로 사용자 조회 |
| PUT | `/api/users/{userId}` | ✓ | 사용자 정보 수정 |
| DELETE | `/api/users/me` | ✓ | 계정 삭제 (soft) |

### Profiles

| Method | Endpoint | Auth | Description |
|--------|----------|:----:|-------------|
| GET | `/api/profiles/me` | ✓ | 내 프로필 조회 |
| PUT | `/api/profiles/me` | ✓ | 프로필 수정 |
| POST | `/api/profiles/me/avatar` | ✓ | 프로필 이미지 업로드 |

### Workspaces

| Method | Endpoint | Auth | Description |
|--------|----------|:----:|-------------|
| GET | `/api/workspaces` | ✓ | 내 워크스페이스 목록 |
| POST | `/api/workspaces` | ✓ | 워크스페이스 생성 |
| GET | `/api/workspaces/{id}` | ✓ | 워크스페이스 상세 |
| PUT | `/api/workspaces/{id}` | ✓ | 워크스페이스 수정 |
| DELETE | `/api/workspaces/{id}` | ✓ | 워크스페이스 삭제 |
| GET | `/api/workspaces/{id}/members` | ✓ | 멤버 목록 |
| POST | `/api/workspaces/{id}/members` | ✓ | 멤버 초대 |
| DELETE | `/api/workspaces/{id}/members/{userId}` | ✓ | 멤버 제거 |

---

## Request/Response

### POST /api/users

새 사용자를 생성합니다.

**Request**
```json
{
  "email": "user@example.com",
  "name": "John Doe",
  "password": "password123"
}
```

**Response** (201 Created)
```json
{
  "id": "uuid",
  "email": "user@example.com",
  "name": "John Doe",
  "createdAt": "2026-01-05T10:00:00Z"
}
```

**Errors**
| Code | Message |
|------|---------|
| 400 | Validation error |
| 409 | Email already exists |

---

### GET /api/users/me

현재 인증된 사용자 정보를 조회합니다.

**Headers**
```
Authorization: Bearer <access_token>
```

**Response** (200 OK)
```json
{
  "id": "uuid",
  "email": "user@example.com",
  "name": "John Doe",
  "profileImage": "https://cdn.wealist.co.kr/...",
  "createdAt": "2026-01-05T10:00:00Z",
  "updatedAt": "2026-01-05T10:00:00Z"
}
```

---

### POST /api/workspaces

새 워크스페이스를 생성합니다.

**Request**
```json
{
  "name": "My Workspace",
  "description": "Team workspace for project"
}
```

**Response** (201 Created)
```json
{
  "id": "uuid",
  "name": "My Workspace",
  "description": "Team workspace for project",
  "ownerId": "user-uuid",
  "createdAt": "2026-01-05T10:00:00Z"
}
```

---

### GET /api/workspaces/{id}/members

워크스페이스 멤버 목록을 조회합니다.

**Response** (200 OK)
```json
{
  "members": [
    {
      "id": "uuid",
      "userId": "user-uuid",
      "name": "John Doe",
      "email": "john@example.com",
      "role": "owner",
      "joinedAt": "2026-01-05T10:00:00Z"
    },
    {
      "id": "uuid",
      "userId": "user-uuid",
      "name": "Jane Doe",
      "email": "jane@example.com",
      "role": "member",
      "joinedAt": "2026-01-05T11:00:00Z"
    }
  ]
}
```

---

### POST /api/workspaces/{id}/members

워크스페이스에 멤버를 초대합니다.

**Request**
```json
{
  "email": "newmember@example.com",
  "role": "member"
}
```

**Response** (201 Created)
```json
{
  "id": "uuid",
  "userId": "user-uuid",
  "role": "member",
  "joinedAt": "2026-01-05T12:00:00Z"
}
```

**Errors**
| Code | Message |
|------|---------|
| 403 | Not workspace owner |
| 404 | User not found |
| 409 | Already a member |

---

## Data Models

### User

| Field | Type | Description |
|-------|------|-------------|
| id | UUID | 사용자 ID |
| email | string | 이메일 (unique) |
| name | string | 이름 |
| profileImage | string | 프로필 이미지 URL |
| createdAt | datetime | 생성일 |
| updatedAt | datetime | 수정일 |

### Workspace

| Field | Type | Description |
|-------|------|-------------|
| id | UUID | 워크스페이스 ID |
| name | string | 이름 |
| description | string | 설명 |
| ownerId | UUID | 소유자 ID |
| createdAt | datetime | 생성일 |
| updatedAt | datetime | 수정일 |

### WorkspaceMember

| Field | Type | Description |
|-------|------|-------------|
| id | UUID | 멤버십 ID |
| workspaceId | UUID | 워크스페이스 ID |
| userId | UUID | 사용자 ID |
| role | enum | owner, admin, member |
| joinedAt | datetime | 가입일 |

---

## 관련 문서

- [API 개요](./README.md)
- [Auth Service API](./auth-service.md)
