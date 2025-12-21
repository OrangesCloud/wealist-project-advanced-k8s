# syntax=docker/dockerfile:1.4
# Build stage
FROM ghcr.io/orangescloud/base/gradle-8-jdk21 AS builder
WORKDIR /home/gradle/src

# Use Gradle Wrapper for consistent builds
# Copy wrapper files
COPY --chown=gradle:gradle gradlew build.gradle settings.gradle ./
COPY --chown=gradle:gradle gradle ./gradle

# Download dependencies with cache mount (dramatically speeds up rebuilds)
RUN --mount=type=cache,target=/home/gradle/.gradle \
    ./gradlew dependencies --no-daemon

# Copy source code and build the application
COPY --chown=gradle:gradle src ./src
RUN --mount=type=cache,target=/home/gradle/.gradle \
    ./gradlew bootJar --no-daemon

# Runtime stage
FROM ghcr.io/orangescloud/base/eclipse-temurin-21-jre-jammy as final
ENV TZ=Asia/Seoul

# Create a non-root user
RUN groupadd -r spring && useradd -r -g spring spring
WORKDIR /app
USER spring:spring

# Copy the application JAR from the build stage
COPY --from=builder --chown=spring:spring /home/gradle/src/build/libs/*.jar app.jar

# Set up health check to use liveness probe
# liveness: Redis 연결과 무관하게 서비스 자체 상태만 체크 (EKS에서 pod 생존 보장)
# readiness는 /actuator/health/readiness로 별도 확인 가능
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD wget -q --spider http://localhost:8080/actuator/health/liveness || exit 1

EXPOSE 8080

# Run the application
ENTRYPOINT ["java", "-XX:+UseContainerSupport", "-XX:MaxRAMPercentage=75.0", "-jar", "app.jar"]
