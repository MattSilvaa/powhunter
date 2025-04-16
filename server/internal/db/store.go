package db

import (
	"context"
	"database/sql"
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

// GetAlertMatches finds alerts that match a specific resort, date, and snow amount
func (s *Store) GetAlertMatches(
	ctx context.Context,
	resortUUID string,
	forecastDate time.Time,
	predictedSnowAmount int32,
	daysAhead int32,
) ([]AlertToSend, error) {
	parsedUUID, err := uuid.Parse(resortUUID)
	if err != nil {
		return nil, fmt.Errorf("error parsing resort UUID %s: %w", resortUUID, err)
	}

	// Get all active alerts
	activeAlerts, err := s.queries.ListActiveAlerts(ctx)
	if err != nil {
		return nil, fmt.Errorf("error listing active alerts: %w", err)
	}

	var alerts []AlertToSend
	for _, alert := range activeAlerts {
		// Filter for matching resort
		if alert.ResortUuid.UUID != parsedUUID {
			continue
		}

		// Check if snow amount is sufficient
		if predictedSnowAmount < alert.MinSnowAmount {
			continue
		}

		// Check if days ahead is within notification preferences
		if daysAhead > alert.NotificationDays {
			continue
		}

		// Check if we've already sent an alert for this user/resort/date
		alreadySent, err := s.queries.CheckAlertSent(ctx, dbgen.CheckAlertSentParams{
			UserID: alert.UserID,
			ResortUuid: alert.ResortUuid,
			ForecastDate: forecastDate,
		})
		if err != nil {
			return nil, fmt.Errorf("error checking if alert was sent: %w", err)
		}
		if alreadySent {
			continue
		}

		// Create AlertToSend object
		alertToSend := AlertToSend{
			UserID:       alert.UserID.Int32,
			UserEmail:    alert.Email,
			ResortName:   alert.ResortName,
			ResortUUID:   alert.ResortUuid.UUID.String(),
			SnowAmount:   predictedSnowAmount,
			ForecastDate: forecastDate,
		}

		if alert.Phone.Valid {
			alertToSend.UserPhone = alert.Phone.String
		}

		alerts = append(alerts, alertToSend)
	}

	return alerts, nil
}

// FindAlertsToSend finds alerts that should be sent based on forecast data
// This is kept for backward compatibility
func (s *Store) FindAlertsToSend(ctx context.Context, daysAhead int) ([]AlertToSend, error) {
	alerts := []AlertToSend{}
	return alerts, nil
}

// RecordAlertSent records that an alert was sent to avoid sending duplicates
func (s *Store) RecordAlertSent(ctx context.Context, alert AlertToSend) error {
	query := `
	INSERT INTO alert_history (user_id, resort_uuid, forecast_date, snow_amount)
	VALUES ($1, $2, $3, $4)
	`

	_, err := s.db.ExecContext(ctx, query,
		alert.UserID,
		alert.ResortUUID,
		alert.ForecastDate,
		alert.SnowAmount,
	)

	if err != nil {
		return fmt.Errorf("error recording alert sent: %w", err)
	}

	return nil
}

type AlertToSend struct {
	UserID       int32
	UserEmail    string
	UserPhone    string
	ResortName   string
	ResortUUID   string
	SnowAmount   int32
	ForecastDate time.Time
}
