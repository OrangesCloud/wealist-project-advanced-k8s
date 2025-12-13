# syntax=docker/dockerfile:1.4
# Builder stage
FROM golang:1.24-bookworm AS builder

# Install build dependencies
RUN apt-get update && apt-get install -y --no-install-recommends git ca-certificates tzdata && rm -rf /var/lib/apt/lists/*

# Set working directory to project root
WORKDIR /workspace

# Copy shared package first
COPY packages/wealist-advanced-go-pkg/ ./packages/wealist-advanced-go-pkg/

# Copy service files
WORKDIR /workspace/services/noti-service

# Copy source code first (needed for go mod tidy to work)
COPY services/noti-service/ ./

# Add replace directive for local package
RUN echo 'replace github.com/OrangesCloud/wealist-advanced-go-pkg => ../../packages/wealist-advanced-go-pkg' >> go.mod

# Tidy and download dependencies with cache mount (use GOWORK=off to ignore go.work)
RUN --mount=type=cache,target=/go/pkg/mod \
    GOWORK=off go mod tidy && \
    GOWORK=off go mod download

# Build with cache mount (use GOWORK=off)
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOWORK=off go build -ldflags="-w -s" -o noti-api ./cmd/api

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata wget

# Create non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

# Set working directory
WORKDIR /home/appuser

# Copy binary from builder
COPY --from=builder --chown=appuser:appuser /workspace/services/noti-service/noti-api .

# Copy configuration files
COPY --from=builder --chown=appuser:appuser /workspace/services/noti-service/configs ./configs

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8002

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8002/health/live || exit 1

# Run the application
CMD ["./noti-api"]
