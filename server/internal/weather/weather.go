package weather

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

//go:generate mockgen -destination=mocks/mock_weather.go -package=mocks github.com/MattSilvaa/powhunter/internal/weather WeatherService,ArchiveService

// WeatherService defines the interface for weather service operations.
type WeatherService interface {
	// GetSnowForecast gets the snow forecast for a location: per-day
	// predictions for alerts, and daily snowfall totals for the snow report
	// covering the past PastDays days and the next ForecastDays.
	GetSnowForecast(ctx context.Context, lat, lon float64) (*Forecast, error)
}

// ArchiveService reads past snowfall from Open-Meteo's historical archive.
type ArchiveService interface {
	// GetSeasonSnowfall returns daily snowfall from start to end, inclusive.
	GetSeasonSnowfall(ctx context.Context, lat, lon float64, start, end time.Time) ([]DailySnowfall, error)
}

const (
	// PastDays is how many days of past snowfall each forecast request also
	// returns. The snow report looks back three weeks to decide whether it is
	// still the season, and asking for them up front means that works from the
	// first run rather than after three weeks of stored forecasts.
	PastDays = 21
	// ForecastDays is how far ahead forecasts reach. Signup offers up to ten
	// days of notice, so anything shorter could never honour it.
	ForecastDays = 10
)

// DailySnowfall is one day's snowfall total in inches. Dates are UTC days.
type DailySnowfall struct {
	Date       time.Time
	SnowfallIn float64
}

// Forecast is everything one forecast request returns.
type Forecast struct {
	// Predictions are days with snow, from hourly data, for alert matching.
	// They can include past days, so callers must ignore those.
	Predictions []WeatherPrediction
	// Daily is every day's total, past and future, including dry days.
	Daily []DailySnowfall
}

// ErrRateLimited reports that Open-Meteo rejected a request with 429 Too Many
// Requests. Its free tier limits requests per IP, and Render's shared egress
// IPs mean other tenants can exhaust that quota, so callers should back off for
// longer than they would for other transient errors.
var ErrRateLimited = errors.New("rate limited by Open-Meteo")

// RateLimitError is returned for a 429 response. RetryAfter is the wait the
// API asked for, or zero if it sent no usable Retry-After header.
type RateLimitError struct {
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	return "error from Open-Meteo API: 429 Too Many Requests"
}

// Is lets callers match any RateLimitError with errors.Is(err, ErrRateLimited).
func (e *RateLimitError) Is(target error) bool {
	return target == ErrRateLimited
}

// parseRetryAfter reads a Retry-After header given in seconds. The HTTP-date
// form is not used by Open-Meteo, so it is treated as absent.
func parseRetryAfter(value string) time.Duration {
	seconds, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || seconds <= 0 {
		return 0
	}

	return time.Duration(seconds) * time.Second
}

// OpenMeteoClient provides access to the Open-Meteo API.
type OpenMeteoClient struct {
	client         *http.Client
	baseURL        string
	archiveBaseURL string
}

// NewOpenMeteoClient creates a new Open-Meteo API client.
func NewOpenMeteoClient() *OpenMeteoClient {
	return &OpenMeteoClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL:        "https://api.open-meteo.com/v1",
		archiveBaseURL: "https://archive-api.open-meteo.com/v1",
	}
}

// CustomTime handles Open-Meteo's time format "2006-01-02T15:04".
type OpenMeteoTime struct {
	time.Time
}

// UnmarshalJSON implements custom JSON unmarshaling for Open-Meteo's time format.
func (ct *OpenMeteoTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "null" || s == "" {
		return nil
	}

	// Try parsing with seconds first (in case format changes)
	t, err := time.Parse("2006-01-02T15:04:05", s)
	if err != nil {
		// Fall back to format without seconds
		t, err = time.Parse("2006-01-02T15:04", s)
		if err != nil {
			return fmt.Errorf("parsing time %q: %w", s, err)
		}
	}

	ct.Time = t
	return nil
}

// MarshalJSON implements JSON marshaling.
func (ct OpenMeteoTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(ct.Time.Format("2006-01-02T15:04:05"))
}

// OpenMeteoResponse represents the response from the Open-Meteo API.
type OpenMeteoResponse struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Elevation float64 `json:"elevation"`
	Current   struct {
		Time        OpenMeteoTime `json:"time"`
		Temperature float64       `json:"temperature_2m"`
		Snowfall    float64       `json:"snowfall"`
	} `json:"current"`
	Hourly struct {
		Time        []OpenMeteoTime `json:"time"`
		Temperature []float64       `json:"temperature_2m"`
		Snowfall    []float64       `json:"snowfall"`
	} `json:"hourly"`
	Daily dailyResponse `json:"daily"`
}

// dailyResponse is the daily block. A value is null when the model has no
// data for that day, which is read as no snow.
type dailyResponse struct {
	Time        []string   `json:"time"`
	SnowfallSum []*float64 `json:"snowfall_sum"`
}

// WeatherPrediction represents a predicted snowfall and temperature for a specific date.
type WeatherPrediction struct {
	Date           time.Time
	SnowAmount     float64 // in inches
	AvgTemperature float64 // in fahrenheit
	MinTemperature float64 // in fahrenheit
	MaxTemperature float64 // in fahrenheit
}

