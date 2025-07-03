package flow_guard

import (
	"context"
	"fmt"
	"github.com/OttoApoklis/flow_guard/config"
	"github.com/OttoApoklis/flow_guard/galileo"
	"github.com/OttoApoklis/flow_guard/limiter"
	logger "github.com/OttoApoklis/flow_guard/log"
	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

// Init 启动 flow_guard，只需传入配置文件路径
func Init(configPath string) (*limiter.RedisLimiter, error) {
	// 1. 加载配置
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load flow_guard config: %w", err)
	}

	// 2. 初始化日志配置
	if err := logger.InitLogger(cfg); err != nil {
		fmt.Printf("Error initializing logger: %v\n", err)
		return nil, fmt.Errorf("failed to load flow_guard log config: %w", err)
	}

	// 记录日志
	logger.GlobalLogger.Info("FlowGuard started successfully.")

	// 3. 初始化 Redis 客户端
	var rdb redis.Cmdable
	if !cfg.FlowGuard.Redis.IsCluster {
		rdb = redis.NewClient(&redis.Options{
			Addr:     cfg.FlowGuard.Redis.RedisAddrs[0],
			Password: cfg.FlowGuard.Redis.RedisPassword,
			DB:       0,
		})
	} else {
		rdb = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    cfg.FlowGuard.Redis.RedisAddrs,
			Password: cfg.FlowGuard.Redis.RedisPassword,
		})
	}

	// 简单连接测试
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	// 4. 初始化 伽利略 客户端
	galileo.GalileoClient = galileo.NewReporter(cfg.FlowGuard.Galileo.AppID, cfg.FlowGuard.Galileo.Token, cfg.FlowGuard.Galileo.APIURL)
	// 暂时不测试，无技术文档无法测试

	// 5. 创建 Redis 限流器
	rl := limiter.NewRedisLimiter(rdb, cfg.FlowGuard.Rules)

	return rl, nil
}
