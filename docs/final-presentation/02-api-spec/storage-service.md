# Storage Service API

> **기술**: Go + Gin
>
> **포트**: 8003
>
> **역할**: 파일 저장소 관리 (Google Drive 스타일)

---

## Endpoints

### Files

| Method | Endpoint | Auth | Description |
|--------|----------|:----:|-------------|
| GET | `/api/storage/files` | ✓ | 파일 목록 |
| POST | `/api/storage/files` | ✓ | 파일 업로드 |
| GET | `/api/storage/files/{id}` | ✓ | 파일 정보 |
| DELETE | `/api/storage/files/{id}` | ✓ | 파일 삭제 |
| GET | `/api/storage/files/{id}/download` | ✓ | 파일 다운로드 |
| PUT | `/api/storage/files/{id}/move` | ✓ | 파일 이동 |

### Folders

| Method | Endpoint | Auth | Description |
|--------|----------|:----:|-------------|
| GET | `/api/storage/folders` | ✓ | 폴더 목록 |
| POST | `/api/storage/folders` | ✓ | 폴더 생성 |
| GET | `/api/storage/folders/{id}` | ✓ | 폴더 정보 |
| PUT | `/api/storage/folders/{id}` | ✓ | 폴더 수정 |
| DELETE | `/api/storage/folders/{id}` | ✓ | 폴더 삭제 |
| GET | `/api/storage/folders/{id}/contents` | ✓ | 폴더 내용 |

### Sharing

| Method | Endpoint | Auth | Description |
|--------|----------|:----:|-------------|
| POST | `/api/storage/files/{id}/share` | ✓ | 공유 링크 생성 |
| DELETE | `/api/storage/files/{id}/share` | ✓ | 공유 링크 삭제 |

---

## Request/Response

### POST /api/storage/files

파일을 업로드합니다.

**Content-Type**: `multipart/form-data`

**Form Fields**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| file | file | ✓ | 업로드할 파일 |
| folderId | string | - | 대상 폴더 ID |
| workspaceId | string | ✓ | 워크스페이스 ID |

**Response** (201 Created)
```json
{
  "id": "uuid",
  "name": "document.pdf",
  "mimeType": "application/pdf",
  "size": 1024000,
  "folderId": "folder-uuid",
  "workspaceId": "workspace-uuid",
  "uploadedBy": "user-uuid",
  "url": "https://cdn.wealist.co.kr/files/...",
  "createdAt": "2026-01-05T10:00:00Z"
}
```

**Errors**
| Code | Message |
|------|---------|
| 400 | File too large (max 50MB) |
| 400 | Invalid file type |
| 403 | Not workspace member |

---

### GET /api/storage/files

파일 목록을 조회합니다.

**Query Parameters**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| workspaceId | string | - | 워크스페이스 ID (필수) |
| folderId | string | - | 폴더 ID |
| search | string | - | 검색어 |
| page | int | 1 | 페이지 번호 |
| limit | int | 50 | 페이지 크기 |

**Response** (200 OK)
```json
{
  "files": [
    {
      "id": "uuid",
      "name": "document.pdf",
      "mimeType": "application/pdf",
      "size": 1024000,
      "folderId": null,
      "url": "https://cdn.wealist.co.kr/files/...",
      "thumbnailUrl": "https://cdn.wealist.co.kr/thumbnails/...",
      "uploadedBy": "user-uuid",
      "createdAt": "2026-01-05T10:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 50,
    "total": 100
  }
}
```

---

### GET /api/storage/files/{id}/download

파일 다운로드 URL을 반환합니다 (Presigned URL).

**Response** (200 OK)
```json
{
  "downloadUrl": "https://s3.amazonaws.com/...",
  "expiresIn": 3600
}
```

---

### POST /api/storage/folders

새 폴더를 생성합니다.

**Request**
```json
{
  "name": "Documents",
  "parentId": null,
  "workspaceId": "workspace-uuid"
}
```

**Response** (201 Created)
```json
{
  "id": "uuid",
  "name": "Documents",
  "parentId": null,
  "workspaceId": "workspace-uuid",
  "createdBy": "user-uuid",
  "createdAt": "2026-01-05T10:00:00Z"
}
```

---

### GET /api/storage/folders/{id}/contents

폴더 내용(파일 + 하위 폴더)을 조회합니다.

**Response** (200 OK)
```json
{
  "folder": {
    "id": "uuid",
    "name": "Documents",
    "parentId": null
  },
  "contents": {
    "folders": [
      {
        "id": "uuid",
        "name": "Subfolder",
        "createdAt": "2026-01-05T10:00:00Z"
      }
    ],
    "files": [
      {
        "id": "uuid",
        "name": "file.pdf",
        "size": 1024000,
        "mimeType": "application/pdf"
      }
    ]
  },
  "path": [
    {"id": "root", "name": "Root"},
    {"id": "uuid", "name": "Documents"}
  ]
}
```

---

### POST /api/storage/files/{id}/share

공유 링크를 생성합니다.

**Request**
```json
{
  "expiresIn": 86400,
  "password": "optional-password"
}
```

**Response** (201 Created)
```json
{
  "shareLink": "https://wealist.co.kr/share/abc123",
  "expiresAt": "2026-01-06T10:00:00Z",
  "hasPassword": true
}
```

---

## Data Models

### File

| Field | Type | Description |
|-------|------|-------------|
| id | UUID | 파일 ID |
| name | string | 파일명 |
| mimeType | string | MIME 타입 |
| size | int64 | 파일 크기 (bytes) |
| folderId | UUID | 폴더 ID |
| workspaceId | UUID | 워크스페이스 ID |
| uploadedBy | UUID | 업로더 ID |
| s3Key | string | S3 키 |
| url | string | CDN URL |
| thumbnailUrl | string | 썸네일 URL |
| createdAt | datetime | 업로드일 |

### Folder

| Field | Type | Description |
|-------|------|-------------|
| id | UUID | 폴더 ID |
| name | string | 폴더명 |
| parentId | UUID | 상위 폴더 ID |
| workspaceId | UUID | 워크스페이스 ID |
| createdBy | UUID | 생성자 ID |
| createdAt | datetime | 생성일 |

---

## 파일 제한

| 항목 | 값 |
|------|---|
| 최대 파일 크기 | 50MB |
| 허용 파일 타입 | 이미지, PDF, 문서, 스프레드시트, 아카이브 |
| 워크스페이스당 용량 | 5GB (Free), 50GB (Pro) |

---

## S3 연동

### Production
- **Bucket**: `wealist-prod-files-{account-id}`
- **CDN**: CloudFront (`cdn.wealist.co.kr`)
- **인증**: IRSA (IAM Role for Service Account)

### Development
- **Storage**: MinIO (S3 호환)
- **Endpoint**: `http://minio:9000`

---

## 관련 문서

- [API 개요](./README.md)
- [Ops Service API](./ops-service.md)
