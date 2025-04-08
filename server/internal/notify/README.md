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
// Initialize Twilio client
twilioClient := notify.NewTwilioClient(
    os.Getenv("TWILIO_ACCOUNT_SID"),
    os.Getenv("TWILIO_AUTH_TOKEN"),
    os.Getenv("TWILIO_FROM_NUMBER"),
)

// Send a snow alert SMS
message := notify.FormatSnowAlertMessage(
    "Crystal Mountain", 
    8.5, 
    time.Now().AddDate(0, 0, 2),
)
err := twilioClient.SendSMS("+12025551234", message)
if err != nil {
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