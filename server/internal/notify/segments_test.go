package notify_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/MattSilvaa/powhunter/internal/notify"
)

func TestSMSSegments(t *testing.T) {
	tests := []struct {
		name    string
		message string
		want    int32
	}{
		{"empty still bills one segment", "", 1},
		{"plain text at the single-segment limit", strings.Repeat("a", 160), 1},
		{"plain text one over the limit splits", strings.Repeat("a", 161), 2},
		{"multi-part segments hold 153", strings.Repeat("a", 307), 3},
		{"extension characters count twice", strings.Repeat("€", 80) + "a", 2},
		{"an emoji forces UCS-2", strings.Repeat("a", 70) + "❄", 2},
		{"UCS-2 at the single-segment limit", strings.Repeat("a", 69) + "❄", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, notify.SMSSegments(tt.message))
		})
	}
}
