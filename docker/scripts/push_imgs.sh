#!/bin/bash

echo "🔨 [1/8] Building auth-service..."
docker build -t your-docker-id/wealist-auth-service:v1 -f services/auth-service/Dockerfile services/auth-service
echo "📤 Pushing auth-service..."
docker push your-docker-id/wealist-auth-service:v1

echo "🔨 [2/8] Building board-service..."
docker build -t your-docker-id/wealist-board-service:v1 -f services/board-service/docker/Dockerfile services/board-service
echo "📤 Pushing board-service..."
docker push your-docker-id/wealist-board-service:v1

echo "🔨 [3/8] Building chat-service..."
docker build -t your-docker-id/wealist-chat-service:v1 -f services/chat-service/docker/Dockerfile services/chat-service
echo "📤 Pushing chat-service..."
docker push your-docker-id/wealist-chat-service:v1

echo "🔨 [4/8] Building user-service..."
docker build -t your-docker-id/wealist-user-service:v1 -f services/user-service/docker/Dockerfile services/user-service
echo "📤 Pushing user-service..."
docker push your-docker-id/wealist-user-service:v1

echo "🔨 [5/8] Building noti-service..."
docker build -t your-docker-id/wealist-noti-service:v1 -f services/noti-service/docker/Dockerfile services/noti-service
echo "📤 Pushing noti-service..."
docker push your-docker-id/wealist-noti-service:v1

echo "🔨 [6/8] Building storage-service..."
docker build -t your-docker-id/wealist-storage-service:v1 -f services/storage-service/docker/Dockerfile services/storage-service
echo "📤 Pushing storage-service..."
docker push your-docker-id/wealist-storage-service:v1


echo "🔨 [7/8] Building video-service..."
docker build -t your-docker-id/wealist-video-service:v1 -f services/video-service/docker/Dockerfile services/video-service
echo "📤 Pushing video-service..."
docker push your-docker-id/wealist-video-service:v1

echo "🔨 [8/8] Building frontend..."
docker build -t your-docker-id/wealist-frontend:v1 -f services/frontend/Dockerfile services/frontend
echo "📤 Pushing frontend..."
docker push your-docker-id/wealist-frontend:v1
