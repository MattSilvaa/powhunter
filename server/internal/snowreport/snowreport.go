// Package snowreport turns stored snowfall into the home page's snow report:
// a live board while it is snowing somewhere, and last season's recap when it
// is not. It holds the rules only; the forecaster fills the tables and the API
// serves the result.
package snowreport

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	dbgen "github.com/MattSilvaa/powhunter/internal/db/generated"
	"github.com/MattSilvaa/powhunter/internal/weather"
)

const (
	// SnowDayThresholdIn is the daily snowfall that counts as a snow day when
	// deciding whether it is the season.
	SnowDayThresholdIn = 2.0
	// LookbackDays is how far back a snow day keeps the live report on. Three
	// weeks means a dry spell in January does not flip the site to the recap.
	LookbackDays = 21
	// ForecastDays is how many days ahead the live report covers, today
	// included.
	ForecastDays = weather.ForecastDays
	// DeepDayIn is the daily snowfall the recap counts as a deep day.
	DeepDayIn = 6.0
	// StormWindowDays is the length of the recap's "biggest storm".
	StormWindowDays = 3
	// ThroughMinIn is the smallest day that counts toward "by Thursday" in the
	// live report, so a trace of snow at the end of the window does not push
	// the date out.
	ThroughMinIn = 1.0

	// archiveLagDays is how long after a season ends its archive data is
	// treated as complete. Open-Meteo's archive trails real time by a few days.
	archiveLagDays = 7
	dateLayout     = time.DateOnly
	// centuryYears turns 2026 into the "26" of "2025-26".
	centuryYears = 100
	// tenths rounds inches to one decimal place.
	tenths = 10
)

// Mode is which report the home page shows.
type Mode string

const (
	// ModeLive is the in-season board of upcoming snow.
	ModeLive Mode = "live"
	// ModeRecap is last season's stats, shown off season.
	ModeRecap Mode = "recap"
)

// ResortDay is one resort's snowfall on one UTC day.
type ResortDay struct {
	ResortName string
	Date       time.Time
	SnowfallIn float64
}

// DetermineMode reports live when any resort had a snow day in the last
// LookbackDays days or has one forecast from today on, and recap otherwise.
// No calendar is involved: an early storm in October turns the season on by
// itself.
func DetermineMode(days []ResortDay, today time.Time) Mode {
	today = dateOf(today)
	earliest := today.AddDate(0, 0, -LookbackDays)

	for _, day := range days {
		if day.SnowfallIn < SnowDayThresholdIn {
			continue
		}

		if !dateOf(day.Date).Before(earliest) {
			return ModeLive
		}
	}

	return ModeRecap
}

// Day is one day of a resort's live forecast. SnowIn is nil when no forecast
// is stored for that day.
type Day struct {
	Date   string   `json:"date"`
	SnowIn *float64 `json:"snow_in"`
}

// LiveResort is one row of the live board.
type LiveResort struct {
	Name    string  `json:"name"`
	TotalIn float64 `json:"total_in"`
	// Through is the last day with at least ThroughMinIn of snow, for copy
	// such as "18″ by Thursday". Nil when no day reaches it.
	Through *string `json:"through"`
	Daily   []Day   `json:"daily"`
}

// BuildLive totals each resort's forecast from today through ForecastDays
// days and sorts the most snow first.
func BuildLive(days []ResortDay, today time.Time) []LiveResort {
	today = dateOf(today)
	end := today.AddDate(0, 0, ForecastDays)

	byResort := map[string]map[string]float64{}

	for _, day := range days {
		date := dateOf(day.Date)
		if byResort[day.ResortName] == nil {
			byResort[day.ResortName] = map[string]float64{}
		}

		if date.Before(today) || !date.Before(end) {
			continue
		}

		byResort[day.ResortName][date.Format(dateLayout)] = day.SnowfallIn
	}

	resorts := make([]LiveResort, 0, len(byResort))

	for name, snowByDate := range byResort {
		row := LiveResort{Name: name, Daily: make([]Day, 0, ForecastDays)}
		total := 0.0

		for offset := range ForecastDays {
			date := today.AddDate(0, 0, offset).Format(dateLayout)
			day := Day{Date: date}

			if snow, ok := snowByDate[date]; ok {
				rounded := round1(snow)
				day.SnowIn = &rounded
				total += snow

				if snow >= ThroughMinIn {
					row.Through = &date
				}
			}

			row.Daily = append(row.Daily, day)
		}

		row.TotalIn = round1(total)
		resorts = append(resorts, row)
	}

	sort.Slice(resorts, func(i, j int) bool {
		if resorts[i].TotalIn != resorts[j].TotalIn {
			return resorts[i].TotalIn > resorts[j].TotalIn
		}

		return resorts[i].Name < resorts[j].Name
	})

	return resorts
}

// Season is one ski season, November 1 through April 30.
type Season struct {
	Name  string
	Start time.Time
	End   time.Time
}

// LastCompletedSeason is the most recent season whose archive data is
// complete as of now. On 2026-11-08 that is 2025-26.
func LastCompletedSeason(now time.Time) Season {
	endYear := now.UTC().Year()

	cutoff := time.Date(endYear, time.April, 30+archiveLagDays+1, 0, 0, 0, 0, time.UTC)
	if now.UTC().Before(cutoff) {
		endYear--
	}

	return Season{
		Name:  fmt.Sprintf("%d-%02d", endYear-1, endYear%centuryYears),
		Start: time.Date(endYear-1, time.November, 1, 0, 0, 0, 0, time.UTC),
		End:   time.Date(endYear, time.April, 30, 0, 0, 0, 0, time.UTC),
	}
}

