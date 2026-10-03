//nolint:testpackage // exercises unexported handler internals, as the sibling handler tests do
package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MattSilvaa/powhunter/internal/auth"
	dbgen "github.com/MattSilvaa/powhunter/internal/db/generated"
	"github.com/MattSilvaa/powhunter/internal/db/mocks"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func testAccountHandler(t *testing.T) (*AccountHandler, *mocks.MockStoreService) {
	store := mocks.NewMockStoreService(gomock.NewController(t))
	return NewAccountHandler(store, slog.New(slog.NewTextHandler(io.Discard, nil))), store
}

func errorCode(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()

	var body ErrorResponse
	require.NoError(t, json.NewDecoder(recorder.Body).Decode(&body))

	return body.Error
}

func TestAccountEndpointsRejectAnUnauthenticatedCaller(t *testing.T) {
	handler, _ := testAccountHandler(t)

	cases := map[string]struct {
		serve  func(http.ResponseWriter, *http.Request)
		method string
		target string
	}{
		"add alerts": {handler.CreateUserAlerts, http.MethodPost, "/api/user/alerts"},
		"update alert": {
			handler.UpdateUserAlert,
			http.MethodPatch,
			"/api/user/alerts?resort_uuid=" + testResortUUID1,
		},
		"update profile": {handler.UpdateProfile, http.MethodPatch, "/api/user/profile"},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			tc.serve(recorder, httptest.NewRequest(tc.method, tc.target, strings.NewReader(`{}`)))

			assert.Equal(t, http.StatusUnauthorized, recorder.Code)
		})
	}
}

func TestCreateUserAlerts(t *testing.T) {
	signedIn := auth.User{UUID: uuid.New(), Email: "rider@example.com"}
	validBody := fmt.Sprintf(
		`{"minSnowAmount":6,"notificationDays":3,"resortsUuids":[%q]}`, testResortUUID1)

	tests := []struct {
		name       string
		body       string
		storeErr   error
		expectCall bool
		wantStatus int
		wantCode   string
	}{
		{
			name:       "adds alerts to the session's account",
			body:       validBody,
			expectCall: true,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "rejects an email in the body",
			body:       `{"email":"victim@example.com","minSnowAmount":6,"notificationDays":3,"resortsUuids":[]}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "INVALID_REQUEST",
		},
		{
			name:       "requires a resort",
			body:       `{"minSnowAmount":6,"notificationDays":3,"resortsUuids":[]}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "MISSING_RESORTS",
		},
		{
			name: "rejects an out-of-range threshold",
			body: fmt.Sprintf(
				`{"minSnowAmount":0,"notificationDays":3,"resortsUuids":[%q]}`, testResortUUID1),
			wantStatus: http.StatusBadRequest,
			wantCode:   "VALIDATION_ERROR",
		},
		{
			name:       "reports a resort the user already has",
			body:       validBody,
			storeErr:   &pq.Error{Code: "23505", Constraint: "user_alerts_user_uuid_resort_uuid_key"},
			expectCall: true,
			wantStatus: http.StatusConflict,
			wantCode:   "DUPLICATE_ALERT",
		},
		{
			name:       "reports a store failure",
			body:       validBody,
			storeErr:   errors.New("database error"),
			expectCall: true,
			wantStatus: http.StatusInternalServerError,
			wantCode:   "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, store := testAccountHandler(t)

			if tt.expectCall {
				store.EXPECT().
					CreateAlertsForUser(gomock.Any(), signedIn.UUID, 6.0, int32(3), []string{testResortUUID1}).
					Return(tt.storeErr)
			}

			recorder := httptest.NewRecorder()
			handler.CreateUserAlerts(recorder, requestAsUserWithBody(t, http.MethodPost,
				"/api/user/alerts", strings.NewReader(tt.body), signedIn))

			assert.Equal(t, tt.wantStatus, recorder.Code)

			if tt.wantCode != "" {
				assert.Equal(t, tt.wantCode, errorCode(t, recorder))
			}
		})
	}
}

