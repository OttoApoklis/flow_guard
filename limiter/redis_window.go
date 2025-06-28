package limiter

import (
	"context"
	"fmt"
	"github.com/OttoApoklis/flow_guard/config"
	"github.com/OttoApoklis/flow_guard/uuid"
	"github.com/redis/go-redis/v9"
	"strconv"
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
	windowSize := time.Duration(r.Rules[0].Window) * time.Second
	now := time.Now().UnixNano() / int64(time.Millisecond)  // 当前时间戳（毫秒）
	windowStart := now - int64(windowSize/time.Millisecond) // 窗口起始时间
	expireTimeSec := int64(windowSize/time.Second) + 1      // 过期时间（秒）
	id := uuid.GetUUID()                                    // 唯一 ID（如 Snowflake）
	member := fmt.Sprintf("%d-%s", now, id)                 // 唯一成员标识

	// 执行 Lua 脚本
	result, err := slidingWindowScript.Run(ctx, client, []string{key},
		strconv.FormatInt(windowStart, 10),   // ARGV[1] windowStart
		strconv.FormatInt(now, 10),           // ARGV[2] now
		member,                               // ARGV[3] member
		strconv.FormatInt(expireTimeSec, 10), // ARGV[4] expireTime
	).Result()

	if err != nil {
		return false, fmt.Errorf("run lua script failed: %v", err)
	}

	// 解析结果
	count, ok := result.(int64)
	if !ok {
		return false, fmt.Errorf("invalid result type: %T", result)
	}

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
