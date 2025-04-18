package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	dbgen "github.com/MattSilvaa/powhunter/internal/db/generated"
)

//go:generate mockgen -destination=mocks/mock_store.go -package=mocks github.com/MattSilvaa/powhunter/internal/db StoreService

// StoreService defines the interface for database operations
type StoreService interface {
	// ListAllResorts returns all resorts
	ListAllResorts(ctx context.Context) ([]dbgen.Resort, error)

	// GetAlertMatches returns alerts matching forecast criteria
	GetAlertMatches(
		ctx context.Context,
		resortUUID string,
		forecastDate time.Time,
		predictedSnowAmount int32,
		daysAhead int32,
	) ([]AlertToSend, error)

	// RecordAlertSent records that an alert was sent
	RecordAlertSent(ctx context.Context, alert AlertToSend) error

	// CreateUserWithAlerts creates a new user with alert preferences
	CreateUserWithAlerts(
		ctx context.Context,
		email, phone string,
		minSnowAmount, notificationDays int32,
		resortUUIDs []string,
	) error
}

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
				UserUuid:         uuid.NullUUID{UUID: user.Uuid, Valid: true},
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

type AlertToSend struct {
	UserUuid     uuid.UUID
	UserEmail    string
	UserPhone    string
	ResortName   string
	ResortUUID   string
	SnowAmount   int32
	ForecastDate time.Time
	IsUpdate     bool
}

// GetAlertMatches finds alerts that match a specific resort, date, and snow amount
func (s *Store) GetAlertMatches(
	ctx context.Context,
	resortUUID string,
	forecastDate time.Time,
	predictedSnowAmount int32,
	daysAhead int32,
) ([]AlertToSend, error) {
	var alertsToSend []AlertToSend

	err := s.ExecTx(ctx, func(q *dbgen.Queries) error {
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

		alerts, err := q.GetResortAlerts(ctx, ruuid)
		if err != nil {
			return fmt.Errorf("error getting alert for resort %s: %w", resortUUID, err)
		}

		for _, alert := range alerts {
			lastAlertSnowAmount, err := q.GetLastAlertSnowAmount(ctx, dbgen.GetLastAlertSnowAmountParams{
				UserUuid:     alert.UserUuid,
				ResortUuid:   ruuid,
				ForecastDate: forecastDate,
			})
			if err != nil {
				// Alert if no alert has ever been sent to this user
				if errors.Is(err, sql.ErrNoRows) {
					userToAlert, err := q.GetUserByUUID(ctx, alert.UserUuid.UUID)
					if err != nil {
						return fmt.Errorf("error getting user %s: %w", alert.UserUuid.UUID.String(), err)
					}

					resortToAlertUserOn, err := q.GetResortByUUID(ctx, alert.ResortUuid.UUID)
					if err != nil {
						return fmt.Errorf("error getting resort %s: %w", alert.ResortUuid.UUID.String(), err)
					}

					alertsToSend = append(alertsToSend, AlertToSend{
						UserUuid:     userToAlert.Uuid,
						UserEmail:    userToAlert.Email,
						UserPhone:    userToAlert.Phone.String,
						ResortName:   resortToAlertUserOn.Name,
						ResortUUID:   resortToAlertUserOn.Uuid.String(),
						SnowAmount:   predictedSnowAmount,
						ForecastDate: forecastDate,
						IsUpdate:     false,
					})
				} else {
					return fmt.Errorf("error getting latest alert for resort %s: %w", resortUUID, err)
				}
			}
			// Alert when new snow ampunt is greater than or equal to 3 inches.
			if predictedSnowAmount-lastAlertSnowAmount >= 3 {
				userToAlert, err := q.GetUserByUUID(ctx, alert.UserUuid.UUID)
				if err != nil {
					return fmt.Errorf("error getting user %s: %w", alert.UserUuid.UUID.String(), err)
				}

				resortToAlertUserOn, err := q.GetResortByUUID(ctx, alert.ResortUuid.UUID)
				if err != nil {
					return fmt.Errorf("error getting resort %s: %w", alert.ResortUuid.UUID.String(), err)
				}

				alertsToSend = append(alertsToSend, AlertToSend{
					UserUuid:     userToAlert.Uuid,
					UserEmail:    userToAlert.Email,
					UserPhone:    userToAlert.Phone.String,
					ResortName:   resortToAlertUserOn.Name,
					ResortUUID:   resortToAlertUserOn.Uuid.String(),
					SnowAmount:   predictedSnowAmount,
					ForecastDate: forecastDate,
					IsUpdate:     true,
				})

			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return alertsToSend, nil
}

//
//// RecordAlertSent records that an alert was sent to avoid sending duplicates
//func (s *Store) RecordAlertSent(ctx context.Context, alert AlertToSend) error {
//	return s.ExecTx(ctx, func(q *dbgen.Queries) error {
//		var ruuid uuid.NullUUID
//		if alert.ResortUUID != "" {
//			parsedUUID, err := uuid.Parse(alert.ResortUUID)
//			if err != nil {
//				return fmt.Errorf("error parsing resort UUID %s: %w", alert.ResortUUID, err)
//			}
//			ruuid = uuid.NullUUID{UUID: parsedUUID, Valid: true}
//		} else {
//			ruuid = uuid.NullUUID{Valid: false}
//		}
//
//		err := q.InsertAlertHistory(ctx, dbgen.InsertAlertHistoryParams{
//			UserID: sql.NullInt32{
//				Valid: true,
//				Int32: alert.UserID,
//			},
//			ResortUuid:   ruuid,
//			ForecastDate: alert.ForecastDate,
//			SnowAmount:   alert.SnowAmount,
//		})
//
//		return err
//	})
//}
