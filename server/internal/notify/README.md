# Notification Service

This package handles sending notifications to users via SMS and (future) email. It currently implements Twilio for SMS delivery.

## Twilio Integration

### Configuration

To use the Twilio integration, set the following environment variables:

```
TWILIO_ACCOUNT_SID=your_account_sid
TWILIO_AUTH_TOKEN=your_auth_token
TWILIO_FROM_NUMBER=your_twilio_phone_number
```

The phone number should be in E.164 format (e.g., `+12025551234`).

### Usage Example

```go
// Credentials come from TWILIO_ACCOUNT_SID and TWILIO_AUTH_TOKEN.
twilioClient := notify.NewTwilioClient(os.Getenv("TWILIO_FROM_NUMBER"))

// The second argument is an optional tracked link (see LINK_BASE_URL). With a
// link, optional phrases are dropped so the message stays in one segment.
message := notify.FormatSnowAlertMessage(alert, "https://api.example.com/go/Ab12Cd34")
segments := notify.SMSSegments(message) // what Twilio will bill

if err := twilioClient.SendSMS(alert.UserPhone, message); err != nil {
    log.Printf("Error sending SMS: %v", err)
}
```

### Message Formatting

The service includes a helper function to format snow alert messages:

```go
FormatSnowAlertMessage(resortName string, snowAmount float64, forecastDate time.Time) string
```

This generates a formatted message like:
> Powder Alert! Crystal Mountain is expecting 8.5 inches of snow on Wednesday, Jan 15. Time to hit the slopes!

### Error Handling

The Twilio client includes error handling for:
- Invalid phone numbers
- API errors
- Authentication issues

## Future Extensions

The package includes a placeholder for email notifications which can be implemented in the future.