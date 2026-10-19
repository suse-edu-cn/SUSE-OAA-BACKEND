package router

import (
	"net/http/httptest"
	"testing"

	"suseoaa/internal/handler"

	"github.com/gin-gonic/gin"
)

func TestRouter_HealthzAndLivez(t *testing.T) {
	gin.SetMode(gin.TestMode)

	healthHandler := handler.NewHealthHandler(nil, nil)
	authHandler := &handler.AuthHandler{JwtSecret: "test_secret"}

	total := &handler.TotalHandler{
		Health: healthHandler,
		Auth:   authHandler,
		User:   &handler.UserHandler{},
		Department: &handler.DepartmentHandler{},
		Role: &handler.RoleHandler{},
		Announcement: &handler.AnnouncementHandler{},
		Term: &handler.TermHandler{},
		File: &handler.FileHandler{},
	}

	r := RouterInit(total, nil, nil)

	// Case 1: GET /livez
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("GET", "/livez", nil)
	r.ServeHTTP(w1, req1)
	if w1.Code != 200 {
		t.Fatalf("expected /livez 200, got %d", w1.Code)
	}

	// Case 2: GET /v2/livez
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/v2/livez", nil)
	r.ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Fatalf("expected /v2/livez 200, got %d", w2.Code)
	}

	// Case 3: GET /v2/healthz
	w3 := httptest.NewRecorder()
	req3 := httptest.NewRequest("GET", "/v2/healthz", nil)
	r.ServeHTTP(w3, req3)
	if w3.Code != 503 {
		t.Fatalf("expected /v2/healthz 503 with nil clients, got %d", w3.Code)
	}
}
