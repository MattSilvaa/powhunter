# Email Configuration Guide

This document explains how to configure email sending for the contact form.

## Overview

When users submit the contact form, the system:
1. Logs the message to a file (for backup)
2. Sends an email to `support@powhunter.app` with the contact details

The email includes:
- Sender's name and email
- Timestamp
- Message content
- Reply-To header set to the sender's email (so you can reply directly)

## Environment Variables

Configure the following environment variables to enable email sending:

### Required Variables

```bash
# SMTP Server Configuration
SMTP_HOST=smtp.example.com      # Your SMTP server hostname
SMTP_PORT=587                    # SMTP port (587 for TLS, 465 for SSL, 25 for plain)
SMTP_USER=your-email@example.com # SMTP username (often your email)
SMTP_PASSWORD=your-password      # SMTP password or app-specific password
```

### Optional Variables

```bash
# From Email Address (defaults to noreply@powhunter.app)
SMTP_FROM_EMAIL=noreply@powhunter.app

# Contact Log Path (defaults to /tmp/powhunter_contacts.log)
CONTACT_LOG_PATH=/var/log/powhunter/contacts.log
```

## SMTP Provider Examples

### Gmail

If using Gmail, you'll need to use an [App Password](https://support.google.com/accounts/answer/185833):

```bash
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASSWORD=your-app-password
SMTP_FROM_EMAIL=your-email@gmail.com
```

### SendGrid

```bash
SMTP_HOST=smtp.sendgrid.net
SMTP_PORT=587
SMTP_USER=apikey
SMTP_PASSWORD=your-sendgrid-api-key
SMTP_FROM_EMAIL=noreply@powhunter.app
```

### Mailgun

```bash
SMTP_HOST=smtp.mailgun.org
SMTP_PORT=587
SMTP_USER=postmaster@your-domain.mailgun.org
SMTP_PASSWORD=your-mailgun-smtp-password
SMTP_FROM_EMAIL=noreply@powhunter.app
```

### AWS SES

```bash
SMTP_HOST=email-smtp.us-east-1.amazonaws.com
SMTP_PORT=587
SMTP_USER=your-ses-smtp-username
SMTP_PASSWORD=your-ses-smtp-password
SMTP_FROM_EMAIL=noreply@powhunter.app
```

### Custom SMTP Server

```bash
SMTP_HOST=mail.yourdomain.com
SMTP_PORT=587
SMTP_USER=support@powhunter.app
SMTP_PASSWORD=your-password
SMTP_FROM_EMAIL=support@powhunter.app
```

## Security Best Practices

1. **Never commit credentials to git**: Use environment variables or secret management
2. **Use TLS/SSL**: Always use port 587 (TLS) or 465 (SSL)
3. **App-specific passwords**: For Gmail and similar services, use app-specific passwords
4. **Verify sender domain**: Ensure your SMTP provider allows sending from `@powhunter.app`
5. **Monitor rate limits**: Check your SMTP provider's sending limits

## Fallback Behavior

If SMTP is not configured (missing `SMTP_HOST` or `SMTP_PORT`):
- The contact form will still work
- Messages will be logged to the file
- No email will be sent
- A log message will indicate "SMTP not configured, skipping email send"

This ensures the contact form remains functional even without email configuration.

## Testing Email Configuration

### 1. Check Logs

After submitting a contact form, check the server logs:

```bash
# Look for success message
grep "Contact email sent" /path/to/logs

# Or look for errors
grep "Failed to send email" /path/to/logs
```

### 2. Check Contact Log File

```bash
cat /tmp/powhunter_contacts.log
# or
cat $CONTACT_LOG_PATH
```

### 3. Test with curl

```bash
curl -X POST http://localhost:8080/api/contact \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test User",
    "email": "test@example.com",
    "message": "This is a test message"
  }'
```

Expected response:
```json
{
  "status": "success",
  "message": "Thank you for contacting us! We'll get back to you soon."
}
```

### 4. Verify Email Receipt

Check your inbox at `support@powhunter.app` for the test message.

## Troubleshooting

### Email not being sent

1. **Check environment variables are set**:
   ```bash
   echo $SMTP_HOST
   echo $SMTP_PORT
   ```

2. **Check server logs** for error messages

3. **Verify SMTP credentials** are correct

4. **Check firewall rules** - ensure outbound connections to SMTP port are allowed

5. **Verify sender domain** - some SMTP providers require sender verification

### "Authentication failed" error

- Double-check SMTP username and password
- For Gmail, ensure you're using an app-specific password
- Verify your SMTP provider allows the authentication method (PlainAuth)

### "Connection refused" error

- Check SMTP_HOST and SMTP_PORT are correct
- Ensure your server can reach the SMTP server
- Check firewall rules

### Emails going to spam

- Set up SPF, DKIM, and DMARC records for your domain
- Use a reputable SMTP provider
- Ensure from address matches your domain

## Production Deployment

For production, consider using:

1. **Managed Email Services**: SendGrid, Mailgun, AWS SES, Postmark
2. **Environment Variables**: Store credentials in your hosting provider's secret manager
3. **Monitoring**: Set up alerts for email delivery failures
4. **Rate Limiting**: Implement rate limiting on the contact form endpoint
5. **Spam Protection**: Add CAPTCHA or honeypot fields

## Contact Log File

Messages are always logged to a file as backup. The log format is:

```
[2025-10-12T15:30:00Z] Name: John Doe | Email: john@example.com | Message: I have a question
```

Rotate this log file regularly to prevent it from growing too large:

```bash
# Using logrotate (example configuration)
/var/log/powhunter/contacts.log {
    daily
    rotate 30
    compress
    missingok
    notifempty
}
```
