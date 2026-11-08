package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/MattSilvaa/powhunter/internal/snowreport"
)

const (
	// snowReportTTL is how long a built report is served from memory. The
	// forecaster only writes twice a day, so five minutes costs nothing in
	// freshness and caps the database at one read per five minutes, however
	// much traffic arrives.
	snowReportTTL = 5 * time.Minute
	// snowReportCacheControl lets browsers and any CDN in front absorb repeat
	// requests too. The body is the same for every visitor, so public caching
	// is safe.
	snowReportCacheControl = "public, max-age=300, stale-while-revalidate=600"
	// snowReportStaleCacheControl is sent with a stale copy served because a
	// rebuild failed, so caches come back soon rather than holding it.
	snowReportStaleCacheControl = "public, max-age=60"
)

// SnowReportBuilder assembles the report. It is a function so tests can count
// and fail builds without a database.
type SnowReportBuilder func(ctx context.Context, now time.Time) (snowreport.Report, error)

// SnowReportHandler serves the home page's snow report.
//
// It is public and unauthenticated, so its cost must not grow with traffic:
//   - it never calls Open-Meteo; it reads what the forecaster stored,
//   - the encoded response is cached in memory and shared by every visitor,
//   - concurrent requests that find the cache expired share a single rebuild,
//   - it takes no input, so there is nothing to vary the work or the cache on,
//   - if a rebuild fails, the last good copy is served instead of an error.
//
// The global per-IP read limiter in front of every route still applies.
type SnowReportHandler struct {
	build  SnowReportBuilder
	now    func() time.Time
	ttl    time.Duration
	logger *slog.Logger

	group singleflight.Group

	mu       sync.RWMutex
	body     []byte
	builtAt  time.Time
	hasValue bool
}

// NewSnowReportHandler builds a handler that reads from store.
func NewSnowReportHandler(store snowreport.ReportStore, logger *slog.Logger) *SnowReportHandler {
	return NewSnowReportHandlerWithBuilder(
		func(ctx context.Context, now time.Time) (snowreport.Report, error) {
			return snowreport.Build(ctx, store, now)
		},
		time.Now, snowReportTTL, logger)
}

// NewSnowReportHandlerWithBuilder lets tests supply the builder and clock.
func NewSnowReportHandlerWithBuilder(
	build SnowReportBuilder,
	now func() time.Time,
	ttl time.Duration,
	logger *slog.Logger,
) *SnowReportHandler {
	return &SnowReportHandler{build: build, now: now, ttl: ttl, logger: logger}
}

// ServeHTTP answers GET /api/snow-report. The query string is ignored.
func (h *SnowReportHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	setSecurityHeaders(w)

	if body, ok := h.fresh(); ok {
		writeSnowReport(w, body, snowReportCacheControl)

		return
	}

	// A detached context: one visitor closing the tab must not fail the
	// rebuild every other waiting request shares.
	result, err, _ := h.group.Do("report", func() (any, error) {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), handlerTimeout)
		defer cancel()

		return h.refresh(ctx)
	})
	if err == nil {
		body, _ := result.([]byte)
		writeSnowReport(w, body, snowReportCacheControl)

		return
	}

	h.logger.ErrorContext(r.Context(), "failed to build snow report", "error", err)

	if body, ok := h.stale(); ok {
		writeSnowReport(w, body, snowReportStaleCacheControl)

		return
	}

	sendErrorResponse(w, "SNOW_REPORT_UNAVAILABLE",
		"The snow report is unavailable right now", http.StatusServiceUnavailable)
}

func (h *SnowReportHandler) fresh() ([]byte, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if !h.hasValue || h.now().Sub(h.builtAt) >= h.ttl {
		return nil, false
	}

	return h.body, true
}

func (h *SnowReportHandler) stale() ([]byte, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.body, h.hasValue
}

func (h *SnowReportHandler) refresh(ctx context.Context) ([]byte, error) {
	// Another request may have rebuilt it while this one waited for its turn.
	if body, ok := h.fresh(); ok {
		return body, nil
	}

	now := h.now()

	report, err := h.build(ctx, now)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if encodeErr := json.NewEncoder(&buf).Encode(report); encodeErr != nil {
		return nil, fmt.Errorf("encoding snow report: %w", encodeErr)
	}

	h.mu.Lock()
	h.body = buf.Bytes()
	h.builtAt = now
	h.hasValue = true
	h.mu.Unlock()

	return buf.Bytes(), nil
}

func writeSnowReport(w http.ResponseWriter, body []byte, cacheControl string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", cacheControl)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}
