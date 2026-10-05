package middleware

import (
	"suseoaa/pkg/logger"
	"suseoaa/pkg/utils"

	"github.com/gin-gonic/gin"
)

func Trace() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader(logger.TraceIDHeader)
		if traceID == "" {
			var err error
			traceID, err = utils.GetUUID()
			if err != nil {
				traceID = "trace-fallback"
			}
		}

		c.Header(logger.TraceIDHeader, traceID)
		c.Set(string(logger.TraceIDKey), traceID)

		ctx := logger.WithTraceID(c.Request.Context(), traceID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
