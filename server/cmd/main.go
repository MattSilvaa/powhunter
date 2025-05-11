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
	"github.com/MattSilvaa/powhunter/internal/notify"
	"github.com/MattSilvaa/powhunter/internal/scheduler"
	"github.com/MattSilvaa/powhunter/internal/weather"

	_ "github.com/lib/pq"
)

func main() {
	h, err := handlers.NewHandlers()
	if err != nil {
		log.Fatalf("Failed to initialize handlers: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/resorts", h.Resort.ListAllResorts)
	mux.HandleFunc("/api/alerts", h.Alert.CreateAlert)

	handler := corsMiddleware(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	weatherClient := weather.NewOpenMeteoClient()

	var twilioClient notify.NotificationService

	twilioAccountSID := os.Getenv("TWILIO_ACCOUNT_SID")
	twilioAuthToken := os.Getenv("TWILIO_AUTH_TOKEN")
	twilioFromNumber := os.Getenv("TWILIO_FROM_NUMBER", )

	if twilioAccountSID != "" && twilioAuthToken != "" && twilioFromNumber != "" {
		twilioClient = notify.NewTwilioClient(
			twilioAccountSID,
			twilioAuthToken,
			twilioFromNumber,
		)
		log.Println("Twilio client initialized")
	} else {
		log.Println("Twilio credentials not found. SMS notifications will not be sent.")
	}

	var forecastScheduler scheduler.ForecastSchedulerService

	if twilioClient != nil {
		forecastScheduler = scheduler.NewForecastScheduler(
			h.Store(),
			weatherClient,
			twilioClient,
			12*time.Hour,
		)

		forecastScheduler.Start()
		log.Println("Forecast scheduler started")
	} else {
		log.Println("Skipping forecast scheduler initialization due to missing Twilio credentials")
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
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if forecastScheduler != nil {
		forecastScheduler.Stop()
		log.Println("Forecast scheduler stopped")
	}

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited gracefully")
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowedOrigin := "*"
		if os.Getenv("ENVIRONMENT") == "production" {
			allowedOrigin = "https://powhunter.onrender.com"
		}
		origin := r.Header.Get("Origin")
		if origin == allowedOrigin || allowedOrigin == "*" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
