package middleware

import (
	"time"

	"suseoaa/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// rateLimitScript 使用 Lua 脚本保证 INCR 与 EXPIRE 原子执行，杜绝因进程意外退出或网络中断导致键永久遗留
var rateLimitScript = redis.NewScript(`
local current = redis.call('INCR', KEYS[1])
if current == 1 then
    redis.call('EXPIRE', KEYS[1], ARGV[1])
end
return current
`)

func RateLimit(rdb *redis.Client, scene string, limit int64, window time.Duration) gin.HandlerFunc {
	windowSeconds := int64(window.Seconds())
	if windowSeconds <= 0 {
		windowSeconds = 1
	}

	return func(c *gin.Context) {
		ip := c.ClientIP()
		key := "ratelimit:" + scene + ":" + ip
		ctx := c.Request.Context()

		count, err := rateLimitScript.Run(ctx, rdb, []string{key}, windowSeconds).Int64()
		if err != nil {
			c.Next()
			return
		}

		if count > limit {
			response.TooManyRequests(c)
			c.Abort()
			return
		}

		c.Next()
	}
}
