package snowreport_test

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	dbgen "github.com/MattSilvaa/powhunter/internal/db/generated"
	"github.com/MattSilvaa/powhunter/internal/snowreport"
	"github.com/MattSilvaa/powhunter/internal/weather"
	weathermocks "github.com/MattSilvaa/powhunter/internal/weather/mocks"
)

// today is mid-afternoon so tests also cover reducing a time to its day.
func today() time.Time {
	return time.Date(2026, 1, 15, 15, 30, 0, 0, time.UTC)
}

func day(offset int) time.Time {
	return time.Date(2026, 1, 15+offset, 0, 0, 0, 0, time.UTC)
}

func snow(resort string, offset int, inches float64) snowreport.ResortDay {
	return snowreport.ResortDay{ResortName: resort, Date: day(offset), SnowfallIn: inches}
}

func TestDetermineMode(t *testing.T) {
	tests := []struct {
		name string
		days []snowreport.ResortDay
		want snowreport.Mode
	}{
		{"no data is off season", nil, snowreport.ModeRecap},
		{"dry everywhere is off season",
			[]snowreport.ResortDay{snow("Alta", -3, 1), snow("Vail", 4, 1.5)}, snowreport.ModeRecap},
		{"a recent snow day keeps it in season",
			[]snowreport.ResortDay{snow("Alta", -10, 8)}, snowreport.ModeLive},
		{"a forecast snow day puts it in season",
			[]snowreport.ResortDay{snow("Alta", 6, 3)}, snowreport.ModeLive},
		{"snow today counts", []snowreport.ResortDay{snow("Alta", 0, 2)}, snowreport.ModeLive},
		{"exactly the threshold counts", []snowreport.ResortDay{snow("Alta", -1, 2.0)}, snowreport.ModeLive},
		{"just under the threshold does not",
			[]snowreport.ResortDay{snow("Alta", -1, 1.99), snow("Vail", 3, 1.99)}, snowreport.ModeRecap},
		{"21 days back is still in the lookback",
			[]snowreport.ResortDay{snow("Alta", -21, 5)}, snowreport.ModeLive},
		{"22 days back is outside it", []snowreport.ResortDay{snow("Alta", -22, 5)}, snowreport.ModeRecap},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, snowreport.DetermineMode(tt.days, today()))
		})
	}
}

// A dry summer flips to the recap, and the first forecast storm in October
// flips it back on its own with no calendar involved.
func TestAnEarlySeasonStormFlipsBackToLive(t *testing.T) {
	october := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	dry := []snowreport.ResortDay{
		{ResortName: "Alta", Date: october.AddDate(0, 0, -5), SnowfallIn: 0},
		{ResortName: "Alta", Date: october.AddDate(0, 0, 3), SnowfallIn: 0.5},
	}
	require.Equal(t, snowreport.ModeRecap, snowreport.DetermineMode(dry, october))

	storm := append(
		append([]snowreport.ResortDay(nil), dry...),
		snowreport.ResortDay{ResortName: "Alta", Date: october.AddDate(0, 0, 7), SnowfallIn: 4},
	)
	assert.Equal(t, snowreport.ModeLive, snowreport.DetermineMode(storm, october))
}

func TestBuildLiveSortsByUpcomingSnow(t *testing.T) {
	days := []snowreport.ResortDay{
		snow("Alta", -2, 20), // past snow is not part of the forecast total
		snow("Alta", 0, 1),
		snow("Alta", 1, 0.4),
		snow("Palisades Tahoe", 1, 6),
		snow("Palisades Tahoe", 3, 12),
		snow("Palisades Tahoe", 5, 0.5), // a trace does not move "by"
		snow("Palisades Tahoe", 10, 30), // past the window
		snow("Vail", 0, 1.4),
	}

	live := snowreport.BuildLive(days, today())

	require.Len(t, live, 3)
	assert.Equal(t, "Palisades Tahoe", live[0].Name)
	assert.InDelta(t, 18.5, live[0].TotalIn, 0.001)
	require.NotNil(t, live[0].Through)
	assert.Equal(t, "2026-01-18", *live[0].Through)
	assert.Len(t, live[0].Daily, snowreport.ForecastDays)
	assert.Nil(t, live[0].Daily[0].SnowIn, "a day with no stored forecast is unknown, not zero")

	// Equal totals fall back to name order.
	assert.Equal(t, []string{"Alta", "Vail"}, []string{live[1].Name, live[2].Name})
	assert.InDelta(t, 1.4, live[1].TotalIn, 0.001)
}

