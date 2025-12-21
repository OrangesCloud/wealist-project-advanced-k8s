# syntax=docker/dockerfile:1.4
# frontend/Dockerfile
# Multi-stage build for local development and production
#
# Usage:
#   Development (docker-compose): docker build --target development -t frontend:dev .
#   Production: docker build -t frontend:prod .
#
# Environments:
#   - local: docker-compose (development stage)
#   - develop: Kind local K8s (production stage + ConfigMap)
#   - prod: EKS (production stage + ConfigMap)

# === Base Stage ===
FROM node:20-alpine AS base
WORKDIR /app
ENV NPM_CONFIG_LOGLEVEL=warn
RUN npm install -g pnpm

# === Development Stage ===
# For local development with hot reload (docker-compose)
FROM base AS development
RUN apk add --no-cache curl
COPY package.json pnpm-lock.yaml ./
RUN --mount=type=cache,target=/root/.local/share/pnpm/store \
    pnpm install --frozen-lockfile
# Source code is mounted via docker-compose volumes
# CMD is overridden in docker-compose.yml

# === Builder Stage ===
# Builds the production bundle
FROM base AS builder
WORKDIR /app

# Build-time environment variable (fallback, can be overridden by runtime config)
ARG VITE_API_BASE_URL=http://localhost
ENV VITE_API_BASE_URL=${VITE_API_BASE_URL}

COPY package.json pnpm-lock.yaml ./
RUN --mount=type=cache,target=/root/.local/share/pnpm/store \
    pnpm install --frozen-lockfile
COPY . .
RUN pnpm run build || (echo "Build failed. Check Vite config or env vars." && exit 1)

# === Production Stage ===
# Serves static files with nginx
FROM nginx:stable-alpine AS production

# Copy built assets
COPY --from=builder /app/dist /usr/share/nginx/html

# Copy nginx config for SPA routing (handles /oauth/callback etc.)
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Create config.js for runtime configuration
# K8s (Kind/EKS): API_BASE_URL = "" (empty = ingress mode, uses /svc/* paths)
# Docker-compose: Override via volume mount with API_BASE_URL = "http://localhost"
RUN echo 'window.__ENV__ = { API_BASE_URL: "", API_DOMAIN: "" };' > /usr/share/nginx/html/config.js

EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
