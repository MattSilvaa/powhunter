package metrics_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dbgen "github.com/MattSilvaa/powhunter/internal/db/generated"
	"github.com/MattSilvaa/powhunter/internal/metrics"
)

type fakeUsage struct {
	err   error
	calls int
	since []time.Time
}

func (f *fakeUsage) GetUsageTotals(context.Context) (dbgen.GetUsageTotalsRow, error) {
	f.calls++

	return dbgen.GetUsageTotalsRow{Users: 10, UsersWithPhone: 8, Subscribers: 6, ActiveAlerts: 9}, f.err
}

func (f *fakeUsage) GetUsageSince(_ context.Context, since time.Time) (dbgen.GetUsageSinceRow, error) {
	f.since = append(f.since, since)

	return dbgen.GetUsageSinceRow{Signups: 2, ActiveUsers: 3, AlertsSent: 4, UsersAlerted: 3, SmsSegments: 5}, nil
}

func (f *fakeUsage) GetActiveAlertsByResort(context.Context) ([]dbgen.GetActiveAlertsByResortRow, error) {
	return []dbgen.GetActiveAlertsByResortRow{{Name: "Alta", ActiveAlerts: 4}}, nil
}

func newCollector(q *fakeUsage, now *time.Time) *metrics.UsageCollector {
	return metrics.NewUsageCollector(q, slog.New(slog.NewTextHandler(io.Discard, nil)),
		func() time.Time { return *now })
}

func TestUsageCollectorReportsUsage(t *testing.T) {
	now := time.Date(2026, 1, 31, 12, 0, 0, 0, time.UTC)
	q := &fakeUsage{}

	expected := `
# HELP powhunter_usage_subscribers Users with at least one active alert.
# TYPE powhunter_usage_subscribers gauge
powhunter_usage_subscribers 6
# HELP powhunter_usage_sms_segments SMS segments billed in the trailing window.
# TYPE powhunter_usage_sms_segments gauge
powhunter_usage_sms_segments{window="1d"} 5
powhunter_usage_sms_segments{window="7d"} 5
powhunter_usage_sms_segments{window="30d"} 5
# HELP powhunter_usage_resort_active_alerts Active alerts per resort.
# TYPE powhunter_usage_resort_active_alerts gauge
powhunter_usage_resort_active_alerts{resort="Alta"} 4
# HELP powhunter_usage_refresh_success Whether the last usage refresh read the database.
# TYPE powhunter_usage_refresh_success gauge
powhunter_usage_refresh_success 1
`
	require.NoError(t, testutil.CollectAndCompare(newCollector(q, &now), strings.NewReader(expected),
		"powhunter_usage_subscribers", "powhunter_usage_sms_segments",
		"powhunter_usage_resort_active_alerts", "powhunter_usage_refresh_success"))

	assert.Equal(t, []time.Time{
		now.Add(-24 * time.Hour),
		now.Add(-7 * 24 * time.Hour),
		now.Add(-30 * 24 * time.Hour),
	}, q.since)
}

func TestUsageCollectorCachesBetweenScrapes(t *testing.T) {
	now := time.Date(2026, 1, 31, 12, 0, 0, 0, time.UTC)
	q := &fakeUsage{}
	collector := newCollector(q, &now)

	testutil.CollectAndCount(collector)
	testutil.CollectAndCount(collector)
	assert.Equal(t, 1, q.calls, "a scrape within the refresh interval should reuse the last read")

	now = now.Add(time.Minute)
	testutil.CollectAndCount(collector)
	assert.Equal(t, 2, q.calls, "a scrape after the refresh interval should read again")
}

func TestUsageCollectorReportsAFailedRefreshOnly(t *testing.T) {
	now := time.Date(2026, 1, 31, 12, 0, 0, 0, time.UTC)
	q := &fakeUsage{err: errors.New("database down")}

	expected := `
# HELP powhunter_usage_refresh_success Whether the last usage refresh read the database.
# TYPE powhunter_usage_refresh_success gauge
powhunter_usage_refresh_success 0
`
	collector := newCollector(q, &now)
	require.NoError(t, testutil.CollectAndCompare(collector, strings.NewReader(expected)))
	assert.Equal(t, 1, testutil.CollectAndCount(collector), "no stale or partial counts should be served")
}
