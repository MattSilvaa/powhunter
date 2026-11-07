package weather

import (
	"context"
	"fmt"
	"time"
)

// DailySnowfall is the snow that fell at a location on one day.
type DailySnowfall struct {
	Date   time.Time
	Inches float64
}

// archiveResponse is the part of an Open-Meteo archive response we read.
// Snowfall entries are null for the last few days, before the reanalysis that
// backs the archive has caught up.
type archiveResponse struct {
	Daily struct {
		Time        []string   `json:"time"`
		SnowfallSum []*float64 `json:"snowfall_sum"`
	} `json:"daily"`
}

// GetDailySnowfall returns the observed daily snowfall at a location between
// start and end inclusive, from Open-Meteo's historical weather archive. Days
// the archive has no value for yet are left out.
func (c *OpenMeteoClient) GetDailySnowfall(
	ctx context.Context,
	lat, lon float64,
	start, end time.Time,
) ([]DailySnowfall, error) {
	url := fmt.Sprintf(
		"%s/archive?latitude=%.6f&longitude=%.6f&start_date=%s&end_date=%s&daily=snowfall_sum&precipitation_unit=inch&timezone=Etc/UTC",
		c.archiveBaseURL,
		lat,
		lon,
		start.Format(time.DateOnly),
		end.Format(time.DateOnly),
	)

	var archive archiveResponse
	if err := c.getJSON(ctx, url, &archive); err != nil {
		return nil, fmt.Errorf("error getting snowfall history: %w", err)
	}

	days := make([]DailySnowfall, 0, len(archive.Daily.Time))
	for i, dateStr := range archive.Daily.Time {
		if i >= len(archive.Daily.SnowfallSum) || archive.Daily.SnowfallSum[i] == nil {
			continue
		}

		date, err := time.Parse(time.DateOnly, dateStr)
		if err != nil {
			return nil, fmt.Errorf("parsing archive date %q: %w", dateStr, err)
		}

		days = append(days, DailySnowfall{Date: date, Inches: *archive.Daily.SnowfallSum[i]})
	}

	return days, nil
}
