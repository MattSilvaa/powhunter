// Package forecast contains the forecaster's run logic, separated from process
// wiring so it can be tested with the existing store, weather and notifier
// mocks.
package forecast

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/MattSilvaa/powhunter/internal/db"
	dbgen "github.com/MattSilvaa/powhunter/internal/db/generated"
	"github.com/MattSilvaa/powhunter/internal/metrics"
	"github.com/MattSilvaa/powhunter/internal/notify"
	"github.com/MattSilvaa/powhunter/internal/weather"
)

const (
	// defaultConcurrency bounds parallel resort checks so the weather API is not
	// hammered while still preventing one slow resort from stalling the run.
	defaultConcurrency = 4
	// defaultResortTimeout bounds the work for a single resort.
	defaultResortTimeout = 90 * time.Second
	// defaultAttempts is how many times a forecast fetch is tried.
	defaultAttempts = 3
	// retryBackoff is the base delay between forecast fetch attempts.
	retryBackoff = 2 * time.Second
	// hoursPerDay converts a duration to whole days.
	hoursPerDay = 24
)

// resortRow is the resort shape the runner consumes.
type resortRow = dbgen.Resort

// Options configures a Runner.
type Options struct {
	Concurrency   int
	ResortTimeout time.Duration
	Attempts      int
}

// Runner executes one forecaster pass.
type Runner struct {
	store    db.StoreService
	weather  weather.WeatherService
	notifier notify.NotificationService
	logger   *slog.Logger
	now      func() time.Time
	opts     Options
}

// NewRunner builds a Runner. Now is injected so the notification-window
// arithmetic is testable and stable across day boundaries.
func NewRunner(
	store db.StoreService,
	weatherClient weather.WeatherService,
	notifier notify.NotificationService,
	logger *slog.Logger,
	now func() time.Time,
	opts Options,
) *Runner {
	if opts.Concurrency <= 0 {
		opts.Concurrency = defaultConcurrency
	}

	if opts.ResortTimeout <= 0 {
		opts.ResortTimeout = defaultResortTimeout
	}

	if opts.Attempts <= 0 {
		opts.Attempts = defaultAttempts
	}

	if now == nil {
		now = time.Now
	}

	return &Runner{
		store:    store,
		weather:  weatherClient,
		notifier: notifier,
		logger:   logger,
		now:      now,
		opts:     opts,
	}
}

// summary accumulates run counters across goroutines.
type summary struct {
	mu  sync.Mutex
	run metrics.ForecastRun
}

func (s *summary) add(apply func(*metrics.ForecastRun)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	apply(&s.run)
}

// Run checks every resort and delivers any alerts that match.
//
// A failure for one resort does not abort the others; the returned summary
// reports what happened so the caller can set an exit code and publish metrics.
func (r *Runner) Run(ctx context.Context) (metrics.ForecastRun, error) {
	start := r.now()

	resorts, err := r.store.ListAllResorts(ctx)
	if err != nil {
		return metrics.ForecastRun{Duration: r.now().Sub(start)}, fmt.Errorf("listing resorts: %w", err)
	}

	totals := &summary{}

	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(r.opts.Concurrency)

	for _, resort := range resorts {
		group.Go(func() error {
			// Each resort gets its own deadline. A single shared deadline for the
			// whole run meant that once it expired, every remaining resort failed
			// instantly and the job still reported success.
			resortCtx, cancel := context.WithTimeout(groupCtx, r.opts.ResortTimeout)
			defer cancel()

			r.processResort(resortCtx, resort, totals)

			return nil
		})
	}

	if waitErr := group.Wait(); waitErr != nil {
		totals.add(func(run *metrics.ForecastRun) { run.Succeeded = false })

		return totals.run, fmt.Errorf("forecast run: %w", waitErr)
	}

	totals.add(func(run *metrics.ForecastRun) {
		run.Duration = r.now().Sub(start)
		// A run that could not reach any resort is a failure, not a quiet success.
		run.Succeeded = run.ResortsFailed == 0
	})

	return totals.run, nil
}

