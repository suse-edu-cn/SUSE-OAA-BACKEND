package middleware

import (
	"suseoaa/pkg/logger"
	"suseoaa/pkg/utils"

	"github.com/gin-gonic/gin"
)

func isValidTraceID(id string) bool {
	if len(id) < 16 || len(id) > 64 {
		return false
	}
	for i := 0; i < len(id); i++ {
		ch := id[i]
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_') {
			return false
		}
	}
	return true
}

func Trace() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader(logger.TraceIDHeader)
		if traceID == "" || !isValidTraceID(traceID) {
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
