package notify

import (
	"strings"
	"testing"
	"time"

	"github.com/MattSilvaa/powhunter/internal/db"
	"github.com/stretchr/testify/assert"
)

const testLink = "https://powhunter-public-api.onrender.com/go/Ab12Cd34"

func TestFormatSnowAlertMessageHeadlineScalesWithSnow(t *testing.T) {
	tests := []struct {
		amount   float64
		headline string
	}{
		{1.2, "FRESH SNOW at Vail"},
		{3, "POWDER ALERT at Vail"},
		{8.5, "POW DAY at Vail"},
		{14, "DEEP DAY at Vail"},
	}

	for _, tt := range tests {
		alert := db.AlertToSend{ResortName: "Vail", SnowAmount: tt.amount, ForecastDate: time.Now()}
		lines := strings.Split(FormatSnowAlertMessage(alert, ""), "\n")
		assert.Equal(t, tt.headline, lines[0])
	}
}

func TestFormatSnowAlertMessageFacts(t *testing.T) {
	alert := db.AlertToSend{ResortName: "Vail", SnowAmount: 12, ForecastDate: time.Now()}
	assert.Contains(t, FormatSnowAlertMessage(alert, ""), "\n12\" forecast today. ",
		"a whole number drops its .0 and the full message keeps a tagline")

	alert.ForecastDate = time.Now().AddDate(0, 0, 1)
	assert.Contains(t, FormatSnowAlertMessage(alert, ""), "forecast tomorrow.")

	alert.ForecastDate = time.Date(2025, 12, 25, 0, 0, 0, 0, time.UTC)
	assert.Contains(t, FormatSnowAlertMessage(alert, ""), "forecast on Thu, Dec 25.")

	update := db.AlertToSend{
		ResortName: "Vail", SnowAmount: 9.5, PreviousSnowAmount: 4, ForecastDate: time.Now(), IsUpdate: true,
	}
	message := FormatSnowAlertMessage(update, "")
	assert.True(t, strings.HasPrefix(message, "STORM UPGRADE at Vail\nNow 9.5\" forecast today, up from 4\"."), message)
}

func TestFormatSnowAlertMessageIsStableForTheSameForecast(t *testing.T) {
	alert := db.AlertToSend{ResortName: "Vail", SnowAmount: 7, ForecastDate: time.Now()}
	assert.Equal(t, FormatSnowAlertMessage(alert, testLink), FormatSnowAlertMessage(alert, testLink))
}

func TestFormatSnowAlertMessageFitsOneSegment(t *testing.T) {
	short := db.AlertToSend{ResortName: "Vail", SnowAmount: 12, ForecastDate: time.Now()}
	message := FormatSnowAlertMessage(short, testLink)
	assert.Equal(t, FormatSnowAlertMessage(short, "")+"\n"+testLink, message,
		"a short alert keeps its tagline and ends with the link")

	long := db.AlertToSend{
		ResortName:         "Summit at Snoqualmie",
		SnowAmount:         24.5,
		PreviousSnowAmount: 18,
		ForecastDate:       time.Date(2025, 12, 24, 0, 0, 0, 0, time.UTC),
		IsUpdate:           true,
	}
	message = FormatSnowAlertMessage(long, strings.Repeat("x", 60))
	assert.Equal(t,
		"STORM UPGRADE at Summit at Snoqualmie\nNow 24.5\" forecast on Wed, Dec 24, up from 18\".\n"+
			strings.Repeat("x", 60),
		message, "the tagline is dropped to keep a long link in one segment")
	assert.Equal(t, int32(1), SMSSegments(message))
}
