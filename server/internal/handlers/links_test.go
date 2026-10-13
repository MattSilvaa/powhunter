//nolint:testpackage // exercises unexported handler internals, as the sibling handler tests do
package handlers

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	testLinkCode     = "Ab12Cd34"
	testDestination  = "https://www.vail.com/snow-report"
	testFallbackURL  = "https://powhunter.app"
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

func followLink(t *testing.T, queries *fakeLinkQueries, method, code, userAgent string) *httptest.ResponseRecorder {
	t.Helper()

	handler := NewLinkHandler(queries, testFallbackURL, slog.New(slog.NewTextHandler(io.Discard, nil)))
	mux := http.NewServeMux()
	mux.HandleFunc("GET /go/{code}", handler.Redirect)

	req := httptest.NewRequest(method, "/go/"+code, nil)
	req.Header.Set("User-Agent", userAgent)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	return rec
}

func TestLinkRedirectCountsAClickAndRedirects(t *testing.T) {
	queries := &fakeLinkQueries{}

	rec := followLink(t, queries, http.MethodGet, testLinkCode, browserUserAgent)

	assert.Equal(t, http.StatusFound, rec.Code)
	assert.Equal(t, testDestination, rec.Header().Get("Location"))
	assert.Equal(t, 1, queries.clicks)
}

func TestLinkRedirectDoesNotCountPreviews(t *testing.T) {
	queries := &fakeLinkQueries{}

	for _, agent := range []string{"facebookexternalhit/1.1 Facebot Twitterbot/1.0", "Slackbot-LinkExpanding 1.0", ""} {
		rec := followLink(t, queries, http.MethodGet, testLinkCode, agent)
		assert.Equal(t, testDestination, rec.Header().Get("Location"), "a preview still follows the link")
	}

	followLink(t, queries, http.MethodHead, testLinkCode, browserUserAgent)

	assert.Zero(t, queries.clicks)
}

func TestLinkRedirectSendsUnknownLinksToTheApp(t *testing.T) {
	queries := &fakeLinkQueries{}

	for _, code := range []string{"Zz99Yy88", "not-a-code"} {
		rec := followLink(t, queries, http.MethodGet, code, browserUserAgent)
		assert.Equal(t, http.StatusFound, rec.Code)
		assert.Equal(t, testFallbackURL, rec.Header().Get("Location"))
	}
}
