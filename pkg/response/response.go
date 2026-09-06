package response

import "github.com/gin-gonic/gin"

type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

func Fail(c *gin.Context, code int, message string, data any) {
	c.JSON(code, Response{
		Code:    code,
		Message: message,
		Data:    data,
	})
}

func Success(c *gin.Context, data any) {
	c.JSON(200, Response{
		Code:    200,
		Message: "success",
		Data:    data,
	})
}
