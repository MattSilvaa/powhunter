package metrics

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
)

// ForecasterJob is the Prometheus job label for forecaster runs.
const ForecasterJob = "powhunter_forecaster"

// ForecastRun summarises one forecaster invocation.
type ForecastRun struct {
	ResortsChecked int
	ResortsFailed  int
	AlertsMatched  int
	AlertsSent     int
	AlertsFailed   int
	RecordsFailed  int
	Duration       time.Duration
	Succeeded      bool
}

// PushForecastRun publishes a run summary to a Pushgateway.
//
// The forecaster is a one-shot job, so there is nothing to scrape. Without
// this, a cron that stops running is completely invisible, which is how the
// alert pipeline could stay broken unnoticed.
func PushForecastRun(gatewayURL string, run ForecastRun) error {
	registry := prometheus.NewRegistry()

	gauge := func(name, help string, value float64) {
		g := prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: "forecaster",
			Name:      name,
			Help:      help,
		})
		g.Set(value)
		registry.MustRegister(g)
	}

	gauge("resorts_checked", "Resorts examined in the last run.", float64(run.ResortsChecked))
	gauge("resorts_failed", "Resorts whose forecast could not be fetched.", float64(run.ResortsFailed))
	gauge("alerts_matched_total", "Alerts matching the forecast in the last run.", float64(run.AlertsMatched))
	gauge("alerts_sent_total", "Alerts successfully delivered in the last run.", float64(run.AlertsSent))
	gauge("alerts_failed_total", "Alerts that could not be delivered.", float64(run.AlertsFailed))
	gauge("records_failed_total", "Delivered alerts whose history write failed.", float64(run.RecordsFailed))
	gauge("run_duration_seconds", "Duration of the last run.", run.Duration.Seconds())

	success := 0.0
	if run.Succeeded {
		success = 1
		gauge("last_success_timestamp_seconds", "Unix time of the last successful run.",
			float64(time.Now().Unix()))
	}

	gauge("run_success", "Whether the last run succeeded.", success)

	if err := push.New(gatewayURL, ForecasterJob).Gatherer(registry).Push(); err != nil {
		return fmt.Errorf("pushing forecaster metrics: %w", err)
	}

	return nil
}
