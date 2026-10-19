package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"suseoaa/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestHealthHandler_Livez(t *testing.T) {
	h := NewHealthHandler(nil, nil)

	r := gin.New()
	r.GET("/livez", h.Livez)

	req := httptest.NewRequest(http.MethodGet, "/livez", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp response.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response JSON: %v", err)
	}

	if resp.Code != 200 || resp.Message != "success" || resp.Data != nil {
		t.Fatalf("expected code 200, message success, data nil, got %+v", resp)
	}
}

func TestHealthHandler_Healthz_UnhealthyWhenNilClients(t *testing.T) {
	h := NewHealthHandler(nil, nil)

	r := gin.New()
	r.GET("/healthz", h.Healthz)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	// 当核心客户端未初始化时，必须返回 503 Service Unavailable
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", w.Code)
	}

	var resp struct {
		Code    int               `json:"code"`
		Message string            `json:"message"`
		Data    map[string]string `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response JSON: %v", err)
	}

	if resp.Code != 503 {
		t.Fatalf("expected code 503, got %d", resp.Code)
	}
	if resp.Data["mysql"] != "mysql client not initialized" {
		t.Fatalf("expected mysql failure message, got %q", resp.Data["mysql"])
	}
	if resp.Data["redis"] != "redis client not initialized" {
		t.Fatalf("expected redis failure message, got %q", resp.Data["redis"])
	}
}

func TestHealthHandler_Healthz_UnhealthyWhenRedisDown(t *testing.T) {
	// 指向一个不可用的端口模拟 Redis 宕机
	deadRdb := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:54321",
	})
	defer deadRdb.Close()

	h := NewHealthHandler(nil, deadRdb)

	r := gin.New()
	r.GET("/healthz", h.Healthz)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503 when redis is down, got %d", w.Code)
	}

	var resp struct {
		Code    int               `json:"code"`
		Message string            `json:"message"`
		Data    map[string]string `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response JSON: %v", err)
	}

	if resp.Data["redis"] == "" {
		t.Fatalf("expected redis error details in data, got empty")
	}
}
