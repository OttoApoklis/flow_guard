package middleware

import (
	"context"
	"fmt"
	"github.com/OttoApoklis/flow_guard/limiter"
	logger "github.com/OttoApoklis/flow_guard/log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func NewRateLimiter(l *limiter.RedisLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		rule := l.GetMatchedRule(c.FullPath()) // 根据 FullPath 或原始 URL 匹配
		if rule == nil {
			logger.GlobalLogger.Infof("no rate limit rule matched for path: %s", c.FullPath())
			c.Next()
			return
		}

		key := rule.Path // 使用规则中定义的限流 key 作为 Redis key
		fmt.Sprintf("限流规则路由：%s", key)
		logger.GlobalLogger.Info(fmt.Sprintf("this path: %s", key))
		if key == "" {
			key = c.Request.URL.Path
		}
		ok, err := l.Allow(context.Background(), key)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "rate limiter error"})
			return
		}

		if !ok {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}

		c.Next()
	}
}
