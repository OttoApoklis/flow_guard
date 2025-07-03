package limiter

import (
	"context"
	"errors"
	"fmt"
	"github.com/OttoApoklis/flow_guard/config"
	logger "github.com/OttoApoklis/flow_guard/log"
	"github.com/OttoApoklis/flow_guard/snowflack"
	"github.com/redis/go-redis/v9"
	"math"
	"net"
	"strconv"
	"strings"
	"time"
)

type RedisLimiter struct {
	Client  redis.Cmdable
	Rules   []config.Rule
	Galileo config.GalileoConfig
}

func NewRedisLimiter(client redis.Cmdable, rules []config.Rule) *RedisLimiter {
	return &RedisLimiter{
		Client: client,
		Rules:  rules,
	}
}

// 最长路由规则优先
func (r *RedisLimiter) GetMatchedRule(path string) *config.Rule {
	var match *config.Rule
	var longest int

	for _, rule := range r.Rules {
		rulePath := rule.Path
		isWildcard := strings.HasSuffix(rulePath, "*")
		base := strings.TrimSuffix(rulePath, "*")

		if rulePath == path || (isWildcard && strings.HasPrefix(path, base)) {
			// 计算实际匹配长度
			matchLen := len(base)
			if !isWildcard {
				matchLen = len(rulePath)
			}

			if match == nil || matchLen > longest {
				match = &rule
				longest = matchLen
			}
		}
	}
	return match

}

// 定义 Lua 脚本
var slidingWindowScript = redis.NewScript(`
    redis.call('ZREMRANGEBYSCORE', KEYS[1], 0, ARGV[1])
    local count = redis.call('ZCARD', KEYS[1])
    redis.call('ZADD', KEYS[1], ARGV[2], ARGV[3])
    redis.call('EXPIRE', KEYS[1], tonumber(ARGV[4]))
    return count
`)

//func (r *RedisLimiter) SingleZSetAllow(ctx context.Context, key string) (bool, error) {
//	client := r.Client
//	client.Ping(ctx)
//	windowSize := time.Duration(r.Rules[0].Window) * time.Second
//	now := time.Now().UnixNano() / int64(time.Millisecond)      // 当前时间戳（毫秒）
//	windowStart := now - int64(windowSize/time.Millisecond)     // 窗口起始时间
//	expireTimeSec := int64(math.Ceil(windowSize.Seconds())) + 1 // 过期时间（秒）
//	id := snowflack.GetSnowFlackID()                            // 唯一 ID（如 Snowflake）
//	member := fmt.Sprintf("%d-%d", now, id)                     // 唯一成员标识
//	logger.GlobalLogger.Info(fmt.Sprintf("redis current limiter info: member: %s", member))
//	var (
//		result interface{}
//		err    error
//		netErr net.Error
//	)
//	// 执行 Lua 脚本
//	for i := 0; i < 3; i++ {
//		ctx, cancel := context.WithTimeout(ctx, 1000*time.Millisecond)
//		defer cancel()
//		result, err = slidingWindowScript.Run(ctx, client, []string{key},
//			strconv.FormatInt(windowStart, 10),   // ARGV[1] windowStart
//			strconv.FormatInt(now, 10),           // ARGV[2] now
//			member,                               // ARGV[3] member
//			strconv.FormatInt(expireTimeSec, 10), // ARGV[4] expireTime
//		).Result()
//		// 对于上下文超时、网络超时、连接中断、Redis TRYAGAIN 情况进行重试
//		if err != nil {
//			if errors.Is(err, context.DeadlineExceeded) {
//				logger.GlobalLogger.Info(fmt.Sprintf("redis current limiter err: deadline Exceeded"))
//				continue
//			}
//			if errors.As(err, &netErr) && netErr != nil && netErr.Timeout() {
//				logger.GlobalLogger.Info(fmt.Sprintf("redis current limiter err: net timeout"))
//				continue
//			}
//			if strings.Contains(err.Error(), "connection refused") {
//				logger.GlobalLogger.Info(fmt.Sprintf("redis current limiter err: connection refused"))
//				continue
//			}
//			if strings.Contains(err.Error(), "EOF") {
//				logger.GlobalLogger.Info(fmt.Sprintf("redis current limiter err: connection refused by redis server"))
//				continue
//			}
//			if strings.Contains(err.Error(), "TRYAGAIN") {
//				logger.GlobalLogger.Info(fmt.Sprintf("redis current limiter err: redis sever advise TRYAGAIN"))
//				continue
//			}
//			// 出错时直接放行， 防止redis故障导致正常请求被拦截
//			return true, fmt.Errorf("redis concurrent limiter err: run lua script failed: %v", err)
//		}
//		break
//	}
//
//	// 解析结果
//	count, ok := result.(int64)
//	if !ok {
//		return false, fmt.Errorf("redis concurrent limiter err: invalid result type: %T", result)
//	}
//	logger.GlobalLogger.Info(fmt.Sprintf("redis current limiter info: count: %d", count))
//	// 判断是否超过最大请求数
//	return count < int64(r.Rules[0].Limit), nil
//}

