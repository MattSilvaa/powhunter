//go:build integration
// +build integration

package db_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MattSilvaa/powhunter/internal/db"
	dbgen "github.com/MattSilvaa/powhunter/internal/db/generated"
	"github.com/MattSilvaa/powhunter/internal/snowreport"
	"github.com/MattSilvaa/powhunter/internal/testutil"
)

func date(month time.Month, day int) time.Time {
	return time.Date(2026, month, day, 0, 0, 0, 0, time.UTC)
}

func TestStoreIntegration_DailySnowfall(t *testing.T) {
	testDB, store, cleanup := testutil.SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	queries := dbgen.New(testDB)
	alta := testutil.SeedTestResort(t, queries, "Alta", 40.58, -111.63)

	first := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	require.NoError(t, store.SaveDailySnowfall(ctx, alta.Uuid, []db.DailySnowfall{
		{Date: date(time.January, 1), SnowfallIn: 3},
		{Date: date(time.January, 16), SnowfallIn: 4},
	}, first))

	// A later run replaces the values for the days it fetched.
	second := first.Add(12 * time.Hour)
	require.NoError(t, store.SaveDailySnowfall(ctx, alta.Uuid, []db.DailySnowfall{
		{Date: date(time.January, 16), SnowfallIn: 9},
	}, second))

	rows, err := queries.ListDailySnowfallSince(ctx, date(time.January, 10))
	require.NoError(t, err)
	require.Len(t, rows, 1, "days before the cutoff are left out")
	assert.Equal(t, "Alta", rows[0].ResortName)
	assert.InDelta(t, 9, rows[0].SnowfallIn, 0.001)
	assert.True(t, second.Equal(rows[0].FetchedAt))

	pruned, err := queries.PruneDailySnowfall(ctx, date(time.January, 10))
	require.NoError(t, err)
	assert.Equal(t, int64(1), pruned)

	// The report reads straight from the stored rows.
	report, err := snowreport.Build(ctx, queries, date(time.January, 15))
	require.NoError(t, err)
	assert.Equal(t, snowreport.ModeLive, report.Mode)
	assert.Equal(t, "Alta", report.Live.Resorts[0].Name)
}

func TestStoreIntegration_SeasonStats(t *testing.T) {
	testDB, _, cleanup := testutil.SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	queries := dbgen.New(testDB)
	alta := testutil.SeedTestResort(t, queries, "Alta", 40.58, -111.63)
	vail := testutil.SeedTestResort(t, queries, "Vail", 39.64, -106.37)

	missing, err := queries.ListResortsMissingSeasonStats(ctx, "2025-26")
	require.NoError(t, err)
	assert.Len(t, missing, 2)

	stats := dbgen.UpsertSeasonStatsParams{
		ResortUuid:        alta.Uuid,
		Season:            "2025-26",
		SeasonStart:       time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC),
		SeasonEnd:         date(time.April, 30),
		TotalSnowfallIn:   300,
		Days6inPlus:       12,
		BiggestStormIn:    30,
		BiggestStormStart: sql.NullTime{Time: date(time.January, 3), Valid: true},
		BiggestStormEnd:   sql.NullTime{Time: date(time.January, 5), Valid: true},
	}
	require.NoError(t, queries.UpsertSeasonStats(ctx, stats))

	stats.TotalSnowfallIn = 320
	require.NoError(t, queries.UpsertSeasonStats(ctx, stats), "recomputing a season replaces it")

	require.NoError(t, queries.UpsertSeasonStats(ctx, dbgen.UpsertSeasonStatsParams{
		ResortUuid: vail.Uuid, Season: "2025-26",
		SeasonStart: stats.SeasonStart, SeasonEnd: stats.SeasonEnd, TotalSnowfallIn: 150,
	}))

	rows, err := queries.ListSeasonStats(ctx, "2025-26")
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, "Alta", rows[0].ResortName, "most snow first")
	assert.InDelta(t, 320, rows[0].TotalSnowfallIn, 0.001)
	assert.False(t, rows[1].BiggestStormStart.Valid)

	missing, err = queries.ListResortsMissingSeasonStats(ctx, "2025-26")
	require.NoError(t, err)
	assert.Empty(t, missing)

	other, err := queries.ListResortsMissingSeasonStats(ctx, "2026-27")
	require.NoError(t, err)
	assert.Len(t, other, 2, "each season is tracked separately")
}
