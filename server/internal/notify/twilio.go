package notify

import (
	"errors"
	"fmt"
	"hash/fnv"
	"strings"
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

// snowTier is the headline and sign-off lines for one range of snowfall.
type snowTier struct {
	minInches float64
	headline  string
	taglines  []string
}

// snowTiers runs from deepest to lightest so the first match wins. Every line
// sticks to the GSM character set: one emoji or curly quote would switch the
// whole message to UCS-2 and split it into two or three billed segments.
var snowTiers = []snowTier{
	{minInches: 12, headline: "DEEP DAY", taglines: []string{
		"Bring a snorkel.",
		"Cancel everything.",
		"This is the one.",
	}},
	{minInches: 6, headline: "POW DAY", taglines: []string{
		"No friends on a pow day.",
		"Set the alarm early.",
		"Time to call in sick.",
	}},
	{minInches: 3, headline: "POWDER ALERT", taglines: []string{
		"First chair is calling.",
		"Worth the drive.",
		"Go wax your skis.",
	}},
	{minInches: 0, headline: "FRESH SNOW", taglines: []string{
		"A soft refresh on top.",
		"Get there early.",
		"Fresh is fresh.",
	}},
}

var updateTaglines = []string{
	"It keeps getting better.",
	"The storm is building.",
	"Even more powder.",
}

// FormatSnowAlertMessage formats a snow alert SMS. The headline scales with
// how much snow is coming, for example:
//
//	POW DAY at Crystal Mountain
//	8" forecast tomorrow. Set the alarm early.
//	https://example.com/go/Ab12Cd34
//	Share: powhunter.app
//
// It adds link when one is given, then a line inviting the reader to share
// shareHost (such as "powhunter.app") when that is given.
//
// Extra text can push an alert into a second segment, which doubles what that
// message costs. So the share line is only added when the message still fits
// in one segment, trying the shorter share line and then dropping the
// tagline first. The link is kept either way: with a link, the tagline is dropped
// when that makes it fit.
func FormatSnowAlertMessage(alert db.AlertToSend, link, shareHost string) string {
	when := forecastDay(alert.ForecastDate, time.Now())
	tagline := pickLine(alert, updateTaglines)

	var headline, facts string
	if alert.IsUpdate {
		headline = "STORM UPGRADE at " + alert.ResortName
		facts = fmt.Sprintf("Now %s forecast %s", inches(alert.SnowAmount), when)
		if alert.PreviousSnowAmount > 0 {
			facts += ", up from " + inches(alert.PreviousSnowAmount)
		}
	} else {
		tier := tierFor(alert.SnowAmount)
		headline = tier.headline + " at " + alert.ResortName
		facts = fmt.Sprintf("%s forecast %s", inches(alert.SnowAmount), when)
		tagline = pickLine(alert, tier.taglines)
	}

	// Wordings from fullest to shortest.
	lead := headline + "\n" + facts + "."
	wordings := []string{lead + " " + tagline, lead}

	var withLink string
	if link != "" {
		withLink = "\n" + link
	}

	// The tagline gives an alert its character, so it outranks the longer share
	// wording, but any share line outranks the tagline: a friend signing up is
	// worth more than a quip.
	if shareHost != "" {
		for _, wording := range wordings {
			for _, share := range []string{"\nSend to your ski crew: ", "\nShare: "} {
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
		if message := wording + withLink; SMSSegments(message) == 1 {
			return message
		}
	}

	return wordings[len(wordings)-1] + withLink
}

// forecastDay describes the forecast date relative to now: "today",
// "tomorrow" or "on Tue, Nov 24".
func forecastDay(forecastDate, now time.Time) string {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	day := time.Date(forecastDate.Year(), forecastDate.Month(), forecastDate.Day(), 0, 0, 0, 0, now.Location())

	switch {
	case day.Equal(today):
		return "today"
	case day.Equal(today.AddDate(0, 0, 1)):
		return "tomorrow"
	default:
		return "on " + forecastDate.Format("Mon, Jan 2")
	}
}

// inches formats a snow amount like 8" or 8.5", dropping a trailing ".0".
func inches(amount float64) string {
	return strings.TrimSuffix(fmt.Sprintf("%.1f", amount), ".0") + `"`
}

func tierFor(amount float64) snowTier {
	for _, tier := range snowTiers {
		if amount >= tier.minInches {
			return tier
		}
	}

	return snowTiers[len(snowTiers)-1]
}

// pickLine chooses one of lines so alerts don't all read the same. The choice
// depends only on the resort and day, so a retried alert keeps its wording.
func pickLine(alert db.AlertToSend, lines []string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(alert.ResortName + alert.ForecastDate.Format(time.DateOnly)))

	return lines[h.Sum32()%uint32(len(lines))] //nolint:gosec // len is small and positive
}
