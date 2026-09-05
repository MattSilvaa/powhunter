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

	if err := run(logger); err != nil {
		// A non-zero exit is what makes this usable as a deploy gate: the
		// release stops rather than serving against a schema it does not have.
		logger.Error("migration failed", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	conn, err := db.Open(cfg.Database)
	if err != nil {
		return err
	}
	defer conn.Close()

	from, to, err := db.MigrateToLatest(conn)
	if err != nil {
		return err
	}

	if from == to {
		logger.Info("schema already up to date", "version", to)
		return nil
	}

	logger.Info("migrations applied", "from_version", from, "to_version", to)

	return nil
}
