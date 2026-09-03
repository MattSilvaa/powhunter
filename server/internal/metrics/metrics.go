// Package metrics exposes Prometheus instrumentation for the API and the
// forecaster.
package metrics

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Namespace prefixes every metric this application publishes.
const Namespace = "powhunter"

// durationBuckets are the latency buckets for request duration histograms.
func durationBuckets() []float64 {
	return []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}
}

// Registry holds the API's collectors.
type Registry struct {
	registry        *prometheus.Registry
	requestsTotal   *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
	inFlight        prometheus.Gauge
}

// NewRegistry builds a registry with the Go runtime and process collectors.
func NewRegistry() *Registry {
	reg := prometheus.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	m := &Registry{
		registry: reg,
		requestsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "Total HTTP requests by route, method and status.",
		}, []string{"route", "method", "status"}),
		requestDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: Namespace,
			Subsystem: "http",
			Name:      "request_duration_seconds",
			Help:      "HTTP request duration by route and method.",
			Buckets:   durationBuckets(),
		}, []string{"route", "method"}),
		inFlight: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: "http",
			Name:      "requests_in_flight",
			Help:      "HTTP requests currently being served.",
		}),
	}

	reg.MustRegister(m.requestsTotal, m.requestDuration, m.inFlight)

	return m
}

// RegisterDBStats publishes connection pool statistics, so pool exhaustion is
// visible rather than showing up only as slow requests.
func (m *Registry) RegisterDBStats(database *sql.DB) {
	m.registry.MustRegister(collectors.NewDBStatsCollector(database, "powhunter"))
}

// Handler serves the metrics endpoint.
func (m *Registry) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

// Registerer exposes the underlying registry for additional collectors.
func (m *Registry) Registerer() prometheus.Registerer {
	return m.registry
}

// statusRecorder captures the status code for metric labelling.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(status int) {
	if s.status == 0 {
		s.status = status
		s.ResponseWriter.WriteHeader(status)
	}
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}

	n, err := s.ResponseWriter.Write(b)

	return n, err //nolint:wrapcheck // pass through the underlying writer's error
}

// Instrument records metrics for one route. The route label is supplied rather
// than taken from the URL so that path values cannot create unbounded label
// cardinality.
func (m *Registry) Instrument(route string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.inFlight.Inc()
		defer m.inFlight.Dec()

		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w}

		next.ServeHTTP(recorder, r)

		status := recorder.status
		if status == 0 {
			status = http.StatusOK
		}

		m.requestDuration.WithLabelValues(route, r.Method).Observe(time.Since(start).Seconds())
		m.requestsTotal.WithLabelValues(route, r.Method, strconv.Itoa(status)).Inc()
	})
}
