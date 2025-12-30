// Package middleware는 Gin 기반 서비스를 위한 HTTP 미들웨어를 제공합니다.
// 이 파일은 요청 로깅 미들웨어를 포함합니다.
package middleware

import (
	"time"

	"github.com/OrangesCloud/wealist-advanced-go-pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// RequestIDKey는 컨텍스트에서 요청 ID를 저장/조회하기 위한 키입니다.
const RequestIDKey = "request_id"

// Logger는 HTTP 요청을 구조화된 로그로 기록하는 미들웨어를 반환합니다.
// 각 요청에 대해 UUID 기반 request_id를 생성하고, 요청/응답 정보를 로깅합니다.
// 상태 코드에 따라 로그 레벨이 결정됩니다:
//   - 5xx: Error
//   - 4xx: Warn
//   - 그 외: Info
func Logger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 요청 ID 생성 및 설정
		requestID := uuid.New().String()
		c.Set(RequestIDKey, requestID)
		c.Header("X-Request-ID", requestID)

		// 타이머 시작
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// 요청 처리
		c.Next()

		// 처리 시간 계산
		duration := time.Since(start)

		// 상태 코드 가져오기
		statusCode := c.Writer.Status()

		// 로그 필드 구성
		fields := []zap.Field{
			zap.String("request_id", requestID),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Int("status", statusCode),
			zap.Duration("duration", duration),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
			zap.Int("body_size", c.Writer.Size()),
		}

		// 인증 미들웨어에서 설정한 user_id가 있으면 추가
		if userID, exists := c.Get("user_id"); exists {
			fields = append(fields, zap.Any("user_id", userID))
		}

		// 에러가 있으면 추가
		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("error", c.Errors.String()))
		}

		// 상태 코드에 따른 로그 레벨 결정
		if statusCode >= 500 {
			logger.Error("Server error", fields...)
		} else if statusCode >= 400 {
			logger.Warn("Client error", fields...)
		} else {
			logger.Info("Request completed", fields...)
		}
	}
}

// LoggerWithTracing은 OpenTelemetry trace context를 추출하고 로그에 포함하는 미들웨어입니다.
// 기존 Logger 미들웨어에 trace_id, span_id를 추가합니다.
// W3C Trace Context 헤더(traceparent, tracestate)를 자동으로 처리합니다.
//
// 사용 예시:
//
//	router.Use(middleware.LoggerWithTracing(zapLogger, "board-service"))
func LoggerWithTracing(zapLogger *zap.Logger, serviceName string) gin.HandlerFunc {
	tracer := otel.Tracer(serviceName)
	propagator := otel.GetTextMapPropagator()

	return func(c *gin.Context) {
		// W3C Trace Context 헤더에서 trace context 추출
		ctx := propagator.Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))

		// 새 span 시작
		ctx, span := tracer.Start(ctx, c.Request.Method+" "+c.FullPath())
		defer span.End()

		// context를 request에 설정
		c.Request = c.Request.WithContext(ctx)

		// 요청 ID 생성 및 설정
		requestID := uuid.New().String()
		c.Set(RequestIDKey, requestID)
		c.Header("X-Request-ID", requestID)

		// 타이머 시작
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// 요청 처리
		c.Next()

		// 처리 시간 계산
		duration := time.Since(start)

		// 상태 코드 가져오기
		statusCode := c.Writer.Status()

		// trace context 추출
		spanCtx := trace.SpanContextFromContext(ctx)

		// 로그 필드 구성 (OTEL Semantic Conventions)
		fields := []zap.Field{
			// Trace context
			zap.String(logger.FieldTraceID, spanCtx.TraceID().String()),
			zap.String(logger.FieldSpanID, spanCtx.SpanID().String()),
			zap.Bool(logger.FieldTraceSampled, spanCtx.IsSampled()),
			// HTTP fields
			zap.String(logger.FieldHTTPRequestID, requestID),
			zap.String(logger.FieldHTTPMethod, c.Request.Method),
			zap.String(logger.FieldHTTPRoute, c.FullPath()),
			zap.String(logger.FieldHTTPTarget, path),
			zap.Int(logger.FieldHTTPStatusCode, statusCode),
			zap.Duration(logger.FieldHTTPDuration, duration),
			zap.String(logger.FieldHTTPClientIP, c.ClientIP()),
			zap.String(logger.FieldHTTPUserAgent, c.Request.UserAgent()),
			// Query string (if present)
			zap.String("http.query", query),
			zap.Int("http.response_content_length", c.Writer.Size()),
		}

		// 인증 미들웨어에서 설정한 user_id가 있으면 추가
		if userID, exists := c.Get("user_id"); exists {
			fields = append(fields, zap.Any(logger.FieldUserID, userID))
		}

		// 에러가 있으면 추가
		if len(c.Errors) > 0 {
			fields = append(fields, zap.String(logger.FieldErrorMessage, c.Errors.String()))
			span.RecordError(c.Errors.Last().Err)
		}

		// 상태 코드에 따른 로그 레벨 결정
		if statusCode >= 500 {
			zapLogger.Error("Server error", fields...)
		} else if statusCode >= 400 {
			zapLogger.Warn("Client error", fields...)
		} else {
			zapLogger.Info("Request completed", fields...)
		}
	}
}

// SkipPathLoggerWithTracing은 특정 경로를 제외하고 LoggerWithTracing을 적용합니다.
func SkipPathLoggerWithTracing(zapLogger *zap.Logger, serviceName string, skipPaths ...string) gin.HandlerFunc {
	skipMap := make(map[string]bool)
	for _, path := range skipPaths {
		skipMap[path] = true
	}

	loggerMiddleware := LoggerWithTracing(zapLogger, serviceName)

	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// 스킵 경로면 로깅 없이 다음 핸들러로 진행
		if skipMap[path] {
			c.Next()
			return
		}

		loggerMiddleware(c)
	}
}

// SkipPathLogger는 특정 경로를 로깅에서 제외하는 미들웨어를 반환합니다.
// /health, /metrics 등 노이즈가 많은 경로를 제외할 때 사용합니다.
func SkipPathLogger(logger *zap.Logger, skipPaths ...string) gin.HandlerFunc {
	// 스킵할 경로를 map으로 변환하여 O(1) 조회
	skipMap := make(map[string]bool)
	for _, path := range skipPaths {
		skipMap[path] = true
	}

	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// 스킵 경로면 로깅 없이 다음 핸들러로 진행
		if skipMap[path] {
			c.Next()
			return
		}

		// 일반 로거 사용
		Logger(logger)(c)
	}
}

// GetRequestID는 컨텍스트에서 요청 ID를 가져옵니다.
// 요청 ID가 없으면 새로운 UUID를 생성하여 반환합니다.
func GetRequestID(c *gin.Context) string {
	if requestID, exists := c.Get(RequestIDKey); exists {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	return uuid.New().String()
}
