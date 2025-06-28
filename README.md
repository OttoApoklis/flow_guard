# flow_guard

简洁 Redis 单机滑动窗口限流中间件，使用者只需加载配置文件并引入中间件即可启用限流。

## ✅ 特性

- 支持用户自定义配置文件（YAML）
- 基于 Redis 单机，使用 Lua 脚本保障原子性
- 支持精确路径和通配符路径
- 主目录控制所有引用，包之间零耦合

## 🧩 使用示例

```yml
flow_guard:
  redis_addr: "192.168.10.38:6379"
  redis_password: "your_strong_password_here"

  rules:
    - path: /*
      limit: 2
      window: 1
  log:
    level: "info"
    file: "./logs/flow_guard.log"
    max_size: 10
    max_backups: 5
    max_age: 7

```

```
import "github.com/OttoApoklis/flow_guard"
// 初始化 FlowGuard
	rl, err := flow_guard.Init("./config.yaml")
	if err != nil {
		log.Fatalf("Error initializing flow_guard: %v", err)
	}

	// 传入请求路径，检查是否需要限流
	path := "/user/get"
	ctx := context.Background()
	allow, err := rl.Allow(ctx, path)
	if err != nil {
		log.Fatalf("Error checking limit: %v", err)
	}

	if allow {
		fmt.Println("Request is allowed.")
	} else {
		// 如果请求被限流，返回429状态码
		c.Header("Retry-After", "60") // 例如返回60秒后可以重试
		c.JSON(429, gin.H{
			"error":       "Rate limit exceeded",
			"message":     "You have made too many requests. Please try again later.",
			"retry_after": 60, // 返回重试时间，单位秒
		})
		return
	}

	_, err = client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(newUser).          // 请求体：User 对象
		SetResult(&responseUsers). // 响应体：[]User
		Get("http://localhost:8080/user/get")

	if err != nil {
		panic(err)
	}
```

---
