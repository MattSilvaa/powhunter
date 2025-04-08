package weather

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetPoints(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check the request
		if r.URL.Path != "/points/45.3300,-121.6700" {
			t.Errorf("Expected URL path %q, got %q", "/points/45.3300,-121.6700", r.URL.Path)
		}

		// Check headers
		if r.Header.Get("User-Agent") == "" {
			t.Error("Expected User-Agent header to be set")
		}

		// Return a mock response
		w.Header().Set("Content-Type", "application/json")
		mockResponse := PointsResponse{}
		mockResponse.Properties.GridID = "PQR"
		mockResponse.Properties.GridX = 86
		mockResponse.Properties.GridY = 70
		mockResponse.Properties.ForecastURL = "https://api.weather.gov/gridpoints/PQR/86,70/forecast"
		mockResponse.Properties.RelativeLocation.Properties.City = "Mount Hood"
		mockResponse.Properties.RelativeLocation.Properties.State = "OR"

		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	// Create a client that uses the mock server
	client := &WeatherGovClient{
		client:  server.Client(),
		baseURL: server.URL,
	}

	// Test the function
	ctx := context.Background()
	response, err := client.GetPoints(ctx, 45.33, -121.67)

	// Check for errors
	if err != nil {
		t.Fatalf("GetPoints returned an error: %v", err)
	}

	// Check the response
	if response.Properties.GridID != "PQR" {
		t.Errorf("Expected GridID %q, got %q", "PQR", response.Properties.GridID)
	}
	if response.Properties.GridX != 86 {
		t.Errorf("Expected GridX %d, got %d", 86, response.Properties.GridX)
	}
	if response.Properties.GridY != 70 {
		t.Errorf("Expected GridY %d, got %d", 70, response.Properties.GridY)
	}
}

func TestGetGridpointForecast(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check the request
		if r.URL.Path != "/gridpoints/PQR/86,70/forecast" {
			t.Errorf("Expected URL path %q, got %q", "/gridpoints/PQR/86,70/forecast", r.URL.Path)
		}

		// Return a mock response
		w.Header().Set("Content-Type", "application/json")
		mockResponse := GridpointForecastResponse{}
		mockResponse.Properties.Updated = "2023-04-07T18:00:00+00:00"
		mockResponse.Properties.SnowfallAmount.UnitCode = "wmoUnit:mm"
		mockResponse.Properties.SnowfallAmount.Values = []struct {
			ValidTime string  `json:"validTime"`
			Value     float64 `json:"value"`
		}{
			{
				ValidTime: "2023-04-08T06:00:00+00:00/PT6H",
				Value:     25.4, // 1 inch in mm
			},
			{
				ValidTime: "2023-04-09T06:00:00+00:00/PT6H",
				Value:     50.8, // 2 inches in mm
			},
		}

		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	// Create a client that uses the mock server
	client := &WeatherGovClient{
		client:  server.Client(),
		baseURL: server.URL,
	}

	// Test the function
	ctx := context.Background()
	response, err := client.GetGridpointForecast(ctx, "PQR", 86, 70)

	// Check for errors
	if err != nil {
		t.Fatalf("GetGridpointForecast returned an error: %v", err)
	}

	// Check the response
	if len(response.Properties.SnowfallAmount.Values) != 2 {
		t.Errorf("Expected 2 snowfall values, got %d", len(response.Properties.SnowfallAmount.Values))
	}
	if response.Properties.SnowfallAmount.Values[0].Value != 25.4 {
		t.Errorf("Expected snowfall value 25.4, got %f", response.Properties.SnowfallAmount.Values[0].Value)
	}
}

