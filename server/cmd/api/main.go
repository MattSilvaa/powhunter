package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MattSilvaa/powhunter/internal/auth"
	"github.com/MattSilvaa/powhunter/internal/config"
	"github.com/MattSilvaa/powhunter/internal/handlers"
	"github.com/MattSilvaa/powhunter/internal/metrics"
	"github.com/MattSilvaa/powhunter/internal/server"
)

// shutdownTimeout bounds how long in-flight requests may finish on shutdown.
const shutdownTimeout = 10 * time.Second

// reapInterval is how often expired sessions and login tokens are swept. Expiry
// is already enforced on read, so this only keeps the tables from growing.
const reapInterval = time.Hour

// reapTimeout bounds one sweep.
const reapTimeout = 30 * time.Second

// reapExpiredCredentials periodically deletes expired sessions and login
// tokens. A failed sweep is logged and retried on the next tick rather than
// taking the server down: stale rows are untidy, not dangerous.
func reapExpiredCredentials(ctx context.Context, service *auth.Service, logger *slog.Logger) {
	ticker := time.NewTicker(reapInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sweepCtx, cancel := context.WithTimeout(ctx, reapTimeout)

			if err := service.Reap(sweepCtx); err != nil {
				logger.ErrorContext(sweepCtx, "failed to reap expired credentials", "error", err)
			}

			cancel()
		}
	}
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	if err := run(logger); err != nil {
		logger.Error("server exited with error", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	h, err := handlers.NewHandlers(cfg, logger)
	if err != nil {
		return err
	}

	registry := metrics.NewRegistry()
	registry.RegisterDBStats(h.Store().DB())

	handler, cleanup := server.New(server.Deps{
		Config:   cfg,
		Handlers: h,
		Metrics:  registry,
		Logger:   logger,
		DB:       h.Store().DB(),
		Auth:     h.AuthService(),
	})
	defer cleanup()

	reaperCtx, stopReaper := context.WithCancel(context.Background())
	defer stopReaper()

	go reapExpiredCredentials(reaperCtx, h.AuthService(), logger)

	httpServer := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadTimeout:       cfg.ReadTimeout,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	serverErrors := make(chan error, 1)

	go func() {
		logger.Info("server starting",
			"addr", httpServer.Addr,
			"environment", cfg.Environment,
			"allowed_origins", cfg.AllowedOrigins,
		)

		if listenErr := httpServer.ListenAndServe(); listenErr != nil &&
			!errors.Is(listenErr, http.ErrServerClosed) {
			serverErrors <- listenErr
		}
	}()

	select {
	case listenErr := <-serverErrors:
		return listenErr
	case <-stop:
		logger.Info("shutting down api server")
	}

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if shutdownErr := httpServer.Shutdown(ctx); shutdownErr != nil {
		return shutdownErr
	}

	logger.Info("server exited gracefully")

	return nil
}
