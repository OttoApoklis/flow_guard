package limiter

import (
	"context"
	"flow_guard/config"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisLimiter struct {
	Client *redis.Client
	Rules  []config.Rule
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

func (r *RedisLimiter) Allow(ctx context.Context, path string) (bool, error) {
	rule := r.GetMatchedRule(path)
	if rule == nil {
		return true, nil
	}

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
	res, err := script.Run(ctx, r.Client, []string{zKey}, now, window, rule.Limit, expire).Int()
	if err != nil {
		return false, err
	}

	return res == 1, nil
}

func matchPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
