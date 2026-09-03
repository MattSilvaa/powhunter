package notify

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/resend/resend-go/v2"
)

// Email is one outbound message.
type Email struct {
	To      []string
	Subject string
	HTML    string
	ReplyTo string
}

// Mailer sends transactional email. The interface exists so the magic-link and
// contact flows share one implementation, and so tests can assert on what would
// have been sent without reaching the network.
type Mailer interface {
	Send(ctx context.Context, msg Email) error
}

// ResendMailer sends through Resend.
type ResendMailer struct {
	client *resend.Client
	from   string
}

// NewResendMailer builds a mailer for the given API key.
func NewResendMailer(apiKey, from string) *ResendMailer {
	return &ResendMailer{client: resend.NewClient(apiKey), from: from}
}

// Send delivers the message.
func (m *ResendMailer) Send(ctx context.Context, msg Email) error {
	sent, err := m.client.Emails.SendWithContext(ctx, &resend.SendEmailRequest{
		From:    m.from,
		To:      msg.To,
		ReplyTo: msg.ReplyTo,
		Subject: msg.Subject,
		Html:    msg.HTML,
	})
	if err != nil {
		return fmt.Errorf("sending email: %w", err)
	}

	slog.Default().Info("email sent", "id", sent.Id, "subject", msg.Subject)

	return nil
}

// LogMailer stands in when no API key is configured. It logs what it would have
// sent, which keeps local development working: the magic link ends up in the
// server log rather than an inbox.
type LogMailer struct {
	logger *slog.Logger
}

// NewLogMailer builds a mailer that only logs.
func NewLogMailer(logger *slog.Logger) *LogMailer {
	return &LogMailer{logger: logger}
}

// Send logs the message instead of delivering it.
func (m *LogMailer) Send(_ context.Context, msg Email) error {
	m.logger.Info("email not sent (no mail provider configured)",
		"to", msg.To,
		"subject", msg.Subject,
		"body", msg.HTML,
	)

	return nil
}