const shardCount = 50

// 计算分片 key，比如用雪花ID做 mod
func getShardKey(baseKey string) string {
	id := snowflack.GetSnowFlackID()
	shard := int(id % shardCount)
	return fmt.Sprintf("%s:shard:%d", baseKey, shard)
}

// 修改后的 Allow 函数
func (r *RedisLimiter) Allow(ctx context.Context, baseKey string) (bool, error) {
	client := r.Client
	client.Ping(ctx)

	windowSize := time.Duration(r.Rules[0].Window) * time.Second
	now := time.Now().UnixNano() / int64(time.Millisecond)      // 当前时间戳（毫秒）
	windowStart := now - int64(windowSize/time.Millisecond)     // 窗口起始时间
	expireTimeSec := int64(math.Ceil(windowSize.Seconds())) + 1 // 过期时间（秒）

	// 生成分片后的 key
	key := getShardKey(baseKey)

	id := snowflack.GetSnowFlackID()
	member := fmt.Sprintf("%d-%d", now, id) // 唯一成员标识

	logger.GlobalLogger.Info(fmt.Sprintf("redis current limiter info: member: %s, shard key: %s", member, key))

	var (
		result interface{}
		err    error
		netErr net.Error
	)

	for i := 0; i < 3; i++ {
		ctx, cancel := context.WithTimeout(ctx, 1000*time.Millisecond)
		defer cancel()

		result, err = slidingWindowScript.Run(ctx, client, []string{key},
			strconv.FormatInt(windowStart, 10),   // ARGV[1] windowStart
			strconv.FormatInt(now, 10),           // ARGV[2] now
			member,                               // ARGV[3] member
			strconv.FormatInt(expireTimeSec, 10), // ARGV[4] expireTime
		).Result()

		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				logger.GlobalLogger.Info("redis current limiter err: deadline Exceeded")
				continue
			}
			if errors.As(err, &netErr) && netErr != nil && netErr.Timeout() {
				logger.GlobalLogger.Info("redis current limiter err: net timeout")
				continue
			}
			if strings.Contains(err.Error(), "connection refused") {
				logger.GlobalLogger.Info("redis current limiter err: connection refused")
				continue
			}
			if strings.Contains(err.Error(), "EOF") {
				logger.GlobalLogger.Info("redis current limiter err: connection refused by redis server")
				continue
			}
			if strings.Contains(err.Error(), "TRYAGAIN") {
				logger.GlobalLogger.Info("redis current limiter err: redis sever advise TRYAGAIN")
				continue
			}
			return true, errors.New(fmt.Sprintf("redis concurrent limiter err: run lua script failed: %v", err))
		}
		break
	}

	count, ok := result.(int64)
	if !ok {
		return false, errors.New(fmt.Sprintf("redis concurrent limiter err: invalid result type: %T", result))
	}

	logger.GlobalLogger.Info(fmt.Sprintf("redis current limiter info: count: %d", count))

	// 由于分片，限制是单 shard 的 limit，必须调整阈值
	// 例如总 limit = r.Rules[0].Limit，单 shard 限制为 limit / shardCount
	perShardLimit := int64(r.Rules[0].Limit) / int64(shardCount)
	if perShardLimit == 0 {
		perShardLimit = 1
	}

	return count < perShardLimit, nil
}

func matchPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
