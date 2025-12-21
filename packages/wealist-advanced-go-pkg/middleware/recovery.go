// Package middleware는 HTTP 미들웨어를 제공합니다.
// 이 파일은 panic 복구 미들웨어를 포함합니다.
package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Recovery는 panic을 복구하는 미들웨어를 반환합니다.
// panic 발생 시 스택 트레이스를 로깅하고 500 에러 응답을 반환합니다.
// 서버가 panic으로 인해 죽지 않도록 보호합니다.
func Recovery(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 요청 ID 가져오기 (없으면 새로 생성)
				requestID := GetRequestID(c)

				// panic 로깅 (스택 트레이스 포함)
				logger.Error("Panic recovered",
					zap.String("request_id", requestID),
					zap.Any("error", err),
					zap.String("stack", string(debug.Stack())),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
				)

				// 500 에러 응답 반환
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": map[string]interface{}{
						"code":    "INTERNAL_ERROR",
						"message": "Internal server error",
					},
					"requestId": requestID,
				})
			}
		}()

		c.Next()
	}
}
