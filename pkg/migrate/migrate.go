package migrate

import (
	"cmp"
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

const migrateTable = "migrate"

var logger *slog.Logger = slog.Default()

func initialMigration(ctx context.Context, db *sql.DB) error {
	initialTable := fmt.Sprintf(
		`
	CREATE TABLE IF NOT EXISTS %s (	
    	id INTEGER
	);
	
	INSERT INTO %s (id) VALUES(0)
	`, migrateTable, migrateTable)

	if _, err := db.ExecContext(ctx, initialTable); err != nil {
		return fmt.Errorf("create initial table: %w", err)
	}

	return nil
}

func Current(ctx context.Context, db *sql.DB) (int, error) {
	q := fmt.Sprintf("SELECT id FROM %s LIMIT 1;", migrateTable)

	row := db.QueryRowContext(ctx, q)
	var current int
	err := row.Scan(
		&current,
	)

	return current, err
}

func setCurrent(ctx context.Context, db *sql.DB, id int) error {
	q := fmt.Sprintf("UPDATE %s SET id=%d;", migrateTable, id)

	_, err := db.ExecContext(ctx, q)
	return err
}

type Migration struct {
	ID   int
	Name string
	Path string
}

const (
	migrationSep = "_"
	migrationDir = "migrations"
)

func GetAll(fsys fs.FS) ([]Migration, error) {
	entries, err := fs.ReadDir(fsys, migrationDir)
	if err != nil {
		return nil, fmt.Errorf("read migrations dir: %w", err)
	}

	migrations := make([]Migration, len(entries))
	for i, entry := range entries {
		if entry.IsDir() {
			return nil, fmt.Errorf("directory not allowed: %q", entry.Name())
		}

		idS, name, found := strings.Cut(entry.Name(), migrationSep)
		if !found {
			return nil, fmt.Errorf("migration file must be separated by %q", migrationSep)
		}

		sqlSuffix := ".sql"
		name, isSQL := strings.CutSuffix(name, sqlSuffix)
		if !isSQL {
			return nil, fmt.Errorf("migrations must end with %s, file: %s", sqlSuffix, entry.Name())
		}

		id, err := strconv.Atoi(idS)
		if err != nil {
			return nil, fmt.Errorf("id must be an integer: %w", err)
		}

		migrations[i] = Migration{
			ID:   id,
			Name: name,
			Path: filepath.Join(migrationDir, entry.Name()),
		}
	}

	// sort by the id
	slices.SortFunc(migrations, func(l Migration, r Migration) int {
		return cmp.Compare(l.ID, r.ID)
	})

	// make sure migrations start from 1
	if migrations[0].ID != 1 {
		return nil, fmt.Errorf("migrations ids must start from 1, found: %d", migrations[0].ID)
	}

	// make sure increasing and by 1
	current := 0
	for _, migration := range migrations {
		if migration.ID-1 != current {
			return nil, fmt.Errorf("migrations must be in increasing order: id: %d, name: %s", migration.ID, migration.Name)
		}

		current += 1
	}

	return migrations, nil
}

func runMigration(ctx context.Context, db *sql.DB, fsys fs.FS, migration *Migration) error {
	query, err := fs.ReadFile(fsys, migration.Path)
	if err != nil {
		return fmt.Errorf("read migration contents: %w", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.ExecContext(ctx, string(query)); err != nil {
		return fmt.Errorf("execute query: %w", err)
	}

	return tx.Commit()
}

func migrate(ctx context.Context, db *sql.DB, fsys fs.FS) error {
	current, err := Current(ctx, db)
	if err != nil {
		return fmt.Errorf("current migration: %w", err)
	}

	migrations, err := GetAll(fsys)
	if err != nil {
		return fmt.Errorf("get all migrations: %w", err)
	}

	// example: 1/migration_name up
	const format string = "%d/%s %s"

	for _, migration := range migrations {
		if migration.ID <= current {
			logger.DebugContext(ctx, fmt.Sprintf(format, migration.ID, migration.Name, "skip"))
			continue
		}

		if err := runMigration(ctx, db, fsys, &migration); err != nil {
			return fmt.Errorf("running migration: %w", err)
		}

		if err := setCurrent(ctx, db, migration.ID); err != nil {
			return fmt.Errorf("set current migration id to %d: %w", migration.ID, err)
		}
		logger.DebugContext(ctx, fmt.Sprintf(format, migration.ID, migration.Name, "up"))
	}

	return nil
}

func Run(ctx context.Context, db *sql.DB, fsys fs.FS) error {
	if err := initialMigration(ctx, db); err != nil {
		return fmt.Errorf("initial migration: %w", err)
	}

	if err := migrate(ctx, db, fsys); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	return nil
}