func (r *Runner) processResort(ctx context.Context, resort resortRow, totals *summary) {
	logger := r.logger.With("resort", resort.Name)

	if !resort.Latitude.Valid || !resort.Longitude.Valid {
		logger.WarnContext(ctx, "skipping resort with missing coordinates")

		return
	}

	totals.add(func(run *metrics.ForecastRun) { run.ResortsChecked++ })

	predictions, err := r.fetchForecast(ctx, resort.Latitude.Float64, resort.Longitude.Float64)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get forecast", "error", err)
		totals.add(func(run *metrics.ForecastRun) { run.ResortsFailed++ })

		return
	}

	for _, prediction := range predictions {
		r.processPrediction(ctx, resort, prediction, logger, totals)
	}
}

func (r *Runner) processPrediction(
	ctx context.Context,
	resort resortRow,
	prediction weather.WeatherPrediction,
	logger *slog.Logger,
	totals *summary,
) {
	daysAhead := r.daysAhead(prediction.Date)

	alerts, err := r.store.GetAlertMatches(
		ctx, resort.Uuid.String(), prediction.Date, prediction.SnowAmount, daysAhead,
	)
	if err != nil {
		logger.ErrorContext(ctx, "failed to find matching alerts", "error", err)
		totals.add(func(run *metrics.ForecastRun) { run.ResortsFailed++ })

		return
	}

	totals.add(func(run *metrics.ForecastRun) { run.AlertsMatched += len(alerts) })

	for _, alert := range alerts {
		r.deliver(ctx, alert, logger, totals)
	}
}

// deliver sends one alert and records it only if the send succeeded.
func (r *Runner) deliver(
	ctx context.Context,
	alert db.AlertToSend,
	logger *slog.Logger,
	totals *summary,
) {
	if alert.UserPhone == "" {
		// Recording history here would permanently suppress this forecast for the
		// user even if they add a phone number later.
		logger.WarnContext(ctx, "skipping alert for user with no phone number")

		return
	}

	message := notify.FormatSnowAlertMessage(alert)
	if err := r.notifier.SendSMS(alert.UserPhone, message); err != nil {
		// Do not record an alert that was never delivered.
		logger.ErrorContext(ctx, "failed to send SMS", "error", err)
		totals.add(func(run *metrics.ForecastRun) { run.AlertsFailed++ })

		return
	}

	totals.add(func(run *metrics.ForecastRun) { run.AlertsSent++ })

	if err := r.store.RecordAlertSent(ctx, alert); err != nil {
		// The SMS is already delivered, so the next run would send a duplicate.
		logger.ErrorContext(ctx, "SMS delivered but recording history failed, next run may duplicate",
			"error", err)
		totals.add(func(run *metrics.ForecastRun) { run.RecordsFailed++ })
	}
}

// fetchForecast retries transient weather API failures, which previously
// dropped a resort for the entire run.
func (r *Runner) fetchForecast(
	ctx context.Context,
	lat, lon float64,
) ([]weather.WeatherPrediction, error) {
	var lastErr error

	for attempt := 1; attempt <= r.opts.Attempts; attempt++ {
		predictions, err := r.weather.GetSnowForecast(ctx, lat, lon)
		if err == nil {
			return predictions, nil
		}

		lastErr = err

		// A cancelled context will not recover, so stop immediately.
		if ctx.Err() != nil {
			break
		}

		if attempt < r.opts.Attempts {
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("forecast fetch cancelled: %w", ctx.Err())
			case <-time.After(time.Duration(attempt) * retryBackoff):
			}
		}
	}

	if lastErr == nil {
		lastErr = errors.New("forecast unavailable")
	}

	return nil, fmt.Errorf("after %d attempts: %w", r.opts.Attempts, lastErr)
}

// daysAhead reports how many days separate today from the forecast date.
//
// Both sides are reduced to a calendar date in UTC, matching how forecast dates
// are parsed. Subtracting raw instants shifted the notification window by a day
// for part of each day.
func (r *Runner) daysAhead(forecastDate time.Time) int32 {
	today := truncateToDate(r.now().UTC())
	target := truncateToDate(forecastDate.UTC())

	days := int32(target.Sub(today).Hours() / hoursPerDay)
	if days < 0 {
		return 0
	}

	return days
}

func truncateToDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
