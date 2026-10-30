package response

import (
	"fmt"
	"strings"
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

func cleanErrorMessage(message string) string {
	if message == "" {
		return "请求参数错误"
	}
	if message == "EOF" || strings.Contains(message, "unexpected EOF") {
		return "请求体不能为空或内容不完整"
	}
	if strings.HasPrefix(message, "Key: '") && strings.Contains(message, "Error:Field validation for '") {
		return "参数校验失败，请检查必填项与格式规范"
	}
	if strings.Contains(message, "cannot unmarshal") {
		return "参数数据类型不匹配"
	}
	if strings.HasPrefix(message, "invalid character") {
		return "请求数据不符合合法 JSON 规范"
	}
	if strings.HasPrefix(message, "json: ") {
		return "JSON 请求参数解析失败"
	}
	if strings.Contains(message, "record not found") {
		return "数据不存在"
	}
	if strings.Contains(message, "strconv.Parse") || strings.Contains(message, "invalid syntax") {
		return "数值格式不合法"
	}
	if strings.Contains(message, "UNIQUE constraint failed") || strings.Contains(message, "Duplicate entry") {
		return "数据已存在，请勿重复添加"
	}
	return message
}

func BadRequest(c *gin.Context, message string) {
	message = cleanErrorMessage(message)
	c.Set(ContextErrorKey, message)
	c.JSON(400, Response{
		Code:    400,
		Message: message,
		Data:    nil,
	})
}

func BadRequestWithData(c *gin.Context, message string, data any) {
	message = cleanErrorMessage(message)
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
