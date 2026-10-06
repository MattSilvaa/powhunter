package metrics

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	dbgen "github.com/MattSilvaa/powhunter/internal/db/generated"
)

const (
	// usageRefreshInterval caps how often the usage queries run. Prometheus
	// scrapes far more often than these numbers meaningfully change.
	usageRefreshInterval = time.Minute
	// usageQueryTimeout bounds one refresh so a slow database cannot hold a
	// scrape open.
	usageQueryTimeout = 5 * time.Second
	hoursPerDay       = 24
)

// usageWindow is a trailing period activity is reported over.
type usageWindow struct {
	label  string
	length time.Duration
}

// usageWindows lists the reporting windows, keyed by the window label.
func usageWindows() []usageWindow {
	return []usageWindow{
		{"1d", 1 * hoursPerDay * time.Hour},
		{"7d", 7 * hoursPerDay * time.Hour},
		{"30d", 30 * hoursPerDay * time.Hour},
	}
}

// UsageQuerier is the slice of the generated queries the usage collector reads.
type UsageQuerier interface {
	GetUsageTotals(ctx context.Context) (dbgen.GetUsageTotalsRow, error)
	GetUsageSince(ctx context.Context, since time.Time) (dbgen.GetUsageSinceRow, error)
	GetActiveAlertsByResort(ctx context.Context) ([]dbgen.GetActiveAlertsByResortRow, error)
}

// UsageCollector publishes product usage read from the database: who is
// signed up, who is subscribed, how many alerts went out and how many SMS
// segments they cost. These are the numbers that decide pricing and show
// resorts their audience, as opposed to the operational metrics elsewhere.
type UsageCollector struct {
	querier UsageQuerier
	logger  *slog.Logger
	now     func() time.Time

	mu          sync.Mutex
	cached      []prometheus.Metric
	refreshedAt time.Time

	users          *prometheus.Desc
	usersWithPhone *prometheus.Desc
	subscribers    *prometheus.Desc
	activeAlerts   *prometheus.Desc
	resortAlerts   *prometheus.Desc
	signups        *prometheus.Desc
	activeUsers    *prometheus.Desc
	alertsSent     *prometheus.Desc
	usersAlerted   *prometheus.Desc
	smsSegments    *prometheus.Desc
	refreshSuccess *prometheus.Desc
}

// NewUsageCollector builds a collector. Now is injectable so tests can pin the
// reporting windows.
func NewUsageCollector(querier UsageQuerier, logger *slog.Logger, now func() time.Time) *UsageCollector {
	if now == nil {
		now = time.Now
	}

	desc := func(name, help string, labels ...string) *prometheus.Desc {
		return prometheus.NewDesc(prometheus.BuildFQName(Namespace, "usage", name), help, labels, nil)
	}

	return &UsageCollector{
		querier: querier,
		logger:  logger,
		now:     now,

		users:          desc("users", "Registered users."),
		usersWithPhone: desc("users_with_phone", "Users with a phone number SMS alerts can reach."),
		subscribers:    desc("subscribers", "Users with at least one active alert."),
		activeAlerts:   desc("active_alerts", "Active alerts across all users."),
		resortAlerts:   desc("resort_active_alerts", "Active alerts per resort.", "resort"),
		signups:        desc("signups", "Users created in the trailing window.", "window"),
		activeUsers:    desc("active_users", "Users who used the web app in the trailing window.", "window"),
		alertsSent:     desc("alerts_sent", "Alerts delivered in the trailing window.", "window"),
		usersAlerted:   desc("users_alerted", "Distinct users alerted in the trailing window.", "window"),
		smsSegments:    desc("sms_segments", "SMS segments billed in the trailing window.", "window"),
		refreshSuccess: desc("refresh_success", "Whether the last usage refresh read the database."),
	}
}

// Describe implements prometheus.Collector.
func (c *UsageCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, d := range []*prometheus.Desc{
		c.users, c.usersWithPhone, c.subscribers, c.activeAlerts, c.resortAlerts,
		c.signups, c.activeUsers, c.alertsSent, c.usersAlerted, c.smsSegments,
		c.refreshSuccess,
	} {
		ch <- d
	}
}

// Collect implements prometheus.Collector. A failed refresh reports only
// refresh_success 0: serving stale counts as if they were current would hide
// the outage, and failing the whole scrape would take the HTTP metrics with it.
func (c *UsageCollector) Collect(ch chan<- prometheus.Metric) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cached == nil || c.now().Sub(c.refreshedAt) >= usageRefreshInterval {
		ctx, cancel := context.WithTimeout(context.Background(), usageQueryTimeout)
		defer cancel()

		metrics, err := c.read(ctx)
		if err != nil {
			c.logger.ErrorContext(ctx, "failed to read usage metrics", "error", err)
			c.cached = nil
			ch <- prometheus.MustNewConstMetric(c.refreshSuccess, prometheus.GaugeValue, 0)

			return
		}

		c.cached = metrics
		c.refreshedAt = c.now()
	}

	for _, m := range c.cached {
		ch <- m
	}
}

func (c *UsageCollector) read(ctx context.Context) ([]prometheus.Metric, error) {
	gauge := func(desc *prometheus.Desc, value int64, labels ...string) prometheus.Metric {
		return prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, float64(value), labels...)
	}

	totals, err := c.querier.GetUsageTotals(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // logged by the caller with context
	}

	metrics := []prometheus.Metric{
		gauge(c.users, totals.Users),
		gauge(c.usersWithPhone, totals.UsersWithPhone),
		gauge(c.subscribers, totals.Subscribers),
		gauge(c.activeAlerts, totals.ActiveAlerts),
	}

	for _, window := range usageWindows() {
		since, sinceErr := c.querier.GetUsageSince(ctx, c.now().Add(-window.length))
		if sinceErr != nil {
			return nil, sinceErr //nolint:wrapcheck // logged by the caller with context
		}

		metrics = append(metrics,
			gauge(c.signups, since.Signups, window.label),
			gauge(c.activeUsers, since.ActiveUsers, window.label),
			gauge(c.alertsSent, since.AlertsSent, window.label),
			gauge(c.usersAlerted, since.UsersAlerted, window.label),
			gauge(c.smsSegments, since.SmsSegments, window.label),
		)
	}

	resorts, err := c.querier.GetActiveAlertsByResort(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // logged by the caller with context
	}

	for _, resort := range resorts {
		metrics = append(metrics, gauge(c.resortAlerts, resort.ActiveAlerts, resort.Name))
	}

	return append(metrics, gauge(c.refreshSuccess, 1)), nil
}
