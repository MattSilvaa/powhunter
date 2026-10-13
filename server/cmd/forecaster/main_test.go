package main

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MattSilvaa/powhunter/internal/metrics"
)

// publishMetrics used to return silently when PUSHGATEWAY_URL was unset, which
// made a misconfigured cron look identical to a working one in the logs. These
// tests pin the fact that every outcome says something.
func TestPublishMetricsLogsWhenGatewayIsUnset(t *testing.T) {
	t.Setenv("PUSHGATEWAY_URL", "")

	var out bytes.Buffer
	publishMetrics(newTestLogger(&out), metrics.ForecastRun{Succeeded: true})

	if !strings.Contains(out.String(), "PUSHGATEWAY_URL is not set") {
		t.Errorf("expected a warning that the gateway is unset, got: %s", out.String())
	}
}

func TestPublishMetricsLogsSuccessfulPush(t *testing.T) {
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer gateway.Close()

	t.Setenv("PUSHGATEWAY_URL", gateway.URL)

	var out bytes.Buffer
	publishMetrics(newTestLogger(&out), metrics.ForecastRun{Succeeded: true})

	if !strings.Contains(out.String(), "published forecaster metrics") {
		t.Errorf("expected a success log line, got: %s", out.String())
	}
}

func TestPublishMetricsLogsFailedPush(t *testing.T) {
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer gateway.Close()

	t.Setenv("PUSHGATEWAY_URL", gateway.URL)

	var out bytes.Buffer
	publishMetrics(newTestLogger(&out), metrics.ForecastRun{Succeeded: true})

	if !strings.Contains(out.String(), "failed to publish forecaster metrics") {
		t.Errorf("expected a failure log line, got: %s", out.String())
	}
}

func newTestLogger(out *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewTextHandler(out, nil))
}

func TestLinkBaseURL(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{"", ""},
		{"https://api.example.com/", "https://api.example.com"},
		{"http://localhost:8080", "http://localhost:8080"},
		{"api.example.com", ""},
		{"javascript:alert(1)", ""},
		{"https://api.example.com/?ref=sms", ""},
	}

	for _, tt := range tests {
		var out bytes.Buffer
		if got := linkBaseURL(newTestLogger(&out), tt.raw); got != tt.want {
			t.Errorf("linkBaseURL(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}
