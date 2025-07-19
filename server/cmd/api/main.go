package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MattSilvaa/powhunter/internal/handlers"
)

func main() {
	h, err := handlers.NewHandlers()
	if err != nil {
		log.Fatalf("Failed to initialize handlers: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	mux.HandleFunc("/api/resorts", h.Resort.ListAllResorts)
	mux.HandleFunc("/api/alerts", h.Alert.CreateAlert)
	mux.HandleFunc("/api/user/alerts", h.Alert.GetUserAlerts)
	mux.HandleFunc("/api/user/alerts/delete", h.Alert.DeleteUserAlert)
	mux.HandleFunc("/api/user/alerts/delete-all", h.Alert.DeleteAllUserAlerts)

	handler := corsMiddleware(mux)

	server := &http.Server{
		Addr:         ":8080",
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Server starting on %s", server.Addr)

		serverErr := server.ListenAndServe()
		if serverErr != nil && !errors.Is(serverErr, http.ErrServerClosed) {
			log.Fatalf("Server failed to start: %v", serverErr)
		}
	}()

	<-stop
	log.Println("Shutting down API server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited gracefully")
}

func corsMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowedOrigin := "*"

		if os.Getenv("ENVIRONMENT") == "production" {
			allowedOrigin = "https://www.powhunter.app"
		}
		origin := r.Header.Get("Origin")

		if allowedOrigin == "*" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			if origin == allowedOrigin {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().
			Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		h.ServeHTTP(w, r)
	})
}
