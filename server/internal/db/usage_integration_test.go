//go:build integration
// +build integration

package db_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MattSilvaa/powhunter/internal/db"
	dbgen "github.com/MattSilvaa/powhunter/internal/db/generated"
	"github.com/MattSilvaa/powhunter/internal/testutil"
)

func TestUsageQueries(t *testing.T) {
	testDB, store, cleanup := testutil.SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	queries := dbgen.New(testDB)

	alta := testutil.SeedTestResort(t, queries, "Alta", 40.5884, -111.6386)
	testutil.SeedTestResort(t, queries, "Unfollowed", 39.6403, -106.3742)

	subscriber := testutil.SeedTestUser(t, queries, "subscriber@example.com", "+15551234567")
	testutil.SeedTestAlert(t, queries, subscriber.Uuid, alta.Uuid, 6, 3)

	// An old signup with no phone and no alerts counts as a user only.
	lapsed := testutil.SeedTestUser(t, queries, "lapsed@example.com", "")
	_, err := testDB.ExecContext(ctx,
		"UPDATE users SET created_at = NOW() - INTERVAL '60 days' WHERE uuid = $1", lapsed.Uuid)
	require.NoError(t, err)

	_, err = testDB.ExecContext(ctx,
		"INSERT INTO sessions (user_uuid, token_hash, expires_at) VALUES ($1, 'hash', NOW() + INTERVAL '1 day')",
		subscriber.Uuid)
	require.NoError(t, err)

	alert := db.AlertToSend{UserUuid: subscriber.Uuid, ResortUUID: alta.Uuid, SnowAmount: 8,
		ForecastDate: time.Now().Add(24 * time.Hour)}
	require.NoError(t, store.RecordAlertSent(ctx, alert, 2))

	totals, err := queries.GetUsageTotals(ctx)
	require.NoError(t, err)
	assert.Equal(t, dbgen.GetUsageTotalsRow{Users: 2, UsersWithPhone: 1, Subscribers: 1, ActiveAlerts: 1}, totals)

	recent, err := queries.GetUsageSince(ctx, time.Now().Add(-30*24*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, dbgen.GetUsageSinceRow{
		Signups: 1, ActiveUsers: 1, AlertsSent: 1, UsersAlerted: 1, SmsSegments: 2,
	}, recent)

	future, err := queries.GetUsageSince(ctx, time.Now().Add(time.Hour))
	require.NoError(t, err)
	assert.Equal(t, dbgen.GetUsageSinceRow{}, future, "nothing happened after the window starts")

	byResort, err := queries.GetActiveAlertsByResort(ctx)
	require.NoError(t, err)
	assert.Equal(t, []dbgen.GetActiveAlertsByResortRow{
		{Name: "Alta", ActiveAlerts: 1},
		{Name: "Unfollowed", ActiveAlerts: 0},
	}, byResort)
}
