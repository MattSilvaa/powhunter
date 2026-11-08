//nolint:testpackage // exercises unexported handler internals, as the sibling handler tests do
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MattSilvaa/powhunter/internal/snowreport"
)

// countingBuilder returns a live report and counts how often it is asked.
type countingBuilder struct {
	calls atomic.Int32
	fail  atomic.Bool
	gate  chan struct{}
}

func (b *countingBuilder) build(_ context.Context, _ time.Time) (snowreport.Report, error) {
	b.calls.Add(1)

	if b.gate != nil {
		<-b.gate
	}

	if b.fail.Load() {
		return snowreport.Report{}, errors.New("database down")
	}

	return testReport(), nil
}

// testReport is a live board of 12 resorts, more than the home page shows.
func testReport() snowreport.Report {
	resorts := make([]snowreport.LiveResort, 12)
	for i := range resorts {
		resorts[i] = snowreport.LiveResort{Rank: i + 1, Name: fmt.Sprintf("Peak %02d", i+1)}
	}

	resorts[6].Name = "Alta"

	return snowreport.Report{
		Mode: snowreport.ModeLive,
		Live: &snowreport.Live{Days: 10, ResortCount: len(resorts), Resorts: resorts},
	}
}

type testClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *testClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.now
}

func (c *testClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.now = c.now.Add(d)
}

func newSnowReportTest(builder *countingBuilder) (*SnowReportHandler, *testClock) {
	clock := &testClock{now: time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)}
	handler := NewSnowReportHandlerWithBuilder(builder.build, clock.Now, time.Minute,
		slog.New(slog.DiscardHandler))

	return handler, clock
}

func getSnowReport(handler http.Handler, query string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/snow-report"+query, nil))

	return rec
}

func TestSnowReportIsServedFromCache(t *testing.T) {
	builder := &countingBuilder{}
	handler, clock := newSnowReportTest(builder)

	first := getSnowReport(handler, "")
	// Varying the query string must not get around the cache.
	second := getSnowReport(handler, "?bust=1")

	require.Equal(t, http.StatusOK, first.Code)
	assert.Equal(t, first.Body.String(), second.Body.String())
	assert.Equal(t, int32(1), builder.calls.Load())
	assert.Equal(t, snowReportCacheControl, first.Header().Get("Cache-Control"))
	assert.Equal(t, "nosniff", first.Header().Get("X-Content-Type-Options"))

	var body snowreport.Report
	require.NoError(t, json.Unmarshal(first.Body.Bytes(), &body))
	assert.Equal(t, snowreport.ModeLive, body.Mode)

	clock.advance(2 * time.Minute)
	getSnowReport(handler, "")
	assert.Equal(t, int32(2), builder.calls.Load(), "an expired cache is rebuilt")
}

// A burst arriving with the cache empty must cost one database read, not one
// per request.
func TestConcurrentRequestsShareOneBuild(t *testing.T) {
	builder := &countingBuilder{gate: make(chan struct{})}
	handler, _ := newSnowReportTest(builder)

	const requests = 50

	var wg sync.WaitGroup

	codes := make(chan int, requests)

	for range requests {
		wg.Go(func() {
			codes <- getSnowReport(handler, "").Code
		})
	}

	// Let the requests pile up behind the first build, then release it.
	require.Eventually(t, func() bool { return builder.calls.Load() == 1 }, time.Second, time.Millisecond)
	time.Sleep(20 * time.Millisecond)
	close(builder.gate)
	wg.Wait()
	close(codes)

	for code := range codes {
		assert.Equal(t, http.StatusOK, code)
	}

	assert.Equal(t, int32(1), builder.calls.Load(), "requests that miss the cache together share a build")
}

func TestSnowReportServesTheLastGoodCopyWhenARebuildFails(t *testing.T) {
	builder := &countingBuilder{}
	handler, clock := newSnowReportTest(builder)

	good := getSnowReport(handler, "")

	builder.fail.Store(true)
	clock.advance(2 * time.Minute)

	stale := getSnowReport(handler, "")

	assert.Equal(t, http.StatusOK, stale.Code)
	assert.Equal(t, good.Body.String(), stale.Body.String())
	assert.Equal(t, snowReportStaleCacheControl, stale.Header().Get("Cache-Control"))
}