func TestLastCompletedSeason(t *testing.T) {
	tests := []struct {
		now  time.Time
		want string
	}{
		{time.Date(2026, 11, 8, 0, 0, 0, 0, time.UTC), "2025-26"},
		{time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), "2024-25"},
		// The archive trails real time, so the season that just ended waits a week.
		{time.Date(2026, 5, 7, 23, 0, 0, 0, time.UTC), "2024-25"},
		{time.Date(2026, 5, 8, 0, 0, 0, 0, time.UTC), "2025-26"},
	}

	for _, tt := range tests {
		season := snowreport.LastCompletedSeason(tt.now)
		assert.Equal(t, tt.want, season.Name, tt.now)
	}

	season := snowreport.LastCompletedSeason(time.Date(2026, 11, 8, 0, 0, 0, 0, time.UTC))
	assert.Equal(t, time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC), season.Start)
	assert.Equal(t, time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC), season.End)
}

func season(inches ...float64) []weather.DailySnowfall {
	days := make([]weather.DailySnowfall, len(inches))
	for i, in := range inches {
		days[i] = weather.DailySnowfall{Date: time.Date(2025, 11, 1+i, 0, 0, 0, 0, time.UTC), SnowfallIn: in}
	}

	return days
}

func TestComputeSeasonStats(t *testing.T) {
	stats := snowreport.ComputeSeasonStats(season(0, 6, 5.9, 2, 0, 10, 4, 4, 0))

	assert.InDelta(t, 31.9, stats.TotalIn, 0.001)
	assert.Equal(t, 2, stats.DeepDays, "6″ counts as deep, 5.9″ does not")
	assert.InDelta(t, 18, stats.BiggestStormIn, 0.001)
	assert.Equal(t, time.Date(2025, 11, 6, 0, 0, 0, 0, time.UTC), stats.StormStart)
	assert.Equal(t, time.Date(2025, 11, 8, 0, 0, 0, 0, time.UTC), stats.StormEnd)
}

func TestComputeSeasonStatsAtTheEdges(t *testing.T) {
	// The biggest storm is the last two days: the window is cut short by the
	// end of the season rather than skipped.
	stats := snowreport.ComputeSeasonStats(season(1, 0, 0, 7, 9))
	assert.InDelta(t, 16, stats.BiggestStormIn, 0.001)
	assert.Equal(t, time.Date(2025, 11, 4, 0, 0, 0, 0, time.UTC), stats.StormStart)
	assert.Equal(t, time.Date(2025, 11, 5, 0, 0, 0, 0, time.UTC), stats.StormEnd)

	// Equal storms go to the earlier one.
	tied := snowreport.ComputeSeasonStats(season(5, 0, 0, 0, 5))
	assert.Equal(t, time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC), tied.StormStart)

	// A season with no snow has no storm dates.
	dry := snowreport.ComputeSeasonStats(season(0, 0))
	assert.Zero(t, dry.BiggestStormIn)
	assert.True(t, dry.StormStart.IsZero())
}

type fakeStore struct {
	daily    []dbgen.ListDailySnowfallSinceRow
	stats    []dbgen.ListSeasonStatsRow
	missing  []dbgen.Resort
	saved    []dbgen.UpsertSeasonStatsParams
	sinceArg time.Time
	season   string
}

func (f *fakeStore) ListDailySnowfallSince(
	_ context.Context,
	date time.Time,
) ([]dbgen.ListDailySnowfallSinceRow, error) {
	f.sinceArg = date
	return f.daily, nil
}

func (f *fakeStore) ListSeasonStats(_ context.Context, season string) ([]dbgen.ListSeasonStatsRow, error) {
	f.season = season
	return f.stats, nil
}

