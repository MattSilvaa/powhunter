// +build integration

package db

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	dbgen "github.com/MattSilvaa/powhunter/internal/db/generated"
	"github.com/MattSilvaa/powhunter/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoreIntegration_CreateUserWithAlerts(t *testing.T) {
	testDB, store, cleanup := testutil.SetupTestDB(t)
	defer cleanup()

	queries := dbgen.New(testDB)

	// Seed resorts
	resort1 := testutil.SeedTestResort(t, queries, "Test Resort 1", 39.6403, -106.3742)
	resort2 := testutil.SeedTestResort(t, queries, "Test Resort 2", 39.4817, -106.0384)

	t.Run("Create user with multiple alerts", func(t *testing.T) {
		ctx := context.Background()

		err := store.CreateUserWithAlerts(
			ctx,
			"test@example.com",
			"+15551234567",
			8.0,
			3,
			[]string{resort1.Uuid.String(), resort2.Uuid.String()},
		)
		require.NoError(t, err)

		// Verify user was created
		user, err := queries.GetUserByEmail(ctx, "test@example.com")
		require.NoError(t, err)
		assert.Equal(t, "test@example.com", user.Email)
		assert.Equal(t, "+15551234567", user.Phone.String)

		// Verify alerts were created
		alerts, err := queries.ListActiveAlerts(ctx)
		require.NoError(t, err)
		assert.Len(t, alerts, 2)

		for _, alert := range alerts {
			assert.Equal(t, user.Uuid, alert.UserUuid.UUID)
			assert.Equal(t, 8.0, alert.MinSnowAmount)
			assert.Equal(t, int32(3), alert.NotificationDays)
		}
	})

	t.Run("Duplicate email returns error", func(t *testing.T) {
		ctx := context.Background()

		err := store.CreateUserWithAlerts(
			ctx,
			"test@example.com", // Same email
			"+15559876543",
			10.0,
			5,
			[]string{resort1.Uuid.String()},
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error creating user")
	})
}

func TestStoreIntegration_GetAlertMatches(t *testing.T) {
	testDB, store, cleanup := testutil.SetupTestDB(t)
	defer cleanup()

	queries := dbgen.New(testDB)

	// Seed data
	resort := testutil.SeedTestResort(t, queries, "Test Resort", 39.6403, -106.3742)
	user := testutil.SeedTestUser(t, queries, "alert@example.com", "+15551234567")
	testutil.SeedTestAlert(t, queries, user.Uuid, resort.Uuid, 5.0, 3)

	t.Run("First alert for resort matches", func(t *testing.T) {
		ctx := context.Background()
		forecastDate := time.Now().Add(24 * time.Hour).Truncate(24 * time.Hour)

		matches, err := store.GetAlertMatches(
			ctx,
			resort.Uuid.String(),
			forecastDate,
			8.0, // More than minimum
			1,   // 1 day ahead
		)
		require.NoError(t, err)
		require.Len(t, matches, 1)

		match := matches[0]
		assert.Equal(t, user.Uuid, match.UserUuid)
		assert.Equal(t, user.Email, match.UserEmail)
		assert.Equal(t, user.Phone.String, match.UserPhone)
		assert.Equal(t, resort.Name, match.ResortName)
		assert.Equal(t, 8.0, match.SnowAmount)
		assert.False(t, match.IsUpdate)
	})

	t.Run("No match when snow amount too low", func(t *testing.T) {
		ctx := context.Background()
		forecastDate := time.Now().Add(24 * time.Hour).Truncate(24 * time.Hour)

		matches, err := store.GetAlertMatches(
			ctx,
			resort.Uuid.String(),
			forecastDate,
			3.0, // Less than minimum
			1,
		)
		require.NoError(t, err)
		assert.Len(t, matches, 0)
	})

	t.Run("Update alert when snow increases by 3+ inches", func(t *testing.T) {
		ctx := context.Background()
		forecastDate := time.Now().Add(24 * time.Hour).Truncate(24 * time.Hour)

		// Record first alert
		firstMatch := AlertToSend{
			UserUuid:     user.Uuid,
			UserEmail:    user.Email,
			UserPhone:    user.Phone.String,
			ResortName:   resort.Name,
			ResortUUID:   resort.Uuid,
			SnowAmount:   8.0,
			ForecastDate: forecastDate,
			IsUpdate:     false,
		}
		err := store.RecordAlertSent(ctx, firstMatch)
		require.NoError(t, err)

		// Now check with increased snow amount
		matches, err := store.GetAlertMatches(
			ctx,
			resort.Uuid.String(),
			forecastDate,
			11.5, // 3.5 inches more than previous
			1,
		)
		require.NoError(t, err)
		require.Len(t, matches, 1)

		match := matches[0]
		assert.Equal(t, 11.5, match.SnowAmount)
		assert.True(t, match.IsUpdate)
	})

	t.Run("No update when snow increase is less than 3 inches", func(t *testing.T) {
		ctx := context.Background()
		forecastDate := time.Now().Add(48 * time.Hour).Truncate(24 * time.Hour)

		// Record first alert
		firstMatch := AlertToSend{
			UserUuid:     user.Uuid,
			UserEmail:    user.Email,
			UserPhone:    user.Phone.String,
			ResortName:   resort.Name,
			ResortUUID:   resort.Uuid,
			SnowAmount:   8.0,
			ForecastDate: forecastDate,
			IsUpdate:     false,
		}
		err := store.RecordAlertSent(ctx, firstMatch)
		require.NoError(t, err)

		// Check with small increase
		matches, err := store.GetAlertMatches(
			ctx,
			resort.Uuid.String(),
			forecastDate,
			10.0, // Only 2 inches more
			2,
		)
		require.NoError(t, err)
		assert.Len(t, matches, 0)
	})
}

