package database

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var redisClient *redis.Client

// InitRedis initializes the Redis connection
func InitRedis(logger *zap.Logger) error {
	host := os.Getenv("REDIS_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("REDIS_PORT")
	if port == "" {
		port = "6379"
	}
	password := os.Getenv("REDIS_PASSWORD")
	dbStr := os.Getenv("REDIS_DB")
	db := 0
	if dbStr != "" {
		var err error
		db, err = strconv.Atoi(dbStr)
		if err != nil {
			db = 0
		}
	}

	redisClient = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", host, port),
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Warn("Redis connection failed", zap.Error(err))
		redisClient = nil
		return err
	}

	logger.Info("Redis connected successfully",
		zap.String("host", host),
		zap.String("port", port),
		zap.Int("db", db),
	)

	return nil
}

// GetRedis returns the Redis client
func GetRedis() *redis.Client {
	return redisClient
}
