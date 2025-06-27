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
		path := c.FullPath()
		logger.GlobalLogger.Info(fmt.Sprintf("this path: %s", path))
		if path == "" {
			path = c.Request.URL.Path
		}

		ok, err := l.Allow(context.Background(), path)
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
