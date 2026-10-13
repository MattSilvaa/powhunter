//nolint:testpackage // exercises unexported handler internals, as the sibling handler tests do
package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dbgen "github.com/MattSilvaa/powhunter/internal/db/generated"
)

const (
	testLinkCode     = "Ab12Cd34"
	testDestination  = "https://www.vail.com/snow-report"
	testAppBaseURL   = "https://powhunter.app"
	testAlertPageURL = testAppBaseURL + "/alert/" + testLinkCode
	browserUserAgent = "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X)"
)

// fakeLinkQueries knows one link and counts the clicks recorded against it.
type fakeLinkQueries struct {
	clicks int
}

func (f *fakeLinkQueries) RecordAlertLinkClick(_ context.Context, code string) (string, error) {
	if code != testLinkCode {
		return "", sql.ErrNoRows
	}

	f.clicks++

	return testDestination, nil
}

func (f *fakeLinkQueries) GetAlertLinkDestination(_ context.Context, code string) (string, error) {
	if code != testLinkCode {
		return "", sql.ErrNoRows
	}

	return testDestination, nil
}

func (f *fakeLinkQueries) GetAlertLinkDetails(_ context.Context, code string) (dbgen.GetAlertLinkDetailsRow, error) {
	if code != testLinkCode {
		return dbgen.GetAlertLinkDetailsRow{}, sql.ErrNoRows
	}

	return dbgen.GetAlertLinkDetailsRow{
		ResortUuid:     uuid.MustParse("11111111-1111-4111-8111-111111111111"),
		ResortName:     "Alta",
		SnowAmount:     14,
		ForecastDate:   time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC),
		DestinationUrl: testDestination,
	}, nil
}

func newTestLinkMux(queries *fakeLinkQueries) *http.ServeMux {
	handler := NewLinkHandler(queries, testAppBaseURL, slog.New(slog.NewTextHandler(io.Discard, nil)))
	mux := http.NewServeMux()
	mux.HandleFunc("GET /go/{code}", handler.Redirect)
	mux.HandleFunc("GET /api/alert-links/{code}", handler.Details)

	return mux
}

func followLink(t *testing.T, queries *fakeLinkQueries, method, code, userAgent string) *httptest.ResponseRecorder {
	t.Helper()

	mux := newTestLinkMux(queries)

	req := httptest.NewRequest(method, "/go/"+code, nil)
	req.Header.Set("User-Agent", userAgent)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	return rec
}

func TestLinkRedirectCountsAClickAndOpensTheAlertPage(t *testing.T) {
	queries := &fakeLinkQueries{}

	rec := followLink(t, queries, http.MethodGet, testLinkCode, browserUserAgent)

	assert.Equal(t, http.StatusFound, rec.Code)
	assert.Equal(t, testAlertPageURL, rec.Header().Get("Location"))
	assert.Equal(t, 1, queries.clicks)
}

func TestLinkRedirectDoesNotCountPreviews(t *testing.T) {
	queries := &fakeLinkQueries{}

	for _, agent := range []string{"facebookexternalhit/1.1 Facebot Twitterbot/1.0", "Slackbot-LinkExpanding 1.0", ""} {
		rec := followLink(t, queries, http.MethodGet, testLinkCode, agent)
		assert.Equal(t, testAlertPageURL, rec.Header().Get("Location"), "a preview still follows the link")
	}

	followLink(t, queries, http.MethodHead, testLinkCode, browserUserAgent)

	assert.Zero(t, queries.clicks)
}

func TestLinkRedirectSendsUnknownLinksToTheApp(t *testing.T) {
	queries := &fakeLinkQueries{}

	for _, code := range []string{"Zz99Yy88", "not-a-code"} {
		rec := followLink(t, queries, http.MethodGet, code, browserUserAgent)
		assert.Equal(t, http.StatusFound, rec.Code)
		assert.Equal(t, testAppBaseURL, rec.Header().Get("Location"))
	}
}

func TestAlertLinkDetails(t *testing.T) {
	mux := newTestLinkMux(&fakeLinkQueries{})

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/alert-links/"+testLinkCode, nil))

	require.Equal(t, http.StatusOK, rec.Code)

	var body AlertLinkDetails
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, AlertLinkDetails{
		ResortUUID:   "11111111-1111-4111-8111-111111111111",
		ResortName:   "Alta",
		SnowAmount:   14,
		ForecastDate: "2026-10-16",
		ReportURL:    testDestination,
	}, body)
}

func TestAlertLinkDetailsNotFound(t *testing.T) {
	mux := newTestLinkMux(&fakeLinkQueries{})

	for _, code := range []string{"Zz99Yy88", "not-a-code"} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/alert-links/"+code, nil))
		assert.Equal(t, http.StatusNotFound, rec.Code, code)
	}
}
