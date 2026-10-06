package response

import (
	"suseoaa/pkg/logger"

	"github.com/gin-gonic/gin"
)

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
	c.JSON(400, Response{
		Code:    400,
		Message: message,
		Data:    nil,
	})
}

func BadRequestWithData(c *gin.Context, message string, data any) {
	c.JSON(400, Response{
		Code:    400,
		Message: message,
		Data:    data,
	})
}

func Unauthorized(c *gin.Context, message string) {
	c.JSON(401, Response{
		Code:    401,
		Message: message,
		Data:    nil,
	})
}

func Forbidden(c *gin.Context, message string) {
	c.JSON(403, Response{
		Code:    403,
		Message: message,
		Data:    nil,
	})
}

func TooManyRequests(c *gin.Context) {
	c.JSON(429, Response{
		Code:    429,
		Message: "请求过于频繁，请稍后再试",
		Data:    nil,
	})
}

func ServerError(c *gin.Context, action string, err error) {
	logger.ErrorContext(c.Request.Context(), action+"失败", "err", err)
	c.JSON(500, Response{
		Code:    500,
		Message: "服务开小差了，请稍后再试",
		Data:    nil,
	})
}
