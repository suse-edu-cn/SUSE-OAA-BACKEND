package middleware

import (
	"log/slog"
	"net"
	"strings"
	"time"

	"suseoaa/pkg/logger"
	"suseoaa/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// rateLimitScript 使用 Lua 脚本保证 INCR 与 EXPIRE 原子执行，杜绝因进程意外退出或网络中断导致键永久遗留
var rateLimitScript = redis.NewScript(`
local current = redis.call('INCR', KEYS[1])
if current == 1 then
    redis.call('EXPIRE', KEYS[1], ARGV[1])
else
    local ttl = redis.call('TTL', KEYS[1])
    if ttl == -1 then
        redis.call('EXPIRE', KEYS[1], ARGV[1])
    end
end
return current
`)

// GetRealClientIP 提取客户端真实 IP
// 优先提取 CDN / 反向代理透传头（Ali-CDN-Real-IP, X-Real-IP, X-Forwarded-For），兜底使用 c.ClientIP()
func GetRealClientIP(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}

	// 1. 阿里云 CDN 专用回源头
	if ip := strings.TrimSpace(c.GetHeader("Ali-CDN-Real-IP")); ip != "" {
		if parsed := net.ParseIP(ip); parsed != nil {
			return parsed.String()
		}
	}

	// 2. 常见反向代理头 X-Real-IP
	if ip := strings.TrimSpace(c.GetHeader("X-Real-IP")); ip != "" {
		if parsed := net.ParseIP(ip); parsed != nil {
			return parsed.String()
		}
	}

	// 3. 标准代理链 X-Forwarded-For（取首个有效客户端 IP）
	if xff := strings.TrimSpace(c.GetHeader("X-Forwarded-For")); xff != "" {
		for _, rawIP := range strings.Split(xff, ",") {
			clientIP := strings.TrimSpace(rawIP)
			if parsed := net.ParseIP(clientIP); parsed != nil {
				return parsed.String()
			}
		}
	}

	// 4. Gin 原生兜底
	return c.ClientIP()
}

func RateLimit(rdb *redis.Client, scene string, limit int64, window time.Duration) gin.HandlerFunc {
	windowSeconds := int64(window.Seconds())
	if windowSeconds <= 0 {
		windowSeconds = 1
	}

	return func(c *gin.Context) {
		ip := GetRealClientIP(c)
		key := "ratelimit:" + scene + ":" + ip
		ctx := c.Request.Context()

		count, err := rateLimitScript.Run(ctx, rdb, []string{key}, windowSeconds).Int64()
		if err != nil {
			logger.WarnContext(ctx, "限流执行失败，已自动降级放行",
				slog.String("scene", scene),
				slog.String("ip", ip),
				slog.String("error", err.Error()),
			)
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
