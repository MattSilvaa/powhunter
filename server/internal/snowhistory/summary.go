// Package snowhistory summarises how much snow a resort has had in past
// seasons, so people can see what a normal winter there looks like and pick an
// alert threshold that fits.
package snowhistory

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/MattSilvaa/powhunter/internal/weather"
)

// CompleteSeasons is how many finished seasons the summary averages over.
const CompleteSeasons = 5

// Every resort in the catalog is in North America, where the ski season runs
// from October through May. Snow outside these months is left out.
const (
	seasonStartMonth = time.October
	seasonEndMonth   = time.May
	seasonMonths     = 8
	monthsInYear     = 12

	// precision keeps one decimal place, which is all a snowfall reading
	// warrants.
	precision = 10
)

// Season is one winter's snowfall.
type Season struct {
	Label       string  `json:"label"`
	StartYear   int     `json:"startYear"`
	Partial     bool    `json:"partial"`
	TotalInches float64 `json:"totalInches"`
	// DaysAtOrAbove counts days with at least the requested threshold. It is nil
	// when no threshold was asked for.
	DaysAtOrAbove *int `json:"daysAtOrAbove"`
}

// MonthAverage is the average snowfall in one calendar month, across the
// complete seasons.
type MonthAverage struct {
	Month  time.Month `json:"month"`
	Inches float64    `json:"inches"`
}

// Summary is a resort's snowfall history, newest season first.
type Summary struct {
	Seasons              []Season       `json:"seasons"`
	MonthlyAverages      []MonthAverage `json:"monthlyAverages"`
	AverageSeasonInches  float64        `json:"averageSeasonInches"`
	AverageDaysAtOrAbove *float64       `json:"averageDaysAtOrAbove"`
}

// seasonStartYear returns the year a date's season began in, and false when
// the date falls in the off-season.
func seasonStartYear(date time.Time) (int, bool) {
	switch {
	case date.Month() >= seasonStartMonth:
		return date.Year(), true
	case date.Month() <= seasonEndMonth:
		return date.Year() - 1, true
	default:
		return 0, false
	}
}

// latestSeason returns the start year of the most recent season to have begun
// by now, and whether it is still under way.
func latestSeason(now time.Time) (int, bool) {
	if now.Month() >= seasonStartMonth {
		return now.Year(), true
	}

	return now.Year() - 1, now.Month() <= seasonEndMonth
}

// Window returns the date range a summary at now needs: the last
// CompleteSeasons finished seasons plus the current one so far.
func Window(now time.Time) (time.Time, time.Time) {
	latest, partial := latestSeason(now)

	first := latest - CompleteSeasons + 1
	if partial {
		first = latest - CompleteSeasons
	}

	start := time.Date(first, seasonStartMonth, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(latest+1, seasonEndMonth, 31, 0, 0, 0, 0, time.UTC)

	// The archive has nothing for today or later.
	yesterday := time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, time.UTC)
	if end.After(yesterday) {
		end = yesterday
	}

	return start, end
}

// Summarize buckets daily snowfall into seasons and months. A threshold of zero
// leaves the day counts out.
func Summarize(days []weather.DailySnowfall, threshold float64, now time.Time) Summary {
	start, _ := Window(now)
	latest, latestPartial := latestSeason(now)
	first := start.Year()

	totals := make(map[int]float64)
	bigDays := make(map[int]int)
	monthTotals := make(map[time.Month]float64)

	for _, day := range days {
		year, inSeason := seasonStartYear(day.Date)
		if !inSeason || year < first || year > latest {
			continue
		}

		totals[year] += day.Inches
		if threshold > 0 && day.Inches >= threshold {
			bigDays[year]++
		}

		if year != latest || !latestPartial {
			monthTotals[day.Date.Month()] += day.Inches
		}
	}

	summary := Summary{
		Seasons:         make([]Season, 0, latest-first+1),
		MonthlyAverages: make([]MonthAverage, 0, seasonMonths),
	}

	var completeTotal float64
	var completeBigDays int
	for year := latest; year >= first; year-- {
		partial := year == latest && latestPartial
		season := Season{
			Label:       fmt.Sprintf("%d–%s", year, strconv.Itoa(year + 1)[2:]),
			StartYear:   year,
			Partial:     partial,
			TotalInches: round(totals[year]),
		}

		if threshold > 0 {
			count := bigDays[year]
			season.DaysAtOrAbove = &count
		}

		if !partial {
			completeTotal += totals[year]
			completeBigDays += bigDays[year]
		}

		summary.Seasons = append(summary.Seasons, season)
	}

	for month := seasonStartMonth; ; month = month%monthsInYear + 1 {
		summary.MonthlyAverages = append(summary.MonthlyAverages, MonthAverage{
			Month:  month,
			Inches: round(monthTotals[month] / CompleteSeasons),
		})

		if month == seasonEndMonth {
			break
		}
	}

	summary.AverageSeasonInches = round(completeTotal / CompleteSeasons)
	if threshold > 0 {
		average := round(float64(completeBigDays) / CompleteSeasons)
		summary.AverageDaysAtOrAbove = &average
	}

	return summary
}

// round rounds a reading to the summary's precision.
func round(value float64) float64 {
	return math.Round(value*precision) / precision
}
