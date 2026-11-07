//go:build integration
// +build integration

package db_test

import (
	"testing"

	"github.com/MattSilvaa/powhunter/internal/db"
	"github.com/MattSilvaa/powhunter/internal/testutil"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The pre-deploy step reports the schema version so a deploy log can be checked
// for whether the release's schema is actually present. A release shipped
// against an un-migrated database once already, and the only symptom was the
// forecaster failing on a missing column half a day later.
func TestMigrateToLatestReportsAVersion(t *testing.T) {
	database, _, teardown := testutil.SetupTestDB(t)
	defer teardown()

	from, to, err := db.MigrateToLatest(database)

	require.NoError(t, err)
	assert.Positive(t, to, "a migrated database should report a non-zero version")
	assert.GreaterOrEqual(t, to, from)
}

// Re-running must be a no-op, since every deploy runs this whether or not the
// release carries a migration.
func TestMigrateToLatestIsIdempotent(t *testing.T) {
	database, _, teardown := testutil.SetupTestDB(t)
	defer teardown()

	_, first, err := db.MigrateToLatest(database)
	require.NoError(t, err)

	from, second, err := db.MigrateToLatest(database)

	require.NoError(t, err)
	assert.Equal(t, first, from, "the second run should start where the first finished")
	assert.Equal(t, first, second, "a second run must not change the schema version")
}

// Alerts now go only to confirmed accounts. Everyone who signed up before that
// rule must stay confirmed, or the deploy would silently stop their alerts.
func TestMigration008ConfirmsExistingAccounts(t *testing.T) {
	database, _, teardown := testutil.SetupTestDB(t)
	defer teardown()

	goose.SetBaseFS(db.MigrationsFS)
	defer goose.SetBaseFS(nil)

	require.NoError(t, goose.SetDialect("postgres"))
	require.NoError(t, goose.DownTo(database, "migrations", 7))

	_, err := database.Exec(`INSERT INTO users (email, phone) VALUES ('existing@example.com', '+15552345678')`)
	require.NoError(t, err)

	require.NoError(t, goose.UpTo(database, "migrations", 8))

	var confirmed bool

	require.NoError(t, database.QueryRow(
		`SELECT email_verified_at IS NOT NULL FROM users WHERE email = 'existing@example.com'`,
	).Scan(&confirmed))
	assert.True(t, confirmed)

	// Leave the schema at the latest version for whatever runs next.
	_, _, err = db.MigrateToLatest(database)
	require.NoError(t, err)
}
