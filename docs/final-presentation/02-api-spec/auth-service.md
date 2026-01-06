# Auth Service API

> **기술**: Spring Boot 3.4
>
> **포트**: 8080
>
> **역할**: JWT 토큰 발급/검증, OAuth2 인증

---

## Endpoints

### Authentication

| Method | Endpoint | Auth | Description |
|--------|----------|:----:|-------------|
| POST | `/api/auth/login` | - | 이메일/비밀번호 로그인 |
| POST | `/api/auth/refresh` | - | 토큰 갱신 |
| POST | `/api/auth/logout` | ✓ | 로그아웃 (토큰 무효화) |
| GET | `/api/auth/me` | ✓ | 현재 사용자 정보 |

### OAuth2

| Method | Endpoint | Auth | Description |
|--------|----------|:----:|-------------|
| GET | `/oauth2/authorization/google` | - | Google 로그인 시작 |
| GET | `/api/oauth2/callback/google` | - | Google 콜백 |

### JWKS (공개 키)

| Method | Endpoint | Auth | Description |
|--------|----------|:----:|-------------|
| GET | `/.well-known/jwks.json` | - | JWT 검증용 공개 키 |

---

## Request/Response

### POST /api/auth/login

로그인하여 JWT 토큰을 발급받습니다.

**Request**
```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

**Response** (200 OK)
```json
{
  "accessToken": "eyJhbGciOiJSUzI1NiIs...",
  "refreshToken": "eyJhbGciOiJSUzI1NiIs...",
  "tokenType": "Bearer",
  "expiresIn": 900
}
```

**Errors**
| Code | Message |
|------|---------|
| 401 | Invalid credentials |

---

### POST /api/auth/refresh

Refresh Token으로 새 Access Token을 발급받습니다.

**Request**
```json
{
  "refreshToken": "eyJhbGciOiJSUzI1NiIs..."
}
```

**Response** (200 OK)
```json
{
  "accessToken": "eyJhbGciOiJSUzI1NiIs...",
  "refreshToken": "eyJhbGciOiJSUzI1NiIs...",
  "tokenType": "Bearer",
  "expiresIn": 900
}
```

**Errors**
| Code | Message |
|------|---------|
| 401 | Invalid or expired refresh token |

---

### POST /api/auth/logout

현재 토큰을 무효화합니다.

**Headers**
```
Authorization: Bearer <access_token>
```

**Response** (200 OK)
```json
{
  "message": "Logged out successfully"
}
```

---

### GET /api/auth/me

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
  "name": "John Doe"
}
```

---

## OAuth2 Flow

### Google 로그인

1. 클라이언트가 `/oauth2/authorization/google`로 리다이렉트
2. 사용자가 Google에서 인증
3. Google이 `/api/oauth2/callback/google`으로 콜백
4. auth-service가 JWT 토큰 발급
5. 클라이언트로 리다이렉트 (토큰 포함)

```
Client → /oauth2/authorization/google
         ↓
      Google Auth
         ↓
/api/oauth2/callback/google?code=xxx
         ↓
    JWT 발급
         ↓
Client → /oauth/callback?token=xxx
```

---

## JWT 구조

### Access Token Payload

```json
{
  "sub": "user-uuid",
  "email": "user@example.com",
  "name": "John Doe",
  "iat": 1704412800,
  "exp": 1704413700
}
```

### Claims

| Claim | Type | Description |
|-------|------|-------------|
| sub | string | 사용자 UUID |
| email | string | 이메일 |
| name | string | 이름 |
| iat | number | 발급 시간 (Unix timestamp) |
| exp | number | 만료 시간 (Unix timestamp) |

---

## JWKS Endpoint

다른 서비스에서 JWT를 검증할 때 사용하는 공개 키입니다.

**GET /.well-known/jwks.json**

```json
{
  "keys": [
    {
      "kty": "RSA",
      "kid": "wealist-key-1",
      "use": "sig",
      "alg": "RS256",
      "n": "...",
      "e": "AQAB"
    }
  ]
}
```

---

## 관련 문서

- [API 개요](./README.md)
- [User Service API](./user-service.md)
