// Command migrate applies pending database migrations.
//
// Deploys previously required running the goose binary out of band, which meant
// the schema step was unversioned and easy to skip. This uses the same embedded
// migrations the tests do.
package main

import (
	"log/slog"
	"os"

	"github.com/MattSilvaa/powhunter/internal/config"
	"github.com/MattSilvaa/powhunter/internal/db"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := run(); err != nil {
		logger.Error("migration failed", "error", err)
		os.Exit(1)
	}

	logger.Info("migrations applied")
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	conn, err := db.Open(cfg.Database)
	if err != nil {
		return err
	}
	defer conn.Close()

	return db.Migrate(conn)
}
