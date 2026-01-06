# syntax=docker/dockerfile:1.4
# 1. Builder Stage
FROM golang:1.23-bookworm AS builder

ARG TARGETARCH

# 데비안용 설치 명령어 (apt-get)
RUN apt-get update && apt-get install -y --no-install-recommends git ca-certificates tzdata && rm -rf /var/lib/apt/lists/*

WORKDIR /workspace
COPY packages/wealist-advanced-go-pkg/ ./packages/wealist-advanced-go-pkg/
WORKDIR /workspace/services/board-service
COPY services/board-service/go.mod services/board-service/go.sum ./
RUN echo 'replace github.com/OrangesCloud/wealist-advanced-go-pkg => ../../packages/wealist-advanced-go-pkg' >> go.mod
RUN --mount=type=cache,target=/go/pkg/mod GOWORK=off go mod download -x
COPY services/board-service/ ./
RUN echo 'replace github.com/OrangesCloud/wealist-advanced-go-pkg => ../../packages/wealist-advanced-go-pkg' >> go.mod
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build GOWORK=off go mod tidy && \
    CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} GOWORK=off go build -ldflags="-w -s" -o board-api ./cmd/api

# 2. Runtime Stage
FROM debian:buster-slim

# [수정됨] EOL된 Debian 10을 위해 저장소 주소를 아카이브로 변경하고 설치
RUN echo "deb http://archive.debian.org/debian buster main" > /etc/apt/sources.list && \
    echo "deb http://archive.debian.org/debian-security buster/updates main" >> /etc/apt/sources.list && \
    apt-get -o Acquire::Check-Valid-Until=false update && \
    apt-get -o Acquire::Check-Valid-Until=false install -y --no-install-recommends ca-certificates tzdata wget && \
    rm -rf /var/lib/apt/lists/*

# 사용자 생성
RUN groupadd -g 1000 appuser && \
    useradd -u 1000 -g appuser -m -s /bin/bash appuser

WORKDIR /home/appuser

# 빌드 결과물 복사
COPY --from=builder --chown=appuser:appuser /workspace/services/board-service/board-api .
COPY --from=builder --chown=appuser:appuser /workspace/services/board-service/configs ./configs

USER appuser

EXPOSE 8000

# 헬스 체크
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8000/health/live || exit 1

CMD ["./board-api"]