# syntax=docker/dockerfile:1.4
# 1. Builder Stage
FROM golang:1.16 AS builder

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

# [수정됨] 여기가 핵심입니다. 데비안이니까 apt-get을 씁니다.
RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates tzdata wget && \
    rm -rf /var/lib/apt/lists/*

# 사용자 생성 (데비안 문법)
RUN groupadd -g 1000 appuser && \
    useradd -u 1000 -g appuser -m -s /bin/bash appuser

WORKDIR /home/appuser
COPY --from=builder --chown=appuser:appuser /workspace/services/board-service/board-api .
COPY --from=builder --chown=appuser:appuser /workspace/services/board-service/configs ./configs

USER appuser
EXPOSE 8000
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 CMD wget --no-verbose --tries=1 --spider http://localhost:8000/health/live || exit 1
CMD ["./board-api"]