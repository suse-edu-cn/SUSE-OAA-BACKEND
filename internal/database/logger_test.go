package database

import (
	"context"
	"errors"
	"suseoaa/pkg/logger"
	"testing"
	"time"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func TestGormSlogLogger_Modes(t *testing.T) {
	l := NewGormSlogLogger(100*time.Millisecond, gormlogger.Warn)
	if l.SlowThreshold != 100*time.Millisecond {
		t.Errorf("expected slow threshold 100ms, got %v", l.SlowThreshold)
	}

	newLogger := l.LogMode(gormlogger.Info).(*GormSlogLogger)
	if newLogger.LogLevel != gormlogger.Info {
		t.Errorf("expected info log level, got %v", newLogger.LogLevel)
	}

	// 验证默认阈值兜底
	defaultL := NewGormSlogLogger(0, gormlogger.Warn)
	if defaultL.SlowThreshold != 200*time.Millisecond {
		t.Errorf("expected default slow threshold 200ms, got %v", defaultL.SlowThreshold)
	}
}

func TestGormSlogLogger_Trace(t *testing.T) {
	ctx := logger.WithTraceID(context.Background(), "test-trace-id-gorm-123")
	l := NewGormSlogLogger(50*time.Millisecond, gormlogger.Warn)

	fc := func() (string, int64) {
		return "SELECT * FROM users WHERE id = 1", 1
	}

	// 1. 静默模式
	silentL := l.LogMode(gormlogger.Silent).(*GormSlogLogger)
	silentL.Trace(ctx, time.Now().Add(-100*time.Millisecond), fc, nil)

	// 2. 正常快查询（未超阈值，不打印）
	l.Trace(ctx, time.Now().Add(-10*time.Millisecond), fc, nil)

	// 3. 慢查询（超过 50ms 阈值，输出 Warn）
	l.Trace(ctx, time.Now().Add(-80*time.Millisecond), fc, nil)

	// 4. gorm.ErrRecordNotFound（正常业务结果，不记录为 Error）
	l.Trace(ctx, time.Now().Add(-10*time.Millisecond), fc, gorm.ErrRecordNotFound)

	// 5. 真实的 SQL 错误（输出 Error）
	realErr := errors.New("connection reset by peer")
	l.Trace(ctx, time.Now().Add(-10*time.Millisecond), fc, realErr)

	// 6. Info 级别输出日常信息
	infoL := l.LogMode(gormlogger.Info).(*GormSlogLogger)
	infoL.Trace(ctx, time.Now().Add(-10*time.Millisecond), fc, nil)
	infoL.Info(ctx, "info test message %s", "arg")
	infoL.Warn(ctx, "warn test message %s", "arg")
	infoL.Error(ctx, "error test message %s", "arg")
}
