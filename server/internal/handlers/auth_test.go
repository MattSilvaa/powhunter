package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MattSilvaa/powhunter/internal/auth"
	"github.com/MattSilvaa/powhunter/internal/db"
	dbgen "github.com/MattSilvaa/powhunter/internal/db/generated"
	"github.com/MattSilvaa/powhunter/internal/db/mocks"
	"github.com/MattSilvaa/powhunter/internal/middleware"
	"github.com/MattSilvaa/powhunter/internal/notify"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// stubAuthService records what it was asked to do and returns canned answers.
type stubAuthService struct {
	issuedFor   string
	issueErr    error
	redeemErr   error
	loggedOut   bool
	cookieSet   bool
	cookieClear bool
}

func (s *stubAuthService) IssueLoginToken(_ context.Context, email string) (string, error) {
	s.issuedFor = email
	if s.issueErr != nil {
		return "", s.issueErr
	}

	return "login-token", nil
}

func (s *stubAuthService) Redeem(context.Context, string) (string, time.Time, error) {
	if s.redeemErr != nil {
		return "", time.Time{}, s.redeemErr
	}

	return "session-token", time.Now().Add(time.Hour), nil
}

func (s *stubAuthService) Logout(context.Context, string) error {
	s.loggedOut = true
	return nil
}

func (s *stubAuthService) SetCookie(http.ResponseWriter, string, time.Time) { s.cookieSet = true }
func (s *stubAuthService) ClearCookie(http.ResponseWriter)                  { s.cookieClear = true }

// captureMailer records messages instead of sending them.
type captureMailer struct {
	sent []notify.Email
	err  error
}

func (m *captureMailer) Send(_ context.Context, msg notify.Email) error {
	if m.err != nil {
		return m.err
	}

	m.sent = append(m.sent, msg)

	return nil
}

func testAuthHandler(service AuthService, mailer notify.Mailer) *AuthHandler {
	return NewAuthHandler(service, mailer, "https://powhunter.app",
		slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func postJSON(t *testing.T, path string, body any) *http.Request {
	t.Helper()

	encoded, err := json.Marshal(body)
	require.NoError(t, err)

	return httptest.NewRequest(http.MethodPost, path, bytes.NewReader(encoded))
}

func TestRequestLinkEmailsASignInLink(t *testing.T) {
	service := &stubAuthService{}
	mailer := &captureMailer{}

	recorder := httptest.NewRecorder()
	testAuthHandler(service, mailer).RequestLink(recorder,
		postJSON(t, "/api/auth/request-link", map[string]string{"email": "Rider@Example.com"}))

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "rider@example.com", service.issuedFor, "the address should be normalised")
	require.Len(t, mailer.sent, 1)
	assert.Equal(t, []string{"rider@example.com"}, mailer.sent[0].To)
	assert.Contains(t, mailer.sent[0].HTML, "https://powhunter.app/login?token=login-token")
}

// Reporting whether an address has an account would turn this endpoint into a
// way to enumerate registered users, so the response must not vary.
func TestRequestLinkResponseDoesNotRevealWhetherSendingWorked(t *testing.T) {
	failing := testAuthHandler(&stubAuthService{issueErr: errors.New("database down")}, &captureMailer{})
	working := testAuthHandler(&stubAuthService{}, &captureMailer{})

	failed := httptest.NewRecorder()
	failing.RequestLink(failed, postJSON(t, "/api/auth/request-link", map[string]string{"email": "rider@example.com"}))

	succeeded := httptest.NewRecorder()
	working.RequestLink(succeeded, postJSON(t, "/api/auth/request-link", map[string]string{"email": "rider@example.com"}))

	assert.Equal(t, http.StatusOK, failed.Code)
	assert.Equal(t, succeeded.Code, failed.Code)
	assert.JSONEq(t, succeeded.Body.String(), failed.Body.String())
}

func TestRequestLinkRejectsAnInvalidAddress(t *testing.T) {
	mailer := &captureMailer{}

	recorder := httptest.NewRecorder()
	testAuthHandler(&stubAuthService{}, mailer).RequestLink(recorder,
		postJSON(t, "/api/auth/request-link", map[string]string{"email": "not-an-email"}))

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Empty(t, mailer.sent)
}

func TestCallbackSetsTheSessionCookie(t *testing.T) {
	service := &stubAuthService{}

	recorder := httptest.NewRecorder()
	testAuthHandler(service, &captureMailer{}).Callback(recorder,
		postJSON(t, "/api/auth/callback", map[string]string{"token": "login-token"}))

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.True(t, service.cookieSet)
}

func TestCallbackRejectsASpentOrExpiredToken(t *testing.T) {
	service := &stubAuthService{redeemErr: auth.ErrInvalidToken}

	recorder := httptest.NewRecorder()
	testAuthHandler(service, &captureMailer{}).Callback(recorder,
		postJSON(t, "/api/auth/callback", map[string]string{"token": "used-token"}))

	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	assert.False(t, service.cookieSet)
}

func TestLogoutClearsTheCookie(t *testing.T) {
	service := &stubAuthService{}

	recorder := httptest.NewRecorder()
	testAuthHandler(service, &captureMailer{}).Logout(recorder,
		httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil))

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.True(t, service.loggedOut)
	assert.True(t, service.cookieClear)
}

