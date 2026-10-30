package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"suseoaa/pkg/response"
	"suseoaa/pkg/utils"

	"github.com/gin-gonic/gin"
)

func TestJWTAuth_Middleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "jwt-test-secret-key-123456"

	r := gin.New()
	r.Use(JWTAuth(secret))
	r.GET("/protected", func(c *gin.Context) {
		userID := c.GetUint64("user_id")
		name := c.GetString("name")
		studentID := c.GetString("student_id")
		response.Success(c, gin.H{
			"user_id":    userID,
			"name":       name,
			"student_id": studentID,
		})
	})

	t.Run("未携带 Token 拦截", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
		var resp response.Response
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Message != "请求未携带Token，访问被拒绝" {
			t.Fatalf("expected '请求未携带Token，访问被拒绝', got %q", resp.Message)
		}
	})

	t.Run("Token 格式非 Bearer", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Basic abcdefg")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
		var resp response.Response
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Message != "Token 格式错误，必须是 Bearer 模式" {
			t.Fatalf("expected 'Token 格式错误，必须是 Bearer 模式', got %q", resp.Message)
		}
	})

	t.Run("已过期 Token 拦截并返回中文", func(t *testing.T) {
		tokenStr, _ := utils.GenerateToken("张三", 100, "20240001", secret, -1)
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+tokenStr)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
		var resp response.Response
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Message != "登录状态已过期，请重新登录" {
			t.Fatalf("expected '登录状态已过期，请重新登录', got %q", resp.Message)
		}
	})

	t.Run("无效/损坏 Token 拦截并返回中文", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer invalid-token-12345")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
		var resp response.Response
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Message != "登录凭证无效，请重新登录" {
			t.Fatalf("expected '登录凭证无效，请重新登录', got %q", resp.Message)
		}
	})

	t.Run("合法 Token 放行并注入上下文", func(t *testing.T) {
		tokenStr, _ := utils.GenerateToken("张三", 100, "20240001", secret, 60)
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+tokenStr)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		var resp struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Data    struct {
				UserID    uint64 `json:"user_id"`
				Name      string `json:"name"`
				StudentID string `json:"student_id"`
			} `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Data.UserID != 100 || resp.Data.Name != "张三" || resp.Data.StudentID != "20240001" {
			t.Fatalf("unexpected context injection: %+v", resp.Data)
		}
	})
}
