# syntax=docker/dockerfile:1.4
# Builder stage - optimized layer caching (2025-12-18)
FROM ghcr.io/orangescloud/base/golang-1.24-bookworm AS builder

# Install build dependencies (cached layer)
RUN apt-get update && apt-get install -y --no-install-recommends git ca-certificates tzdata && rm -rf /var/lib/apt/lists/*

# Set working directory to project root
WORKDIR /workspace

# Copy shared package first (for replace directive)
COPY packages/wealist-advanced-go-pkg/ ./packages/wealist-advanced-go-pkg/

# Copy service files
WORKDIR /workspace/services/chat-service

# OPTIMIZATION: Copy only go.mod/go.sum first for dependency caching
COPY services/chat-service/go.mod services/chat-service/go.sum ./

# Add replace directive for local package
RUN echo 'replace github.com/OrangesCloud/wealist-advanced-go-pkg => ../../packages/wealist-advanced-go-pkg' >> go.mod

# Download dependencies (cached if go.mod/go.sum unchanged)
RUN --mount=type=cache,target=/go/pkg/mod \
    GOWORK=off go mod download -x

# Copy source code (changes here don't invalidate dependency cache)
COPY services/chat-service/ ./

# Re-add replace directive (COPY overwrites go.mod)
RUN echo 'replace github.com/OrangesCloud/wealist-advanced-go-pkg => ../../packages/wealist-advanced-go-pkg' >> go.mod

# Verify dependencies and build
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOWORK=off go mod tidy && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOWORK=off go build -ldflags="-w -s" -o chat-api ./cmd/api

# Runtime stage
FROM ghcr.io/orangescloud/base/alpine-latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata wget

# Create non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

# Set working directory
WORKDIR /home/appuser

# Copy binary from builder
COPY --from=builder --chown=appuser:appuser /workspace/services/chat-service/chat-api .

# Copy configuration files
COPY --from=builder --chown=appuser:appuser /workspace/services/chat-service/configs ./configs

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8001

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8001/health || exit 1

# Run the application
CMD ["./chat-api"]
