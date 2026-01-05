# Frontend Changelog

All notable changes to the frontend will be documented in this file.

## [1.0.5] - 2026-01-02

### Fixed
- K8s ingress 환경에서 JWT 토큰 갱신 실패 문제 수정
  - `/api/svc/auth`가 `/`로 rewrite되는 환경에서 auth-service 경로가 잘못 설정됨
  - `refreshAccessToken()`: `/refresh` → `/api/auth/refresh`
  - `logout()`: `/logout` → `/api/auth/logout`
- 토큰 만료 시 무한 로그인 페이지 리다이렉트 문제 해결

## [1.0.4] - 2026-01-01

### Fixed
- WebSocket 재연결 시 토큰 갱신 로직 추가

## [1.0.3] - 2025-12-31

### Added
- Initial production release
