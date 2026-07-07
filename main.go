package main

import (
	"context"
	"database/sql"
	_ "embed"
	"log"
	"log/slog"
	"os"

	"github.com/kamil-koziol/pingo/internal/alerting"
	"github.com/kamil-koziol/pingo/internal/configuration"
	"github.com/kamil-koziol/pingo/internal/db"
	"github.com/kamil-koziol/pingo/internal/monitoring"
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var ddl string

func main() {
	f, err := os.Open("config.yml")
	if err != nil {
		log.Fatalf("unable to read config: %v", err)
	}

	config, err := configuration.Parse(f)
	if err != nil {
		log.Fatalf("unable to parse config: %v", err)
	}

	var alerter alerting.Alerter

	alerters := make([]alerting.Alerter, len(config.Alerts))
	for i, alert := range config.Alerts {
		alerter, err := alert.Build()
		if err != nil {
			log.Fatalf("unable to build alerter: %v", err)
		}
		alerters[i] = alerter
	}
	alerter = alerting.NewMultiAlerter(alerters...)

	logger := slog.Default()
	alerter = alerting.NewLoggingAlerter(logger, alerter)

	conn, err := sql.Open("sqlite", "pingo.db")
	if err != nil {
		log.Fatalf("unable to open db: %v", err)
	}

	// create tables
	ctx := context.Background()
	if _, err := conn.ExecContext(ctx, ddl); err != nil {
		log.Fatalf("unable to migrate db: %v", err)
	}

	q := db.New(conn)

	for _, check := range config.Checks {
		service, err := q.UpsertService(ctx, db.UpsertServiceParams{
			Name:            check.Name,
			Url:             check.URL.String(),
			IntervalSeconds: int64(check.Interval.Seconds()),
			ExpectedStatus:  int64(check.ExpectedStatus),
		})

		if err != nil {
			log.Fatalf("unable to upsert service: %v", err)
		}

		m := monitoring.NewMonitor(service, q, alerter)
		go m.Run(ctx)
	}

	<-ctx.Done()
}
