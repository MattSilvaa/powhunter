package snowhistory_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MattSilvaa/powhunter/internal/snowhistory"
	"github.com/MattSilvaa/powhunter/internal/weather"
)

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func TestWindow(t *testing.T) {
	// Mid-season: five finished seasons plus this one so far.
	start, end := snowhistory.Window(date(2026, time.January, 15))
	assert.Equal(t, date(2020, time.October, 1), start)
	assert.Equal(t, date(2026, time.January, 14), end)

	// Summer: the season that just ended is the latest complete one.
	start, end = snowhistory.Window(date(2026, time.July, 1))
	assert.Equal(t, date(2021, time.October, 1), start)
	assert.Equal(t, date(2026, time.May, 31), end)
}

func TestSummarize(t *testing.T) {
	now := date(2026, time.January, 15)
	days := []weather.DailySnowfall{
		{Date: date(2024, time.December, 20), Inches: 10},
		{Date: date(2025, time.January, 5), Inches: 4},
		{Date: date(2025, time.July, 4), Inches: 3},       // off-season, ignored
		{Date: date(2020, time.November, 1), Inches: 6},   // oldest complete season
		{Date: date(2025, time.December, 1), Inches: 8},   // current season
		{Date: date(2019, time.December, 1), Inches: 100}, // before the window
	}

	summary := snowhistory.Summarize(days, 6, now)

	require.Len(t, summary.Seasons, snowhistory.CompleteSeasons+1)

	current := summary.Seasons[0]
	assert.Equal(t, "2025–26", current.Label)
	assert.True(t, current.Partial)
	assert.InDelta(t, 8.0, current.TotalInches, 0.001)
	assert.Equal(t, 1, *current.DaysAtOrAbove)

	lastSeason := summary.Seasons[1]
	assert.Equal(t, "2024–25", lastSeason.Label, "December and January share a season")
	assert.InDelta(t, 14.0, lastSeason.TotalInches, 0.001)
	assert.Equal(t, 1, *lastSeason.DaysAtOrAbove)

	assert.Equal(t, "2020–21", summary.Seasons[snowhistory.CompleteSeasons].Label)

	// The partial season stays out of the averages: (14 + 6) / 5.
	assert.InDelta(t, 4.0, summary.AverageSeasonInches, 0.001)
	assert.InDelta(t, 0.4, *summary.AverageDaysAtOrAbove, 0.001)

	require.Len(t, summary.MonthlyAverages, 8)
	assert.Equal(t, time.October, summary.MonthlyAverages[0].Month)
	assert.Equal(t, time.May, summary.MonthlyAverages[7].Month)
	assert.InDelta(t, 2.0, summary.MonthlyAverages[2].Inches, 0.001, "December: 10 / 5")
}

func TestSummarizeWithoutThreshold(t *testing.T) {
	summary := snowhistory.Summarize(nil, 0, date(2026, time.January, 15))

	assert.Nil(t, summary.AverageDaysAtOrAbove)
	assert.Nil(t, summary.Seasons[0].DaysAtOrAbove)
}

type fakeSource struct {
	calls int
	err   error
}

func (f *fakeSource) GetDailySnowfall(
	context.Context, float64, float64, time.Time, time.Time,
) ([]weather.DailySnowfall, error) {
	f.calls++
	return []weather.DailySnowfall{{Date: date(2025, time.December, 1), Inches: 5}}, f.err
}

func TestServiceCachesPerResort(t *testing.T) {
	source := &fakeSource{}
	service := snowhistory.NewService(source)
	now := date(2026, time.January, 15)
	service.SetClock(func() time.Time { return now })

	_, err := service.Summary(t.Context(), "resort-a", 1, 2, 0)
	require.NoError(t, err)
	summary, err := service.Summary(t.Context(), "resort-a", 1, 2, 0)
	require.NoError(t, err)
	assert.InDelta(t, 5.0, summary.Seasons[0].TotalInches, 0.001)
	assert.Equal(t, 1, source.calls, "the second request is served from the cache")

	_, err = service.Summary(t.Context(), "resort-b", 1, 2, 0)
	require.NoError(t, err)
	assert.Equal(t, 2, source.calls, "each resort is cached separately")

	now = now.Add(snowhistory.CacheTTL)
	_, err = service.Summary(t.Context(), "resort-a", 1, 2, 0)
	require.NoError(t, err)
	assert.Equal(t, 3, source.calls, "an expired entry is refetched")
}

func TestServiceDoesNotCacheErrors(t *testing.T) {
	source := &fakeSource{err: errors.New("boom")}
	service := snowhistory.NewService(source)

	_, err := service.Summary(t.Context(), "resort-a", 1, 2, 0)
	require.Error(t, err)

	source.err = nil
	_, err = service.Summary(t.Context(), "resort-a", 1, 2, 0)
	require.NoError(t, err)
	assert.Equal(t, 2, source.calls)
}
