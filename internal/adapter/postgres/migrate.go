package postgres

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// RunMigrations применяет новые файлы из migrations/ по порядку имён;
// применённые фиксируются в schema_migrations.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			name       TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`); err != nil {
		return fmt.Errorf("ensure migrations table: %w", err)
	}

	applied, err := loadApplied(ctx, pool)
	if err != nil {
		return err
	}

	names, err := collectSQLFiles()
	if err != nil {
		return err
	}

	for _, name := range names {
		if applied[name] {
			continue
		}
		if err := applyFile(ctx, pool, name); err != nil {
			return fmt.Errorf("migration %s: %w", name, err)
		}
		fmt.Printf("migration applied: %s\n", name)
	}
	return nil
}

func collectSQLFiles() ([]string, error) {
	var names []string
	err := fs.WalkDir(migrationsFS, "migrations", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".sql") {
			names = append(names, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk migrations dir: %w", err)
	}
	sort.Strings(names)
	return names, nil
}

func loadApplied(ctx context.Context, pool *pgxpool.Pool) (map[string]bool, error) {
	rows, err := pool.Query(ctx, `SELECT name FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("load applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		applied[name] = true
	}
	return applied, rows.Err()
}

func applyFile(ctx context.Context, pool *pgxpool.Pool, path string) error {
	data, err := migrationsFS.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Без аргументов pgx использует simple protocol — допускает несколько statement'ов
	if _, err := tx.Exec(ctx, string(data)); err != nil {
		return fmt.Errorf("exec: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (name) VALUES ($1)`, path); err != nil {
		return fmt.Errorf("record migration: %w", err)
	}
	return tx.Commit(ctx)
}
