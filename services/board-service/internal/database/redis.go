package database

import (
	"context"
	"fmt"
	"os"
	"project-board-api/internal/config"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var RedisClient *redis.Client

func InitRedis(cfg config.Config, log *zap.Logger) error {
	var client *redis.Client
	var addr string

	// redis:// 형식 URL 있으면 우선 사용
	if cfg.Redis.URL != "" {
		log.Info("Using REDIS_URL for connection", zap.String("url", cfg.Redis.URL))
		opts, err := redis.ParseURL(cfg.Redis.URL)
		if err != nil {
			log.Error("Failed to parse REDIS_URL", zap.Error(err))
			return err
		}
		client = redis.NewClient(opts)
		addr = opts.Addr
	} else {
		// REDIS_HOST 환경변수 사용, 없으면 기본값 redis
		redisHost := os.Getenv("REDIS_HOST")
		if redisHost == "" {
			redisHost = "redis"
		}
		redisPort := os.Getenv("REDIS_PORT")
		if redisPort == "" {
			redisPort = "6379"
		}
		addr = fmt.Sprintf("%s:%s", redisHost, redisPort)

		// Password 처리: "NONE"이나 플레이스홀더는 빈 문자열로 처리
		password := cfg.Redis.Password
		if password == "NONE" || password == "CHANGE_THIS_REDIS_AUTH_TOKEN_32_CHAR_MIN" {
			password = ""
		}

		log.Info("Using REDIS_HOST/PORT for connection",
			zap.String("host", redisHost),
			zap.String("port", redisPort),
			zap.String("addr", addr))

		client = redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: password,
			DB:       1,
		})
	}

	// 연결 테스트 (실패해도 서비스는 시작)
	if err := client.Ping(context.Background()).Err(); err != nil {
		log.Warn("Redis connection failed, but service will continue without Redis",
			zap.String("addr", addr),
			zap.Error(err))
		RedisClient = nil
		return nil
	}

	RedisClient = client
	log.Info("Redis connection established successfully",
		zap.String("addr", addr),
		zap.Int("db", cfg.Redis.DB))
	return nil
}

func GetRedis() *redis.Client {
	// Return nil instead of panicking to allow tests to run without Redis
	return RedisClient
}
