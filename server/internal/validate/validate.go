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

	// nanpDigits is the length of a North American number without its
	// country code.
	nanpDigits = 10
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

// Phone normalizes a US or Canadian phone number to E.164.
//
// Only North American numbers are accepted. Alerts are sent by SMS at the
// service's expense, and sending to premium international numbers is the usual
// way an open SMS form is abused to run up a bill ("SMS pumping"). The signup
// form is presented as North American, so this matches what users are told.
func Phone(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", invalid("phone", "is required")
	}

	var digits strings.Builder

	for _, r := range trimmed {
		switch {
		case r >= '0' && r <= '9':
			digits.WriteRune(r)
		case r == '+' || r == '-' || r == ' ' || r == '(' || r == ')' || r == '.':
			// Formatting characters are ignored.
		default:
			return "", invalid("phone", "contains unsupported characters")
		}
	}

	number := digits.String()

	// Accept a bare 10-digit number or one written with the +1 / 1 country code.
	if len(number) == nanpDigits+1 && number[0] == '1' {
		number = number[1:]
	}

	if len(number) != nanpDigits {
		return "", invalid("phone", "must be a US or Canadian number")
	}

	// In the North American Numbering Plan neither the area code nor the
	// exchange may start with 0 or 1.
	if number[0] < '2' || number[3] < '2' {
		return "", invalid("phone", "must be a US or Canadian number")
	}

	if _, blocked := nonNorthAmericanAreaCodes[number[:3]]; blocked {
		return "", invalid("phone", "must be a US or Canadian number")
	}

	return "+1" + number, nil
}

// nonNorthAmericanAreaCodes are +1 area codes that are not the United States
// or Canada: the Caribbean and Atlantic countries that share the North
// American Numbering Plan, plus premium-rate 900. Texting them is
// international SMS at international prices, and they are a well-known
// target for SMS pumping fraud. US territories such as Puerto Rico and Guam
// are the US and stay allowed.
//
//nolint:gochecknoglobals // a fixed lookup table
var nonNorthAmericanAreaCodes = map[string]struct{}{
	"242": {}, // Bahamas
	"246": {}, // Barbados
	"264": {}, // Anguilla
	"268": {}, // Antigua and Barbuda
	"284": {}, // British Virgin Islands
	"345": {}, // Cayman Islands
	"441": {}, // Bermuda
	"473": {}, // Grenada
	"649": {}, // Turks and Caicos
	"658": {}, // Jamaica
	"664": {}, // Montserrat
	"721": {}, // Sint Maarten
	"758": {}, // Saint Lucia
	"767": {}, // Dominica
	"784": {}, // Saint Vincent and the Grenadines
	"809": {}, // Dominican Republic
	"829": {}, // Dominican Republic
	"849": {}, // Dominican Republic
	"868": {}, // Trinidad and Tobago
	"869": {}, // Saint Kitts and Nevis
	"876": {}, // Jamaica
	"900": {}, // premium rate
}

// SingleLine trims a value and removes control characters such as CR and LF.
// Use it for anything that ends up in an email header, where a newline would
// let the submitter add headers of their own.
func SingleLine(raw string) string {
	cleaned := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}

		return r
	}, raw)

	return strings.Join(strings.Fields(cleaned), " ")
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

// maxTagLength bounds a marketing tag such as utm_source.
const maxTagLength = 40

// Tag normalizes a marketing link tag such as utm_source=Reddit to "reddit".
// It returns "" for anything that is not a short run of letters, digits, dots,
// dashes and underscores: a tag is only a label for counting signups, so an
// odd one is dropped rather than rejected.
func Tag(raw string) string {
	tag := strings.ToLower(strings.TrimSpace(raw))
	if tag == "" || len(tag) > maxTagLength {
		return ""
	}

	for _, r := range tag {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '.' && r != '-' && r != '_' {
			return ""
		}
	}

	return tag
}
