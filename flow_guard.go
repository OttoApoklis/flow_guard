package flow_guard

import (
	"context"
	"fmt"
	"github.com/OttoApoklis/flow_guard/config"
	"github.com/OttoApoklis/flow_guard/limiter"
	"github.com/OttoApoklis/flow_guard/middleware"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

// Init 启动 flow_guard，只需传入配置文件路径
func Init(configPath string) error {
	// 1. 加载配置
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// 2. 初始化 Redis 客户端
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.FlowGuard.RedisAddr,
		Password: cfg.FlowGuard.Password,
		DB:       0,
	})

	// 简单连接测试
	if err := rdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to redis: %w", err)
	}

	rl := limiter.NewRedisLimiter(rdb, cfg.FlowGuard.Rules)

	r := gin.Default()
	r.Use(middleware.NewRateLimiter(rl))
	return nil
}
