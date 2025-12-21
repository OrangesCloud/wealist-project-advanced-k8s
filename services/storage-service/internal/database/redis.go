package database

import (
	"context"
	"os"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var RedisClient *redis.Client

// InitRedis initializes Redis connection
func InitRedis(logger *zap.Logger) error {
	var client *redis.Client

	// redis:// URL 형식 지원
	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		opts, err := redis.ParseURL(redisURL)
		if err != nil {
			return err
		}
		client = redis.NewClient(opts)
	} else {
		// 기본 연결 설정
		redisHost := os.Getenv("REDIS_HOST")
		if redisHost == "" {
			redisHost = "redis"
		}
		redisPort := os.Getenv("REDIS_PORT")
		if redisPort == "" {
			redisPort = "6379"
		}
		password := os.Getenv("REDIS_PASSWORD")
		if password == "NONE" || password == "" {
			password = ""
		}

		client = redis.NewClient(&redis.Options{
			Addr:     redisHost + ":" + redisPort,
			Password: password,
			DB:       0,
		})
	}

	// 연결 테스트 (실패해도 서비스는 시작)
	if err := client.Ping(context.Background()).Err(); err != nil {
		logger.Warn("Redis connection failed, rate limiting will be disabled", zap.Error(err))
		RedisClient = nil
		return nil
	}

	RedisClient = client
	logger.Info("Redis connection established successfully")
	return nil
}

// GetRedis returns the Redis client (may be nil if not connected)
func GetRedis() *redis.Client {
	return RedisClient
}
