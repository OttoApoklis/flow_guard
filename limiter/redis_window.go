package limiter

import (
	"context"
	"fmt"
	"github.com/OttoApoklis/flow_guard/config"
	logger "github.com/OttoApoklis/flow_guard/log"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisLimiter struct {
	Client *redis.Client
	Rules  []config.Rule
}

// 创建一个新的 Redis 限流器
func NewRedisLimiter(client *redis.Client, rules []config.Rule) *RedisLimiter {
	return &RedisLimiter{
		Client: client,
		Rules:  rules,
	}
}

// 获取匹配的规则，如果路径是通配符路径，也会进行匹配
func (r *RedisLimiter) GetMatchedRule(path string) *config.Rule {
	var match *config.Rule
	for _, rule := range r.Rules {
		// 精确匹配路径或匹配通配符路径
		if rule.Path == path || (len(rule.Path) > 0 && rule.Path[len(rule.Path)-1] == '*' && matchPrefix(path, rule.Path[:len(rule.Path)-1])) {
			if match == nil || len(rule.Path) > len(match.Path) {
				match = &rule
			}
		}
	}
	return match
}

// 判断路径是否需要限流（拦截）
func (r *RedisLimiter) ShouldLimit(path string) (bool, error) {
	rule := r.GetMatchedRule(path)
	if rule == nil {
		// 如果没有匹配的规则，则不进行限流
		return false, nil
	}

	// 如果找到了匹配的规则，则执行限流判断
	now := time.Now().UnixNano()
	zKey := "limiter:" + path
	window := int64(rule.Window) * int64(time.Second)
	expire := rule.Window

	luaScript := `
        local key = KEYS[1]
        local now = tonumber(ARGV[1])
        local window = tonumber(ARGV[2])
        local limit = tonumber(ARGV[3])
        local expire = tonumber(ARGV[4])

        redis.call("ZREMRANGEBYSCORE", key, 0, now - window)
        redis.call("ZADD", key, now, now)
        local count = redis.call("ZCARD", key)
        redis.call("EXPIRE", key, expire)

        if count <= limit then
            return 1
        else
            return 0
        end
    `

	script := redis.NewScript(luaScript)
	res, err := script.Run(context.Background(), r.Client, []string{zKey}, now, window, rule.Limit, expire).Int()
	if err != nil {
		logger.GlobalLogger.Info(fmt.Sprintf("%s pass failed.", path))
		return false, err
	}
	logger.GlobalLogger.Info(fmt.Sprintf("%s pass successfully.", path))
	return res == 1, nil
}

// 辅助函数：判断路径前缀是否匹配
func matchPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
