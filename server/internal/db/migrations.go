package db

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
)

// MigrationsFS holds the goose migrations so they can be applied from Go
// without depending on the goose binary or the repository layout on disk.
//
//go:embed all:migrations
var MigrationsFS embed.FS

// migrationsDir is the path of the migrations inside MigrationsFS.
const migrationsDir = "migrations"

// Migrate applies every pending migration to the given database.
func Migrate(database *sql.DB) error {
	goose.SetBaseFS(MigrationsFS)
	defer goose.SetBaseFS(nil)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("setting goose dialect: %w", err)
	}

	goose.SetLogger(goose.NopLogger())

	if err := goose.Up(database, migrationsDir); err != nil {
		return fmt.Errorf("running migrations: %w", err)
	}

	return nil
}