func (c *OpenMeteoClient) GetForecast(ctx context.Context, lat, lon float64) (*OpenMeteoResponse, error) {
	url := fmt.Sprintf(
		"%s/forecast?latitude=%.6f&longitude=%.6f&current=temperature_2m,snowfall&hourly=snowfall,temperature_2m"+
			"&daily=snowfall_sum&past_days=%d&forecast_days=%d"+
			"&temperature_unit=fahrenheit&precipitation_unit=inch&temporal_resolution=hourly_6&timezone=Etc/UTC",
		c.baseURL,
		lat,
		lon,
		PastDays,
		ForecastDays,
	)

	var forecastResp OpenMeteoResponse
	if err := c.getJSON(ctx, url, &forecastResp); err != nil {
		return nil, err
	}

	return &forecastResp, nil
}

// GetSeasonSnowfall reads daily snowfall from the historical archive. It is
// called once per resort per season, never per page view.
func (c *OpenMeteoClient) GetSeasonSnowfall(
	ctx context.Context,
	lat, lon float64,
	start, end time.Time,
) ([]DailySnowfall, error) {
	url := fmt.Sprintf(
		"%s/archive?latitude=%.6f&longitude=%.6f&start_date=%s&end_date=%s"+
			"&daily=snowfall_sum&precipitation_unit=inch&timezone=Etc/UTC",
		c.archiveBaseURL,
		lat,
		lon,
		start.Format(time.DateOnly),
		end.Format(time.DateOnly),
	)

	var archive struct {
		Daily dailyResponse `json:"daily"`
	}
	if err := c.getJSON(ctx, url, &archive); err != nil {
		return nil, fmt.Errorf("error getting archive: %w", err)
	}

	return parseDaily(archive.Daily), nil
}

// getJSON performs a GET against Open-Meteo and decodes the JSON body.
func (c *OpenMeteoClient) getJSON(ctx context.Context, url string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("User-Agent", "Powhunter/1.0 (Language=Go 1.24)")
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("error getting data from Open-Meteo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return &RateLimitError{RetryAfter: parseRetryAfter(resp.Header.Get("Retry-After"))}
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error from Open-Meteo API: %s", resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("error decoding Open-Meteo response: %w", err)
	}

	return nil
}

func (c *OpenMeteoClient) GetSnowForecast(ctx context.Context, lat, lon float64) (*Forecast, error) {
	forecast, err := c.GetForecast(ctx, lat, lon)
	if err != nil {
		return nil, fmt.Errorf("error getting forecast: %w", err)
	}

	return &Forecast{
		Predictions: ParseWeatherData(forecast),
		Daily:       parseDaily(forecast.Daily),
	}, nil
}

// parseDaily turns the daily block into totals. Days that fail to parse are
// skipped, and a null value counts as no snow.
func parseDaily(daily dailyResponse) []DailySnowfall {
	days := make([]DailySnowfall, 0, len(daily.Time))

	for i, raw := range daily.Time {
		date, err := time.Parse(time.DateOnly, raw)
		if err != nil {
			continue
		}

		snow := 0.0
		if i < len(daily.SnowfallSum) && daily.SnowfallSum[i] != nil && *daily.SnowfallSum[i] > 0 {
			snow = *daily.SnowfallSum[i]
		}

		days = append(days, DailySnowfall{Date: date, SnowfallIn: snow})
	}

	return days
}

// ParseWeatherData parses the snowfall and temperature data from the response.
func ParseWeatherData(forecast *OpenMeteoResponse) []WeatherPrediction {
	snowByDate := make(map[string]float64)
	tempSumByDate := make(map[string]float64)
	tempMinByDate := make(map[string]float64)
	tempMaxByDate := make(map[string]float64)
	countByDate := make(map[string]int)

	for i, timestamp := range forecast.Hourly.Time {
		if i >= len(forecast.Hourly.Snowfall) || i >= len(forecast.Hourly.Temperature) {
			fmt.Printf("mismatch between items in time and snowfall/temperature")
			break
		}
		dateStr := timestamp.Format("2006-01-02")
		snowByDate[dateStr] += forecast.Hourly.Snowfall[i]

		temp := forecast.Hourly.Temperature[i]

		if _, exists := countByDate[dateStr]; !exists {
			tempMinByDate[dateStr] = temp
			tempMaxByDate[dateStr] = temp
		} else {
			if temp < tempMinByDate[dateStr] {
				tempMinByDate[dateStr] = temp
			}
			if temp > tempMaxByDate[dateStr] {
				tempMaxByDate[dateStr] = temp
			}
		}

		tempSumByDate[dateStr] += temp
		countByDate[dateStr]++
	}
	var predictions []WeatherPrediction
	for dateStr, snowAmount := range snowByDate {
		if snowAmount <= 0 {
			continue
		}

		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}

		avgTemp := 0.0
		if count := countByDate[dateStr]; count > 0 {
			avgTemp = tempSumByDate[dateStr] / float64(count)
		}

		predictions = append(predictions, WeatherPrediction{
			Date:           date,
			SnowAmount:     snowAmount,
			AvgTemperature: avgTemp,
			MinTemperature: tempMinByDate[dateStr],
			MaxTemperature: tempMaxByDate[dateStr],
		})
	}

	return predictions
}
