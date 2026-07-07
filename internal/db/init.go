package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
)

func Create(ctx context.Context) (*sql.DB, error) {
	conn, err := sql.Open("sqlite", "pingo.db")
	if err != nil {
		return nil, fmt.Errorf("unable to open db: %w", err)
	}

	// create tables
	ddl, err := os.ReadFile("schema.sql")
	if err != nil {
		return nil, fmt.Errorf("unable to load schema: %w", err)
	}

	if _, err := conn.ExecContext(ctx, string(ddl)); err != nil {
		log.Fatalf("unable to migrate db: %v", err)
	}

	if _, err := conn.ExecContext(ctx, "PRAGMA journal_mode=WAL;"); err != nil {
		return nil, fmt.Errorf("unable to pragma: %w", err)
	}

	if _, err := conn.ExecContext(ctx, "PRAGMA synchronous=NORMAL;"); err != nil {
		return nil, fmt.Errorf("unable to pragma: %w", err)
	}

	if _, err := conn.ExecContext(ctx, "PRAGMA busy_timeout=5000;"); err != nil {
		return nil, fmt.Errorf("unable to pragma: %w", err)
	}

	return conn, nil
}
