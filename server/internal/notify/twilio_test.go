package notify

import (
	"testing"
	"time"
)

func TestFormatSnowAlertMessage(t *testing.T) {
	tests := []struct {
		name         string
		resortName   string
		snowAmount   float64
		forecastDate time.Time
		expected     string
	}{
		{
			name:         "Today's forecast",
			resortName:   "Whistler Blackcomb",
			snowAmount:   8.5,
			forecastDate: time.Now(),
			expected:     "Powder Alert! Whistler Blackcomb is expecting 8.5 inches of snow today. Time to hit the slopes!",
		},
		{
			name:         "Tomorrow's forecast",
			resortName:   "Vail",
			snowAmount:   12.0,
			forecastDate: time.Now().Add(24 * time.Hour),
			expected:     "Powder Alert! Vail is expecting 12.0 inches of snow tomorrow. Time to hit the slopes!",
		},
		{
			name:         "Future date forecast",
			resortName:   "Mammoth Mountain",
			snowAmount:   6.2,
			forecastDate: time.Date(2025, 12, 25, 0, 0, 0, 0, time.UTC),
			expected:     "Powder Alert! Mammoth Mountain is expecting 6.2 inches of snow on Thursday, Dec 25. Time to hit the slopes!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatSnowAlertMessage(tt.resortName, tt.snowAmount, tt.forecastDate)
			if result != tt.expected {
				t.Errorf("FormatSnowAlertMessage() = %q, want %q", result, tt.expected)
			}
		})
	}
}
