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

func TestGetDailySnowfall(t *testing.T) {
	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		assert.Equal(t, "/archive", r.URL.Path)
		_, _ = w.Write([]byte(`{"daily":{` +
			`"time":["2025-01-01","2025-01-02","2025-01-03"],` +
			`"snowfall_sum":[4.5,0,null]}}`))
	}))
	defer server.Close()

	client := weather.NewClientForTest(server.URL)
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC)

	days, err := client.GetDailySnowfall(t.Context(), 39.64, -106.37, start, end)

	require.NoError(t, err)
	assert.Contains(t, gotQuery, "start_date=2025-01-01")
	assert.Contains(t, gotQuery, "end_date=2025-01-03")
	assert.Equal(t, []weather.DailySnowfall{
		{Date: start, Inches: 4.5},
		{Date: start.AddDate(0, 0, 1), Inches: 0},
	}, days, "the null day is skipped")
}

func TestGetDailySnowfallReportsRateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := weather.NewClientForTest(server.URL)

	_, err := client.GetDailySnowfall(t.Context(), 39.64, -106.37, time.Now(), time.Now())

	require.ErrorIs(t, err, weather.ErrRateLimited)
}
