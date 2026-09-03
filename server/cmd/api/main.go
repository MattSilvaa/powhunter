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

	"github.com/MattSilvaa/powhunter/internal/config"
	"github.com/MattSilvaa/powhunter/internal/handlers"
	"github.com/MattSilvaa/powhunter/internal/metrics"
	"github.com/MattSilvaa/powhunter/internal/server"
)

// shutdownTimeout bounds how long in-flight requests may finish on shutdown.
const shutdownTimeout = 10 * time.Second

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

	h, err := handlers.NewHandlers()
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
	})
	defer cleanup()

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
