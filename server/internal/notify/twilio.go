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

// FormatSnowAlertMessage formats a snow alert SMS message. It adds link when
// one is given, then a line inviting the reader to share shareHost (such as
// "powhunter.app") when that is given.
//
// Extra text can push an alert into a second segment, which doubles what that
// message costs. So the share line is only added when the message still fits
// in one segment, trying the shorter share line and dropping the optional
// closing phrases first. The link is kept either way: with a link, closing
// phrases are dropped one at a time until it fits. With neither a link nor a
// share line that fits, the wording is unchanged.
func FormatSnowAlertMessage(alert db.AlertToSend, link, shareHost string) string {
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

	// Wordings from fullest to shortest.
	var wordings []string

	if alert.IsUpdate {
		lead := fmt.Sprintf("Powder Alert Update! %s is now expecting %.1f inches of snow %s",
			resortName, snowAmount, timeStr)
		wordings = []string{
			lead + " - even more powder than before! Time to hit the slopes!",
			lead + " - even more powder than before!",
			lead + ".",
		}
	} else {
		lead := fmt.Sprintf("Powder Alert! %s is expecting %.1f inches of snow %s.",
			resortName, snowAmount, timeStr)
		wordings = []string{
			lead + " Time to hit the slopes!",
			lead,
		}
	}

	var withLink string
	if link != "" {
		withLink = " " + link
	}

	// Sharing beats the closing phrases: a friend signing up is worth more than
	// "Time to hit the slopes!".
	if shareHost != "" {
		for _, share := range []string{" Send to your ski crew: ", " Share: "} {
			for _, wording := range wordings {
				if message := wording + withLink + share + shareHost; SMSSegments(message) == 1 {
					return message
				}
			}
		}
	}

	if link == "" {
		return wordings[0]
	}

	for _, wording := range wordings {
		if message := wording + " " + link; SMSSegments(message) == 1 {
			return message
		}
	}

	return wordings[len(wordings)-1] + " " + link
}
