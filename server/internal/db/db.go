package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/MattSilvaa/powhunter/internal/config"

	_ "github.com/lib/pq"
)

const (
	dbTimeout       = 10 * time.Second
	maxOpenConns    = 25
	maxIdleConns    = 25
	connMaxLifetime = 5 * time.Minute
	connMaxIdleTime = 2 * time.Minute
)

// New opens a verified connection pool using the process configuration.
//
// Credentials come from config.Load, which refuses to fall back to defaults in
// production. Previously a missing DB_PASSWORD silently connected with a
// well-known credential instead of failing.
func New() (*sql.DB, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("loading configuration: %w", err)
	}

	return Open(cfg.Database)
}

// Open connects to the given database and verifies the connection.
func Open(settings config.Database) (*sql.DB, error) {
	database, openErr := sql.Open("postgres", settings.DSN())
	if openErr != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", openErr)
	}

	database.SetMaxOpenConns(maxOpenConns)
	database.SetMaxIdleConns(maxIdleConns)
	database.SetConnMaxLifetime(connMaxLifetime)
	database.SetConnMaxIdleTime(connMaxIdleTime)

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	if pingErr := database.PingContext(ctx); pingErr != nil {
		if closeErr := database.Close(); closeErr != nil {
			return nil, fmt.Errorf("failed to ping database (%w) and to close it: %w", pingErr, closeErr)
		}

		return nil, fmt.Errorf("failed to ping database: %w", pingErr)
	}

	return database, nil
}
