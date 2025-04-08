package notify

import (
	"testing"
	"time"
)

// Skip network tests by default
func TestSendSMSValidation(t *testing.T) {
	client := NewTwilioClient("test_account_sid", "test_auth_token", "+18005551234")

	// Test with empty phone number
	err := client.SendSMS("", "Test message")
	if err == nil {
		t.Error("Expected error with empty phone number but got none")
	}

	// Test with empty message
	err = client.SendSMS("+12025551234", "")
	if err == nil {
		t.Error("Expected error with empty message but got none")
	}
}

func TestFormatSnowAlertMessage(t *testing.T) {
	// Create a test date
	date := time.Date(2023, 4, 8, 0, 0, 0, 0, time.UTC)

	// Test formatting with whole inch amount
	message := FormatSnowAlertMessage("Crystal Mountain", 8.0, date)
	expected := "Powder Alert! Crystal Mountain is expecting 8.0 inches of snow on Saturday, Apr 8. Time to hit the slopes!"
	if message != expected {
		t.Errorf("Expected message %q, got %q", expected, message)
	}

	// Test formatting with fractional inch amount
	message = FormatSnowAlertMessage("Crystal Mountain", 8.5, date)
	expected = "Powder Alert! Crystal Mountain is expecting 8.5 inches of snow on Saturday, Apr 8. Time to hit the slopes!"
	if message != expected {
		t.Errorf("Expected message %q, got %q", expected, message)
	}
}