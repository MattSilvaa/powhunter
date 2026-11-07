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
		{name: "too short", input: "12345", wantErr: true},
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