func TestUpdateUserAlert(t *testing.T) {
	signedIn := auth.User{UUID: uuid.New()}
	target := "/api/user/alerts?resort_uuid=" + testResortUUID1
	validBody := `{"minSnowAmount":8,"notificationDays":5}`

	tests := []struct {
		name       string
		target     string
		body       string
		storeErr   error
		expectCall bool
		wantStatus int
		wantCode   string
	}{
		{
			name:       "updates the session user's alert",
			target:     target,
			body:       validBody,
			expectCall: true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "rejects a malformed resort",
			target:     "/api/user/alerts?resort_uuid=not-a-uuid",
			body:       validBody,
			wantStatus: http.StatusBadRequest,
			wantCode:   "MISSING_RESORT",
		},
		{
			name:       "rejects an out-of-range window",
			target:     target,
			body:       `{"minSnowAmount":8,"notificationDays":11}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "VALIDATION_ERROR",
		},
		{
			name:       "reports an alert the user does not have",
			target:     target,
			body:       validBody,
			storeErr:   fmt.Errorf("wrapped: %w", sql.ErrNoRows),
			expectCall: true,
			wantStatus: http.StatusNotFound,
			wantCode:   "ALERT_NOT_FOUND",
		},
		{
			name:       "reports a store failure",
			target:     target,
			body:       validBody,
			storeErr:   errors.New("database error"),
			expectCall: true,
			wantStatus: http.StatusInternalServerError,
			wantCode:   "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, store := testAccountHandler(t)

			if tt.expectCall {
				store.EXPECT().
					UpdateAlertForUser(gomock.Any(), signedIn.UUID, testResortUUID1, 8.0, int32(5)).
					Return(dbgen.UserAlert{MinSnowAmount: 8, NotificationDays: 5}, tt.storeErr)
			}

			recorder := httptest.NewRecorder()
			handler.UpdateUserAlert(recorder, requestAsUserWithBody(t, http.MethodPatch,
				tt.target, strings.NewReader(tt.body), signedIn))

			assert.Equal(t, tt.wantStatus, recorder.Code)

			if tt.wantCode != "" {
				assert.Equal(t, tt.wantCode, errorCode(t, recorder))
			}
		})
	}
}

func TestUpdateProfile(t *testing.T) {
	signedIn := auth.User{UUID: uuid.New(), Email: "rider@example.com", EmailVerified: true}

	t.Run("stores the normalized number and returns the account", func(t *testing.T) {
		handler, store := testAccountHandler(t)
		store.EXPECT().UpdateUserPhone(gomock.Any(), signedIn.UUID, "+11234567890").Return(nil)

		recorder := httptest.NewRecorder()
		handler.UpdateProfile(recorder, requestAsUserWithBody(t, http.MethodPatch,
			"/api/user/profile", strings.NewReader(`{"phone":"(123) 456-7890"}`), signedIn))

		require.Equal(t, http.StatusOK, recorder.Code)

		var body userResponse
		require.NoError(t, json.NewDecoder(recorder.Body).Decode(&body))
		assert.Equal(t, userResponse{Email: "rider@example.com", Phone: "+11234567890", EmailVerified: true}, body)
	})

	t.Run("rejects an invalid number", func(t *testing.T) {
		handler, _ := testAccountHandler(t)

		recorder := httptest.NewRecorder()
		handler.UpdateProfile(recorder, requestAsUserWithBody(t, http.MethodPatch,
			"/api/user/profile", strings.NewReader(`{"phone":"123"}`), signedIn))

		assert.Equal(t, http.StatusBadRequest, recorder.Code)
		assert.Equal(t, "MISSING_PHONE", errorCode(t, recorder))
	})

	t.Run("reports a store failure", func(t *testing.T) {
		handler, store := testAccountHandler(t)
		store.EXPECT().UpdateUserPhone(gomock.Any(), signedIn.UUID, "+11234567890").Return(errors.New("database error"))

		recorder := httptest.NewRecorder()
		handler.UpdateProfile(recorder, requestAsUserWithBody(t, http.MethodPatch,
			"/api/user/profile", strings.NewReader(`{"phone":"1234567890"}`), signedIn))

		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	})
}
