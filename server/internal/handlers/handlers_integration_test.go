//go:build integration
// +build integration

package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/MattSilvaa/powhunter/internal/auth"
	"github.com/MattSilvaa/powhunter/internal/db"
	dbgen "github.com/MattSilvaa/powhunter/internal/db/generated"
	"github.com/MattSilvaa/powhunter/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAlertHandlerIntegration_CreateAlert(t *testing.T) {
	testDB, store, cleanup := testutil.SetupTestDB(t)
	defer cleanup()

	queries := dbgen.New(testDB)

	// Seed resorts
	resort1 := testutil.SeedTestResort(t, queries, "Vail", 39.6403, -106.3742)
	resort2 := testutil.SeedTestResort(t, queries, "Breckenridge", 39.4817, -106.0384)

	handler, err := NewAlertHandler(store, auth.NewService(queries, false, false), &captureMailer{}, "https://powhunter.app")
	require.NoError(t, err)

	t.Run("Successfully create alert with real database", func(t *testing.T) {
		requestBody := CreateAlertRequest{
			Email:            "integration@test.com",
			Phone:            "+15552345678",
			NotificationDays: 3,
			MinSnowAmount:    8.0,
			ResortsUuids:     []string{resort1.Uuid.String(), resort2.Uuid.String()},
		}

		body, err := json.Marshal(requestBody)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/alert", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		handler.CreateAlert(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)

		var response map[string]string
		err = json.NewDecoder(rr.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, "success", response["status"])

		// Verify data was actually written to database
		ctx := context.Background()
		user, err := queries.GetUserByEmail(ctx, "integration@test.com")
		require.NoError(t, err)
		assert.Equal(t, "integration@test.com", user.Email)

		alerts, err := queries.ListActiveAlerts(ctx)
		require.NoError(t, err)
		assert.Len(t, alerts, 2)
	})

	// A lost or filtered confirmation email must not strand someone: signing up
	// again before confirming succeeds and sends a fresh link.
	t.Run("Repeat unconfirmed signup succeeds and sends a fresh link", func(t *testing.T) {
		mailer := &captureMailer{}
		retryHandler, handlerErr := NewAlertHandler(store, auth.NewService(queries, false, false), mailer,
			"https://powhunter.app")
		require.NoError(t, handlerErr)

		requestBody := CreateAlertRequest{
			Email:            "duplicate@test.com",
			Phone:            "+15559876543",
			NotificationDays: 2,
			MinSnowAmount:    5.0,
			ResortsUuids:     []string{resort1.Uuid.String()},
		}

		body, err := json.Marshal(requestBody)
		require.NoError(t, err)

		for range 2 {
			req := httptest.NewRequest(http.MethodPost, "/api/alert", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			retryHandler.CreateAlert(rr, req)
			assert.Equal(t, http.StatusCreated, rr.Code)
		}

		assert.Len(t, mailer.sent, 2)
	})

	// A malformed resort identifier is a client mistake. It previously reached
	// the database and surfaced as a 500.
	t.Run("Invalid resort UUID is rejected as a bad request", func(t *testing.T) {
		requestBody := CreateAlertRequest{
			Email:            "invalidresort@test.com",
			Phone:            "+15552221111",
			NotificationDays: 3,
			MinSnowAmount:    8.0,
			ResortsUuids:     []string{"not-a-valid-uuid"},
		}

		body, err := json.Marshal(requestBody)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/alert", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		handler.CreateAlert(rr, req)
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestResortHandlerIntegration_ListAllResorts(t *testing.T) {
	testDB, store, cleanup := testutil.SetupTestDB(t)
	defer cleanup()

	queries := dbgen.New(testDB)

	// Seed resorts
	testutil.SeedTestResort(t, queries, "Whistler Blackcomb", 50.1163, -122.9574)
	testutil.SeedTestResort(t, queries, "Vail", 39.6403, -106.3742)
	testutil.SeedTestResort(t, queries, "Park City", 40.6514, -111.5081)

	handler := &ResortHandler{store: store}

	t.Run("List all resorts returns real data", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/resorts", nil)
		rr := httptest.NewRecorder()

		handler.ListAllResorts(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var resorts []dbgen.Resort
		err := json.NewDecoder(rr.Body).Decode(&resorts)
		require.NoError(t, err)
		assert.Len(t, resorts, 3)

		names := make(map[string]bool)
		for _, r := range resorts {
			names[r.Name] = true
			assert.True(t, r.Latitude.Valid)
			assert.True(t, r.Longitude.Valid)
		}

		assert.True(t, names["Whistler Blackcomb"])
		assert.True(t, names["Vail"])
		assert.True(t, names["Park City"])
	})

	t.Run("Returns empty array when no resorts", func(t *testing.T) {
		// Clean up all resorts
		ctx := context.Background()
		_, err := testDB.ExecContext(ctx, "DELETE FROM resorts")
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/resorts", nil)
		rr := httptest.NewRecorder()

		handler.ListAllResorts(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var resorts []dbgen.Resort
		err = json.NewDecoder(rr.Body).Decode(&resorts)
		require.NoError(t, err)
		assert.Len(t, resorts, 0)
	})
}

func TestEndToEndFlow_CreateAlertAndVerifyInDatabase(t *testing.T) {
	testDB, store, cleanup := testutil.SetupTestDB(t)
	defer cleanup()

	queries := dbgen.New(testDB)

	// Seed resort
	resort := testutil.SeedTestResort(t, queries, "Test Mountain", 39.6403, -106.3742)

	// Create alert via API, with the real auth service behind the emailed link.
	authService := auth.NewService(queries, false, false)
	mailer := &captureMailer{}
	alertHandler, err := NewAlertHandler(store, authService, mailer, "https://powhunter.app")
	require.NoError(t, err)

	requestBody := CreateAlertRequest{
		Email:            "endtoend@test.com",
		Phone:            "+15557654321",
		NotificationDays: 5,
		MinSnowAmount:    10.0,
		ResortsUuids:     []string{resort.Uuid.String()},
	}

	body, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/alert", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	alertHandler.CreateAlert(rr, req)
	require.Equal(t, http.StatusCreated, rr.Code)

	// Verify data using store methods (simulating forecaster logic)
	ctx := context.Background()
	now := time.Now()
	forecastDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	matchAlerts := func() []db.AlertToSend {
		matches, matchErr := store.GetAlertMatches(
			ctx,
			resort.Uuid.String(),
			forecastDate,
			12.0, // More than minimum
			5,    // Within notification window
		)
		require.NoError(t, matchErr)

		return matches
	}

	// Nothing is texted until the address is confirmed.
	assert.Empty(t, matchAlerts())

	// Click the emailed link.
	require.Len(t, mailer.sent, 1)
	link := regexp.MustCompile(`token=([A-Za-z0-9_-]+)`).FindStringSubmatch(mailer.sent[0].HTML)
	require.Len(t, link, 2)
	_, _, err = authService.Redeem(ctx, link[1])
	require.NoError(t, err)

	matches := matchAlerts()
	require.Len(t, matches, 1)

	match := matches[0]
	assert.Equal(t, "endtoend@test.com", match.UserEmail)
	assert.Equal(t, "+15557654321", match.UserPhone)
	assert.Equal(t, "Test Mountain", match.ResortName)

	// Record alert sent
	err = store.RecordAlertSent(ctx, matches[0], db.SentAlert{SMSSegments: 1})
	require.NoError(t, err)

	// Verify alert history was recorded
	snowAmount, err := queries.GetLastAlertSnowAmount(ctx, dbgen.GetLastAlertSnowAmountParams{
		UserUuid:     uuid.NullUUID{UUID: match.UserUuid, Valid: true},
		ResortUuid:   uuid.NullUUID{UUID: resort.Uuid, Valid: true},
		ForecastDate: forecastDate,
	})
	require.NoError(t, err)
	assert.Equal(t, 12.0, snowAmount)
}
