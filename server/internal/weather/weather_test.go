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
