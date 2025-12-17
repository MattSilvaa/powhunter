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

	"github.com/MattSilvaa/powhunter/internal/auth"
	"github.com/MattSilvaa/powhunter/internal/handlers"
)

func main() {
	h, err := handlers.NewHandlers()
	if err != nil {
		log.Fatalf("Failed to initialize handlers: %v", err)
	}

	// Initialize authentication
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "your-secret-key" // Default for development - change in production
	}

	authService := auth.NewAuthService(jwtSecret)
	authHandlers := auth.NewAuthHandlers(h.Store().Queries(), authService)
	authMiddleware := auth.NewAuthMiddleware(authService, h.Store().Queries())

	mux := http.NewServeMux()
	
	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Authentication routes (no auth required)
	mux.HandleFunc("/api/auth/magic-link", authHandlers.SendMagicLink)
	mux.HandleFunc("/api/auth/verify", authHandlers.VerifyMagicLink)

	// Public routes (no auth required)
	mux.HandleFunc("/api/resorts", h.Resort.ListAllResorts)
	mux.HandleFunc("/api/contact", h.Contact.HandleContact)

	// Protected routes (auth required)
	mux.Handle("/api/alerts", authMiddleware.RequireAuth(http.HandlerFunc(h.Alert.CreateAlert)))
	mux.Handle("/api/user/alerts", authMiddleware.RequireAuth(http.HandlerFunc(h.Alert.GetUserAlerts)))
	mux.Handle("/api/user/alerts/delete", authMiddleware.RequireAuth(http.HandlerFunc(h.Alert.DeleteUserAlert)))
	mux.Handle("/api/user/alerts/delete-all", authMiddleware.RequireAuth(http.HandlerFunc(h.Alert.DeleteAllUserAlerts)))
	mux.Handle("/api/auth/logout", authMiddleware.RequireAuth(http.HandlerFunc(authHandlers.Logout)))

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
			allowedOrigin = "https://powhunter.app"
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
