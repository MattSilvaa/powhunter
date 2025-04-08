package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

//go:generate mockgen -destination=mocks/mock_weather.go -package=mocks github.com/MattSilvaa/powhunter/internal/weather WeatherService

// WeatherService defines the interface for weather service operations
type WeatherService interface {
	// GetSnowForecast gets the snow forecast for a location
	GetSnowForecast(ctx context.Context, lat, lon float64) ([]SnowPrediction, error)
}

// WeatherGovClient provides access to the Weather.gov API
type WeatherGovClient struct {
	client  *http.Client
	baseURL string
}

// NewWeatherGovClient creates a new Weather.gov API client
func NewWeatherGovClient() *WeatherGovClient {
	return &WeatherGovClient{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: "https://api.weather.gov",
	}
}

// PointsResponse contains information about a weather location
type PointsResponse struct {
	Properties struct {
		GridID       string `json:"gridId"`
		GridX        int    `json:"gridX"`
		GridY        int    `json:"gridY"`
		ForecastURL  string `json:"forecast"`
		RelativeLocation struct {
			Properties struct {
				City     string `json:"city"`
				State    string `json:"state"`
			} `json:"properties"`
		} `json:"relativeLocation"`
	} `json:"properties"`
}

// ForecastResponse from the Weather.gov API
type ForecastResponse struct {
	Properties struct {
		Updated    string `json:"updated"`
		Periods    []ForecastPeriod `json:"periods"`
	} `json:"properties"`
}

// ForecastPeriod represents a single forecast period
type ForecastPeriod struct {
	Number           int       `json:"number"`
	Name             string    `json:"name"`
	StartTime        time.Time `json:"startTime"`
	EndTime          time.Time `json:"endTime"`
	DetailedForecast string    `json:"detailedForecast"`
	ShortForecast    string    `json:"shortForecast"`
	Temperature      int       `json:"temperature"`
	TemperatureUnit  string    `json:"temperatureUnit"`
	ProbabilityOfPrecipitation struct {
		Value int `json:"value"`
	} `json:"probabilityOfPrecipitation"`
}

// GridpointForecastResponse from the Weather.gov API's gridpoint endpoint
type GridpointForecastResponse struct {
	Properties struct {
		Updated    string `json:"updated"`
		Periods    []ForecastPeriod `json:"periods"`
		Elevation  struct {
			Value float64 `json:"value"`
			UnitCode string `json:"unitCode"`
		} `json:"elevation"`
		QuantitativePrecipitation struct {
			Values []struct {
				ValidTime string `json:"validTime"`
				Value     float64 `json:"value"`
			} `json:"values"`
		} `json:"quantitativePrecipitation"`
		SnowfallAmount struct {
			Values []struct {
				ValidTime string `json:"validTime"`
				Value     float64 `json:"value"`
			} `json:"values"`
			UnitCode string `json:"unitCode"`
		} `json:"snowfallAmount"`
		WinterWeather struct {
			Values []struct {
				ValidTime string `json:"validTime"`
				Value     int `json:"value"`
			} `json:"values"`
		} `json:"winterWeather"`
	} `json:"properties"`
}

// SnowPrediction represents a predicted snowfall for a specific date
type SnowPrediction struct {
	Date       time.Time
	SnowAmount float64 // in inches
}

// GetPoints gets the gridpoint data for a location
func (c *WeatherGovClient) GetPoints(ctx context.Context, lat, lon float64) (*PointsResponse, error) {
	url := fmt.Sprintf("%s/points/%.4f,%.4f", c.baseURL, lat, lon)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	
	req.Header.Set("User-Agent", "Powhunter/1.0 (powhunter@example.com)")
	req.Header.Set("Accept", "application/json")
	
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error getting points data: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error from weather.gov API: %s", resp.Status)
	}
	
	var pointsResp PointsResponse
	if err := json.NewDecoder(resp.Body).Decode(&pointsResp); err != nil {
		return nil, fmt.Errorf("error decoding points response: %w", err)
	}
	
	return &pointsResp, nil
}

