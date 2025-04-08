package notify

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

//go:generate mockgen -destination=mocks/mock_notify.go -package=mocks github.com/MattSilvaa/powhunter/internal/notify NotificationService

// NotificationService defines the interface for notification services
type NotificationService interface {
	// SendSMS sends an SMS message
	SendSMS(to, message string) error
}

// TwilioClient handles SMS notifications via Twilio
type TwilioClient struct {
	accountSID   string
	authToken    string
	fromNumber   string
	client       *http.Client
}

// NewTwilioClient creates a new Twilio client
func NewTwilioClient(accountSID, authToken, fromNumber string) *TwilioClient {
	return &TwilioClient{
		accountSID: accountSID,
		authToken:  authToken,
		fromNumber: fromNumber,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SendSMS sends an SMS message using Twilio
func (t *TwilioClient) SendSMS(to, message string) error {
	if to == "" || message == "" {
		return fmt.Errorf("phone number and message are required")
	}

	// Format the phone number if needed
	if !strings.HasPrefix(to, "+") {
		to = "+" + to
	}
	
	// Build the request form data
	data := url.Values{}
	data.Set("To", to)
	data.Set("From", t.fromNumber)
	data.Set("Body", message)
	
	// Set up the HTTP request
	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", t.accountSID),
		strings.NewReader(data.Encode()),
	)
	if err != nil {
		return fmt.Errorf("error creating Twilio request: %w", err)
	}
	
	// Set headers and auth
	req.SetBasicAuth(t.accountSID, t.authToken)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	
	// Make the request
	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending SMS: %w", err)
	}
	defer resp.Body.Close()
	
	// Check response
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errResp struct {
			Code     int    `json:"code"`
			Message  string `json:"message"`
			MoreInfo string `json:"more_info"`
			Status   int    `json:"status"`
		}
		
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return fmt.Errorf("error parsing Twilio error: %w (status: %d)", err, resp.StatusCode)
		}
		
		return fmt.Errorf("Twilio error: %s (code: %d)", errResp.Message, errResp.Code)
	}
	
	return nil
}

// FormatSnowAlertMessage formats a snow alert SMS message
func FormatSnowAlertMessage(resortName string, snowAmount float64, forecastDate time.Time) string {
	dateStr := forecastDate.Format("Monday, Jan 2")
	return fmt.Sprintf("Powder Alert! %s is expecting %.1f inches of snow on %s. Time to hit the slopes!", 
		resortName, snowAmount, dateStr)
}

// EmailClient handles email notifications
type EmailClient struct {
	// Email sending configuration would go here
	// This is a placeholder for future implementation
}

// SendEmail sends an email notification
// This is a placeholder for future implementation
func (e *EmailClient) SendEmail(to, subject, body string) error {
	// Implementation would depend on chosen email provider
	return fmt.Errorf("email sending not yet implemented")
}