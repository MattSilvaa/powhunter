package notify

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/twilio/twilio-go"
	twilioAPI "github.com/twilio/twilio-go/rest/api/v2010"
)

//go:generate mockgen -destination=mocks/mock_notify.go -package=mocks github.com/MattSilvaa/powhunter/internal/notify NotificationService

// NotificationService defines the interface for notification services.
type NotificationService interface {
	// SendSMS sends an SMS message
	SendSMS(to, message string) error
}

// TwilioClient handles SMS notifications via Twilio.
type TwilioClient struct {
	fromNumber string
}

// NewTwilioClient creates a new Twilio client.
func NewTwilioClient(fromNumber string) *TwilioClient {
	return &TwilioClient{
		fromNumber: fromNumber,
	}
}

// SendSMS sends an SMS message using Twilio.
func (t *TwilioClient) SendSMS(to, message string) error {
	if to == "" || message == "" {
		return errors.New("phone number and message are required")
	}

	// This will look for `TWILIO_ACCOUNT_SID` and `TWILIO_AUTH_TOKEN` variables inside the current environment to initialize the constructor
	client := twilio.NewRestClient()
	params := &twilioAPI.CreateMessageParams{}
	params.SetTo("6195733405")
	params.SetFrom(t.fromNumber)
	params.SetBody(message)

	resp, err := client.Api.CreateMessage(params)
	if err != nil {
		return fmt.Errorf("error sending SMS: %w", err)
	}

	response, _ := json.Marshal(*resp)
	fmt.Println("Response: " + string(response))

	return nil
}

// FormatSnowAlertMessage formats a snow alert SMS message.
func FormatSnowAlertMessage(resortName string, snowAmount float64, forecastDate time.Time) string {
	dateStr := forecastDate.Format("Monday, Jan 2")
	return fmt.Sprintf("Powder Alert! %s is expecting %.1f inches of snow on %s. Time to hit the slopes!",
		resortName, snowAmount, dateStr)
}
