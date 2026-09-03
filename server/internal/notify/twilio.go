package notify

import (
	"errors"
	"fmt"
	"time"

	"github.com/MattSilvaa/powhunter/internal/db"
	"github.com/twilio/twilio-go"
	twilioAPI "github.com/twilio/twilio-go/rest/api/v2010"
)

//go:generate mockgen -destination=mocks/mock_notify.go -package=mocks github.com/MattSilvaa/powhunter/internal/notify NotificationService

// NotificationService defines the interface for notification services.
type NotificationService interface {
	// SendSMS sends an SMS message
	SendSMS(to, message string) error
}

// messageAPI is the slice of the Twilio REST client this package uses. Keeping
// it as an interface lets tests assert on the parameters actually sent.
type messageAPI interface {
	CreateMessage(params *twilioAPI.CreateMessageParams) (*twilioAPI.ApiV2010Message, error)
}

// TwilioClient handles SMS notifications via Twilio.
type TwilioClient struct {
	fromNumber string
	api        messageAPI
}

// NewTwilioClient creates a new Twilio client. Credentials are read from the
// TWILIO_ACCOUNT_SID and TWILIO_AUTH_TOKEN environment variables.
func NewTwilioClient(fromNumber string) *TwilioClient {
	return &TwilioClient{
		fromNumber: fromNumber,
		api:        twilio.NewRestClient().Api,
	}
}

// SendSMS sends an SMS message using Twilio.
func (t *TwilioClient) SendSMS(to, message string) error {
	if to == "" || message == "" {
		return errors.New("phone number and message are required")
	}

	params := &twilioAPI.CreateMessageParams{}
	params.SetTo(to)
	params.SetFrom(t.fromNumber)
	params.SetBody(message)

	if _, err := t.api.CreateMessage(params); err != nil {
		return fmt.Errorf("error sending SMS: %w", err)
	}

	return nil
}

// FormatSnowAlertMessage formats a snow alert SMS message.
func FormatSnowAlertMessage(alert db.AlertToSend) string {
	forecastDate := alert.ForecastDate
	resortName := alert.ResortName
	snowAmount := alert.SnowAmount

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	tomorrow := today.Add(24 * time.Hour)
	forecastDay := time.Date(forecastDate.Year(), forecastDate.Month(), forecastDate.Day(), 0, 0, 0, 0, forecastDate.Location())

	var timeStr string
	switch {
	case forecastDay.Equal(today):
		timeStr = "today"
	case forecastDay.Equal(tomorrow):
		timeStr = "tomorrow"
	default:
		timeStr = "on " + forecastDate.Format("Monday, Jan 2")
	}

	if alert.IsUpdate {
		return fmt.Sprintf("Powder Alert Update! %s is now expecting %.1f inches of snow %s - even more powder than before! Time to hit the slopes!",
			resortName, snowAmount, timeStr)
	}

	return fmt.Sprintf("Powder Alert! %s is expecting %.1f inches of snow %s. Time to hit the slopes!",
		resortName, snowAmount, timeStr)
}
