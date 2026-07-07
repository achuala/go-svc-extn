package messaging

import (
	"log/slog"

	"github.com/ThreeDotsLabs/watermill"
)

type WatermillLoggerAdapter struct {
	logger *slog.Logger
	fields watermill.LogFields
}

func NewWatermillLoggerAdapter(logger *slog.Logger) watermill.LoggerAdapter {
	if logger == nil {
		logger = slog.Default()
	}
	return &WatermillLoggerAdapter{logger: logger}
}

func (w *WatermillLoggerAdapter) Debug(msg string, fields watermill.LogFields) {
	w.logger.Debug(msg, logAttrsFromFields(fields)...)
}

func (w *WatermillLoggerAdapter) Error(msg string, err error, fields watermill.LogFields) {
	args := logAttrsFromFields(fields)
	if err != nil {
		args = append(args, "error", err)
	}
	w.logger.Error(msg, args...)
}

func (w *WatermillLoggerAdapter) Info(msg string, fields watermill.LogFields) {
	w.logger.Info(msg, logAttrsFromFields(fields)...)
}

func (w *WatermillLoggerAdapter) Trace(msg string, fields watermill.LogFields) {
}

func (w *WatermillLoggerAdapter) With(fields watermill.LogFields) watermill.LoggerAdapter {
	return &WatermillLoggerAdapter{logger: w.logger, fields: fields}
}

func logAttrsFromFields(fields watermill.LogFields) []any {
	result := make([]any, 0, len(fields)*2)

	for key, value := range fields {
		result = append(result, key, value)
	}

	return result
}
