package middleware

import (
	"github.com/OttoApoklis/flow_guard/limiter"
	logger "github.com/OttoApoklis/flow_guard/log"
	"github.com/gin-gonic/gin"
	"strings"
)

// ThirdPartyInterceptor 是针对第三方接口的限流中间件
func ThirdPartyInterceptor(limiter *limiter.RedisLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 仅拦截特定路由，例如 /third-party-proxy
		if strings.HasPrefix(c.Request.URL.Path, "/third-party-proxy") {
			// 获取API密钥或其他标识
			apiKey := c.GetHeader("X-API-Key")
			if apiKey == "" {
				c.AbortWithStatusJSON(400, gin.H{"error": "API Key is required"})
				return
			}

			// 使用Redis限流器检查该API请求是否超过限流阈值
			ok, err := limiter.Allow(c, apiKey)
			if err != nil {
				logger.GlobalLogger.Error("Error while checking rate limit: %v", err)
				c.AbortWithStatusJSON(500, gin.H{"error": "internal server error"})
				return
			}

			// 若超过限制，返回429 Too Many Requests
			if !ok {
				c.AbortWithStatusJSON(429, gin.H{"error": "rate limit exceeded"})
				return
			}
		}

		// 继续执行后续的处理
		c.Next()
	}
}