func TestSnowReportUnavailableWithNothingCached(t *testing.T) {
	builder := &countingBuilder{}
	builder.fail.Store(true)
	handler, _ := newSnowReportTest(builder)

	rec := getSnowReport(handler, "")

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.Contains(t, rec.Body.String(), "SNOW_REPORT_UNAVAILABLE")
	assert.NotContains(t, rec.Body.String(), "database down", "internal errors are not leaked")
}

func TestSnowReportServesOnlyTheTopResorts(t *testing.T) {
	handler, _ := newSnowReportTest(&countingBuilder{})

	var body snowreport.Report
	require.NoError(t, json.Unmarshal(getSnowReport(handler, "").Body.Bytes(), &body))

	assert.Len(t, body.Live.Resorts, snowreport.TopResorts)
	assert.Equal(t, 12, body.Live.ResortCount)
}

func searchSnowReport(handler *SnowReportHandler, query string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	handler.Search(rec, httptest.NewRequest(http.MethodGet, "/api/snow-report/search"+query, nil))

	return rec
}

// Search reads the cached report, so a stream of searches costs no more
// database reads than the home page does.
func TestSnowReportSearchIsAnsweredFromTheCache(t *testing.T) {
	builder := &countingBuilder{}
	handler, _ := newSnowReportTest(builder)

	getSnowReport(handler, "")
	rec := searchSnowReport(handler, "?q=alta")
	searchSnowReport(handler, "?q=peak")

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, int32(1), builder.calls.Load())
	assert.Equal(t, snowReportCacheControl, rec.Header().Get("Cache-Control"))

	var result snowreport.SearchResult
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
	require.Len(t, result.Live, 1)
	assert.Equal(t, 7, result.Live[0].Rank, "a resort outside the top 5 is still found")
}

func TestSnowReportSearchCapsResults(t *testing.T) {
	handler, _ := newSnowReportTest(&countingBuilder{})

	var result snowreport.SearchResult
	require.NoError(t, json.Unmarshal(searchSnowReport(handler, "?q=peak").Body.Bytes(), &result))

	assert.Len(t, result.Live, snowreport.SearchMaxResults)
}

func TestSnowReportSearchRejectsBadQueries(t *testing.T) {
	builder := &countingBuilder{}
	handler, _ := newSnowReportTest(builder)

	for _, query := range []string{"", "?q=", "?q=a", "?q=%20a%20", "?q=" + strings.Repeat("a", 65)} {
		rec := searchSnowReport(handler, query)

		assert.Equal(t, http.StatusBadRequest, rec.Code, query)
		assert.Contains(t, rec.Body.String(), "INVALID_QUERY", query)
	}

	assert.Zero(t, builder.calls.Load(), "a rejected query never builds the report")
}

func TestSnowReportSearchUnavailableWithNothingCached(t *testing.T) {
	builder := &countingBuilder{}
	builder.fail.Store(true)
	handler, _ := newSnowReportTest(builder)

	rec := searchSnowReport(handler, "?q=alta")

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
}

// During an outage, requests keep getting the last good copy without each one
// trying the database again. Search is called on every keystroke, so without
// this backoff an outage would multiply database attempts.
func TestAFailedRebuildIsNotRetriedOnEveryRequest(t *testing.T) {
	builder := &countingBuilder{}
	handler, clock := newSnowReportTest(builder)

	getSnowReport(handler, "")
	builder.fail.Store(true)
	clock.advance(2 * time.Minute)

	for range 5 {
		assert.Equal(t, http.StatusOK, searchSnowReport(handler, "?q=alta").Code)
		assert.Equal(t, http.StatusOK, getSnowReport(handler, "").Code)
	}

	assert.Equal(t, int32(2), builder.calls.Load(), "one good build, then one failed attempt")

	clock.advance(snowReportRetryAfter)
	builder.fail.Store(false)
	getSnowReport(handler, "")

	assert.Equal(t, int32(3), builder.calls.Load(), "it tries again once the backoff has passed")
	assert.Equal(t, snowReportCacheControl, getSnowReport(handler, "").Header().Get("Cache-Control"))
}

func TestAFailedFirstBuildBacksOffToo(t *testing.T) {
	builder := &countingBuilder{}
	builder.fail.Store(true)
	handler, _ := newSnowReportTest(builder)

	for range 5 {
		assert.Equal(t, http.StatusServiceUnavailable, getSnowReport(handler, "").Code)
	}

	assert.Equal(t, int32(1), builder.calls.Load())
}
