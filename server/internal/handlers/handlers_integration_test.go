// +build integration

package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	dbgen "github.com/MattSilvaa/powhunter/internal/db/generated"
	"github.com/MattSilvaa/powhunter/internal/testutil"
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

	handler := &AlertHandler{store: store}

	t.Run("Successfully create alert with real database", func(t *testing.T) {
		requestBody := CreateAlertRequest{
			Email:            "integration@test.com",
			Phone:            "+15551234567",
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

	t.Run("Duplicate email returns conflict", func(t *testing.T) {
		// First request
		requestBody := CreateAlertRequest{
			Email:            "duplicate@test.com",
			Phone:            "+15559876543",
			NotificationDays: 2,
			MinSnowAmount:    5.0,
			ResortsUuids:     []string{resort1.Uuid.String()},
		}

		body, err := json.Marshal(requestBody)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/alert", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		handler.CreateAlert(rr, req)
		assert.Equal(t, http.StatusCreated, rr.Code)

		// Duplicate request
		req2 := httptest.NewRequest(http.MethodPut, "/api/alert", bytes.NewReader(body))
		req2.Header.Set("Content-Type", "application/json")
		rr2 := httptest.NewRecorder()

		handler.CreateAlert(rr2, req2)
		assert.Equal(t, http.StatusConflict, rr2.Code)

		var errorResponse ErrorResponse
		err = json.NewDecoder(rr2.Body).Decode(&errorResponse)
		require.NoError(t, err)
		assert.Equal(t, "DUPLICATE_EMAIL", errorResponse.Error)
	})

	t.Run("Invalid resort UUID returns error", func(t *testing.T) {
		requestBody := CreateAlertRequest{
			Email:            "invalidresort@test.com",
			Phone:            "+15551111111",
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
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
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

	// Create alert via API
	alertHandler := &AlertHandler{store: store}

	requestBody := CreateAlertRequest{
		Email:            "endtoend@test.com",
		Phone:            "+15557654321",
		NotificationDays: 5,
		MinSnowAmount:    10.0,
		ResortsUuids:     []string{resort.Uuid.String()},
	}

	body, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPut, "/api/alert", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	alertHandler.CreateAlert(rr, req)
	require.Equal(t, http.StatusCreated, rr.Code)

	// Verify data using store methods (simulating forecaster logic)
	ctx := context.Background()
	now := time.Now()
	forecastDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	matches, err := store.GetAlertMatches(
		ctx,
		resort.Uuid.String(),
		forecastDate,
		12.0, // More than minimum
		5,    // Within notification window
	)
	require.NoError(t, err)
	require.Len(t, matches, 1)

	match := matches[0]
	assert.Equal(t, "endtoend@test.com", match.UserEmail)
	assert.Equal(t, "+15557654321", match.UserPhone)
	assert.Equal(t, "Test Mountain", match.ResortName)

	// Record alert sent
	err = store.RecordAlertSent(ctx, matches[0])
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
