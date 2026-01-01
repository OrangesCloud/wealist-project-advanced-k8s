# =============================================================================
# Ops Portal Dockerfile (Multi-stage build)
# =============================================================================
# Non-root nginx 실행을 위한 설정 포함

# Stage 1: Build
FROM node:20-alpine AS builder

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm ci

# Copy source code
COPY . .

# Build
RUN npm run build

# Stage 2: Serve with nginx (non-root compatible)
FROM nginx:alpine

# nginx non-root 실행을 위한 권한 설정
# Kubernetes securityContext: runAsUser: 101 (nginx)
RUN mkdir -p /var/cache/nginx/client_temp \
             /var/cache/nginx/proxy_temp \
             /var/cache/nginx/fastcgi_temp \
             /var/cache/nginx/uwsgi_temp \
             /var/cache/nginx/scgi_temp && \
    chown -R nginx:nginx /var/cache/nginx && \
    chown -R nginx:nginx /var/log/nginx && \
    chown -R nginx:nginx /etc/nginx/conf.d && \
    touch /var/run/nginx.pid && \
    chown nginx:nginx /var/run/nginx.pid

# Copy custom nginx config
COPY docker/nginx.conf /etc/nginx/conf.d/default.conf

# Copy built files
COPY --from=builder /app/dist /usr/share/nginx/html

# 정적 파일 권한 설정
RUN chown -R nginx:nginx /usr/share/nginx/html

# Expose port (non-privileged port 사용 시 변경)
EXPOSE 80

# non-root user로 실행
USER nginx

# Start nginx
CMD ["nginx", "-g", "daemon off;"]
