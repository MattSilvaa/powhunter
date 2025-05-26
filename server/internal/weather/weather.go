package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

//go:generate mockgen -destination=mocks/mock_weather.go -package=mocks github.com/MattSilvaa/powhunter/internal/weather WeatherService

// WeatherService defines the interface for weather service operations
type WeatherService interface {
	// GetSnowForecast gets the snow forecast for a location
	GetSnowForecast(ctx context.Context, lat, lon float64) ([]WeatherPrediction, error)
}

// OpenMeteoClient provides access to the Open-Meteo API
type OpenMeteoClient struct {
	client  *http.Client
	baseURL string
}

// NewOpenMeteoClient creates a new Open-Meteo API client
func NewOpenMeteoClient() *OpenMeteoClient {
	return &OpenMeteoClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "https://api.open-meteo.com/v1",
	}
}

// CustomTime handles Open-Meteo's time format "2006-01-02T15:04"
type OpenMeteoTime struct {
	time.Time
}

// UnmarshalJSON implements custom JSON unmarshaling for Open-Meteo's time format
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

// MarshalJSON implements JSON marshaling
func (ct OpenMeteoTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(ct.Time.Format("2006-01-02T15:04:05"))
}

// OpenMeteoResponse represents the response from the Open-Meteo API
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
}

// WeatherPrediction represents a predicted snowfall and temperature for a specific date
type WeatherPrediction struct {
	Date           time.Time
	SnowAmount     float64 // in inches
	AvgTemperature float64 // in fahrenheit
	MinTemperature float64 // in fahrenheit
	MaxTemperature float64 // in fahrenheit
}

func (c *OpenMeteoClient) GetForecast(ctx context.Context, lat, lon float64) (*OpenMeteoResponse, error) {
	url := fmt.Sprintf("%s/forecast?latitude=%.6f&longitude=%.6f&current=temperature_2m,snowfall&hourly=snowfall,temperature_2m&temperature_unit=fahrenheit&precipitation_unit=inch&temporal_resolution=hourly_6", c.baseURL, lat, lon)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("User-Agent", "Powhunter/1.0 (Language=Go 1.24)")
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error getting forecast data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error from Open-Meteo API: %s", resp.Status)
	}

	var forecastResp OpenMeteoResponse
	if err := json.NewDecoder(resp.Body).Decode(&forecastResp); err != nil {
		return nil, fmt.Errorf("error decoding forecast response: %w", err)
	}

	return &forecastResp, nil
}

func (c *OpenMeteoClient) GetSnowForecast(ctx context.Context, lat, lon float64) ([]WeatherPrediction, error) {
	forecast, err := c.GetForecast(ctx, lat, lon)
	if err != nil {
		return nil, fmt.Errorf("error getting forecast: %w", err)
	}

	return ParseWeatherData(forecast), nil
}

// ParseWeatherData parses the snowfall and temperature data from the response
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
