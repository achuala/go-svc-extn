package messaging

import (
	"github.com/ThreeDotsLabs/watermill"
	"github.com/go-kratos/kratos/v2/log"
)

type WatermillLoggerAdapter struct {
	logHelper *log.Helper
	fields    watermill.LogFields
}

func NewWatermillLoggerAdapter(logger log.Logger) watermill.LoggerAdapter {
	return &WatermillLoggerAdapter{logHelper: log.NewHelper(logger)}
}

// Debug implements watermill.LoggerAdapter.
func (w *WatermillLoggerAdapter) Debug(msg string, fields watermill.LogFields) {
	if w.logHelper.Enabled(log.LevelDebug) {
		logFields := logAttrsFromFields(fields)
		w.logHelper.Debug(msg, logFields)
	}
}

// Error implements watermill.LoggerAdapter.
func (w *WatermillLoggerAdapter) Error(msg string, err error, fields watermill.LogFields) {
	logFields := logAttrsFromFields(fields)
	w.logHelper.Error(msg, logFields)
}

// Info implements watermill.LoggerAdapter.
func (w *WatermillLoggerAdapter) Info(msg string, fields watermill.LogFields) {
	if w.logHelper.Enabled(log.LevelInfo) {
		logFields := logAttrsFromFields(fields)
		w.logHelper.Info(msg, logFields)
	}
}

// Trace implements watermill.LoggerAdapter.
func (w *WatermillLoggerAdapter) Trace(msg string, fields watermill.LogFields) {
	if w.logHelper.Enabled(log.LevelDebug) {
		logFields := logAttrsFromFields(fields)
		w.logHelper.Debug(msg, logFields)
	}
}

// With implements watermill.LoggerAdapter.
func (w *WatermillLoggerAdapter) With(fields watermill.LogFields) watermill.LoggerAdapter {
	return &WatermillLoggerAdapter{logHelper: w.logHelper, fields: fields}
}

func logAttrsFromFields(fields watermill.LogFields) []any {
	result := make([]any, 0, len(fields)*2)

	for key, value := range fields {
		result = append(result, key, value)
	}

	return result
}
