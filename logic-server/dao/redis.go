package dao

import (
	"context"
	"fmt"
	"log"

	"geekedu/common/config"

	"github.com/redis/go-redis/v9"
)

// RedisClient 全局 Redis 客户端实例
var RedisClient *redis.Client

// InitRedis 初始化 Redis 连接
func InitRedis() {
	cfg := config.GetConfig()
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// 测试连接是否畅通
	_, err := rdb.Ping(context.Background()).Result()
	if err != nil {
		log.Printf("Warning: Failed to connect to Redis: %v. Service will degrade to DB only.", err)
	} else {
		log.Println("Redis connected successfully")
	}

	RedisClient = rdb
}
