# Forecast Scheduler

This package implements a scheduler that periodically checks for snow forecasts and notifies users when their alert criteria are met.

## How It Works

The forecast scheduler:

1. Runs at a configurable interval (default: every 12 hours)
2. Fetches all resorts with their coordinates
3. For each resort, gets the snow forecast from Weather.gov
4. For each snow prediction:
   - Finds users who have alerts set up for that resort
   - Checks if the predicted snow amount meets or exceeds the user's minimum
   - Verifies the forecast date is within the user's notification days window
   - Ensures the same alert hasn't been sent already (using alert_history)
5. Sends notifications to matching users via SMS (using Twilio)
6. Records sent alerts in the database to prevent duplicates

## Configuration

The scheduler accepts the following parameters:

- `store` - Database store for querying alerts and recording history
- `weatherClient` - Client for fetching weather forecasts
- `twilioClient` - Client for sending SMS notifications
- `interval` - How often to check for new forecasts (e.g., 12 hours)

## Usage Example

```go
// Initialize dependencies
store := db.NewStore(dbConn)
weatherClient := weather.NewWeatherGovClient()
twilioClient := notify.NewTwilioClient(accountSID, authToken, fromNumber)

// Create and start the scheduler
scheduler := scheduler.NewForecastScheduler(
    store,
    weatherClient,
    twilioClient,
    12*time.Hour,
)
scheduler.Start()

// Gracefully stop the scheduler when shutting down
scheduler.Stop()
```

## Concurrency and Safety

The scheduler:

- Uses a worker goroutine to check forecasts periodically
- Implements safe shutdown with a wait group
- Handles context cancellation for graceful termination
- Uses proper error handling to prevent crashes

## Logging

The scheduler logs important events:

- Scheduler start and stop
- Forecast check errors
- Notification sending status
- Alert recording confirmations

This provides transparency into the scheduler's operation and helps with debugging.