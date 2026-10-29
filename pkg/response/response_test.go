package response

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestResponseHelpers_SetContextError(t *testing.T) {
	tests := []struct {
		name          string
		call          func(c *gin.Context)
		expectedCode  int
		expectedError string
	}{
		{
			name: "BadRequest",
			call: func(c *gin.Context) {
				BadRequest(c, "参数校验失败")
			},
			expectedCode:  http.StatusBadRequest,
			expectedError: "参数校验失败",
		},
		{
			name: "BadRequestWithData",
			call: func(c *gin.Context) {
				BadRequestWithData(c, "表单错误", map[string]string{"field": "username"})
			},
			expectedCode:  http.StatusBadRequest,
			expectedError: "表单错误",
		},
		{
			name: "Unauthorized",
			call: func(c *gin.Context) {
				Unauthorized(c, "Token失效")
			},
			expectedCode:  http.StatusUnauthorized,
			expectedError: "Token失效",
		},
		{
			name: "Forbidden",
			call: func(c *gin.Context) {
				Forbidden(c, "无权访问")
			},
			expectedCode:  http.StatusForbidden,
			expectedError: "无权访问",
		},
		{
			name: "NotFound",
			call: func(c *gin.Context) {
				NotFound(c, "资源未找到")
			},
			expectedCode:  http.StatusNotFound,
			expectedError: "资源未找到",
		},
		{
			name: "TooManyRequests",
			call: func(c *gin.Context) {
				TooManyRequests(c)
			},
			expectedCode:  http.StatusTooManyRequests,
			expectedError: "请求过于频繁，请稍后再试",
		},
		{
			name: "ServiceUnavailable",
			call: func(c *gin.Context) {
				ServiceUnavailable(c, "服务暂时不可用", nil)
			},
			expectedCode:  http.StatusServiceUnavailable,
			expectedError: "服务暂时不可用",
		},
		{
			name: "ServerError",
			call: func(c *gin.Context) {
				ServerError(c, "数据库查询", errors.New("timeout"))
			},
			expectedCode:  http.StatusInternalServerError,
			expectedError: "数据库查询失败: timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			c.Request = req

			tt.call(c)

			if w.Code != tt.expectedCode {
				t.Fatalf("expected status %d, got %d", tt.expectedCode, w.Code)
			}
			errVal, ok := c.Get(ContextErrorKey)
			if !ok {
				t.Fatalf("expected ContextErrorKey to be set in context")
			}
			if errVal != tt.expectedError {
				t.Fatalf("expected error message %q, got %q", tt.expectedError, errVal)
			}
		})
	}
}

func TestSuccess_NoContextError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Request = req

	Success(c, "ok")

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if _, ok := c.Get(ContextErrorKey); ok {
		t.Fatalf("expected ContextErrorKey not to be set on Success")
	}
}
