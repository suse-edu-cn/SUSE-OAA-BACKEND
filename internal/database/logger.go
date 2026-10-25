package database

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// GormSlogLogger 实现 GORM 的 logger.Interface，将 SQL 执行日志、慢查警告与执行异常桥接至标准库 log/slog
// 自动继承当前 Context 中的 trace_id，形成全链路结构化追踪
type GormSlogLogger struct {
	SlowThreshold time.Duration
	LogLevel      gormlogger.LogLevel
}

func NewGormSlogLogger(slowThreshold time.Duration, level gormlogger.LogLevel) *GormSlogLogger {
	if slowThreshold <= 0 {
		slowThreshold = 200 * time.Millisecond // 默认 200ms 判定为慢 SQL
	}
	return &GormSlogLogger{
		SlowThreshold: slowThreshold,
		LogLevel:      level,
	}
}

func (l *GormSlogLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	newLogger := *l
	newLogger.LogLevel = level
	return &newLogger
}

func (l *GormSlogLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Info {
		slog.InfoContext(ctx, fmt.Sprintf(msg, data...))
	}
}

func (l *GormSlogLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Warn {
		slog.WarnContext(ctx, fmt.Sprintf(msg, data...))
	}
}

func (l *GormSlogLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Error {
		slog.ErrorContext(ctx, fmt.Sprintf(msg, data...))
	}
}

// Trace 捕获每一条 SQL 的执行耗时与影响行数
func (l *GormSlogLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= gormlogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	switch {
	// 1. 真实 SQL 执行报错（排除正常的 ErrRecordNotFound）
	case err != nil && !errors.Is(err, gorm.ErrRecordNotFound) && l.LogLevel >= gormlogger.Error:
		sql, rows := fc()
		slog.ErrorContext(ctx, "GORM SQL 异常",
			slog.String("err", err.Error()),
			slog.String("sql", sql),
			slog.Int64("rows", rows),
			slog.Duration("latency", elapsed),
			slog.Int64("latency_ms", elapsed.Milliseconds()),
		)

	// 2. 慢 SQL 告警（超过配置的阈值，例如 200ms）
	case l.SlowThreshold != 0 && elapsed > l.SlowThreshold && l.LogLevel >= gormlogger.Warn:
		sql, rows := fc()
		slog.WarnContext(ctx, "GORM 慢查询告警",
			slog.String("sql", sql),
			slog.Int64("rows", rows),
			slog.Duration("latency", elapsed),
			slog.Int64("latency_ms", elapsed.Milliseconds()),
			slog.Duration("slow_threshold", l.SlowThreshold),
		)

	// 3. Info 级别输出日常 SQL 追踪
	case l.LogLevel == gormlogger.Info:
		sql, rows := fc()
		slog.InfoContext(ctx, "GORM 执行",
			slog.String("sql", sql),
			slog.Int64("rows", rows),
			slog.Duration("latency", elapsed),
			slog.Int64("latency_ms", elapsed.Milliseconds()),
		)
	}
}
