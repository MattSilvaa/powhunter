// Package server wires the HTTP routes and middleware for the API.
package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/MattSilvaa/powhunter/internal/config"
	"github.com/MattSilvaa/powhunter/internal/handlers"
	"github.com/MattSilvaa/powhunter/internal/metrics"
	"github.com/MattSilvaa/powhunter/internal/middleware"
)

// readinessTimeout bounds the database check behind the readiness probe.
const readinessTimeout = 2 * time.Second

// Deps are the collaborators the router needs.
type Deps struct {
	Config   config.Config
	Handlers *handlers.Handlers
	Metrics  *metrics.Registry
	Logger   *slog.Logger
	DB       *sql.DB
}

// New builds the fully wired HTTP handler.
//
// Routes are registered with method patterns so the mux rejects the wrong verb,
// and every route is instrumented with a static label so path values cannot
// produce unbounded metric cardinality.
func New(deps Deps) (http.Handler, func()) {
	mux := http.NewServeMux()

	// Liveness stays deliberately cheap and dependency-free. Tying it to the
	// database would turn a brief database blip into a restart loop.
	mux.Handle("GET /health", deps.Metrics.Instrument("health", http.HandlerFunc(healthHandler)))
	mux.Handle("GET /ready", deps.Metrics.Instrument("ready", readyHandler(deps.DB)))
	mux.Handle("GET /metrics", deps.Metrics.Handler())

	writeLimiter := middleware.NewRateLimiter(middleware.RateLimitConfig{
		RequestsPerSecond: deps.Config.WriteRatePerSec,
		Burst:             deps.Config.WriteRateBurst,
	}, deps.Config.TrustProxyHeaders)

	// Writes get a tighter bucket than reads: creating alerts and sending
	// contact email both cost money downstream.
	write := func(handler http.HandlerFunc) http.Handler {
		return writeLimiter.Middleware(handler)
	}

	mux.Handle("GET /api/resorts",
		deps.Metrics.Instrument("resorts", http.HandlerFunc(deps.Handlers.Resort.ListAllResorts)))
	mux.Handle("POST /api/alerts",
		deps.Metrics.Instrument("alerts_create", write(deps.Handlers.Alert.CreateAlert)))
	mux.Handle("GET /api/user/alerts",
		deps.Metrics.Instrument("user_alerts", middleware.NoStore(http.HandlerFunc(deps.Handlers.Alert.GetUserAlerts))))
	mux.Handle("DELETE /api/user/alerts/delete",
		deps.Metrics.Instrument("user_alerts_delete", write(deps.Handlers.Alert.DeleteUserAlert)))
	mux.Handle("DELETE /api/user/alerts/delete-all",
		deps.Metrics.Instrument("user_alerts_delete_all", write(deps.Handlers.Alert.DeleteAllUserAlerts)))
	mux.Handle("POST /api/contact",
		deps.Metrics.Instrument("contact", write(deps.Handlers.Contact.HandleContact)))

	readLimiter := middleware.NewRateLimiter(middleware.RateLimitConfig{
		RequestsPerSecond: deps.Config.RatePerSecond,
		Burst:             deps.Config.RateBurst,
	}, deps.Config.TrustProxyHeaders)

	handler := middleware.Chain(mux,
		middleware.RequestID,
		middleware.Recover(deps.Logger),
		middleware.Logging(deps.Logger),
		middleware.SecurityHeaders(deps.Config.IsProduction()),
		middleware.CORS(middleware.NewCORSConfig(deps.Config.AllowedOrigins, true)),
		readLimiter.Middleware,
		middleware.MaxBytes(deps.Config.MaxBodyBytes),
	)

	cleanup := func() {
		readLimiter.Close()
		writeLimiter.Close()
	}

	return handler, cleanup
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// readyHandler reports whether the service can actually serve traffic.
func readyHandler(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), readinessTimeout)
		defer cancel()

		if database == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{
				"status": "unavailable",
				"reason": "no database configured",
			})

			return
		}

		if err := database.PingContext(ctx); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{
				"status": "unavailable",
				"reason": "database unreachable",
			})

			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
