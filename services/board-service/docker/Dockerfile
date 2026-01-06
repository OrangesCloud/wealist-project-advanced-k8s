# syntax=docker/dockerfile:1.4
# 1. Builder Stage
FROM golang:1.23-bookworm AS builder

ARG TARGETARCH

# 기본 패키지 설치
RUN apt-get update && apt-get install -y --no-install-recommends git ca-certificates tzdata && rm -rf /var/lib/apt/lists/*

WORKDIR /workspace

# 패키지 먼저 복사
COPY packages/wealist-advanced-go-pkg/ ./packages/wealist-advanced-go-pkg/

WORKDIR /workspace/services/board-service

# go.mod, go.sum 복사
COPY services/board-service/go.mod services/board-service/go.sum ./

# [핵심 수정 1] 줄바꿈 강제 추가 후 replace 적용 (파일 깨짐 방지)
RUN echo "" >> go.mod && \
    echo 'replace github.com/OrangesCloud/wealist-advanced-go-pkg => ../../packages/wealist-advanced-go-pkg' >> go.mod

# [핵심 수정 2] 디버깅을 위해 go.mod 내용 확인 (빌드 로그에서 확인 가능)
RUN cat go.mod

# 의존성 다운로드 (GOPROXY 추가)
RUN --mount=type=cache,target=/go/pkg/mod \
    GOWORK=off go mod download

# 소스 복사 및 빌드
COPY services/board-service/ ./
# 소스 복사 후 go.mod가 덮어씌워지므로 replace 다시 적용 (줄바꿈 포함)
RUN echo "" >> go.mod && \
    echo 'replace github.com/OrangesCloud/wealist-advanced-go-pkg => ../../packages/wealist-advanced-go-pkg' >> go.mod

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOWORK=off go mod tidy && \
    CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} GOWORK=off go build -ldflags="-w -s" -o board-api ./cmd/api

# -----------------------------------------------------------
# 2. Runtime Stage: 취약한 Debian 구버전 (Trivy용)
# -----------------------------------------------------------
FROM debian:buster-slim

RUN echo "deb http://archive.debian.org/debian buster main" > /etc/apt/sources.list && \
    echo "deb http://archive.debian.org/debian-security buster/updates main" >> /etc/apt/sources.list && \
    apt-get -o Acquire::Check-Valid-Until=false update && \
    apt-get -o Acquire::Check-Valid-Until=false install -y --no-install-recommends ca-certificates tzdata wget && \
    rm -rf /var/lib/apt/lists/*

RUN groupadd -g 1000 appuser && \
    useradd -u 1000 -g appuser -m -s /bin/bash appuser

WORKDIR /home/appuser

COPY --from=builder --chown=appuser:appuser /workspace/services/board-service/board-api .
COPY --from=builder --chown=appuser:appuser /workspace/services/board-service/configs ./configs

USER appuser
EXPOSE 8000
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8000/health/live || exit 1

CMD ["./board-api"]