func TestParseSnowfallAmount(t *testing.T) {
	// Create a test forecast response
	forecast := &GridpointForecastResponse{}
	forecast.Properties.SnowfallAmount.UnitCode = "wmoUnit:mm"
	forecast.Properties.SnowfallAmount.Values = []struct {
		ValidTime string  `json:"validTime"`
		Value     float64 `json:"value"`
	}{
		{
			ValidTime: "2023-04-08T06:00:00+00:00/PT6H",
			Value:     25.4, // 1 inch in mm
		},
		{
			ValidTime: "2023-04-08T12:00:00+00:00/PT6H",
			Value:     25.4, // another 1 inch for the same day
		},
		{
			ValidTime: "2023-04-09T06:00:00+00:00/PT6H",
			Value:     50.8, // 2 inches in mm
		},
	}

	// Parse the snowfall amounts
	predictions, err := ParseSnowfallAmount(forecast)
	if err != nil {
		t.Fatalf("ParseSnowfallAmount returned an error: %v", err)
	}

	// Check the predictions
	if len(predictions) != 2 {
		t.Fatalf("Expected 2 day predictions, got %d", len(predictions))
	}

	// Check April 8 prediction (should combine two 1-inch periods)
	april8 := time.Date(2023, 4, 8, 0, 0, 0, 0, time.UTC)
	found := false
	for _, pred := range predictions {
		if pred.Date.Equal(april8) {
			found = true
			if pred.SnowAmount != 2.0 {
				t.Errorf("Expected April 8 snowfall to be 2.0 inches, got %f", pred.SnowAmount)
			}
		}
	}
	if !found {
		t.Error("April 8 prediction not found")
	}

	// Check April 9 prediction
	april9 := time.Date(2023, 4, 9, 0, 0, 0, 0, time.UTC)
	found = false
	for _, pred := range predictions {
		if pred.Date.Equal(april9) {
			found = true
			if pred.SnowAmount != 2.0 {
				t.Errorf("Expected April 9 snowfall to be 2.0 inches, got %f", pred.SnowAmount)
			}
		}
	}
	if !found {
		t.Error("April 9 prediction not found")
	}
}

func TestGetSnowForecast(t *testing.T) {
	// Save the original functions and restore them at the end of the test
	origGetPoints := getPoints
	origGetGridpointForecast := getGridpointForecast
	defer func() {
		getPoints = origGetPoints
		getGridpointForecast = origGetGridpointForecast
	}()

	// Override the functions with test implementations
	getPoints = func(ctx context.Context, c *WeatherGovClient, lat, lon float64) (*PointsResponse, error) {
		resp := &PointsResponse{}
		resp.Properties.GridID = "TEST"
		resp.Properties.GridX = 1
		resp.Properties.GridY = 2
		return resp, nil
	}

	getGridpointForecast = func(ctx context.Context, c *WeatherGovClient, gridID string, gridX, gridY int) (*GridpointForecastResponse, error) {
		resp := &GridpointForecastResponse{}
		resp.Properties.SnowfallAmount.UnitCode = "wmoUnit:mm"
		resp.Properties.SnowfallAmount.Values = []struct {
			ValidTime string  `json:"validTime"`
			Value     float64 `json:"value"`
		}{
			{
				ValidTime: "2023-04-08T06:00:00+00:00/PT6H",
				Value:     25.4, // 1 inch in mm
			},
		}
		return resp, nil
	}

	// Create a client
	client := NewWeatherGovClient()

	// Test the function
	ctx := context.Background()
	predictions, err := client.GetSnowForecast(ctx, 45.33, -121.67)

	// Check for errors
	if err != nil {
		t.Fatalf("GetSnowForecast returned an error: %v", err)
	}

	// Check the predictions
	if len(predictions) != 1 {
		t.Fatalf("Expected 1 day prediction, got %d", len(predictions))
	}

	april8 := time.Date(2023, 4, 8, 0, 0, 0, 0, time.UTC)
	if !predictions[0].Date.Equal(april8) {
		t.Errorf("Expected prediction for April 8, got %s", predictions[0].Date.Format("2006-01-02"))
	}

	if predictions[0].SnowAmount != 1.0 {
		t.Errorf("Expected April 8 snowfall to be 1.0 inches, got %f", predictions[0].SnowAmount)
	}
}

func TestSplitTimeRange(t *testing.T) {
	testCases := []struct {
		input    string
		expected []string
	}{
		{
			input:    "2023-04-08T06:00:00+00:00/PT6H",
			expected: []string{"2023-04-08T06:00:00+00:00", "PT6H"},
		},
		{
			input:    "2023-04-08T06:00:00+00:00",
			expected: []string{"2023-04-08T06:00:00+00:00"},
		},
		{
			input:    "",
			expected: []string{""},
		},
	}

	for _, tc := range testCases {
		result := splitTimeRange(tc.input)
		if len(result) != len(tc.expected) {
			t.Errorf("splitTimeRange(%q) returned %d parts, expected %d", tc.input, len(result), len(tc.expected))
			continue
		}

		for i, v := range result {
			if v != tc.expected[i] {
				t.Errorf("splitTimeRange(%q) part %d is %q, expected %q", tc.input, i, v, tc.expected[i])
			}
		}
	}
}