package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	dbgen "github.com/MattSilvaa/powhunter/internal/db/generated"
)

type Store struct {
	db      *sql.DB
	queries *dbgen.Queries
}

func NewStore(db *sql.DB) *Store {
	return &Store{
		db:      db,
		queries: dbgen.New(db),
	}
}

func (s *Store) ExecTx(ctx context.Context, fn func(*dbgen.Queries) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}

	q := dbgen.New(tx)
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

func (s *Store) ListAllResorts(ctx context.Context) ([]dbgen.Resort, error) {
	resorts, err := s.queries.ListResorts(ctx)
	if err != nil {
		return []dbgen.Resort{}, fmt.Errorf("error getting all alerts: %w", err)
	}

	return resorts, nil
}

func (s *Store) CreateUserWithAlerts(ctx context.Context, email, phone string,
	minSnowAmount, notificationDays int32, resortUUIDs []string) error {

	return s.ExecTx(ctx, func(q *dbgen.Queries) error {
		phoneParam := sql.NullString{
			String: phone,
			Valid:  phone != "",
		}

		user, err := q.CreateUser(ctx, dbgen.CreateUserParams{
			Email: email,
			Phone: phoneParam,
		})

		if err != nil {
			return fmt.Errorf("error creating user: %w", err)
		}

		for _, resortUUID := range resortUUIDs {
			var ruuid uuid.NullUUID
			if resortUUID != "" {
				parsedUUID, err := uuid.Parse(resortUUID)
				if err != nil {
					return fmt.Errorf("error parsing resort UUID %s: %w", resortUUID, err)
				}
				ruuid = uuid.NullUUID{UUID: parsedUUID, Valid: true}
			} else {
				ruuid = uuid.NullUUID{Valid: false}
			}

			_, err = q.CreateUserAlert(ctx, dbgen.CreateUserAlertParams{
				UserID: sql.NullInt32{
					Int32: user.ID,
					Valid: true,
				},
				ResortUuid:       ruuid,
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

func (s *Store) UpdateSnowForecasts(ctx context.Context, resortUUID string,
	forecasts map[time.Time]int32) error {
	var ruuid uuid.NullUUID
	if resortUUID != "" {
		parsedUUID, err := uuid.Parse(resortUUID)
		if err != nil {
			return fmt.Errorf("error parsing resort UUID %s: %w", resortUUID, err)
		}
		ruuid = uuid.NullUUID{UUID: parsedUUID, Valid: true}
	} else {
		ruuid = uuid.NullUUID{Valid: false}
	}

	return s.ExecTx(ctx, func(q *dbgen.Queries) error {
		for date, amount := range forecasts {
			_, err := q.CreateSnowForecast(ctx, dbgen.CreateSnowForecastParams{
				ResortUuid:          ruuid,
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
	alerts := []AlertToSend{}
	return alerts, nil
}

type AlertToSend struct {
	UserEmail    string
	UserPhone    string
	ResortName   string
	SnowAmount   int32
	ForecastDate time.Time
}
