package contextx

import (
	"context"
	"log/slog"
)

type loggerContextKey struct{}

var loggerKey loggerContextKey

func WithLogger(ctx context.Context, log *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, log)
}

func Logger(ctx context.Context) *slog.Logger {
	if log, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return log
	}

	return slog.Default()
}
