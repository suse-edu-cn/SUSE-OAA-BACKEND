package response

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type TestReq struct {
	Name  string `json:"name" binding:"required"`
	Age   int    `json:"age" binding:"min=1,max=120"`
	Email string `json:"email" binding:"omitempty,email"`
}

func TestFormatBindError_Validation(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")

	var req TestReq
	err := c.ShouldBindJSON(&req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	msg := FormatBindError(err)
	t.Logf("formatted validation msg: %s", msg)
	if !strings.Contains(msg, "name") || !strings.Contains(msg, "必填") {
		t.Fatalf("expected translated error for name, got %q", msg)
	}
}

func TestFormatBindError_EOF(t *testing.T) {
	msg := FormatBindError(io.EOF)
	if msg != "请求体不能为空" {
		t.Fatalf("expected '请求体不能为空', got %q", msg)
	}
}

func TestFormatBindError_TypeMismatch(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name": 123}`))
	c.Request.Header.Set("Content-Type", "application/json")

	var req TestReq
	err := c.ShouldBindJSON(&req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	msg := FormatBindError(err)
	if !strings.Contains(msg, "类型错误") || !strings.Contains(msg, "name") {
		t.Fatalf("expected type error message for name, got %q", msg)
	}
}

func TestCleanErrorMessage(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "请求参数错误"},
		{"EOF", "请求体不能为空或内容不完整"},
		{"unexpected EOF", "请求体不能为空或内容不完整"},
		{"Key: 'Req.Name' Error:Field validation for 'Name' failed on the 'required' tag", "参数校验失败，请检查必填项与格式规范"},
		{"json: cannot unmarshal string into Go struct field", "参数数据类型不匹配"},
		{"invalid character 'a' looking for beginning of value", "请求数据不符合合法 JSON 规范"},
		{"json: syntax error", "JSON 请求参数解析失败"},
		{"record not found", "数据不存在"},
		{"strconv.ParseUint: parsing \"abc\": invalid syntax", "数值格式不合法"},
		{"UNIQUE constraint failed: users.student_id", "数据已存在，请勿重复添加"},
		{"账号不存在", "账号不存在"},
	}

	for _, tt := range tests {
		got := cleanErrorMessage(tt.input)
		if got != tt.expected {
			t.Errorf("cleanErrorMessage(%q) = %q, expected %q", tt.input, got, tt.expected)
		}
	}
}
