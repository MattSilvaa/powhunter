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
	_, _, err := MigrateToLatest(database)
	return err
}

// MigrateToLatest applies every pending migration and reports the schema
// version before and after. A deploy step that only says "done" cannot be
// checked: reporting the versions is what makes it possible to confirm from a
// deploy log that the schema a release needs is actually present. Equal
// versions mean there was nothing to apply.
func MigrateToLatest(database *sql.DB) (int64, int64, error) {
	goose.SetBaseFS(MigrationsFS)
	defer goose.SetBaseFS(nil)

	if dialectErr := goose.SetDialect("postgres"); dialectErr != nil {
		return 0, 0, fmt.Errorf("setting goose dialect: %w", dialectErr)
	}

	goose.SetLogger(goose.NopLogger())

	// A database with no goose table yet reports version 0, which is the right
	// starting point rather than an error.
	before, versionErr := goose.GetDBVersion(database)
	if versionErr != nil {
		return 0, 0, fmt.Errorf("reading schema version: %w", versionErr)
	}

	if upErr := goose.Up(database, migrationsDir); upErr != nil {
		return before, before, fmt.Errorf("running migrations: %w", upErr)
	}

	after, versionErr := goose.GetDBVersion(database)
	if versionErr != nil {
		return before, before, fmt.Errorf("reading schema version after migrating: %w", versionErr)
	}

	return before, after, nil
}
