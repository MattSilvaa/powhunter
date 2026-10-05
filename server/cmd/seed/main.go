// Command seed applies the bundled resort catalog to the database.
//
// cmd/migrate already does this on every deploy; this is for applying the
// catalog by hand, for example to a local database.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/MattSilvaa/powhunter/internal/db"
	dbgen "github.com/MattSilvaa/powhunter/internal/db/generated"
	"github.com/MattSilvaa/powhunter/internal/seed"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := run(logger); err != nil {
		logger.Error("resort seeding failed", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	dbConn, err := db.New()
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer dbConn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err = seed.Resorts(ctx, dbgen.New(dbConn), logger)

	return err
}