func TestMeReportsSignedOutWithoutASession(t *testing.T) {
	recorder := httptest.NewRecorder()
	testAuthHandler(&stubAuthService{}, &captureMailer{}).Me(recorder,
		httptest.NewRequest(http.MethodGet, "/api/auth/me", nil))

	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
}

// requestAsUser builds a request carrying an authenticated session, the way the
// session middleware would have left it.
func requestAsUser(t *testing.T, method, target string, user auth.User) *http.Request {
	t.Helper()

	req := httptest.NewRequest(method, target, nil)

	var authenticated *http.Request

	middleware.Session(stubAuthenticator{token: "token", user: user}, slog.New(slog.NewTextHandler(io.Discard, nil)))(
		http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			authenticated = r
		}),
	).ServeHTTP(httptest.NewRecorder(), addSessionCookie(req))

	require.NotNil(t, authenticated)

	return authenticated
}

func addSessionCookie(r *http.Request) *http.Request {
	r.AddCookie(&http.Cookie{Name: auth.CookieName, Value: "token"})
	return r
}

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

// The regression that matters: alerts are read for the session's user, never
// for an address the caller supplies. Passing someone else's email must not
// reach their data.
func TestGetUserAlertsIgnoresACallerSuppliedEmail(t *testing.T) {
	handler, store := testAlertHandler(t)
	signedIn := auth.User{UUID: uuid.New(), Email: "rider@example.com"}

	store.EXPECT().
		GetUserAlerts(gomock.Any(), signedIn.UUID).
		Return([]dbgen.GetUserAlertsByUserUUIDRow{}, nil)

	recorder := httptest.NewRecorder()
	handler.GetUserAlerts(recorder,
		requestAsUser(t, http.MethodGet, "/api/user/alerts?email=victim@example.com", signedIn))

	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestUserAlertEndpointsRejectAnUnauthenticatedCaller(t *testing.T) {
	handler, _ := testAlertHandler(t)

	cases := map[string]struct {
		serve  func(http.ResponseWriter, *http.Request)
		method string
		target string
	}{
		"read":       {handler.GetUserAlerts, http.MethodGet, "/api/user/alerts?email=victim@example.com"},
		"delete":     {handler.DeleteUserAlert, http.MethodDelete, "/api/user/alerts/delete?email=victim@example.com"},
		"delete all": {handler.DeleteAllUserAlerts, http.MethodDelete, "/api/user/alerts/delete-all?email=victim@example.com"},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			tc.serve(recorder, httptest.NewRequest(tc.method, tc.target, nil))

			assert.Equal(t, http.StatusUnauthorized, recorder.Code)
		})
	}
}

func TestDeleteUserAlertDeletesOnlyTheSessionUsersAlert(t *testing.T) {
	handler, store := testAlertHandler(t)
	signedIn := auth.User{UUID: uuid.New(), Email: "rider@example.com"}
	resortUUID := testResortUUID1

	store.EXPECT().
		DeleteAlertForUser(gomock.Any(), signedIn.UUID, resortUUID).
		Return(nil)

	recorder := httptest.NewRecorder()
	handler.DeleteUserAlert(recorder, requestAsUser(t, http.MethodDelete,
		"/api/user/alerts/delete?resort_uuid="+resortUUID+"&email=victim@example.com", signedIn))

	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestDeleteUserAlertRejectsAMalformedResort(t *testing.T) {
	handler, _ := testAlertHandler(t)
	signedIn := auth.User{UUID: uuid.New()}

	recorder := httptest.NewRecorder()
	handler.DeleteUserAlert(recorder, requestAsUser(t, http.MethodDelete,
		"/api/user/alerts/delete?resort_uuid=not-a-uuid", signedIn))

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestDeleteAllUserAlertsScopesToTheSessionUser(t *testing.T) {
	handler, store := testAlertHandler(t)
	signedIn := auth.User{UUID: uuid.New()}

	store.EXPECT().
		DeleteAllAlertsForUser(gomock.Any(), signedIn.UUID).
		Return(nil)

	recorder := httptest.NewRecorder()
	handler.DeleteAllUserAlerts(recorder,
		requestAsUser(t, http.MethodDelete, "/api/user/alerts/delete-all", signedIn))

	assert.Equal(t, http.StatusOK, recorder.Code)
}

// Keep the compile-time link to the store interface the handlers depend on.
var _ db.StoreService = (*mocks.MockStoreService)(nil)
