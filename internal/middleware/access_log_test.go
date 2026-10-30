package middleware

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"suseoaa/pkg/logger"
	"suseoaa/pkg/response"

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

	r.GET("/test/response-bad-request", func(c *gin.Context) {
		response.BadRequest(c, "业务参数校验未通过")
	})


	r.GET("/test/server-error", func(c *gin.Context) {
		c.String(http.StatusInternalServerError, "server error")
	})


	r.GET("/test/panic", func(c *gin.Context) {
		panic("database connection unexpected panic!")
	})

	r.GET("/test/with-user", func(c *gin.Context) {
		c.Set("user_id", uint64(8848))
		c.String(http.StatusOK, "user response")
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

func TestAccessLog_ResponseError(t *testing.T) {
	r := setupRouter()

	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	slog.SetDefault(slog.New(handler))

	req := httptest.NewRequest(http.MethodGet, "/test/response-bad-request", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, "error=业务参数校验未通过") && !strings.Contains(logOutput, "业务参数校验未通过") {
		t.Fatalf("expected log output to contain error message, got: %s", logOutput)
	}
}

func TestAccessLog_WithUserID(t *testing.T) {
	r := setupRouter()

	var buf bytes.Buffer
	baseHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	loggerInitWithHandler := slog.New(&testTraceHandler{Handler: baseHandler})
	slog.SetDefault(loggerInitWithHandler)

	req := httptest.NewRequest(http.MethodGet, "/test/with-user", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, `"user_id":8848`) {
		t.Fatalf("expected log output to contain user_id 8848, got: %s", logOutput)
	}
}

type testTraceHandler struct {
	slog.Handler
}

func (h *testTraceHandler) Handle(ctx context.Context, r slog.Record) error {
	if traceID := logger.GetTraceID(ctx); traceID != "" {
		r.AddAttrs(slog.String("trace_id", traceID))
	}
	if userID := logger.GetUserID(ctx); userID > 0 {
		r.AddAttrs(slog.Uint64("user_id", userID))
	}
	return h.Handler.Handle(ctx, r)
}