func (f *fakeStore) ListResortsMissingSeasonStats(_ context.Context, season string) ([]dbgen.Resort, error) {
	f.season = season
	return f.missing, nil
}

func (f *fakeStore) UpsertSeasonStats(_ context.Context, arg dbgen.UpsertSeasonStatsParams) error {
	f.saved = append(f.saved, arg)
	return nil
}

func TestBuildLiveReport(t *testing.T) {
	fetched := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	store := &fakeStore{daily: []dbgen.ListDailySnowfallSinceRow{
		{ResortName: "Alta", Date: day(2), SnowfallIn: 9, FetchedAt: fetched},
	}}

	report, err := snowreport.Build(t.Context(), store, today())

	require.NoError(t, err)
	assert.Equal(t, snowreport.ModeLive, report.Mode)
	assert.Equal(t, day(-snowreport.LookbackDays), store.sinceArg)
	require.NotNil(t, report.Live)
	assert.Nil(t, report.Recap)
	assert.Equal(t, fetched, *report.UpdatedAt)
	assert.Equal(t, "Alta", report.Live.Resorts[0].Name)
}

func TestBuildRecapReport(t *testing.T) {
	store := &fakeStore{stats: []dbgen.ListSeasonStatsRow{{
		ResortName: "Alta", TotalSnowfallIn: 401.26, Days6inPlus: 22, BiggestStormIn: 41,
		BiggestStormStart: sql.NullTime{Time: time.Date(2025, 12, 20, 0, 0, 0, 0, time.UTC), Valid: true},
		BiggestStormEnd:   sql.NullTime{Time: time.Date(2025, 12, 22, 0, 0, 0, 0, time.UTC), Valid: true},
	}}}
	november := time.Date(2026, 11, 8, 0, 0, 0, 0, time.UTC)

	report, err := snowreport.Build(t.Context(), store, november)

	require.NoError(t, err)
	assert.Equal(t, snowreport.ModeRecap, report.Mode)
	assert.Equal(t, "2025-26", store.season)
	require.NotNil(t, report.Recap)
	assert.Equal(t, "2025-11-01", report.Recap.Start)
	assert.Equal(t, "2026-04-30", report.Recap.End)
	assert.InDelta(t, 401.3, report.Recap.Resorts[0].TotalIn, 0.001)
	assert.Equal(t, "2025-12-20", *report.Recap.Resorts[0].BiggestStormStart)
}

func TestBackfillStoresMissingSeasonsAndKeepsGoingAfterAFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	archive := weathermocks.NewMockArchiveService(ctrl)

	resortAt := func(name string, lat, lon float64) dbgen.Resort {
		return dbgen.Resort{
			Uuid:      uuid.New(),
			Name:      name,
			Latitude:  sql.NullFloat64{Float64: lat, Valid: true},
			Longitude: sql.NullFloat64{Float64: lon, Valid: true},
		}
	}
	broken := resortAt("Broken", 1, 1)
	alta := resortAt("Alta", 40.58, -111.63)
	store := &fakeStore{missing: []dbgen.Resort{broken, alta}}

	start := time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC)
	archive.EXPECT().GetSeasonSnowfall(gomock.Any(), 1.0, 1.0, start, end).Return(nil, errors.New("upstream down"))
	archive.EXPECT().GetSeasonSnowfall(gomock.Any(), 40.58, -111.63, start, end).Return(season(0, 8, 4), nil)

	err := snowreport.Backfill(t.Context(), store, archive,
		time.Date(2026, 11, 8, 0, 0, 0, 0, time.UTC), slog.New(slog.DiscardHandler))

	require.ErrorContains(t, err, "Broken")
	require.Len(t, store.saved, 1)
	assert.Equal(t, alta.Uuid, store.saved[0].ResortUuid)
	assert.Equal(t, "2025-26", store.saved[0].Season)
	assert.InDelta(t, 12, store.saved[0].TotalSnowfallIn, 0.001)
	assert.Equal(t, int32(1), store.saved[0].Days6inPlus)
}