// SeasonStats summarizes one resort's season.
type SeasonStats struct {
	TotalIn        float64
	DeepDays       int
	BiggestStormIn float64
	// StormStart and StormEnd are zero when the season had no snow.
	StormStart time.Time
	StormEnd   time.Time
}

// ComputeSeasonStats totals a season's daily snowfall, counts days of at
// least DeepDayIn, and finds the most snow over StormWindowDays consecutive
// days. Ties go to the earlier storm.
func ComputeSeasonStats(days []weather.DailySnowfall) SeasonStats {
	sorted := append([]weather.DailySnowfall(nil), days...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Date.Before(sorted[j].Date) })

	var stats SeasonStats

	for i, day := range sorted {
		stats.TotalIn += day.SnowfallIn
		if day.SnowfallIn >= DeepDayIn {
			stats.DeepDays++
		}

		windowEnd := dateOf(day.Date).AddDate(0, 0, StormWindowDays)
		storm := 0.0

		// The storm's dates are its first and last snowy days, so a window
		// that opens or closes on a dry day does not say so.
		var first, last time.Time

		for _, next := range sorted[i:] {
			if !dateOf(next.Date).Before(windowEnd) {
				break
			}

			if next.SnowfallIn > 0 {
				storm += next.SnowfallIn
				if first.IsZero() {
					first = next.Date
				}
				last = next.Date
			}
		}

		if storm > stats.BiggestStormIn {
			stats.BiggestStormIn = storm
			stats.StormStart = dateOf(first)
			stats.StormEnd = dateOf(last)
		}
	}

	stats.TotalIn = round1(stats.TotalIn)
	stats.BiggestStormIn = round1(stats.BiggestStormIn)

	return stats
}

// ReportStore reads what the forecaster stored. *dbgen.Queries satisfies it.
type ReportStore interface {
	ListDailySnowfallSince(ctx context.Context, date time.Time) ([]dbgen.ListDailySnowfallSinceRow, error)
	ListSeasonStats(ctx context.Context, season string) ([]dbgen.ListSeasonStatsRow, error)
}

// Report is the /api/snow-report response. Only the block for Mode is set.
type Report struct {
	Mode      Mode       `json:"mode"`
	UpdatedAt *time.Time `json:"updated_at"`
	Live      *Live      `json:"live"`
	Recap     *Recap     `json:"recap"`
}

// Live is the in-season board.
type Live struct {
	Days    int          `json:"days"`
	Resorts []LiveResort `json:"resorts"`
}

// Recap is last season's stats.
type Recap struct {
	Season  string        `json:"season"`
	Start   string        `json:"start"`
	End     string        `json:"end"`
	Resorts []RecapResort `json:"resorts"`
}

// RecapResort is one row of the recap.
type RecapResort struct {
	Name              string  `json:"name"`
	TotalIn           float64 `json:"total_in"`
	DeepDays          int32   `json:"days_6in_plus"`
	BiggestStormIn    float64 `json:"biggest_storm_in"`
	BiggestStormStart *string `json:"biggest_storm_start"`
	BiggestStormEnd   *string `json:"biggest_storm_end"`
}

// Build reads the stored snowfall and assembles the report for now.
func Build(ctx context.Context, store ReportStore, now time.Time) (Report, error) {
	today := dateOf(now)

	rows, err := store.ListDailySnowfallSince(ctx, today.AddDate(0, 0, -LookbackDays))
	if err != nil {
		return Report{}, fmt.Errorf("reading daily snowfall: %w", err)
	}

	days := make([]ResortDay, 0, len(rows))

	var updated *time.Time

	for _, row := range rows {
		days = append(days, ResortDay{ResortName: row.ResortName, Date: row.Date, SnowfallIn: row.SnowfallIn})

		if updated == nil || row.FetchedAt.After(*updated) {
			fetched := row.FetchedAt.UTC()
			updated = &fetched
		}
	}

	report := Report{Mode: DetermineMode(days, today), UpdatedAt: updated}

	if report.Mode == ModeLive {
		report.Live = &Live{Days: ForecastDays, Resorts: BuildLive(days, today)}

		return report, nil
	}

	season := LastCompletedSeason(now)

	stats, err := store.ListSeasonStats(ctx, season.Name)
	if err != nil {
		return Report{}, fmt.Errorf("reading season stats: %w", err)
	}

	report.Recap = &Recap{
		Season:  season.Name,
		Start:   season.Start.Format(dateLayout),
		End:     season.End.Format(dateLayout),
		Resorts: make([]RecapResort, 0, len(stats)),
	}

	for _, stat := range stats {
		report.Recap.Resorts = append(report.Recap.Resorts, RecapResort{
			Name:              stat.ResortName,
			TotalIn:           round1(stat.TotalSnowfallIn),
			DeepDays:          stat.Days6inPlus,
			BiggestStormIn:    round1(stat.BiggestStormIn),
			BiggestStormStart: nullDate(stat.BiggestStormStart.Time, stat.BiggestStormStart.Valid),
			BiggestStormEnd:   nullDate(stat.BiggestStormEnd.Time, stat.BiggestStormEnd.Valid),
		})
	}

	return report, nil
}

func nullDate(t time.Time, valid bool) *string {
	if !valid {
		return nil
	}

	formatted := t.Format(dateLayout)

	return &formatted
}

// dateOf reduces a time to its UTC calendar day, matching how forecast dates
// are stored.
func dateOf(t time.Time) time.Time {
	t = t.UTC()

	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func round1(value float64) float64 {
	return math.Round(value*tenths) / tenths
}
