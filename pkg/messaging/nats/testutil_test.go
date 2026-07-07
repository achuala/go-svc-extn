package nats_test

import (
	"log/slog"
	"os"

	"github.com/go-kratos/kratos/v3/log"
)

func testLogger() *slog.Logger {
	return log.NewLogger(log.NewHandler(log.WithWriter(os.Stdout)))
}
