//go:build integration
// +build integration

package db_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/MattSilvaa/powhunter/internal/db"
	dbgen "github.com/MattSilvaa/powhunter/internal/db/generated"
	"github.com/MattSilvaa/powhunter/internal/testutil"
	"github.com/google/uuid"
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

	t.Run("Existing email reuses the user and rejects a duplicate alert", func(t *testing.T) {
		ctx := context.Background()

		// The same email must not create a second user; it reuses the existing one
		// and then fails on the (user_uuid, resort_uuid) uniqueness constraint,
		// which the handler maps to a 409 DUPLICATE_ALERT.
		err := store.CreateUserWithAlerts(
			ctx,
			"test@example.com", // Same email
			"+15559876543",
			10.0,
			5,
			[]string{resort1.Uuid.String()},
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user_alerts_user_uuid_resort_uuid_key")
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
		firstMatch := db.AlertToSend{
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
		firstMatch := db.AlertToSend{
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

		alertToSend := db.AlertToSend{
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

// Two people signing up at the same moment with the same email previously both
// saw no existing row, both inserted, and one failed on the unique constraint.
func TestStoreIntegration_ConcurrentSignupsShareOneUser(t *testing.T) {
	testDB, store, cleanup := testutil.SetupTestDB(t)
	defer cleanup()

	queries := dbgen.New(testDB)

	resortA := testutil.SeedTestResort(t, queries, "Alta", 40.5883, -111.6372)
	resortB := testutil.SeedTestResort(t, queries, "Snowbird", 40.5830, -111.6556)

	const email = "race@example.com"

	var wg sync.WaitGroup

	errs := make([]error, 2)
	resorts := []string{resortA.Uuid.String(), resortB.Uuid.String()}

	for i, resort := range resorts {
		wg.Add(1)

		go func(index int, resortUUID string) {
			defer wg.Done()

			errs[index] = store.CreateUserWithAlerts(
				context.Background(), email, "+15551234567", 6.0, 3, []string{resortUUID},
			)
		}(i, resort)
	}

	wg.Wait()

	for _, err := range errs {
		require.NoError(t, err, "concurrent signups for the same email must both succeed")
	}

	var userCount int

	require.NoError(t,
		testDB.QueryRow("SELECT count(*) FROM users WHERE email = $1", email).Scan(&userCount))
	assert.Equal(t, 1, userCount, "the same email must map to exactly one user")

	alerts, err := store.GetUserAlertsByEmail(context.Background(), email)
	require.NoError(t, err)
	assert.Len(t, alerts, 2)
}
