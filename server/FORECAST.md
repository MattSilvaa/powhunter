# Snow Forecast Alerts

This feature integrates with the Weather.gov API to send snow forecast alerts to users. The system checks for snow forecasts at ski resorts and sends SMS notifications through Twilio when matching the users' alert criteria.

## Overview

The feature consists of four main components:

1. **Weather Service**: Fetches snow forecasts from Weather.gov API
2. **Notification Service**: Sends SMS alerts via Twilio
3. **Scheduler**: Periodically checks forecasts and sends notifications 
4. **Database**: Stores user alerts and alert history

## Weather.gov API Integration

The system integrates with the [Weather.gov API](https://www.weather.gov/documentation/services-web-api#/default/gridpoint_forecast) to get accurate snow predictions for ski resorts. 

### API Endpoints Used:

- `/points/{lat},{lon}`: Get the grid point for a location
- `/gridpoints/{gridId}/{gridX},{gridY}/forecast`: Get detailed forecast including snow predictions

### Snow Prediction Process:

1. Convert resort latitude/longitude to Weather.gov grid points
2. Fetch detailed forecast from the grid point
3. Extract and process snowfall predictions
4. Convert measurements to inches if necessary
5. Aggregate snow amounts by day

## Notification System

SMS alerts are sent through Twilio when matching alert criteria is found. The Twilio client:

1. Formats snow alert messages with resort name, amount, and date
2. Sends SMS using the configured Twilio account
3. Records sent alerts in the database to prevent duplicates

## Alert Scheduler

The scheduler runs automatically every 12 hours to check for new forecasts. It:

1. Retrieves all resorts from the database
2. Fetches forecasts for each resort
3. Finds alerts matching the forecast criteria
4. Sends notifications to users
5. Tracks sent alerts

## Manual Forecast Checking

You can manually check forecasts using the provided command:

```bash
# Check forecasts without sending SMS
make check-forecasts

# Check forecasts and send SMS notifications
make check-forecasts-send-sms
```

## Configuration

Set the following environment variables to enable SMS notifications:

```
TWILIO_ACCOUNT_SID=your_account_sid
TWILIO_AUTH_TOKEN=your_auth_token  
TWILIO_FROM_NUMBER=your_twilio_phone_number
```

## Database Schema

The feature uses the following database tables:

- `users`: Store user contact information (email, phone)
- `resorts`: Store resort information including lat/long coordinates
- `user_alerts`: Store alert preferences (resort, snow amount, notification days)
- `alert_history`: Track sent alerts to prevent duplicates

## Testing

The code includes comprehensive tests for all components:

- Weather service tests
- Notification service tests
- Scheduler tests

Run the tests with:

```bash
make test
```