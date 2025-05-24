package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/MattSilvaa/powhunter/internal/db"
	"github.com/MattSilvaa/powhunter/internal/notify"
	"github.com/MattSilvaa/powhunter/internal/scheduler"
	"github.com/MattSilvaa/powhunter/internal/weather"

	_ "github.com/lib/pq"
)

func main() {

	dbConn, err := db.New()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	store := db.NewStore(dbConn)
	weatherClient := weather.NewOpenMeteoClient()

	var twilioClient notify.NotificationService
	twilioAccountSID := os.Getenv("TWILIO_ACCOUNT_SID")
	twilioAuthToken := os.Getenv("TWILIO_AUTH_TOKEN")
	twilioFromNumber := os.Getenv("TWILIO_FROM_NUMBER")

	if twilioAccountSID == "" || twilioAuthToken == "" || twilioFromNumber == "" {
		log.Fatalf("Twilio credentials not found. Set TWILIO_ACCOUNT_SID, TWILIO_AUTH_TOKEN, and TWILIO_FROM_NUMBER environment variables.")
	}
	twilioClient = notify.NewTwilioClient(
		twilioFromNumber,
	)

	forecastScheduler := scheduler.NewForecastScheduler(
		store,
		weatherClient,
		twilioClient,
		12*time.Hour,
	)
	forecastScheduler.Start()
	log.Println("Forecast scheduler started")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	resorts, err := store.ListAllResorts(ctx)
	if err != nil {
		log.Fatalf("Failed to list resorts: %v", err)
	}

	for _, resort := range resorts {
		if !resort.Latitude.Valid || !resort.Longitude.Valid {
			log.Printf("Skipping resort %s: missing coordinates", resort.Name)
			continue
		}

		log.Printf("Checking forecast for %s (%.4f, %.4f)", resort.Name, resort.Latitude.Float64, resort.Longitude.Float64)

		predictions, err := weatherClient.GetSnowForecast(ctx, resort.Latitude.Float64, resort.Longitude.Float64)
		if err != nil {
			log.Printf("Error getting forecast for %s: %v", resort.Name, err)
			continue
		}

		if len(predictions) == 0 {
			log.Printf("No snow predicted for %s", resort.Name)
			continue
		}

		log.Printf("Found %d snow predictions for %s:", len(predictions), resort.Name)
		for _, pred := range predictions {
			log.Printf("  %s: %.1f inches", pred.Date.Format("2006-01-02"), pred.SnowAmount)

			daysAhead := int32(pred.Date.Sub(time.Now().Truncate(24*time.Hour)).Hours() / 24)
			if daysAhead < 0 {
				daysAhead = 0
			}

			snowAmount := int32(pred.SnowAmount + 0.5) // Round up for best prediction
			alerts, err := store.GetAlertMatches(ctx, resort.Uuid.String(), pred.Date, snowAmount, daysAhead)
			if err != nil {
				log.Printf("Error finding matching alerts: %v", err)
				continue
			}

			log.Printf("Found %d matching alerts", len(alerts))
			for _, alert := range alerts {
				log.Printf("Alert for %s (Phone: %s)", alert.ResortName, alert.UserPhone)

				// Send notification
				if alert.UserPhone != "" {
					message := notify.FormatSnowAlertMessage(alert.ResortName, pred.SnowAmount, alert.ForecastDate)
					if err := twilioClient.SendSMS(alert.UserPhone, message); err != nil {
						log.Printf("Error sending SMS to %s: %v", alert.UserPhone, err)
					} else {
						log.Printf("Sent SMS alert to %s for %s", alert.UserPhone, alert.ResortName)
					}
				}

				// Record that we sent the alert
				if err := store.RecordAlertSent(ctx, alert); err != nil {
					log.Printf("Error recording alert history: %v", err)
				}
			}
		}
	}

	log.Println("Forecast check complete")
}
