package postgres

import (
	"context"
	"os"
	"path/filepath"
)

// ApplyMigrations reads and applies SQL migration files from the given directory
// in lexical order. For Phase 1C we primarily use 001_init.sql.
func ApplyMigrations(ctx context.Context, db *DB, migrationsDir string) error {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if filepath.Ext(name) != ".sql" {
			continue
		}
		content, err := os.ReadFile(filepath.Join(migrationsDir, name))
		if err != nil {
			return err
		}
		if err := db.Migrate(ctx, string(content)); err != nil {
			return err
		}
	}
	return nil
}
