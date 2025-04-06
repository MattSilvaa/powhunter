package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	db "github.com/MattSilvaa/powhunter/internal/db/generated"
)

// Store handles all database interactions
type Store struct {
	db *sql.DB
	*db.Queries
}

// NewStore creates a new store with database connection
func NewStore(db *sql.DB) *Store {
	return &Store{
		db:      db,
		Queries: db.New(db),
	}
}

// ExecTx executes a function within a database transaction
func (s *Store) ExecTx(ctx context.Context, fn func(*db.Queries) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}

	q := db.New(tx)
	err = fn(q)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("error rolling back transaction: %v (original error: %w)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing transaction: %w", err)
	}

	return nil
}

// CreateUserWithAlerts creates a user and their resort alerts in a single transaction
func (s *Store) CreateUserWithAlerts(ctx context.Context, email, phone string,
	minSnowAmount, notificationDays int32, resortUUIDs []string) error {

	return s.ExecTx(ctx, func(q *db.Queries) error {
		// Create the user
		phoneParam := sql.NullString{
			String: phone,
			Valid:  phone != "",
		}

		user, err := q.CreateUser(ctx, db.CreateUserParams{
			Email: email,
			Phone: phoneParam,
		})
		if err != nil {
			return fmt.Errorf("error creating user: %w", err)
		}

		// Create alerts for each resort
		for _, resortUUID := range resortUUIDs {
			_, err = q.CreateUserAlert(ctx, db.CreateUserAlertParams{
				UserID:           user.ID,
				ResortUuid:       resortUUID,
				MinSnowAmount:    minSnowAmount,
				NotificationDays: notificationDays,
			})
			if err != nil {
				return fmt.Errorf("error creating alert for resort %s: %w", resortUUID, err)
			}
		}

		return nil
	})
}

// UpdateSnowForecasts updates snow forecasts for a resort
func (s *Store) UpdateSnowForecasts(ctx context.Context, resortUUID string,
	forecasts map[time.Time]int32) error {

	return s.ExecTx(ctx, func(q *db.Queries) error {
		for date, amount := range forecasts {
			_, err := q.CreateSnowForecast(ctx, db.CreateSnowForecastParams{
				ResortUuid:          resortUUID,
				ForecastDate:        date,
				PredictedSnowAmount: amount,
			})
			if err != nil {
				return fmt.Errorf("error updating forecast for %s on %s: %w",
					resortUUID, date.Format("2006-01-02"), err)
			}
		}
		return nil
	})
}

// FindAlertsToSend finds alerts that should be sent based on forecast data
func (s *Store) FindAlertsToSend(ctx context.Context, daysAhead int) ([]AlertToSend, error) {
	// This will be a complex query in the future
	// For now, we'll return a placeholder
	alerts := []AlertToSend{}
	return alerts, nil
}

// AlertToSend represents a snow alert that should be sent to a user
type AlertToSend struct {
	UserEmail    string
	UserPhone    string
	ResortName   string
	SnowAmount   int32
	ForecastDate time.Time
}
