package validate_test

import (
	"strings"
	"testing"

	"github.com/MattSilvaa/powhunter/internal/validate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmail(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "plain address", input: "skier@example.com", want: "skier@example.com"},
		{name: "trims and lowercases", input: "  Skier@Example.COM ", want: "skier@example.com"},
		{name: "empty", input: "", wantErr: true},
		{name: "no domain dot", input: "skier@example", wantErr: true},
		{name: "dot at only, previously accepted", input: ".@", wantErr: true},
		{name: "display name form", input: "Skier <skier@example.com>", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validate.Email(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPhone(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "ten digits assumed north american", input: "6195733405", want: "+16195733405"},
		{name: "formatted", input: "(619) 573-3405", want: "+16195733405"},
		{name: "already e164", input: "+16195733405", want: "+16195733405"},
		{name: "country code without plus", input: "1 619 573 3405", want: "+16195733405"},
		{name: "too short", input: "12345", wantErr: true},
		// International numbers are refused: SMS to them is the usual target of
		// toll-fraud abuse, and the service only operates in North America.
		{name: "international", input: "+44 20 7946 0958", wantErr: true},
		{name: "eleven digits not starting with one", input: "26195733405", wantErr: true},
		{name: "area code starting with one", input: "(123) 456-7890", wantErr: true},
		{name: "exchange starting with zero", input: "619-073-3405", wantErr: true},
		// +1 but not the US or Canada: the Caribbean numbers SMS pumping targets.
		{name: "bahamas", input: "+1 242 555 0123", wantErr: true},
		{name: "jamaica", input: "(876) 555-0123", wantErr: true},
		{name: "premium rate", input: "900-555-0123", wantErr: true},
		{name: "us territory", input: "787-555-0123", want: "+17875550123"},
		{name: "non-ascii digits", input: "٦١٩٥٧٣٣٤٠٥", wantErr: true},
		{name: "letters", input: "555-CALL-NOW", wantErr: true},
		{name: "empty", input: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validate.Phone(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNumericBounds(t *testing.T) {
	require.NoError(t, validate.NotificationDays(3))
	require.Error(t, validate.NotificationDays(0))
	require.Error(t, validate.NotificationDays(99))

	require.NoError(t, validate.SnowAmount(6))
	// A zero or negative threshold would alert on every forecast.
	require.Error(t, validate.SnowAmount(0))
	require.Error(t, validate.SnowAmount(-5))
	require.Error(t, validate.SnowAmount(1000))
}

func TestTag(t *testing.T) {
	tests := map[string]string{
		"Reddit":                "reddit",
		" cosnow ":              "cosnow",
		"tiktok_2026-nov.":      "tiktok_2026-nov.",
		"":                      "",
		"two words":             "",
		"<script>":              "",
		strings.Repeat("a", 41): "",
	}

	for raw, want := range tests {
		if got := validate.Tag(raw); got != want {
			t.Errorf("Tag(%q) = %q, want %q", raw, got, want)
		}
	}
}

// A name or subject with a newline in it could add email headers of its own.
func TestSingleLineStripsControlCharacters(t *testing.T) {
	assert.Equal(t, "Ann Bcc: evil@example.com", validate.SingleLine("  Ann\r\nBcc: evil@example.com\t"))
	assert.Equal(t, "plain", validate.SingleLine("plain"))
}
