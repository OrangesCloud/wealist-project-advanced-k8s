# Requirements

weAlist 요구사항 정의서입니다.

---

## Full Document

> 상세 요구사항 정의서: [Google Docs](https://docs.google.com/document/d/1Cmc4fSrtqnJRTxgARCCyQGNgOiVJ-vvIkmSqktE_hx8)

---

## Functional Requirements

### FR-001: 사용자 인증
- OAuth2 (Google) 로그인
- JWT 토큰 기반 인증
- 자동 로그인 (Refresh Token)

### FR-002: 워크스페이스 관리
- 워크스페이스 CRUD
- 멤버 초대/권한 관리
- 역할 기반 접근 제어

### FR-003: 프로젝트/보드
- 프로젝트 생성/관리
- 칸반 보드
- 태스크 생성/할당
- 댓글/첨부파일

### FR-004: 실시간 채팅
- 1:1/그룹 채팅
- 실시간 메시지 (WebSocket)
- 파일 공유

### FR-005: 알림
- 실시간 알림 (SSE)
- 멘션 알림
- 이메일 알림

### FR-006: 파일 스토리지
- 파일 업로드/다운로드
- 폴더 관리
- 공유 링크

### FR-007: 영상통화
- 1:1/그룹 통화
- 화면 공유
- 녹화

---

## Non-Functional Requirements

### NFR-001: 성능
- API 응답 시간 < 200ms (p95)
- WebSocket 연결 < 1s
- 동시 접속자 1000명 지원

### NFR-002: 가용성
- SLA 99.9%
- Multi-AZ 배포
- 자동 복구

### NFR-003: 보안
- TLS 암호화
- JWT 토큰 만료 관리
- SQL Injection 방지

### NFR-004: 확장성
- 수평 확장 (HPA)
- 마이크로서비스 독립 배포
- 데이터베이스 샤딩 준비

---

## Related Pages

- [Cloud Proposal](Cloud-Proposal.md)
- [Architecture Overview](Architecture.md)
