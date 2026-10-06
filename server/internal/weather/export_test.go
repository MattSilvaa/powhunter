package weather

import "time"

// NewClientForTest points a client at a fake server.
func NewClientForTest(baseURL string) *OpenMeteoClient {
	client := NewOpenMeteoClient()
	client.baseURL = baseURL

	return client
}

// ParseRetryAfter exposes parseRetryAfter to the external tests.
func ParseRetryAfter(value string) time.Duration {
	return parseRetryAfter(value)
}
