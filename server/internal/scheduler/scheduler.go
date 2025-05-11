package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/MattSilvaa/powhunter/internal/db"
	"github.com/MattSilvaa/powhunter/internal/notify"
	"github.com/MattSilvaa/powhunter/internal/weather"
)

//go:generate mockgen -destination=mocks/mock_scheduler.go -package=mocks github.com/MattSilvaa/powhunter/internal/scheduler ForecastSchedulerService

// ForecastSchedulerService defines the interface for the forecast scheduler
type ForecastSchedulerService interface {
	// Start begins the scheduler
	Start()

	// Stop shuts down the scheduler
	Stop()

	// CheckForecasts checks for new snow forecasts and sends alerts
	CheckForecasts() error
}

// ForecastScheduler handles regular checking of weather forecasts
// and sending notifications for new snow
type ForecastScheduler struct {
	store         db.StoreService
	weatherClient weather.WeatherService
	twilioClient  notify.NotificationService
	interval      time.Duration
	quit          chan struct{}
	wg            sync.WaitGroup
}

// NewForecastScheduler creates a new forecast scheduler
func NewForecastScheduler(
	store db.StoreService,
	weatherClient weather.WeatherService,
	twilioClient notify.NotificationService,
	interval time.Duration,
) *ForecastScheduler {
	return &ForecastScheduler{
		store:         store,
		weatherClient: weatherClient,
		twilioClient:  twilioClient,
		interval:      interval,
		quit:          make(chan struct{}),
	}
}

// Start begins the scheduler
func (s *ForecastScheduler) Start() {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		if err := s.CheckForecasts(); err != nil {
			log.Printf("Error checking forecasts: %v", err)
		}

		for {
			select {
			case <-ticker.C:
				if err := s.CheckForecasts(); err != nil {
					log.Printf("Error checking forecasts: %v", err)
				}
			case <-s.quit:
				return
			}
		}
	}()
}

func (s *ForecastScheduler) Stop() {
	close(s.quit)
	s.wg.Wait()
}

func (s *ForecastScheduler) CheckForecasts() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	resorts, err := s.store.ListAllResorts(ctx)
	if err != nil {
		return fmt.Errorf("error listing resorts: %w", err)
	}

	for _, resort := range resorts {
		if !resort.Latitude.Valid || !resort.Longitude.Valid {
			log.Printf("Skipping resort %s: missing coordinates", resort.Name)
			continue
		}

		predictions, err := s.weatherClient.GetSnowForecast(ctx, resort.Latitude.Float64, resort.Longitude.Float64)
		if err != nil {
			log.Printf("Error getting forecast for %s: %v", resort.Name, err)
			continue
		}

		if len(predictions) == 0 {
			continue
		}

		for _, pred := range predictions {
			snowAmount := int32(pred.SnowAmount + 0.5)
			if snowAmount <= 0 {
				continue
			}

			/*
				 	We need to do the following:
						1. Check all users that have an alert for this resort
						2. Check whether we've already sent them an alert that it's going to snow on that day?
						3. If we have not sent them an alert, send an alert and add it to alert history for that date
			*/

			alerts, err := s.findMatchingAlerts(ctx, resort.Uuid.String(), pred.Date, snowAmount)
			if err != nil {
				log.Printf("Error finding matching alerts for %s: %v", resort.Name, err)
				continue
			}

			for _, alert := range alerts {
				if err := s.sendAlertNotification(ctx, alert, pred.SnowAmount); err != nil {
					log.Printf("Error sending notification: %v", err)
					continue
				}
			}
		}
	}

	return nil
}

// findMatchingAlerts finds alerts that match the given criteria
func (s *ForecastScheduler) findMatchingAlerts(
	ctx context.Context,
	resortUUID string,
	forecastDate time.Time,
	snowAmount int32,
) ([]db.AlertToSend, error) {
	nowDate := time.Now().UTC().Truncate(24 * time.Hour)
	forecastDateUTC := forecastDate.UTC().Truncate(24 * time.Hour)

	daysAhead := int32(forecastDateUTC.Sub(nowDate).Hours() / 24)
	if daysAhead < 0 {
		daysAhead = 0
	}

	return s.store.GetAlertMatches(ctx, resortUUID, forecastDate, snowAmount, daysAhead)
}

// sendAlertNotification sends a notification for a matching alert
func (s *ForecastScheduler) sendAlertNotification(
	ctx context.Context,
	alert db.AlertToSend,
	snowAmount float64,
) error {
	if alert.UserPhone == "" {
		return fmt.Errorf("no phone number for user ID %d", alert.UserUuid)
	}

	message := notify.FormatSnowAlertMessage(alert.ResortName, snowAmount, alert.ForecastDate)
	if err := s.twilioClient.SendSMS(alert.UserPhone, message); err != nil {
		log.Printf("Error sending SMS to %s: %v", alert.UserPhone, err)
		return fmt.Errorf("error sending SMS: %w", err)
	} else {
		log.Printf("Sent SMS alert to %s for %s", alert.UserPhone, alert.ResortName)
	}

	// We want to keep track of each alert we've sent so we need to persist it
	if err := s.store.RecordAlertSent(ctx, alert); err != nil {
		return fmt.Errorf("error recording alert history: %w", err)
	}

	return nil
}
