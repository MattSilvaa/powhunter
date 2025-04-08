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
	store          db.StoreService
	weatherClient  weather.WeatherService
	twilioClient   notify.NotificationService
	interval       time.Duration
	quit           chan struct{}
	wg             sync.WaitGroup
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

		// Run once at startup
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

// Stop shuts down the scheduler
func (s *ForecastScheduler) Stop() {
	close(s.quit)
	s.wg.Wait()
}

// CheckForecasts checks for new snow forecasts and sends alerts
func (s *ForecastScheduler) CheckForecasts() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Get all resorts
	resorts, err := s.store.ListAllResorts(ctx)
	if err != nil {
		return fmt.Errorf("error listing resorts: %w", err)
	}

	// Process each resort
	for _, resort := range resorts {
		// Skip if missing lat/lon
		if !resort.Latitude.Valid || !resort.Longitude.Valid {
			log.Printf("Skipping resort %s: missing coordinates", resort.Name)
			continue
		}

		// Get snow forecast
		predictions, err := s.weatherClient.GetSnowForecast(ctx, resort.Latitude.Float64, resort.Longitude.Float64)
		if err != nil {
			log.Printf("Error getting forecast for %s: %v", resort.Name, err)
			continue
		}

		// Skip if no snow predicted
		if len(predictions) == 0 {
			continue
		}

		// Process each prediction
		for _, pred := range predictions {
			// Convert to int32 (round up for most favorable prediction)
			snowAmount := int32(pred.SnowAmount + 0.5)
			if snowAmount <= 0 {
				continue
			}

			// Find matching alerts
			alerts, err := s.findMatchingAlerts(ctx, resort.Uuid.String(), pred.Date, snowAmount)
			if err != nil {
				log.Printf("Error finding matching alerts for %s: %v", resort.Name, err)
				continue
			}

			// Send notifications for each alert
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
	// Calculate days ahead
	daysAhead := int32(forecastDate.Sub(time.Now().Truncate(24*time.Hour)).Hours() / 24)
	if daysAhead < 0 {
		daysAhead = 0
	}

	// Find matching alerts
	return s.store.GetAlertMatches(ctx, resortUUID, forecastDate, snowAmount, daysAhead)
}

// sendAlertNotification sends a notification for a matching alert
func (s *ForecastScheduler) sendAlertNotification(
	ctx context.Context,
	alert db.AlertToSend,
	snowAmount float64,
) error {
	// Skip if no contact method
	if alert.UserPhone == "" && alert.UserEmail == "" {
		return fmt.Errorf("no contact method for user ID %d", alert.UserID)
	}

	// Send SMS if phone available
	if alert.UserPhone != "" {
		message := notify.FormatSnowAlertMessage(alert.ResortName, snowAmount, alert.ForecastDate)
		if err := s.twilioClient.SendSMS(alert.UserPhone, message); err != nil {
			log.Printf("Error sending SMS to %s: %v", alert.UserPhone, err)
		} else {
			log.Printf("Sent SMS alert to %s for %s", alert.UserPhone, alert.ResortName)
		}
	}

	// Record that we sent the alert
	if err := s.store.RecordAlertSent(ctx, alert); err != nil {
		return fmt.Errorf("error recording alert history: %w", err)
	}

	return nil
}