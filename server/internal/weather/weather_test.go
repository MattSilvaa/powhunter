package weather_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MattSilvaa/powhunter/internal/weather"
)

func TestGetForecastReportsRateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "45")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := weather.NewClientForTest(server.URL)

	_, err := client.GetSnowForecast(t.Context(), 39.64, -106.37)

	require.ErrorIs(t, err, weather.ErrRateLimited)

	var rateLimited *weather.RateLimitError
	require.ErrorAs(t, err, &rateLimited)
	assert.Equal(t, 45*time.Second, rateLimited.RetryAfter)
}

func TestGetForecastOtherErrorsAreNotRateLimits(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	client := weather.NewClientForTest(server.URL)

	_, err := client.GetSnowForecast(t.Context(), 39.64, -106.37)

	require.Error(t, err)
	assert.NotErrorIs(t, err, weather.ErrRateLimited)
}

func TestParseRetryAfter(t *testing.T) {
	assert.Equal(t, 30*time.Second, weather.ParseRetryAfter("30"))
	assert.Zero(t, weather.ParseRetryAfter(""))
	assert.Zero(t, weather.ParseRetryAfter("Wed, 21 Oct 2015 07:28:00 GMT"))
	assert.Zero(t, weather.ParseRetryAfter("-5"))
}

func TestGetSnowForecastAsksForPastAndTenDaysAndReadsDailyTotals(t *testing.T) {
	var query string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"hourly":{"time":[],"snowfall":[],"temperature_2m":[]},` +
			`"daily":{"time":["2026-01-10","2026-01-11","2026-01-12"],"snowfall_sum":[0,7.5,null]}}`))
	}))
	defer server.Close()

	forecast, err := weather.NewClientForTest(server.URL).GetSnowForecast(t.Context(), 39.64, -106.37)

	require.NoError(t, err)
	assert.Contains(t, query, "daily=snowfall_sum")
	assert.Contains(t, query, "past_days=21")
	assert.Contains(t, query, "forecast_days=10")
	assert.Equal(t, []weather.DailySnowfall{
		{Date: time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC), SnowfallIn: 0},
		{Date: time.Date(2026, 1, 11, 0, 0, 0, 0, time.UTC), SnowfallIn: 7.5},
		{Date: time.Date(2026, 1, 12, 0, 0, 0, 0, time.UTC), SnowfallIn: 0},
	}, forecast.Daily)
}

func TestGetSeasonSnowfallReadsTheArchive(t *testing.T) {
	var path, query string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, query = r.URL.Path, r.URL.RawQuery
		_, _ = w.Write([]byte(`{"daily":{"time":["2025-11-01","2025-11-02"],"snowfall_sum":[1.2,0]}}`))
	}))
	defer server.Close()

	days, err := weather.NewClientForTest(server.URL).GetSeasonSnowfall(t.Context(), 40.58, -111.63,
		time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC))

	require.NoError(t, err)
	assert.Equal(t, "/archive", path)
	assert.Contains(t, query, "start_date=2025-11-01")
	assert.Contains(t, query, "end_date=2026-04-30")
	assert.Contains(t, query, "precipitation_unit=inch")
	require.Len(t, days, 2)
	assert.InDelta(t, 1.2, days[0].SnowfallIn, 0.001)
}
