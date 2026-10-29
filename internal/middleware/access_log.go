package middleware

import (
	"log/slog"
	"suseoaa/pkg/logger"
	"suseoaa/pkg/response"
	"time"

	"github.com/gin-gonic/gin"
)

func AccessLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		rawQuery := c.Request.URL.RawQuery
		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		ctx := c.Request.Context()
		if rawQuery != "" {
			path = path + "?" + rawQuery
		}

		attrs := []any{
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.Int("status", status),
			slog.Duration("latency", latency),
			slog.Int64("latency_ms", latency.Milliseconds()),
			slog.String("client_ip", c.ClientIP()),
			slog.Int("body_size", c.Writer.Size()),
			slog.String("user_agent", c.Request.UserAgent()),
		}

		if len(c.Errors) > 0 {
			attrs = append(attrs, slog.String("gin_errors", c.Errors.String()))
		}

		if status >= 400 {
			var errMsg string
			if v, ok := c.Get(response.ContextErrorKey); ok {
				if msg, okStr := v.(string); okStr && msg != "" {
					errMsg = msg
				}
			}
			if errMsg == "" && len(c.Errors) > 0 {
				errMsg = c.Errors.String()
			}
			if errMsg != "" {
				attrs = append(attrs, slog.String("error", errMsg))
			}
		}

		switch {
		case status >= 500:
			logger.ErrorContext(ctx, "HTTP Server Error", attrs...)
		case status >= 400:
			logger.WarnContext(ctx, "HTTP Client Error", attrs...)
		default:
			logger.InfoContext(ctx, "HTTP Request", attrs...)
		}
	}

}
