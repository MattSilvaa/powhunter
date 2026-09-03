package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/MattSilvaa/powhunter/internal/db"
	dbgen "github.com/MattSilvaa/powhunter/internal/db/generated"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// requireDBEnv names the environment variable that makes a missing test database
// a hard failure instead of a skip. CI must set it: an integration suite that
// silently skips reports green while covering nothing.
const requireDBEnv = "POWHUNTER_TEST_DB"

func unavailable(t *testing.T, format string, args ...any) {
	t.Helper()

	if os.Getenv(requireDBEnv) != "" {
		t.Fatalf("%s is set but the test database is unavailable: "+format, append([]any{requireDBEnv}, args...)...)
	}

	t.Skipf("Skipping integration test: "+format, args...)
}

// RunMigrations applies every migration to the given database.
func RunMigrations(t *testing.T, database *sql.DB) {
	t.Helper()

	if err := db.Migrate(database); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}
}

// TestDBConfig holds configuration for test database
type TestDBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// GetTestDBConfig returns test database configuration from environment
func GetTestDBConfig() TestDBConfig {
	return TestDBConfig{
		Host:     getEnvOrDefault("TEST_DB_HOST", "localhost"),
		Port:     getEnvOrDefault("TEST_DB_PORT", "5432"),
		User:     getEnvOrDefault("TEST_DB_USER", "postgres"),
		Password: getEnvOrDefault("TEST_DB_PASSWORD", "postgres"),
		DBName:   getEnvOrDefault("TEST_DB_NAME", "powhunter_test"),
		SSLMode:  getEnvOrDefault("TEST_DB_SSLMODE", "disable"),
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// SetupTestDB creates a test database connection and runs migrations
func SetupTestDB(t *testing.T) (*sql.DB, *db.Store, func()) {
	t.Helper()

	config := GetTestDBConfig()
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.DBName, config.SSLMode)

	testDB, err := sql.Open("postgres", connStr)
	if err != nil {
		unavailable(t, "cannot connect to test database: %v", err)
		return nil, nil, func() {}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := testDB.PingContext(ctx); err != nil {
		testDB.Close()
		unavailable(t, "test database not available: %v", err)
		return nil, nil, func() {}
	}

	// Bring the schema up to date before touching any table. The doc comment has
	// always claimed this happened; until now it did not.
	RunMigrations(t, testDB)

	// Clean up existing data
	cleanupDB(t, testDB)

	store := db.NewStore(testDB)

	cleanup := func() {
		cleanupDB(t, testDB)
		testDB.Close()
	}

	return testDB, store, cleanup
}

// cleanupDB removes all data from test tables
func cleanupDB(t *testing.T, db *sql.DB) {
	t.Helper()

	ctx := context.Background()
	queries := []string{
		"DELETE FROM alert_history",
		"DELETE FROM user_alerts",
		"DELETE FROM users",
		"DELETE FROM resorts",
	}

	for _, query := range queries {
		if _, err := db.ExecContext(ctx, query); err != nil {
			t.Logf("Warning: failed to clean table: %v", err)
		}
	}
}

// SeedTestResort creates a test resort in the database
func SeedTestResort(t *testing.T, queries *dbgen.Queries, name string, lat, lon float64) dbgen.Resort {
	t.Helper()

	resort, err := queries.InsertResort(context.Background(), dbgen.InsertResortParams{
		Uuid:        uuid.New(),
		Name:        name,
		UrlHost:     sql.NullString{String: "example.com", Valid: true},
		UrlPathname: sql.NullString{String: "/snow", Valid: true},
		Latitude:    sql.NullFloat64{Float64: lat, Valid: true},
		Longitude:   sql.NullFloat64{Float64: lon, Valid: true},
	})
	if err != nil {
		t.Fatalf("Failed to seed test resort: %v", err)
	}

	return resort
}

// SeedTestUser creates a test user in the database
func SeedTestUser(t *testing.T, queries *dbgen.Queries, email, phone string) dbgen.User {
	t.Helper()

	user, err := queries.CreateUser(context.Background(), dbgen.CreateUserParams{
		Email: email,
		Phone: sql.NullString{String: phone, Valid: phone != ""},
	})
	if err != nil {
		t.Fatalf("Failed to seed test user: %v", err)
	}

	return user
}

// SeedTestAlert creates a test alert in the database
func SeedTestAlert(t *testing.T, queries *dbgen.Queries, userUUID, resortUUID uuid.UUID, minSnow float64, days int32) dbgen.UserAlert {
	t.Helper()

	alert, err := queries.CreateUserAlert(context.Background(), dbgen.CreateUserAlertParams{
		UserUuid:         uuid.NullUUID{UUID: userUUID, Valid: true},
		ResortUuid:       uuid.NullUUID{UUID: resortUUID, Valid: true},
		MinSnowAmount:    minSnow,
		NotificationDays: days,
	})
	if err != nil {
		t.Fatalf("Failed to seed test alert: %v", err)
	}

	return alert
}
