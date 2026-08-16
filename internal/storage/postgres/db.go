// Package postgres implements storage ports using PostgreSQL.
package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq"
)

// DB wraps *sql.DB with helpers used by repositories.
type DB struct {
	*sql.DB
}

// Open opens a PostgreSQL connection from the given DSN.
func Open(dsn string) (*DB, error) {
	if dsn == "" {
		return nil, fmt.Errorf("database DSN is empty")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return &DB{DB: db}, nil
}

// DSNFromEnv returns the database URL from CYBEROS_DATABASE_URL or a local default.
func DSNFromEnv() string {
	if v := os.Getenv("CYBEROS_DATABASE_URL"); v != "" {
		return v
	}
	// Default for local/dev (matches the user/db created in Phase 1C setup).
	return "postgres://cyberos:cyberos@localhost:5432/cyberos?sslmode=disable"
}

// Migrate applies the embedded SQL migration file.
// For Phase 1C we apply 001_init.sql in a simple, explicit way.
func (db *DB) Migrate(ctx context.Context, sqlContent string) error {
	_, err := db.ExecContext(ctx, sqlContent)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}
