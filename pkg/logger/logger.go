package logger

import (
	"context"
	"log/slog"
	"os"
)

type contextKey string

const (
	TraceIDKey    contextKey = "trace_id"
	TraceIDHeader            = "X-Request-ID"
	UserIDKey     contextKey = "user_id"
)

func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

func GetTraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(TraceIDKey).(string); ok {
		return v
	}
	return ""
}

func WithUserID(ctx context.Context, userID uint64) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

func GetUserID(ctx context.Context) uint64 {
	if ctx == nil {
		return 0
	}
	if v, ok := ctx.Value(UserIDKey).(uint64); ok {
		return v
	}
	return 0
}

type traceHandler struct {
	slog.Handler
}

func (h *traceHandler) Handle(ctx context.Context, r slog.Record) error {
	if traceID := GetTraceID(ctx); traceID != "" {
		r.AddAttrs(slog.String("trace_id", traceID))
	}
	if userID := GetUserID(ctx); userID > 0 {
		r.AddAttrs(slog.Uint64("user_id", userID))
	}
	return h.Handler.Handle(ctx, r)
}

func Init(level slog.Level) {
	opts := &slog.HandlerOptions{
		Level: level,
	}
	baseHandler := slog.NewJSONHandler(os.Stdout, opts)
	handler := &traceHandler{Handler: baseHandler}
	logger := slog.New(handler)
	slog.SetDefault(logger)
}

func InfoContext(ctx context.Context, msg string, args ...any) {
	slog.InfoContext(ctx, msg, args...)
}

func WarnContext(ctx context.Context, msg string, args ...any) {
	slog.WarnContext(ctx, msg, args...)
}

func ErrorContext(ctx context.Context, msg string, args ...any) {
	slog.ErrorContext(ctx, msg, args...)
}

func DebugContext(ctx context.Context, msg string, args ...any) {
	slog.DebugContext(ctx, msg, args...)
}

func Info(msg string, args ...any) {
	slog.Info(msg, args...)
}

func Warn(msg string, args ...any) {
	slog.Warn(msg, args...)
}

func Error(msg string, args ...any) {
	slog.Error(msg, args...)
}