// GetGridpointForecast gets detailed forecast data including snow predictions
func (c *WeatherGovClient) GetGridpointForecast(ctx context.Context, gridID string, gridX, gridY int) (*GridpointForecastResponse, error) {
	url := fmt.Sprintf("%s/gridpoints/%s/%d,%d/forecast", c.baseURL, gridID, gridX, gridY)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	
	req.Header.Set("User-Agent", "Powhunter/1.0 (powhunter@example.com)")
	req.Header.Set("Accept", "application/json")
	
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error getting forecast data: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error from weather.gov API: %s", resp.Status)
	}
	
	var forecastResp GridpointForecastResponse
	if err := json.NewDecoder(resp.Body).Decode(&forecastResp); err != nil {
		return nil, fmt.Errorf("error decoding forecast response: %w", err)
	}
	
	return &forecastResp, nil
}

// ParseSnowfallAmount parses the snowfall amount from the response
// Returns the amount in inches
func ParseSnowfallAmount(forecast *GridpointForecastResponse) ([]SnowPrediction, error) {
	var predictions []SnowPrediction
	
	// Check if snowfall data is available
	if len(forecast.Properties.SnowfallAmount.Values) == 0 {
		return predictions, nil
	}
	
	// Parse each snowfall prediction
	for _, val := range forecast.Properties.SnowfallAmount.Values {
		// Parse the time range like "2023-04-07T18:00:00+00:00/PT6H"
		timeStr := val.ValidTime
		parts := splitTimeRange(timeStr)
		if len(parts) != 2 {
			continue
		}
		
		startTime, err := time.Parse(time.RFC3339, parts[0])
		if err != nil {
			continue
		}
		
		// Convert from mm to inches if needed
		inches := val.Value
		if forecast.Properties.SnowfallAmount.UnitCode == "wmoUnit:mm" {
			inches = val.Value / 25.4 // Convert mm to inches
		}
		
		// Only add if there's actual snow predicted
		if inches > 0 {
			// Use the date portion only for predictions
			date := time.Date(startTime.Year(), startTime.Month(), startTime.Day(), 0, 0, 0, 0, time.UTC)
			
			// Check if we already have a prediction for this date
			found := false
			for i, p := range predictions {
				if p.Date.Equal(date) {
					// Add to existing prediction for this date
					predictions[i].SnowAmount += inches
					found = true
					break
				}
			}
			
			// If no prediction for this date exists, create a new one
			if !found {
				predictions = append(predictions, SnowPrediction{
					Date:       date,
					SnowAmount: inches,
				})
			}
		}
	}
	
	return predictions, nil
}

// Helper function to split time range string
func splitTimeRange(timeRange string) []string {
	// Implementation depends on the exact format returned by the API
	// This is a simplified version
	for i, c := range timeRange {
		if c == '/' {
			return []string{timeRange[:i], timeRange[i+1:]}
		}
	}
	return []string{timeRange}
}

// GetSnowForecast gets the snow forecast for a location
func (c *WeatherGovClient) GetSnowForecast(ctx context.Context, lat, lon float64) ([]SnowPrediction, error) {
	// Get the grid point for the location
	points, err := getPoints(ctx, c, lat, lon)
	if err != nil {
		return nil, fmt.Errorf("error getting grid points: %w", err)
	}
	
	// Get the detailed forecast
	forecast, err := getGridpointForecast(ctx, c, points.Properties.GridID, points.Properties.GridX, points.Properties.GridY)
	if err != nil {
		return nil, fmt.Errorf("error getting forecast: %w", err)
	}
	
	// Parse the snowfall amounts
	snowPredictions, err := ParseSnowfallAmount(forecast)
	if err != nil {
		return nil, fmt.Errorf("error parsing snowfall: %w", err)
	}
	
	return snowPredictions, nil
}

// Extracted these functions to allow for easier testing
var getPoints = func(ctx context.Context, c *WeatherGovClient, lat, lon float64) (*PointsResponse, error) {
	return c.GetPoints(ctx, lat, lon)
}

var getGridpointForecast = func(ctx context.Context, c *WeatherGovClient, gridID string, gridX, gridY int) (*GridpointForecastResponse, error) {
	return c.GetGridpointForecast(ctx, gridID, gridX, gridY)
}