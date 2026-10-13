package notify

import (
	"testing"
	"time"

	"github.com/MattSilvaa/powhunter/internal/db"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestFormatSnowAlertMessage(t *testing.T) {
	tests := []struct {
		name     string
		alert    db.AlertToSend
		expected string
	}{
		{
			name: "Today's forecast - new alert",
			alert: db.AlertToSend{
				UserUuid:     uuid.New(),
				UserEmail:    "testing@aol.com",
				UserPhone:    "+1234567890",
				ResortName:   "Whistler Blackcomb",
				ResortUUID:   uuid.New(),
				SnowAmount:   8.5,
				ForecastDate: time.Now(),
				IsUpdate:     false,
			},
			expected: "Powder Alert! Whistler Blackcomb is expecting 8.5 inches of snow today. Time to hit the slopes!",
		},
		{
			name: "Tomorrow's forecast - new alert",
			alert: db.AlertToSend{
				UserUuid:     uuid.New(),
				UserEmail:    "testing@aol.com",
				UserPhone:    "+1234567890",
				ResortName:   "Vail",
				ResortUUID:   uuid.New(),
				SnowAmount:   12.0,
				ForecastDate: time.Now().Add(24 * time.Hour),
				IsUpdate:     false,
			},
			expected: "Powder Alert! Vail is expecting 12.0 inches of snow tomorrow. Time to hit the slopes!",
		},
		{
			name: "Future date forecast - new alert",
			alert: db.AlertToSend{
				UserUuid:     uuid.New(),
				UserEmail:    "testing@aol.com",
				UserPhone:    "+1234567890",
				ResortName:   "Mammoth Mountain",
				ResortUUID:   uuid.New(),
				SnowAmount:   6.2,
				ForecastDate: time.Date(2025, 12, 25, 0, 0, 0, 0, time.UTC),
				IsUpdate:     false,
			},
			expected: "Powder Alert! Mammoth Mountain is expecting 6.2 inches of snow on Thursday, Dec 25. Time to hit the slopes!",
		},
		{
			name: "Today's forecast - update alert",
			alert: db.AlertToSend{
				UserUuid:     uuid.New(),
				UserEmail:    "testing@aol.com",
				UserPhone:    "+1234567890",
				ResortName:   "Whistler Blackcomb",
				ResortUUID:   uuid.New(),
				SnowAmount:   15.0,
				ForecastDate: time.Now(),
				IsUpdate:     true,
			},
			expected: "Powder Alert Update! Whistler Blackcomb is now expecting 15.0 inches of snow today - even more powder than before! Time to hit the slopes!",
		},
		{
			name: "Tomorrow's forecast - update alert",
			alert: db.AlertToSend{
				UserUuid:     uuid.New(),
				UserEmail:    "testing@aol.com",
				UserPhone:    "+1234567890",
				ResortName:   "Vail",
				ResortUUID:   uuid.New(),
				SnowAmount:   18.5,
				ForecastDate: time.Now().Add(24 * time.Hour),
				IsUpdate:     true,
			},
			expected: "Powder Alert Update! Vail is now expecting 18.5 inches of snow tomorrow - even more powder than before! Time to hit the slopes!",
		},
		{
			name: "Future date forecast - update alert",
			alert: db.AlertToSend{
				UserUuid:     uuid.New(),
				UserEmail:    "testing@aol.com",
				UserPhone:    "+1234567890",
				ResortName:   "Mammoth Mountain",
				ResortUUID:   uuid.New(),
				SnowAmount:   9.8,
				ForecastDate: time.Date(2025, 12, 25, 0, 0, 0, 0, time.UTC),
				IsUpdate:     true,
			},
			expected: "Powder Alert Update! Mammoth Mountain is now expecting 9.8 inches of snow on Thursday, Dec 25 - even more powder than before! Time to hit the slopes!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatSnowAlertMessage(tt.alert, "")
			if result != tt.expected {
				t.Errorf("FormatSnowAlertMessage() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFormatSnowAlertMessageWithALink(t *testing.T) {
	const link = "https://go.example.com/go/Ab12Cd34"

	newAlert := db.AlertToSend{ResortName: "Vail", SnowAmount: 12, ForecastDate: time.Now()}
	assert.Equal(t,
		"Powder Alert! Vail is expecting 12.0 inches of snow today. Time to hit the slopes! "+link,
		FormatSnowAlertMessage(newAlert, link), "a message that fits keeps its full wording")

	update := db.AlertToSend{
		ResortName:   "Summit at Snoqualmie",
		SnowAmount:   24.5,
		ForecastDate: time.Date(2025, 12, 24, 0, 0, 0, 0, time.UTC),
		IsUpdate:     true,
	}
	message := FormatSnowAlertMessage(update, link)
	assert.Equal(t,
		"Powder Alert Update! Summit at Snoqualmie is now expecting 24.5 inches of snow on Wednesday, Dec 24. "+link,
		message, "optional phrases are dropped to keep the link from adding a segment")
	assert.Equal(t, int32(1), SMSSegments(message))
}