func TestStoreIntegration_RecordAlertSent(t *testing.T) {
	testDB, store, cleanup := testutil.SetupTestDB(t)
	defer cleanup()

	queries := dbgen.New(testDB)

	// Seed data
	resort := testutil.SeedTestResort(t, queries, "Test Resort", 39.6403, -106.3742)
	user := testutil.SeedTestUser(t, queries, "record@example.com", "+15551234567")

	t.Run("Record alert history", func(t *testing.T) {
		ctx := context.Background()
		forecastDate := time.Now().Add(24 * time.Hour).Truncate(24 * time.Hour)

		alertToSend := AlertToSend{
			UserUuid:     user.Uuid,
			UserEmail:    user.Email,
			UserPhone:    user.Phone.String,
			ResortName:   resort.Name,
			ResortUUID:   resort.Uuid,
			SnowAmount:   10.0,
			ForecastDate: forecastDate,
			IsUpdate:     false,
		}

		err := store.RecordAlertSent(ctx, alertToSend)
		require.NoError(t, err)

		// Verify it was recorded
		snowAmount, err := queries.GetLastAlertSnowAmount(ctx, dbgen.GetLastAlertSnowAmountParams{
			UserUuid:     uuid.NullUUID{UUID: user.Uuid, Valid: true},
			ResortUuid:   uuid.NullUUID{UUID: resort.Uuid, Valid: true},
			ForecastDate: forecastDate,
		})
		require.NoError(t, err)
		assert.Equal(t, 10.0, snowAmount)
	})
}

func TestStoreIntegration_ListAllResorts(t *testing.T) {
	testDB, store, cleanup := testutil.SetupTestDB(t)
	defer cleanup()

	queries := dbgen.New(testDB)

	// Seed multiple resorts
	testutil.SeedTestResort(t, queries, "Resort A", 39.6403, -106.3742)
	testutil.SeedTestResort(t, queries, "Resort B", 39.4817, -106.0384)
	testutil.SeedTestResort(t, queries, "Resort C", 37.6487, -119.0650)

	t.Run("List all resorts", func(t *testing.T) {
		ctx := context.Background()

		resorts, err := store.ListAllResorts(ctx)
		require.NoError(t, err)
		assert.Len(t, resorts, 3)

		names := make([]string, len(resorts))
		for i, r := range resorts {
			names[i] = r.Name
		}
		assert.Contains(t, names, "Resort A")
		assert.Contains(t, names, "Resort B")
		assert.Contains(t, names, "Resort C")
	})
}
