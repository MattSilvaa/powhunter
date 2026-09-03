// Package validate holds the input rules shared by the HTTP handlers.
package validate

import (
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"unicode"
)

const (
	// MaxEmailLength matches the users.email column width.
	MaxEmailLength = 255
	// MaxNameLength bounds a contact form name.
	MaxNameLength = 200
	// MaxMessageLength bounds a contact form message.
	MaxMessageLength = 5000
	// MaxResortsPerRequest bounds how many alerts one signup may create, so a
	// single request cannot open a transaction that inserts thousands of rows.
	MaxResortsPerRequest = 50

	// MinNotificationDays and MaxNotificationDays match the signup slider.
	MinNotificationDays = 1
	MaxNotificationDays = 10

	// MinSnowAmount and MaxSnowAmount match the signup slider. A threshold of
	// zero would alert on every forecast, so the minimum is above zero.
	MinSnowAmount = 0.5
	MaxSnowAmount = 24.0

	minPhoneDigits = 10
	maxPhoneDigits = 15
)

// ErrInvalid marks a value that failed validation.
var ErrInvalid = errors.New("invalid value")

func invalid(field, reason string) error {
	return fmt.Errorf("%w: %s %s", ErrInvalid, field, reason)
}

// Email normalizes and validates an email address. Addresses are lowercased so
// that a user cannot end up with two accounts differing only by case.
func Email(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", invalid("email", "is required")
	}

	if len(trimmed) > MaxEmailLength {
		return "", invalid("email", "is too long")
	}

	address, err := mail.ParseAddress(trimmed)
	if err != nil {
		return "", invalid("email", "is not a valid address")
	}

	// mail.ParseAddress accepts a display name; keep only the address itself.
	if address.Address != trimmed {
		return "", invalid("email", "is not a valid address")
	}

	at := strings.LastIndex(address.Address, "@")
	if at < 0 || !strings.Contains(address.Address[at+1:], ".") {
		return "", invalid("email", "is not a valid address")
	}

	return strings.ToLower(address.Address), nil
}

// Phone normalizes a phone number to E.164. The forecaster sends SMS to
// whatever is stored, so an unnormalized number is a delivery failure.
func Phone(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", invalid("phone", "is required")
	}

	var digits strings.Builder

	hasPlus := strings.HasPrefix(trimmed, "+")

	for _, r := range trimmed {
		switch {
		case unicode.IsDigit(r):
			digits.WriteRune(r)
		case r == '+' || r == '-' || r == ' ' || r == '(' || r == ')' || r == '.':
			// Formatting characters are ignored.
		default:
			return "", invalid("phone", "contains unsupported characters")
		}
	}

	number := digits.String()
	if len(number) < minPhoneDigits || len(number) > maxPhoneDigits {
		return "", invalid("phone", "must be between 10 and 15 digits")
	}

	// A bare 10-digit number is assumed to be North American, matching how the
	// signup form is presented today.
	if !hasPlus && len(number) == minPhoneDigits {
		return "+1" + number, nil
	}

	return "+" + number, nil
}

// NotificationDays checks the forecast window is within the supported range.
func NotificationDays(days int) error {
	if days < MinNotificationDays || days > MaxNotificationDays {
		return invalid(
			"notificationDays",
			fmt.Sprintf("must be between %d and %d", MinNotificationDays, MaxNotificationDays),
		)
	}

	return nil
}

// SnowAmount checks the alert threshold is within the supported range.
func SnowAmount(inches float64) error {
	if inches < MinSnowAmount || inches > MaxSnowAmount {
		return invalid(
			"minSnowAmount",
			fmt.Sprintf("must be between %.1f and %.1f inches", MinSnowAmount, MaxSnowAmount),
		)
	}

	return nil
}

// Text trims and length-checks a free-text field.
func Text(field, raw string, maxLength int) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", invalid(field, "is required")
	}

	if len(trimmed) > maxLength {
		return "", invalid(field, fmt.Sprintf("must be at most %d characters", maxLength))
	}

	return trimmed, nil
}
