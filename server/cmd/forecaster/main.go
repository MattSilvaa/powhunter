// Command forecaster checks every resort's forecast and delivers matching
// alerts. It is a one-shot job intended to be run on a schedule.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/MattSilvaa/powhunter/internal/config"
	"github.com/MattSilvaa/powhunter/internal/db"
	dbgen "github.com/MattSilvaa/powhunter/internal/db/generated"
	"github.com/MattSilvaa/powhunter/internal/forecast"
	"github.com/MattSilvaa/powhunter/internal/metrics"
	"github.com/MattSilvaa/powhunter/internal/notify"
	"github.com/MattSilvaa/powhunter/internal/snowreport"
	"github.com/MattSilvaa/powhunter/internal/weather"

	_ "github.com/lib/pq"
)

// runTimeout is a generous ceiling for the whole job. Per-resort deadlines do
// the real work; this only stops a wedged process from running forever.
const runTimeout = 30 * time.Minute

// snowReportTimeout bounds the snow report upkeep after the alerts. The recap
// backfill is at most one archive call per resort, once a season.
const snowReportTimeout = 5 * time.Minute

// dailySnowfallRetentionDays is how long stored daily snowfall is kept. The
// report only reads the last three weeks; the rest is slack for debugging.
const dailySnowfallRetentionDays = 60

// These live here rather than in a sibling file because the deploy builds this
// command as a single file (`go build cmd/forecaster/main.go`), which silently
// excludes the rest of the package and fails to compile.
var (
	errMissingTwilioConfig = errors.New("twilio credentials are not configured")
	errRunIncomplete       = errors.New("forecast run did not complete for every resort")
)

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

	store := db.NewStore(conn)
	weatherClient := weather.NewOpenMeteoClient()

	runner := forecast.NewRunner(
		store,
		weatherClient,
		notify.NewTwilioClient(fromNumber),
		logger,
		time.Now,
		forecast.Options{
			LinkBaseURL: linkBaseURL(logger, os.Getenv("LINK_BASE_URL")),
			ShareHost:   shareHost(logger, cfg.AppBaseURL),
		},
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

	// After the alerts, so nothing here can delay a text. Failures are logged
	// and retried next run; they never change the exit code.
	updateSnowReport(ctx, logger, store.Queries(), weatherClient)

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

// updateSnowReport prunes old daily snowfall and fills in last season's recap
// for any resort that does not have it yet.
func updateSnowReport(
	ctx context.Context,
	logger *slog.Logger,
	queries *dbgen.Queries,
	archive weather.ArchiveService,
) {
	ctx, cancel := context.WithTimeout(ctx, snowReportTimeout)
	defer cancel()

	now := time.Now().UTC()

	cutoff := now.AddDate(0, 0, -dailySnowfallRetentionDays)
	if pruned, err := queries.PruneDailySnowfall(ctx, cutoff); err != nil {
		logger.WarnContext(ctx, "failed to prune daily snowfall", "error", err)
	} else if pruned > 0 {
		logger.InfoContext(ctx, "pruned daily snowfall", "rows", pruned)
	}

	if err := snowreport.Backfill(ctx, queries, archive, now, logger); err != nil {
		logger.WarnContext(ctx, "season recap backfill incomplete; it will retry next run", "error", err)
	}
}

// linkBaseURL validates where alert links point. A bad value logs and turns
// links off rather than failing the run: alerts without a link still do their
// job, and the run exiting non-zero over a typo would deliver nothing.
func linkBaseURL(logger *slog.Logger, raw string) string {
	if raw == "" {
		logger.Info("LINK_BASE_URL is not set; alerts are sent without a tracked link")
		return ""
	}

	parsed, err := url.Parse(strings.TrimRight(raw, "/"))
	// A path is rejected because the API serves /go at its root: a base such as
	// https://api.example.com/go would produce /go/go/<code> and break every link.
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" ||
		parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		logger.Warn("LINK_BASE_URL must be a bare http(s) origin such as https://api.example.com; "+
			"alerts are sent without a tracked link",
			"value", raw)

		return ""
	}

	return parsed.String()
}

// shareHost is the web app address alerts invite readers to share, written
// without its scheme ("powhunter.app") because every character costs room in
// the SMS and phones link a bare domain anyway. A local or invalid address
// turns the share line off: a friend cannot open localhost.
func shareHost(logger *slog.Logger, appBaseURL string) string {
	parsed, err := url.Parse(appBaseURL)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" ||
		parsed.Hostname() == "localhost" || parsed.Hostname() == "127.0.0.1" {
		logger.Info("APP_BASE_URL is not a public address; alerts are sent without a share line",
			"value", appBaseURL)

		return ""
	}

	return strings.TrimPrefix(parsed.Host, "www.") + strings.TrimRight(parsed.Path, "/")
}

// publishMetrics reports the run to a Pushgateway when one is configured.
//
// Every outcome is logged, including the do-nothing one. An unset gateway is
// indistinguishable from a working one in the job's output otherwise, and a
// silent no-op here means ForecasterNotRunning — the alert that catches a dead
// alert pipeline — can never fire, with nothing anywhere saying why.
func publishMetrics(logger *slog.Logger, summary metrics.ForecastRun) {
	gateway := os.Getenv("PUSHGATEWAY_URL")
	if gateway == "" {
		logger.Warn("PUSHGATEWAY_URL is not set; run metrics were not published")
		return
	}

	if err := metrics.PushForecastRun(gateway, summary); err != nil {
		logger.Warn("failed to publish forecaster metrics", "gateway", gateway, "error", err)
		return
	}

	logger.Info("published forecaster metrics", "gateway", gateway)
}
