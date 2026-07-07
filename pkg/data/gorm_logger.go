package data

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/go-kratos/kratos/v3/log"
	glogger "gorm.io/gorm/logger"
)

// GormLogger is a custom logger for GORM that uses Kratos logger
type GormLogger struct {
	logger        *slog.Logger
	LogLevel      glogger.LogLevel
	SlowThreshold time.Duration
	Enabled       *atomic.Bool
}

// NewGormLogger creates a new GORM logger that uses Kratos logger
func NewGormLogger(logger *slog.Logger) *GormLogger {
	enabled := &atomic.Bool{}
	enabled.Store(true)

	if logger == nil {
		logger = slog.Default()
	}

	return &GormLogger{
		logger:        logger.With("module", "gorm"),
		LogLevel:      glogger.Info,
		SlowThreshold: 200 * time.Millisecond,
		Enabled:       enabled,
	}
}

// LogMode sets the log mode
func (l *GormLogger) LogMode(level glogger.LogLevel) glogger.Interface {
	newLogger := *l
	newLogger.LogLevel = level
	return &newLogger
}

// EnableLogging turns on SQL query logging
func (l *GormLogger) EnableLogging() {
	l.Enabled.Store(true)
}

// DisableLogging turns off SQL query logging
func (l *GormLogger) DisableLogging() {
	l.Enabled.Store(false)
}

// IsLogEnabled checks if logging is currently enabled
func (l *GormLogger) IsLogEnabled() bool {
	return l.Enabled.Load()
}

// SetSlowThreshold sets the threshold for slow query logging
func (l *GormLogger) SetSlowThreshold(threshold time.Duration) {
	l.SlowThreshold = threshold
}

// Info prints info
func (l *GormLogger) Info(ctx context.Context, msg string, data ...any) {
	if l.IsLogEnabled() && l.LogLevel >= glogger.Info {
		l.logger.Log(ctx, log.LevelInfo, msg, data...)
	}
}

// Warn prints warn messages
func (l *GormLogger) Warn(ctx context.Context, msg string, data ...any) {
	if l.IsLogEnabled() && l.LogLevel >= glogger.Warn {
		l.logger.Log(ctx, log.LevelWarn, msg, data...)
	}
}

// Error prints error messages
func (l *GormLogger) Error(ctx context.Context, msg string, data ...any) {
	if l.IsLogEnabled() && l.LogLevel >= glogger.Error {
		l.logger.Log(ctx, log.LevelError, msg, data...)
	}
}

// Trace prints trace messages
func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if !l.IsLogEnabled() {
		return
	}

	elapsed := time.Since(begin)

	if err != nil && l.LogLevel < glogger.Error {
		return
	}
	if err == nil && l.LogLevel < glogger.Info && elapsed < l.SlowThreshold {
		return
	}

	sql, rows := fc()

	if err != nil {
		l.logger.Log(ctx, log.LevelError, "gorm query failed",
			"sql", sql,
			"rows", rows,
			"elapsed", elapsed,
			"err", err,
		)
		return
	}

	if elapsed > l.SlowThreshold {
		l.logger.Log(ctx, log.LevelWarn, "gorm slow query",
			"sql", sql,
			"rows", rows,
			"elapsed", elapsed,
			"slow_query", true,
		)
	}
}
