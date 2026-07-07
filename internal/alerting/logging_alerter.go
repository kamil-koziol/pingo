package alerting

import (
	"context"
	"log/slog"
)

type LoggingAlerter struct {
	Logger  *slog.Logger
	Alerter Alerter
}

func NewLoggingAlerter(logger *slog.Logger, alerter Alerter) *LoggingAlerter {
	return &LoggingAlerter{
		Logger:  logger,
		Alerter: alerter,
	}
}

func (l *LoggingAlerter) Publish(ctx context.Context, event Event) error {
	l.Logger.InfoContext(ctx, "sending", "type", event.Type())
	return l.Alerter.Publish(ctx, event)
}
