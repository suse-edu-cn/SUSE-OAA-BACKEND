package response

import (
	"fmt"
	"suseoaa/pkg/logger"

	"github.com/gin-gonic/gin"
)

const ContextErrorKey = "response_error_msg"

type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

func Success(c *gin.Context, data any) {
	c.JSON(200, Response{
		Code:    200,
		Message: "success",
		Data:    data,
	})
}

func BadRequest(c *gin.Context, message string) {
	c.Set(ContextErrorKey, message)
	c.JSON(400, Response{
		Code:    400,
		Message: message,
		Data:    nil,
	})
}

func BadRequestWithData(c *gin.Context, message string, data any) {
	c.Set(ContextErrorKey, message)
	c.JSON(400, Response{
		Code:    400,
		Message: message,
		Data:    data,
	})
}

func Unauthorized(c *gin.Context, message string) {
	c.Set(ContextErrorKey, message)
	c.JSON(401, Response{
		Code:    401,
		Message: message,
		Data:    nil,
	})
}

func Forbidden(c *gin.Context, message string) {
	c.Set(ContextErrorKey, message)
	c.JSON(403, Response{
		Code:    403,
		Message: message,
		Data:    nil,
	})
}

func NotFound(c *gin.Context, message string) {
	c.Set(ContextErrorKey, message)
	c.JSON(404, Response{
		Code:    404,
		Message: message,
		Data:    nil,
	})
}

func TooManyRequests(c *gin.Context) {
	msg := "请求过于频繁，请稍后再试"
	c.Set(ContextErrorKey, msg)
	c.JSON(429, Response{
		Code:    429,
		Message: msg,
		Data:    nil,
	})
}

func ServiceUnavailable(c *gin.Context, message string, data any) {
	c.Set(ContextErrorKey, message)
	c.JSON(503, Response{
		Code:    503,
		Message: message,
		Data:    data,
	})
}

func ServerError(c *gin.Context, action string, err error) {
	errMsg := action + "失败"
	if err != nil {
		errMsg = fmt.Sprintf("%s: %v", errMsg, err)
	}
	c.Set(ContextErrorKey, errMsg)
	logger.ErrorContext(c.Request.Context(), action+"失败", "err", err)
	c.JSON(500, Response{
		Code:    500,
		Message: "服务开小差了，请稍后再试",
		Data:    nil,
	})
}
