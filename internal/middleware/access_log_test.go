package middleware

import (
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"suseoaa/pkg/logger"

	"github.com/gin-gonic/gin"
)


func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	logger.Init(slog.LevelDebug)

	r := gin.New()
	r.Use(Trace())
	r.Use(AccessLog())
	r.Use(gin.Recovery())


	r.GET("/test/success", func(c *gin.Context) {
		c.String(http.StatusOK, "success response")
	})


	r.GET("/test/bad-request", func(c *gin.Context) {
		_ = c.Error(errors.New("mock client validation error"))
		c.String(http.StatusBadRequest, "bad request")
	})


	r.GET("/test/server-error", func(c *gin.Context) {
		c.String(http.StatusInternalServerError, "server error")
	})


	r.GET("/test/panic", func(c *gin.Context) {
		panic("database connection unexpected panic!")
	})

	return r
}

func TestAccessLog_Success(t *testing.T) {
	r := setupRouter()

	req := httptest.NewRequest(http.MethodGet, "/test/success?query=hello", nil)
	req.Header.Set(logger.TraceIDHeader, "trace-id-valid-for-test-123456")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)


	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}


	respTraceID := w.Header().Get(logger.TraceIDHeader)
	if respTraceID != "trace-id-valid-for-test-123456" {
		t.Fatalf("expected trace id to be returned, got %q", respTraceID)
	}
}

func TestAccessLog_BadRequest(t *testing.T) {
	r := setupRouter()

	req := httptest.NewRequest(http.MethodGet, "/test/bad-request", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestAccessLog_ServerError(t *testing.T) {
	r := setupRouter()

	req := httptest.NewRequest(http.MethodGet, "/test/server-error", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}
}

func TestAccessLog_PanicRecovery(t *testing.T) {
	r := setupRouter()

	req := httptest.NewRequest(http.MethodGet, "/test/panic", nil)
	w := httptest.NewRecorder()


	r.ServeHTTP(w, req)


	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500 after panic recovery, got %d", w.Code)
	}
}
