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
	Client  *redis.Client
	Rules   []config.Rule
	Galileo config.GalileoConfig
}

func NewRedisLimiter(client *redis.Client, rules []config.Rule) *RedisLimiter {
	return &RedisLimiter{
		Client: client,
		Rules:  rules,
	}
}

func (r *RedisLimiter) GetMatchedRule(path string) *config.Rule {
	var match *config.Rule
	for _, rule := range r.Rules {
		if rule.Path == path || (len(rule.Path) > 0 && rule.Path[len(rule.Path)-1] == '*' && matchPrefix(path, rule.Path[:len(rule.Path)-1])) {
			if match == nil || len(rule.Path) > len(match.Path) {
				match = &rule
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

// 滑动窗口限流检查
func (r *RedisLimiter) Allow(ctx context.Context, key string) (bool, error) {
	client := r.Client
	client.Ping(ctx)
	windowSize := time.Duration(r.Rules[0].Window) * time.Second
	now := time.Now().UnixNano() / int64(time.Millisecond)      // 当前时间戳（毫秒）
	windowStart := now - int64(windowSize/time.Millisecond)     // 窗口起始时间
	expireTimeSec := int64(math.Ceil(windowSize.Seconds())) + 1 // 过期时间（秒）
	id := snowflack.GetSnowFlackID()                            // 唯一 ID（如 Snowflake）
	member := fmt.Sprintf("%d-%d", now, id)                     // 唯一成员标识
	logger.GlobalLogger.Info(fmt.Sprintf("redis current limiter info: member: %s", member))
	var (
		result interface{}
		err    error
		netErr net.Error
	)
	// 执行 Lua 脚本
	for i := 0; i < 3; i++ {
		ctx, cancel := context.WithTimeout(ctx, 1000*time.Millisecond)
		defer cancel()
		result, err = slidingWindowScript.Run(ctx, client, []string{key},
			strconv.FormatInt(windowStart, 10),   // ARGV[1] windowStart
			strconv.FormatInt(now, 10),           // ARGV[2] now
			member,                               // ARGV[3] member
			strconv.FormatInt(expireTimeSec, 10), // ARGV[4] expireTime
		).Result()
		// 对于上下文超时、网络超时、连接中断、Redis TRYAGAIN 情况进行重试
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				logger.GlobalLogger.Info(fmt.Sprintf("redis current limiter err: deadline Exceeded"))
				continue
			}
			if errors.As(err, &netErr) && netErr != nil && netErr.Timeout() {
				logger.GlobalLogger.Info(fmt.Sprintf("redis current limiter err: net timeout"))
				continue
			}
			if strings.Contains(err.Error(), "connection refused") {
				logger.GlobalLogger.Info(fmt.Sprintf("redis current limiter err: connection refused"))
				continue
			}
			if strings.Contains(err.Error(), "EOF") {
				logger.GlobalLogger.Info(fmt.Sprintf("redis current limiter err: connection refused by redis server"))
				continue
			}
			if strings.Contains(err.Error(), "TRYAGAIN") {
				logger.GlobalLogger.Info(fmt.Sprintf("redis current limiter err: redis sever advise TRYAGAIN"))
				continue
			}
			// 出错时直接放行， 防止redis故障导致正常请求被拦截
			return true, fmt.Errorf("redis concurrent limiter err: run lua script failed: %v", err)
		}
		break
	}

	// 解析结果
	count, ok := result.(int64)
	if !ok {
		return false, fmt.Errorf("redis concurrent limiter err: invalid result type: %T", result)
	}
	logger.GlobalLogger.Info(fmt.Sprintf("redis current limiter info: count: %d", count))
	// 判断是否超过最大请求数
	return count < int64(r.Rules[0].Limit), nil
}

//func (r *RedisLimiter) Allow(ctx context.Context, path string) (bool, error) {
//	rule := r.GetMatchedRule(path)
//	fmt.Sprintf("in allow logic this path: %s", path)
//	logger.GlobalLogger.Info(fmt.Sprintf("in allow logic this path: %s", path))
//	if rule == nil {
//		logger.GlobalLogger.Info(fmt.Sprintf("%s not in rules.", path))
//		return true, nil
//	}
//
//	now := time.Now().UnixNano()
//	zKey := "limiter:" + path
//	window := int64(rule.Window) * int64(time.Second)
//	expire := rule.Window
//
//	luaScript := `
//        local key = KEYS[1]
//        local now = tonumber(ARGV[1])
//        local window = tonumber(ARGV[2])
//        local limit = tonumber(ARGV[3])
//        local expire = tonumber(ARGV[4])
//
//        redis.call("ZREMRANGEBYSCORE", key, 0, now - window)
//        redis.call("ZADD", key, now, now)
//        local count = redis.call("ZCARD", key)
//        redis.call("EXPIRE", key, expire)
//
//        if count <= limit then
//            return 1
//        else
//            return 0
//        end
//    `
//
//	script := redis.NewScript(luaScript)
//	res, err := script.Run(ctx, r.Client, []string{zKey}, now, window, rule.Limit, expire).Int()
//	if err != nil {
//		logger.GlobalLogger.Info(fmt.Sprintf("%s pass failed. limite : %v", path, rule))
//		return false, err
//	}
//	logger.GlobalLogger.Info(fmt.Sprintf("%s pass successfully. limite : %v", path, rule))
//	// 上报限流事件到伽利略
//	report := galileo.GetReporter()
//	report.ReportRateLimitEvent(path, res == 1)
//	return res == 1, nil
//}

func matchPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
