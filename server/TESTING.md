# Testing Guide

This document describes how to run tests in the powhunter server.

## Test Types

### Unit Tests
Unit tests use mocks and don't require external dependencies. They run quickly and are the default test mode.

```bash
# Run all unit tests
cd server && go test ./...

# Run unit tests for a specific package
cd server && go test ./internal/handlers
cd server && go test ./internal/notify
```

### Integration Tests
Integration tests require a PostgreSQL database and test the full stack with real database connections.

## Running Integration Tests

### Prerequisites

1. **PostgreSQL Database**: You need a running PostgreSQL instance for integration tests.

2. **Environment Variables**: Set the following environment variables:
   ```bash
   export TEST_DB_HOST=localhost
   export TEST_DB_PORT=5432
   export TEST_DB_USER=postgres
   export TEST_DB_PASSWORD=postgres
   export TEST_DB_NAME=powhunter_test
   export TEST_DB_SSLMODE=disable
   ```

   Default values are shown above. If these environment variables are not set, the tests will use these defaults.

3. **Database Schema**: Ensure the test database exists and has the correct schema:
   ```bash
   # Create test database (if it doesn't exist)
   createdb powhunter_test

   # Run migrations
   cd server
   goose -dir internal/db/migrations postgres "host=localhost user=postgres password=postgres dbname=powhunter_test sslmode=disable" up
   ```

### Running Integration Tests

```bash
# Run only integration tests
cd server && go test -tags=integration ./...

# Run integration tests for a specific package
cd server && go test -tags=integration ./internal/db
cd server && go test -tags=integration ./internal/handlers

# Run with verbose output
cd server && go test -tags=integration -v ./...
```

### Running All Tests (Unit + Integration)

```bash
# Run both unit and integration tests
cd server && go test -tags=integration ./...
```

Note: Unit tests don't use the `integration` build tag, so they will run alongside integration tests.

## Test Coverage

Integration tests cover:

1. **Database Store Operations** (`internal/db/store_integration_test.go`):
   - Creating users with alerts
   - Finding alert matches based on forecast criteria
   - Recording alert history
   - Duplicate email handling
   - Alert update logic when snow increases

2. **API Handlers** (`internal/handlers/handlers_integration_test.go`):
   - Creating alerts via HTTP API with real database
   - Listing resorts via HTTP API
   - End-to-end flow: API → Database → Alert matching
   - Error handling with real database constraints

3. **Unit Tests** (various `*_test.go` files):
   - Handler validation and error responses (with mocks)
   - Message formatting
   - Individual function behavior

## Test Database Cleanup

Integration tests automatically clean up the test database before and after each test. The cleanup process:
- Deletes all rows from `alert_history`, `user_alerts`, `users`, and `resorts` tables
- Preserves the schema and migrations

## Continuous Integration

If using CI/CD, ensure your pipeline:
1. Starts a PostgreSQL service
2. Creates the test database
3. Runs migrations
4. Sets environment variables
5. Runs tests with `-tags=integration`

Example GitHub Actions snippet:
```yaml
services:
  postgres:
    image: postgres:15
    env:
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: powhunter_test
    options: >-
      --health-cmd pg_isready
      --health-interval 10s
      --health-timeout 5s
      --health-retries 5

steps:
  - name: Run migrations
    run: |
      cd server
      goose -dir internal/db/migrations postgres "$DATABASE_URL" up
    env:
      DATABASE_URL: "host=localhost user=postgres password=postgres dbname=powhunter_test sslmode=disable"

  - name: Run tests
    run: cd server && go test -tags=integration ./...
    env:
      TEST_DB_HOST: localhost
      TEST_DB_USER: postgres
      TEST_DB_PASSWORD: postgres
      TEST_DB_NAME: powhunter_test
```

## Troubleshooting

### "Skipping integration test: cannot connect to test database"

This message appears when the test database is not available. Make sure:
1. PostgreSQL is running
2. Connection parameters are correct
3. The test database exists

### "table does not exist" errors

Run migrations on the test database:
```bash
goose -dir internal/db/migrations postgres "host=localhost user=postgres password=postgres dbname=powhunter_test sslmode=disable" up
```

### Tests fail with unique constraint violations

Integration tests should clean up automatically. If you see constraint violations:
1. Manually clean the test database: `psql powhunter_test -c "DELETE FROM alert_history; DELETE FROM user_alerts; DELETE FROM users; DELETE FROM resorts;"`
2. Or drop and recreate: `dropdb powhunter_test && createdb powhunter_test` then run migrations
