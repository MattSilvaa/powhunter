# Pow Hunter

> Never miss fresh powder at your favorite mountain resorts

Pow Hunter is a web application that sends you SMS alerts when fresh snow is forecasted at your favorite ski resorts. Set your notification preferences, choose your resorts, and get ready to shred!

## Features

- 🏔️ Track snow forecasts at multiple mountain resorts
- 📱 Receive SMS notifications when fresh powder is coming
- ⏱️ Customize notification timing (1-5 days in advance)
- ❄️ Set minimum snow thresholds for alerts
- 🗺️ View detailed resort information and forecasts
- 🌤️ Real-time weather data from Weather.gov API
- 📧 Contact form with email notifications to support team

## Technical Stack

- **Frontend**: TypeScript, React, Material UI
- **Backend**: Go, PostgreSQL
- **APIs**: Weather.gov for forecasts, Twilio for SMS

## Weather.gov Integration

Pow Hunter uses the National Weather Service (NWS) API to get accurate snow forecasts for ski resorts. The application:

1. Retrieves grid coordinates based on resort latitude/longitude
2. Fetches detailed snowfall predictions for each resort
3. Checks forecasts against user alert criteria
4. Delivers timely SMS notifications through Twilio

## Getting Started

### Prerequisites

- Go 1.24+
- PostgreSQL
- Node.js 18+
- Twilio account (for SMS notifications)

### Environment Variables

See `server/.env.example` for a complete list of configuration options.

```
# Database
DATABASE_URL=postgres://user:password@localhost:5432/powhunter

# Twilio (for SMS notifications)
TWILIO_ACCOUNT_SID=your_account_sid
TWILIO_AUTH_TOKEN=your_auth_token
TWILIO_FROM_NUMBER=your_twilio_phone_number

# SMTP (for contact form emails)
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USER=your-email@example.com
SMTP_PASSWORD=your-password
SMTP_FROM_EMAIL=noreply@powhunter.app
```

For detailed email configuration, see [server/EMAIL_CONFIGURATION.md](server/EMAIL_CONFIGURATION.md).

### Setup

```bash
# Install dependencies
make install

# Run development environment
make dev

# Build for production
make build
```

## How It Works

1. Users sign up and set alert preferences:
   - Email and phone number
   - Minimum snow amount (in inches)
   - Notification days in advance (1-5)
   - Selected ski resorts

2. Every 12 hours, the system:
   - Fetches the latest snow forecasts from Weather.gov
   - Identifies matching user alerts
   - Sends SMS notifications for new forecasts
   - Records sent alerts to prevent duplicates

## Contributing

Contributions are welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for details.