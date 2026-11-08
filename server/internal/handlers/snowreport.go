package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

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

// snowReportCache is one built report: the full report, which search reads,
// and the encoded top of it, which the home page reads.
type snowReportCache struct {
	report snowreport.Report
	body   []byte
}

// SnowReportBuilder assembles the report. It is a function so tests can count
// and fail builds without a database.
type SnowReportBuilder func(ctx context.Context, now time.Time) (snowreport.Report, error)

// SnowReportHandler serves the home page's snow report.
//
// It is public and unauthenticated, so its cost must not grow with traffic:
//   - it never calls Open-Meteo; it reads what the forecaster stored,
//   - the encoded response is cached in memory and shared by every visitor,
//   - concurrent requests that find the cache expired share a single rebuild,
//   - the home page route takes no input and always returns the top
//     snowreport.TopResorts, however many resorts there are,
//   - search reads that same in-memory report, with a bounded query and a
//     bounded number of results, and never touches the database itself,
//   - if a rebuild fails, the last good copy is served instead of an error.
//
// The global per-IP read limiter in front of every route still applies.
type SnowReportHandler struct {
	build  SnowReportBuilder
	now    func() time.Time
	ttl    time.Duration
	logger *slog.Logger

	group singleflight.Group

	mu      sync.RWMutex
	cached  *snowReportCache
	builtAt time.Time
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

// ServeHTTP answers GET /api/snow-report with the top of the board. The query
// string is ignored.
func (h *SnowReportHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	setSecurityHeaders(w)

	cached, stale := h.load(r)
	if cached == nil {
		sendErrorResponse(w, "SNOW_REPORT_UNAVAILABLE",
			"The snow report is unavailable right now", http.StatusServiceUnavailable)

		return
	}

	writeSnowReport(w, cached.body, cacheControl(stale))
}

// Search answers GET /api/snow-report/search?q=… with the resorts whose name
// matches, from the same in-memory report.
func (h *SnowReportHandler) Search(w http.ResponseWriter, r *http.Request) {
	setSecurityHeaders(w)

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if length := utf8.RuneCountInString(query); length < snowreport.SearchMinQueryLen ||
		length > snowreport.SearchMaxQueryLen {
		sendErrorResponse(w, "INVALID_QUERY",
			fmt.Sprintf("Search needs %d to %d characters",
				snowreport.SearchMinQueryLen, snowreport.SearchMaxQueryLen),
			http.StatusBadRequest)

		return
	}

	cached, stale := h.load(r)
	if cached == nil {
		sendErrorResponse(w, "SNOW_REPORT_UNAVAILABLE",
			"The snow report is unavailable right now", http.StatusServiceUnavailable)

		return
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(cached.report.Search(query)); err != nil {
		h.logger.ErrorContext(r.Context(), "failed to encode snow report search", "error", err)
		sendErrorResponse(w, "INTERNAL_ERROR", "Something went wrong", http.StatusInternalServerError)

		return
	}

	writeSnowReport(w, buf.Bytes(), cacheControl(stale))
}

// load returns the cached report, rebuilding it when it has expired. If the
// rebuild fails it falls back to the last good copy and reports it as stale;
// it returns nil only when there has never been a good copy.
func (h *SnowReportHandler) load(r *http.Request) (*snowReportCache, bool) {
	if cached, ok := h.fresh(); ok {
		return cached, false
	}

	// A detached context: one visitor closing the tab must not fail the
	// rebuild every other waiting request shares.
	result, err, _ := h.group.Do("report", func() (any, error) {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), handlerTimeout)
		defer cancel()

		return h.refresh(ctx)
	})
	if err == nil {
		cached, _ := result.(*snowReportCache)

		return cached, false
	}

	h.logger.ErrorContext(r.Context(), "failed to build snow report", "error", err)

	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.cached, h.cached != nil
}

func (h *SnowReportHandler) fresh() (*snowReportCache, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.cached == nil || h.now().Sub(h.builtAt) >= h.ttl {
		return nil, false
	}

	return h.cached, true
}

func (h *SnowReportHandler) refresh(ctx context.Context) (*snowReportCache, error) {
	// Another request may have rebuilt it while this one waited for its turn.
	if cached, ok := h.fresh(); ok {
		return cached, nil
	}

	now := h.now()

	report, err := h.build(ctx, now)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if encodeErr := json.NewEncoder(&buf).Encode(report.Top(snowreport.TopResorts)); encodeErr != nil {
		return nil, fmt.Errorf("encoding snow report: %w", encodeErr)
	}

	cached := &snowReportCache{report: report, body: buf.Bytes()}

	h.mu.Lock()
	h.cached = cached
	h.builtAt = now
	h.mu.Unlock()

	return cached, nil
}

func cacheControl(stale bool) string {
	if stale {
		return snowReportStaleCacheControl
	}

	return snowReportCacheControl
}

func writeSnowReport(w http.ResponseWriter, body []byte, cacheControl string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", cacheControl)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}
