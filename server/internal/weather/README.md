# Weather.gov Snow Forecast Integration

This package integrates with the Weather.gov API to fetch snow forecasts for ski resorts. It queries the National Weather Service (NWS) API to get accurate weather predictions for specific geographic locations.

## How It Works

The weather service:

1. Uses the resort's latitude and longitude to get the appropriate weather grid from the Weather.gov API
2. Fetches detailed forecast data from the grid endpoint
3. Parses the response to extract snowfall predictions
4. Converts measurements to inches if needed
5. Aggregates snow predictions by day for easy consumption

## Weather.gov API

This integration uses two primary Weather.gov API endpoints:

1. `/points/{lat},{lon}` - Gets the grid ID, X, and Y coordinates for a location
2. `/gridpoints/{gridId}/{gridX},{gridY}/forecast` - Gets the detailed forecast for a specific grid

The API returns detailed weather data including:
- Forecast periods with timestamps
- Snow accumulation predictions
- Probability of precipitation
- Temperature data

## Configuration

No API key is required for Weather.gov, but we include a User-Agent header to identify our application:

```
User-Agent: Powhunter/1.0 (powhunter@example.com)
```

## Usage Example

```go
// Create a new Weather.gov client
weatherClient := weather.NewWeatherGovClient()

// Get snow forecasts for a location
forecasts, err := weatherClient.GetSnowForecast(ctx, 46.9459, -121.5802)
if err != nil {
    log.Printf("Error getting forecast: %v", err)
    return
}

// Process each forecast
for _, forecast := range forecasts {
    fmt.Printf("Date: %s, Snow Amount: %.1f inches\n", 
        forecast.Date.Format("2006-01-02"), 
        forecast.SnowAmount)
}
```

## Error Handling

The client includes robust error handling for:
- Invalid coordinates
- Network issues
- API rate limiting
- Malformed responses

## Rate Limiting

Weather.gov API has rate limits. To respect these limits:
- Cache responses when possible
- Limit requests to what's necessary
- Use the scheduler to check forecasts at reasonable intervals (e.g., every 12 hours)