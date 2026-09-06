package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MattSilvaa/powhunter/internal/middleware"
	"github.com/stretchr/testify/assert"
)

func guarded(token string) http.Handler {
	return middleware.RequireBearerToken(token)(okHandler())
}

// /metrics was readable by anyone who found it. A scraper is the only caller
// that needs it.
func TestBearerTokenRejectsAnUnauthenticatedCaller(t *testing.T) {
	recorder := httptest.NewRecorder()
	guarded("secret").ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	assert.Equal(t, "Bearer", recorder.Header().Get("WWW-Authenticate"))
}

func TestBearerTokenAcceptsTheConfiguredToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("Authorization", "Bearer secret")

	recorder := httptest.NewRecorder()
	guarded("secret").ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestBearerTokenRejectsAWrongOrMalformedToken(t *testing.T) {
	cases := map[string]string{
		"wrong token":    "Bearer nope",
		"no scheme":      "secret",
		"wrong scheme":   "Basic secret",
		"empty":          "",
		"prefix of real": "Bearer secre",
		"longer":         "Bearer secretsecret",
	}

	for name, header := range cases {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
			if header != "" {
				req.Header.Set("Authorization", header)
			}

			recorder := httptest.NewRecorder()
			guarded("secret").ServeHTTP(recorder, req)

			assert.Equal(t, http.StatusUnauthorized, recorder.Code)
		})
	}
}

// Development runs without a token; requiring one there would mean every
// contributor had to set it to see their own metrics. config.ValidateWeb is
// what stops that default reaching production.
func TestAnEmptyTokenLeavesTheRouteOpen(t *testing.T) {
	recorder := httptest.NewRecorder()
	guarded("").ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	assert.Equal(t, http.StatusOK, recorder.Code)
}
