# flow_guard

简洁 Redis 单机滑动窗口限流中间件，使用者只需加载配置文件并引入中间件即可启用限流。

## ✅ 特性

- 支持用户自定义配置文件（YAML）
- 基于 Redis 单机，使用 Lua 脚本保障原子性
- 支持精确路径和通配符路径
- 主目录控制所有引用，包之间零耦合

## 🧩 使用示例

```
import (
    "flow_guard/config"
    "flow_guard/limiter"
    "flow_guard/middleware"

    "github.com/gin-gonic/gin"
    "github.com/redis/go-redis/v9"
)

func main() {
    cfg, _ := config.LoadConfig("config.yml")

    rdb := redis.NewClient(&redis.Options{Addr: cfg.FlowGuard.RedisAddr,  Password: cfg.FlowGuard.Password})
    rl := limiter.NewRedisLimiter(rdb, cfg.FlowGuard.Rules)

    r := gin.Default()
    r.Use(middleware.NewRateLimiter(rl))

    r.GET("/api/user/info", func(c *gin.Context) {
        c.JSON(200, gin.H{"msg": "ok"})
    })

    r.Run(":8080")
}
```

---
