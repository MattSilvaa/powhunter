// Command forecaster checks every resort's forecast and delivers matching
// alerts. It is a one-shot job intended to be run on a schedule.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MattSilvaa/powhunter/internal/config"
	"github.com/MattSilvaa/powhunter/internal/db"
	"github.com/MattSilvaa/powhunter/internal/forecast"
	"github.com/MattSilvaa/powhunter/internal/metrics"
	"github.com/MattSilvaa/powhunter/internal/notify"
	"github.com/MattSilvaa/powhunter/internal/weather"

	_ "github.com/lib/pq"
)

// runTimeout is a generous ceiling for the whole job. Per-resort deadlines do
// the real work; this only stops a wedged process from running forever.
const runTimeout = 30 * time.Minute

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	if err := run(logger); err != nil {
		logger.Error("forecast run failed", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	fromNumber := os.Getenv("TWILIO_FROM_NUMBER")
	if os.Getenv("TWILIO_ACCOUNT_SID") == "" || os.Getenv("TWILIO_AUTH_TOKEN") == "" || fromNumber == "" {
		logger.Error("twilio credentials not configured",
			"required", "TWILIO_ACCOUNT_SID, TWILIO_AUTH_TOKEN, TWILIO_FROM_NUMBER")

		return errMissingTwilioConfig
	}

	conn, err := db.Open(cfg.Database)
	if err != nil {
		return err
	}
	defer conn.Close()

	// A signal-aware context lets an in-flight run stop cleanly rather than
	// being killed midway between sending an SMS and recording it.
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	ctx, timeoutCancel := context.WithTimeout(ctx, runTimeout)
	defer timeoutCancel()

	runner := forecast.NewRunner(
		db.NewStore(conn),
		weather.NewOpenMeteoClient(),
		notify.NewTwilioClient(fromNumber),
		logger,
		time.Now,
		forecast.Options{},
	)

	summary, runErr := runner.Run(ctx)

	logger.Info("forecast check complete",
		"resorts_checked", summary.ResortsChecked,
		"resorts_failed", summary.ResortsFailed,
		"alerts_matched", summary.AlertsMatched,
		"alerts_sent", summary.AlertsSent,
		"alerts_failed", summary.AlertsFailed,
		"records_failed", summary.RecordsFailed,
		"duration_ms", summary.Duration.Milliseconds(),
		"succeeded", summary.Succeeded,
	)

	publishMetrics(logger, summary)

	if runErr != nil {
		return runErr
	}

	// Exiting non-zero lets the scheduler surface a bad run instead of the job
	// reporting success while having notified nobody.
	if !summary.Succeeded {
		return errRunIncomplete
	}

	return nil
}

// publishMetrics reports the run to a Pushgateway when one is configured.
func publishMetrics(logger *slog.Logger, summary metrics.ForecastRun) {
	gateway := os.Getenv("PUSHGATEWAY_URL")
	if gateway == "" {
		return
	}

	if err := metrics.PushForecastRun(gateway, summary); err != nil {
		logger.Warn("failed to publish forecaster metrics", "error", err)
	}
}
