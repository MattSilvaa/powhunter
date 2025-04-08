package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	"github.com/MattSilvaa/powhunter/internal/db"
	"github.com/MattSilvaa/powhunter/internal/notify"
	"github.com/MattSilvaa/powhunter/internal/weather"

	_ "github.com/lib/pq"
)

func main() {
	// Parse command-line flags
	sendSMS := flag.Bool("send-sms", false, "Actually send SMS notifications")
	flag.Parse()

	// Connect to the database
	dbConn, err := db.New()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	store := db.NewStore(dbConn)

	// Create the weather client
	weatherClient := weather.NewWeatherGovClient()

	// Create the Twilio client if credentials are available and SMS sending is enabled
	var twilioClient notify.NotificationService
	if *sendSMS {
		twilioAccountSID := os.Getenv("TWILIO_ACCOUNT_SID")
		twilioAuthToken := os.Getenv("TWILIO_AUTH_TOKEN")
		twilioFromNumber := os.Getenv("TWILIO_FROM_NUMBER")

		if twilioAccountSID == "" || twilioAuthToken == "" || twilioFromNumber == "" {
			log.Fatalf("Twilio credentials not found. Set TWILIO_ACCOUNT_SID, TWILIO_AUTH_TOKEN, and TWILIO_FROM_NUMBER environment variables.")
		}

		twilioClient = notify.NewTwilioClient(
			twilioAccountSID,
			twilioAuthToken,
			twilioFromNumber,
		)
		log.Println("Twilio client initialized")
	} else {
		// Create a dummy notification service that just logs messages
		twilioClient = &DummyNotificationService{}
		log.Println("Using dummy notification service (SMS will not be sent)")
	}

	// Get all resorts
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	resorts, err := store.ListAllResorts(ctx)
	if err != nil {
		log.Fatalf("Failed to list resorts: %v", err)
	}

	log.Printf("Found %d resorts", len(resorts))

	// Process each resort
	for _, resort := range resorts {
		// Skip if missing lat/lon
		if !resort.Latitude.Valid || !resort.Longitude.Valid {
			log.Printf("Skipping resort %s: missing coordinates", resort.Name)
			continue
		}

		log.Printf("Checking forecast for %s (%.4f, %.4f)", resort.Name, resort.Latitude.Float64, resort.Longitude.Float64)

		// Get snow forecast
		predictions, err := weatherClient.GetSnowForecast(ctx, resort.Latitude.Float64, resort.Longitude.Float64)
		if err != nil {
			log.Printf("Error getting forecast for %s: %v", resort.Name, err)
			continue
		}

		// Report predictions
		if len(predictions) == 0 {
			log.Printf("No snow predicted for %s", resort.Name)
			continue
		}

		log.Printf("Found %d snow predictions for %s:", len(predictions), resort.Name)
		for _, pred := range predictions {
			log.Printf("  %s: %.1f inches", pred.Date.Format("2006-01-02"), pred.SnowAmount)

			// Compute days ahead for this prediction
			daysAhead := int32(pred.Date.Sub(time.Now().Truncate(24*time.Hour)).Hours() / 24)
			if daysAhead < 0 {
				daysAhead = 0
			}

			// Find matching alerts
			snowAmount := int32(pred.SnowAmount + 0.5) // Round up for best prediction
			alerts, err := store.GetAlertMatches(ctx, resort.Uuid.String(), pred.Date, snowAmount, daysAhead)
			if err != nil {
				log.Printf("Error finding matching alerts: %v", err)
				continue
			}

			// Report matching alerts
			log.Printf("  Found %d matching alerts", len(alerts))
			for _, alert := range alerts {
				log.Printf("  Alert for %s (Email: %s, Phone: %s)", alert.ResortName, alert.UserEmail, alert.UserPhone)

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

// DummyNotificationService just logs messages instead of sending them
type DummyNotificationService struct{}

func (d *DummyNotificationService) SendSMS(to, message string) error {
	log.Printf("[DUMMY SMS] To: %s, Message: %s", to, message)
	return nil
}