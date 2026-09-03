package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MattSilvaa/powhunter/internal/auth"
	"github.com/MattSilvaa/powhunter/internal/middleware"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubAuthenticator resolves exactly one token.
type stubAuthenticator struct {
	token string
	user  auth.User
}

func (s stubAuthenticator) Authenticate(_ context.Context, token string) (auth.User, error) {
	if token != s.token {
		return auth.User{}, auth.ErrNoSession
	}

	return s.user, nil
}

func sessionHandler(t *testing.T, authenticator middleware.Authenticator, handler http.Handler) http.Handler {
	t.Helper()

	return middleware.Session(authenticator, discardLogger())(handler)
}

func TestSessionAttachesTheUserForAValidCookie(t *testing.T) {
	want := auth.User{UUID: uuid.New(), Email: "rider@example.com"}
	authenticator := stubAuthenticator{token: "good-token", user: want}

	var got auth.User

	var found bool

	handler := sessionHandler(t, authenticator, http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got, found = middleware.UserFromContext(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/user/alerts", nil)
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: "good-token"})

	handler.ServeHTTP(httptest.NewRecorder(), req)

	require.True(t, found)
	assert.Equal(t, want, got)
}

// An expired or forged cookie is an ordinary signed-out request, not an error.
func TestSessionPassesThroughAnUnknownCookie(t *testing.T) {
	authenticator := stubAuthenticator{token: "good-token"}

	var found bool

	handler := sessionHandler(t, authenticator, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, found = middleware.UserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/resorts", nil)
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: "stale-token"})

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	assert.False(t, found)
	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestRequireUserRejectsAnUnauthenticatedRequest(t *testing.T) {
	called := false
	handler := middleware.RequireUser(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/user/alerts", nil))

	assert.False(t, called, "the protected handler must not run")
	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "UNAUTHENTICATED")
}

func TestRequireUserAllowsAnAuthenticatedRequest(t *testing.T) {
	authenticator := stubAuthenticator{token: "good-token", user: auth.User{UUID: uuid.New()}}

	called := false
	handler := sessionHandler(t, authenticator, middleware.RequireUser(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		}),
	))

	req := httptest.NewRequest(http.MethodGet, "/api/user/alerts", nil)
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: "good-token"})

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusOK, recorder.Code)
}
